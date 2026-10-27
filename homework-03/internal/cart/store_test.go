package cart

import (
	"math"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStoreAggregatesAndSorts(t *testing.T) {
	t.Parallel()
	store := NewStore()
	require.NoError(t, store.Add(1, Product{SKU: 20, Name: "second", Price: 3}, 2))
	require.NoError(t, store.Add(1, Product{SKU: 10, Name: "first", Price: 5}, 1))
	require.NoError(t, store.Add(1, Product{SKU: 20, Name: "second", Price: 3}, 4))
	cart, err := store.Get(1)
	require.NoError(t, err)
	require.Equal(t, []Item{
		{Product: Product{SKU: 10, Name: "first", Price: 5}, Count: 1},
		{Product: Product{SKU: 20, Name: "second", Price: 3}, Count: 6},
	}, cart.Items)
	require.Equal(t, uint32(23), cart.TotalPrice)
}

func TestStoreCases(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name       string
		price      uint32
		first      uint32
		second     uint32
		wantAddErr error
		wantGetErr error
		wantCount  uint32
	}{
		{name: "count overflow", price: 1, first: math.MaxUint32, second: 1, wantAddErr: ErrOverflow, wantCount: math.MaxUint32},
		{name: "total overflow", price: 2, first: math.MaxUint32, wantGetErr: ErrOverflow, wantCount: math.MaxUint32},
		{name: "maximum total", price: 1, first: math.MaxUint32, wantCount: math.MaxUint32},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			store := NewStore()
			require.NoError(t, store.Add(1, Product{SKU: 1, Price: test.price}, test.first))
			if test.second != 0 {
				require.ErrorIs(t, store.Add(1, Product{SKU: 1, Price: test.price}, test.second), test.wantAddErr)
			}
			items, err := store.snapshot(1)
			require.NoError(t, err)
			require.Equal(t, test.wantCount, items[0].Count)
			_, err = store.Get(1)
			if test.wantGetErr != nil {
				require.ErrorIs(t, err, test.wantGetErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStoreMissingAndSnapshot(t *testing.T) {
	t.Parallel()
	store := NewStore()
	_, err := store.Get(1)
	require.ErrorIs(t, err, ErrCartNotFound)
	require.NoError(t, store.Add(1, Product{SKU: 1, Price: 4}, 1))
	items, err := store.snapshot(1)
	require.NoError(t, err)
	require.NoError(t, store.Add(1, Product{SKU: 1, Price: 5}, 1))
	require.Equal(t, uint32(1), items[0].Count)
	require.Equal(t, uint32(4), items[0].Price)
}

func TestStoreIsSafeForConcurrentAdds(t *testing.T) {
	t.Parallel()
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
	require.NoError(t, err)
	require.Equal(t, uint32(100), cart.Items[0].Count)
}
