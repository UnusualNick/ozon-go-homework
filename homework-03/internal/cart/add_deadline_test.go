package cart

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddSetsProductLookupDeadline(t *testing.T) {
	t.Parallel()
	seenDeadline := false
	client := productFunc(func(ctx context.Context, sku int64) (Product, error) {
		_, seenDeadline = ctx.Deadline()
		return Product{SKU: sku, Name: "item", Price: 3}, nil
	})
	store := NewStore()
	service := NewService(client, store)
	require.NoError(t, service.Add(context.Background(), 1, 7, 1))
	require.True(t, seenDeadline, "outgoing product lookup needs a deadline")
	cart, err := store.Get(1)
	require.NoError(t, err)
	require.Equal(t, int64(7), cart.Items[0].SKU)
}
