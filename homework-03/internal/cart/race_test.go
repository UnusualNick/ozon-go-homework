//go:build race

package cart

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConcurrentServiceAdds(t *testing.T) {
	t.Parallel()
	const workers = 128
	service := NewService(productFunc(func(_ context.Context, sku int64) (Product, error) {
		return Product{SKU: sku, Name: "item", Price: 2}, nil
	}), NewStore())
	var group sync.WaitGroup
	errors := make(chan error, workers)
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			errors <- service.Add(context.Background(), 1, 10, 1)
		}()
	}
	group.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}
	cart, err := service.Get(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, uint32(workers), cart.Items[0].Count)
	require.Equal(t, uint32(workers*2), cart.TotalPrice)
}
