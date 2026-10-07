package cart

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

func TestHTTPAddAndGet(t *testing.T) {
	handler := NewHTTPHandler(NewService(fakeProducts{}, NewStore()))

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/user/7/cart/20", strings.NewReader(`{"count":2}`))
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Body.Len() != 0 {
		t.Fatalf("POST status=%d body=%q", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/user/7/cart", nil)
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("GET status=%d body=%q", response.Code, response.Body.String())
	}
	if body := response.Body.String(); !strings.Contains(body, `"sku":20`) || !strings.Contains(body, `"total_price":40`) {
		t.Fatalf("GET body=%q", body)
	}
}

func TestHTTPErrors(t *testing.T) {
	handler := NewHTTPHandler(NewService(fakeProducts{}, NewStore()))
	tests := []struct {
		method string
		path   string
		body   string
		status int
	}{
		{http.MethodPost, "/user/0/cart/1", `{"count":1}`, http.StatusBadRequest},
		{http.MethodPost, "/user/1/cart/0", `{"count":1}`, http.StatusBadRequest},
		{http.MethodPost, "/user/1/cart/1", `{"count":0}`, http.StatusBadRequest},
		{http.MethodPost, "/user/1/cart/999", `{"count":1}`, http.StatusPreconditionFailed},
		{http.MethodPost, "/user/1/cart/500", `{"count":1}`, http.StatusInternalServerError},
		{http.MethodGet, "/user/0/cart", "", http.StatusBadRequest},
		{http.MethodGet, "/user/42/cart", "", http.StatusNotFound},
	}
	for _, test := range tests {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
		handler.ServeHTTP(response, request)
		if response.Code != test.status {
			t.Errorf("%s %s status=%d, want %d, body=%q", test.method, test.path, response.Code, test.status, response.Body.String())
		}
	}
}
