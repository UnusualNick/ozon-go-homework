package product

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	productv1 "github.com/UnusualNick/ozon-go-homework/homework-02/pkg/product/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestLoadAndGet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "products.json")
	if err := os.WriteFile(path, []byte(`{"20":{"name":"second","price":22},"10":{"name":"first","price":11}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	item, err := store.Get(10)
	if err != nil || item.Name != "first" || item.Price != 11 {
		t.Fatalf("Get(10) = %#v, %v", item, err)
	}
	if _, err := store.Get(999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(999) error = %v", err)
	}
	items, total := store.List(0, 10)
	if total != 2 || len(items) != 2 || items[0].SKU != 10 || items[1].SKU != 20 {
		t.Fatalf("List() = %#v, total=%d", items, total)
	}
}

func TestServiceErrorsAndPagination(t *testing.T) {
	service := NewService(New([]Product{{SKU: 2, Name: "two", Price: 20}, {SKU: 1, Name: "one", Price: 10}}))
	if _, err := service.GetProduct(context.Background(), &productv1.GetProductRequest{}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("zero SKU code = %s", status.Code(err))
	}
	if _, err := service.GetProduct(context.Background(), &productv1.GetProductRequest{Sku: 999}); status.Code(err) != codes.NotFound {
		t.Fatalf("missing SKU code = %s", status.Code(err))
	}
	first, err := service.ListProducts(context.Background(), &productv1.ListProductsRequest{PageSize: 1})
	if err != nil || len(first.Products) != 1 || first.Products[0].Sku != 1 || first.NextPageToken == "" {
		t.Fatalf("first page = %#v, %v", first, err)
	}
	second, err := service.ListProducts(context.Background(), &productv1.ListProductsRequest{PageSize: 1, PageToken: first.NextPageToken})
	if err != nil || len(second.Products) != 1 || second.Products[0].Sku != 2 || second.NextPageToken != "" {
		t.Fatalf("second page = %#v, %v", second, err)
	}
}
