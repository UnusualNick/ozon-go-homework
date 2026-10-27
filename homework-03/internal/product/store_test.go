package product

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	productv1 "github.com/UnusualNick/ozon-go-homework/homework-03/pkg/product/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetProduct(t *testing.T) {
	t.Parallel()

	products := []Product{
		{SKU: 1, Name: "first", Price: 0},
		{SKU: math.MaxInt64, Name: "last", Price: math.MaxInt32},
	}
	tests := []struct {
		name     string
		request  *productv1.GetProductRequest
		want     *productv1.GetProductResponse
		wantCode codes.Code
	}{
		{name: "nil request", wantCode: codes.InvalidArgument},
		{name: "zero SKU", request: &productv1.GetProductRequest{}, wantCode: codes.InvalidArgument},
		{name: "negative SKU", request: &productv1.GetProductRequest{Sku: -1}, wantCode: codes.InvalidArgument},
		{name: "missing SKU", request: &productv1.GetProductRequest{Sku: 2}, wantCode: codes.NotFound},
		{name: "first product and zero price", request: &productv1.GetProductRequest{Sku: 1}, want: &productv1.GetProductResponse{Sku: 1, Name: "first", Price: 0}},
		{name: "largest SKU and price", request: &productv1.GetProductRequest{Sku: math.MaxInt64}, want: &productv1.GetProductResponse{Sku: math.MaxInt64, Name: "last", Price: math.MaxInt32}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService(New(products))
			response, err := service.GetProduct(context.Background(), tt.request)
			if tt.wantCode != codes.OK {
				require.Equal(t, tt.wantCode, status.Code(err))
				require.Nil(t, response)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, response)
		})
	}
}

func TestListProducts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		request   *productv1.ListProductsRequest
		count     int
		wantSKUs  []int64
		wantToken string
		wantCode  codes.Code
	}{
		{name: "nil request defaults to fifty", count: 120, wantSKUs: skuRange(1, 50), wantToken: "50"},
		{name: "zero page size defaults to fifty", request: &productv1.ListProductsRequest{PageSize: 0}, count: 120, wantSKUs: skuRange(1, 50), wantToken: "50"},
		{name: "default size on later page", request: &productv1.ListProductsRequest{PageToken: "70"}, count: 120, wantSKUs: skuRange(71, 120)},
		{name: "first single-item page", request: &productv1.ListProductsRequest{PageSize: 1}, count: 120, wantSKUs: []int64{1}, wantToken: "1"},
		{name: "middle page", request: &productv1.ListProductsRequest{PageSize: 3, PageToken: "9"}, count: 120, wantSKUs: []int64{10, 11, 12}, wantToken: "12"},
		{name: "last item", request: &productv1.ListProductsRequest{PageSize: 1, PageToken: "119"}, count: 120, wantSKUs: []int64{120}},
		{name: "partial final page", request: &productv1.ListProductsRequest{PageSize: 10, PageToken: "115"}, count: 120, wantSKUs: skuRange(116, 120)},
		{name: "exact end offset", request: &productv1.ListProductsRequest{PageSize: 1, PageToken: "120"}, count: 120},
		{name: "past end offset", request: &productv1.ListProductsRequest{PageSize: 1, PageToken: "121"}, count: 120},
		{name: "maximum page size", request: &productv1.ListProductsRequest{PageSize: 100}, count: 120, wantSKUs: skuRange(1, 100), wantToken: "100"},
		{name: "maximum page size from next token", request: &productv1.ListProductsRequest{PageSize: 100, PageToken: "100"}, count: 120, wantSKUs: skuRange(101, 120)},
		{name: "empty catalog", request: &productv1.ListProductsRequest{PageSize: 1}, count: 0},
		{name: "negative page size", request: &productv1.ListProductsRequest{PageSize: -1}, count: 120, wantCode: codes.InvalidArgument},
		{name: "page size above maximum", request: &productv1.ListProductsRequest{PageSize: 101}, count: 120, wantCode: codes.InvalidArgument},
		{name: "negative page token", request: &productv1.ListProductsRequest{PageToken: "-1"}, count: 120, wantCode: codes.InvalidArgument},
		{name: "non-numeric page token", request: &productv1.ListProductsRequest{PageToken: "next"}, count: 120, wantCode: codes.InvalidArgument},
		{name: "fractional page token", request: &productv1.ListProductsRequest{PageToken: "1.5"}, count: 120, wantCode: codes.InvalidArgument},
		{name: "overflowed page token", request: &productv1.ListProductsRequest{PageToken: "999999999999999999999999999999999"}, count: 120, wantCode: codes.InvalidArgument},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService(New(testProducts(tt.count)))
			response, err := service.ListProducts(context.Background(), tt.request)
			if tt.wantCode != codes.OK {
				require.Equal(t, tt.wantCode, status.Code(err))
				require.Nil(t, response)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, response)
			require.NotNil(t, response.Products)
			require.Equal(t, tt.wantToken, response.NextPageToken)
			require.Len(t, response.Products, len(tt.wantSKUs))
			for i, sku := range tt.wantSKUs {
				require.Equal(t, &productv1.Product{Sku: sku, Name: fmt.Sprintf("product %d", sku), Price: int32(sku * 10)}, response.Products[i])
			}
		})
	}
}

