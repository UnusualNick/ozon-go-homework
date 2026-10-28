package cart

import (
	"context"
	"testing"
)

func BenchmarkAddToCart(b *testing.B) {
	service := NewService(fakeProducts{}, NewStore())
	ctx := context.Background()
	b.ResetTimer()
	for range b.N {
		if err := service.Add(ctx, 1, 10, 1); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetCart(b *testing.B) {
	store := NewStore()
	for _, sku := range []int64{10, 20, 30, 40} {
		if err := store.Add(1, Product{SKU: sku, Name: "item", Price: uint32(sku)}, 2); err != nil {
			b.Fatal(err)
		}
	}
	service := NewService(fakeProducts{}, store)
	ctx := context.Background()
	b.ResetTimer()
	for range b.N {
		if _, err := service.Get(ctx, 1); err != nil {
			b.Fatal(err)
		}
	}
}
