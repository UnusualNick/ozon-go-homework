//go:build api

package test

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"io"
	"math"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	productv1 "github.com/UnusualNick/ozon-go-homework/homework-03/pkg/product/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const requestTimeout = 5 * time.Second

func TestProductAPI(t *testing.T) {
	t.Parallel()

	t.Run("real product", func(t *testing.T) {
		t.Parallel()
		client, conn, err := connectProduct()
		require.NoError(t, err)
		t.Cleanup(func() { _ = conn.Close() })
		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		defer cancel()

		page, err := client.ListProducts(ctx, &productv1.ListProductsRequest{PageSize: 1})
		require.NoError(t, err)
		require.Len(t, page.GetProducts(), 1)
		listed := page.GetProducts()[0]
		got, err := client.GetProduct(ctx, &productv1.GetProductRequest{Sku: listed.GetSku()})
		require.NoError(t, err)
		require.Equal(t, listed.GetSku(), got.GetSku())
		require.Equal(t, listed.GetName(), got.GetName())
		require.Equal(t, listed.GetPrice(), got.GetPrice())
	})

	t.Run("invalid sku", func(t *testing.T) {
		t.Parallel()
		client, conn, err := connectProduct()
		require.NoError(t, err)
		t.Cleanup(func() { _ = conn.Close() })
		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		defer cancel()
		_, err = client.GetProduct(ctx, &productv1.GetProductRequest{Sku: 0})
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("missing product", func(t *testing.T) {
		t.Parallel()
		client, conn, err := connectProduct()
		require.NoError(t, err)
		t.Cleanup(func() { _ = conn.Close() })
		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		defer cancel()
		_, err = client.GetProduct(ctx, &productv1.GetProductRequest{Sku: math.MaxInt64})
		require.Equal(t, codes.NotFound, status.Code(err))
	})

	t.Run("invalid page token", func(t *testing.T) {
		t.Parallel()
		client, conn, err := connectProduct()
		require.NoError(t, err)
		t.Cleanup(func() { _ = conn.Close() })
		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		defer cancel()
		_, err = client.ListProducts(ctx, &productv1.ListProductsRequest{PageToken: "not-a-number"})
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	})
}

func TestCartAPI(t *testing.T) {
	t.Parallel()

	t.Run("cart through both servers", func(t *testing.T) {
		t.Parallel()
		products, conn, err := connectProduct()
		require.NoError(t, err)
		t.Cleanup(func() { _ = conn.Close() })
		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		defer cancel()
		page, err := products.ListProducts(ctx, &productv1.ListProductsRequest{PageSize: 2})
		require.NoError(t, err)
		require.Len(t, page.GetProducts(), 2)
		first, second := page.GetProducts()[0], page.GetProducts()[1]
		require.Less(t, first.GetSku(), second.GetSku())

		userID := uniqueUserID(t)
		client := &http.Client{Timeout: requestTimeout}
		for _, sku := range []int64{second.GetSku(), first.GetSku()} {
			path := "/user/" + strconv.FormatInt(userID, 10) + "/cart/" + strconv.FormatInt(sku, 10)
			code, body := doHTTP(t, client, http.MethodPost, path, `{"count":1}`)
			require.Equal(t, http.StatusOK, code, string(body))
			require.Empty(t, body)
		}

		code, body := doHTTP(t, client, http.MethodGet, "/user/"+strconv.FormatInt(userID, 10)+"/cart", "")
		require.Equal(t, http.StatusOK, code, string(body))
		var response struct {
			Items []struct {
				SKU   int64  `json:"sku"`
				Name  string `json:"name"`
				Count uint32 `json:"count"`
				Price uint32 `json:"price"`
			} `json:"items"`
			TotalPrice uint32 `json:"total_price"`
		}
		require.NoError(t, json.Unmarshal(body, &response))
		require.Len(t, response.Items, 2)
		require.Equal(t, first.GetSku(), response.Items[0].SKU)
		require.Equal(t, second.GetSku(), response.Items[1].SKU)
		require.Equal(t, first.GetName(), response.Items[0].Name)
		require.Equal(t, uint32(1), response.Items[0].Count)
		require.Equal(t, uint32(first.GetPrice()), response.Items[0].Price)
		require.Equal(t, uint32(first.GetPrice()+second.GetPrice()), response.TotalPrice)
	})

	t.Run("HTTP error codes", func(t *testing.T) {
		t.Parallel()
		client := &http.Client{Timeout: requestTimeout}
		userID := strconv.FormatInt(uniqueUserID(t), 10)
		cases := []struct {
			name, method, path, payload string
			want                        int
		}{
			{name: "missing cart", method: http.MethodGet, path: "/user/" + userID + "/cart", want: http.StatusNotFound},
			{name: "invalid user", method: http.MethodGet, path: "/user/0/cart", want: http.StatusBadRequest},
			{name: "unknown sku", method: http.MethodPost, path: "/user/" + userID + "/cart/" + strconv.FormatInt(math.MaxInt64, 10), payload: `{"count":1}`, want: http.StatusPreconditionFailed},
			{name: "invalid count", method: http.MethodPost, path: "/user/" + userID + "/cart/1", payload: `{"count":0}`, want: http.StatusBadRequest},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				code, body := doHTTP(t, client, tc.method, tc.path, tc.payload)
				require.Equal(t, tc.want, code, string(body))
			})
		}
	})
}

func uniqueUserID(t *testing.T) int64 {
	t.Helper()
	value, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64-1))
	require.NoError(t, err)
	return value.Int64() + 1
}

func doHTTP(t *testing.T, client *http.Client, method, path, payload string) (int, []byte) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, method, cartBaseURL()+path, strings.NewReader(payload))
	require.NoError(t, err)
	if payload != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := client.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	require.NoError(t, err)
	return response.StatusCode, body
}