func TestLoad(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		json    string
		missing bool
		want    []Product
		wantErr bool
	}{
		{name: "sorted numeric SKUs and prices", json: `{"20":{"name":"second","price":22},"10":{"name":"first","price":0}}`, want: []Product{{SKU: 10, Name: "first", Price: 0}, {SKU: 20, Name: "second", Price: 22}}},
		{name: "largest supported price and SKU", json: `{"9223372036854775807":{"name":"max","price":2147483647}}`, want: []Product{{SKU: math.MaxInt64, Name: "max", Price: math.MaxInt32}}},
		{name: "empty object", json: `{}`, want: []Product{}},
		{name: "missing file", missing: true, wantErr: true},
		{name: "empty file", json: ``, wantErr: true},
		{name: "malformed JSON", json: `{"1":`, wantErr: true},
		{name: "trailing JSON value", json: `{"1":{"name":"one","price":1}} {}`, wantErr: true},
		{name: "malformed trailing JSON", json: `{} invalid`, wantErr: true},
		{name: "non-numeric SKU", json: `{"one":{"name":"one","price":1}}`, wantErr: true},
		{name: "zero SKU", json: `{"0":{"name":"zero","price":1}}`, wantErr: true},
		{name: "negative SKU", json: `{"-1":{"name":"negative","price":1}}`, wantErr: true},
		{name: "overflowed SKU", json: `{"9223372036854775808":{"name":"large","price":1}}`, wantErr: true},
		{name: "empty name", json: `{"1":{"name":"","price":1}}`, wantErr: true},
		{name: "negative price", json: `{"1":{"name":"one","price":-1}}`, wantErr: true},
		{name: "price above int32 maximum", json: `{"1":{"name":"one","price":2147483648}}`, wantErr: true},
		{name: "non-numeric price", json: `{"1":{"name":"one","price":"one"}}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "products.json")
			if !tt.missing {
				require.NoError(t, os.WriteFile(path, []byte(tt.json), 0o600))
			}
			store, err := Load(path)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, store)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, store)
			items, total := store.List(0, len(tt.want)+1)
			require.Equal(t, len(tt.want), total)
			require.Equal(t, tt.want, items)
			for _, want := range tt.want {
				item, err := store.Get(want.SKU)
				require.NoError(t, err)
				require.Equal(t, want, item)
			}
			_, err = store.Get(-1)
			require.ErrorIs(t, err, ErrNotFound)
		})
	}
}

func BenchmarkGetProduct(b *testing.B) {
	service := NewService(New([]Product{{SKU: 1, Name: "one", Price: 10}}))
	ctx := context.Background()
	request := &productv1.GetProductRequest{Sku: 1}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := service.GetProduct(ctx, request); err != nil {
			b.Fatal(err)
		}
	}
}

func testProducts(count int) []Product {
	products := make([]Product, 0, count)
	for sku := count; sku > 0; sku-- {
		products = append(products, Product{SKU: int64(sku), Name: fmt.Sprintf("product %d", sku), Price: uint32(sku * 10)})
	}
	return products
}

func skuRange(first, last int64) []int64 {
	skus := make([]int64, 0, last-first+1)
	for sku := first; sku <= last; sku++ {
		skus = append(skus, sku)
	}
	return skus
}
