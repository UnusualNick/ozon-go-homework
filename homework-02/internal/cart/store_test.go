package cart

import (
	"errors"
	"math"
	"sync"
	"testing"
)

func TestStoreAggregatesAndSorts(t *testing.T) {
	store := NewStore()
	if err := store.Add(1, Product{SKU: 20, Name: "second", Price: 3}, 2); err != nil {
		t.Fatal(err)
	}
	if err := store.Add(1, Product{SKU: 10, Name: "first", Price: 5}, 1); err != nil {
		t.Fatal(err)
	}
	if err := store.Add(1, Product{SKU: 20, Name: "second", Price: 3}, 4); err != nil {
		t.Fatal(err)
	}
	cart, err := store.Get(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(cart.Items) != 2 || cart.Items[0].SKU != 10 || cart.Items[1].SKU != 20 {
		t.Fatalf("unexpected order: %#v", cart.Items)
	}
	if cart.Items[1].Count != 6 || cart.TotalPrice != 23 {
		t.Fatalf("unexpected cart: %#v", cart)
	}
}

func TestStoreIsSafeForConcurrentAdds(t *testing.T) {
	store := NewStore()
	var group sync.WaitGroup
	for range 100 {
		group.Add(1)
		go func() {
			defer group.Done()
			if err := store.Add(1, Product{SKU: 10, Name: "item", Price: 1}, 1); err != nil {
				t.Errorf("Add() error = %v", err)
			}
		}()
	}
	group.Wait()
	cart, err := store.Get(1)
	if err != nil || cart.Items[0].Count != 100 {
		t.Fatalf("Get() = %#v, %v", cart, err)
	}
}

func TestStoreRejectsCountOverflow(t *testing.T) {
	store := NewStore()
	if err := store.Add(1, Product{SKU: 1, Name: "item", Price: 1}, math.MaxUint32); err != nil {
		t.Fatal(err)
	}
	if err := store.Add(1, Product{SKU: 1, Name: "item", Price: 1}, 1); !errors.Is(err, ErrOverflow) {
		t.Fatalf("Add() error = %v", err)
	}
}
