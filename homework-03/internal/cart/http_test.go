package cart

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeProducts struct{}

func (fakeProducts) GetProduct(_ context.Context, sku int64) (Product, error) {
	if sku == 999 {
		return Product{}, ErrProductNotFound
	}
	if sku == 500 {
		return Product{}, errors.New("dependency unavailable")
	}
	return Product{SKU: sku, Name: "item", Price: uint32(sku)}, nil
}

func serveCart(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(method, path, strings.NewReader(body)))
	return response
}

func TestHTTPAddAndGet(t *testing.T) {
	t.Parallel()
	handler := NewHTTPHandler(NewService(fakeProducts{}, NewStore()))
	require.Equal(t, http.StatusOK, serveCart(handler, http.MethodPost, "/user/7/cart/20", `{"count":2}`).Code)
	require.Equal(t, http.StatusOK, serveCart(handler, http.MethodPost, "/user/7/cart/10", `{"count":3}`).Code)
	response := serveCart(handler, http.MethodGet, "/user/7/cart", "")
	require.Equal(t, http.StatusOK, response.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	require.Equal(t, float64(70), body["total_price"])
	items, ok := body["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 2)
	require.Equal(t, map[string]any{"sku": float64(10), "name": "item", "count": float64(3), "price": float64(10)}, items[0])
	require.Equal(t, map[string]any{"sku": float64(20), "name": "item", "count": float64(2), "price": float64(20)}, items[1])
}

func TestHTTPErrors(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, method, path, body string
		status                   int
	}{
		{"bad user on add", http.MethodPost, "/user/0/cart/1", `{"count":1}`, http.StatusBadRequest},
		{"bad sku", http.MethodPost, "/user/1/cart/0", `{"count":1}`, http.StatusBadRequest},
		{"zero count", http.MethodPost, "/user/1/cart/1", `{"count":0}`, http.StatusBadRequest},
		{"bad json", http.MethodPost, "/user/1/cart/1", `{"count":"1"}`, http.StatusBadRequest},
		{"missing product on add", http.MethodPost, "/user/1/cart/999", `{"count":1}`, http.StatusPreconditionFailed},
		{"dependency on add", http.MethodPost, "/user/1/cart/500", `{"count":1}`, http.StatusInternalServerError},
		{"bad user on get", http.MethodGet, "/user/0/cart", "", http.StatusBadRequest},
		{"absent cart", http.MethodGet, "/user/42/cart", "", http.StatusNotFound},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			handler := NewHTTPHandler(NewService(fakeProducts{}, NewStore()))
			response := serveCart(handler, test.method, test.path, test.body)
			require.Equal(t, test.status, response.Code)
		})
	}
}

func TestHTTPGetDependencyErrorsHaveNoPartialCart(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		failure error
		status  int
	}{
		{"missing product", ErrProductNotFound, http.StatusPreconditionFailed},
		{"grpc missing product", status.Error(codes.NotFound, "gone"), http.StatusPreconditionFailed},
		{"dependency failure", errors.New("temporary failure"), http.StatusInternalServerError},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var fail atomic.Bool
			client := productFunc(func(_ context.Context, sku int64) (Product, error) {
				if sku == 2 && fail.Load() {
					return Product{}, test.failure
				}
				return Product{SKU: sku, Name: "fresh", Price: 3}, nil
			})
			handler := NewHTTPHandler(NewService(client, NewStore()))
			require.Equal(t, http.StatusOK, serveCart(handler, http.MethodPost, "/user/1/cart/1", `{"count":1}`).Code)
			require.Equal(t, http.StatusOK, serveCart(handler, http.MethodPost, "/user/1/cart/2", `{"count":1}`).Code)
			fail.Store(true)
			response := serveCart(handler, http.MethodGet, "/user/1/cart", "")
			require.Equal(t, test.status, response.Code)
			var body map[string]any
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
			require.NotContains(t, body, "items")
			require.NotContains(t, body, "total_price")
		})
	}
}
