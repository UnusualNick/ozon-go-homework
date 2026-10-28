package cart

import (
	"context"
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type productFunc func(context.Context, int64) (Product, error)

func (f productFunc) GetProduct(ctx context.Context, sku int64) (Product, error) {
	return f(ctx, sku)
}

func TestServiceGetCases(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name       string
		productErr error
		freshPrice uint32
		wantErr    error
	}{
		{name: "fresh prices", freshPrice: 7},
		{name: "missing product", productErr: ErrProductNotFound, wantErr: ErrProductNotFound},
		{name: "grpc not found", productErr: status.Error(codes.NotFound, "gone"), wantErr: ErrProductNotFound},
		{name: "dependency error", productErr: errors.New("offline"), wantErr: errors.New("offline")},
		{name: "total overflow", freshPrice: math.MaxUint32, wantErr: ErrOverflow},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var calls atomic.Int32
			client := productFunc(func(_ context.Context, sku int64) (Product, error) {
				calls.Add(1)
				if sku == 20 && test.productErr != nil {
					return Product{}, test.productErr
				}
				return Product{SKU: sku, Name: "current", Price: test.freshPrice}, nil
			})
			store := NewStore()
			require.NoError(t, store.Add(1, Product{SKU: 20, Name: "old", Price: 1}, 1))
			require.NoError(t, store.Add(1, Product{SKU: 10, Name: "old", Price: 1}, 1))
			cart, err := NewService(client, store).Get(context.Background(), 1)
			require.Equal(t, int32(2), calls.Load())
			if test.wantErr != nil {
				if test.name == "dependency error" {
					require.ErrorContains(t, err, "offline")
				} else {
					require.ErrorIs(t, err, test.wantErr)
				}
				require.Empty(t, cart.Items)
				return
			}
			require.NoError(t, err)
			require.Equal(t, []Item{
				{Product: Product{SKU: 10, Name: "current", Price: 7}, Count: 1},
				{Product: Product{SKU: 20, Name: "current", Price: 7}, Count: 1},
			}, cart.Items)
			require.Equal(t, uint32(14), cart.TotalPrice)
		})
	}
}

func TestServiceGetAbsentNeverCallsProduct(t *testing.T) {
	t.Parallel()
	client := productFunc(func(context.Context, int64) (Product, error) {
		t.Error("unexpected product lookup")
		return Product{}, nil
	})
	_, err := NewService(client, NewStore()).Get(context.Background(), 1)
	require.ErrorIs(t, err, ErrCartNotFound)
}

func TestServiceGetFetchesConcurrently(t *testing.T) {
	t.Parallel()
	started := make(chan int64, 2)
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	client := productFunc(func(ctx context.Context, sku int64) (Product, error) {
		started <- sku
		select {
		case <-release:
			return Product{SKU: sku, Price: 2}, nil
		case <-ctx.Done():
			return Product{}, ctx.Err()
		}
	})
	store := NewStore()
	require.NoError(t, store.Add(1, Product{SKU: 2}, 1))
	require.NoError(t, store.Add(1, Product{SKU: 1}, 1))
	done := make(chan error, 1)
	go func() {
		_, err := NewService(client, store).Get(context.Background(), 1)
		done <- err
	}()
	for range 2 {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("product calls did not run concurrently")
		}
	}
	unblock()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Get did not finish")
	}
}

func TestServiceGetCancellation(t *testing.T) {
	t.Parallel()
	started := make(chan context.Context, 1)
	client := productFunc(func(ctx context.Context, _ int64) (Product, error) {
		started <- ctx
		<-ctx.Done()
		return Product{}, ctx.Err()
	})
	store := NewStore()
	require.NoError(t, store.Add(1, Product{SKU: 1}, 1))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := NewService(client, store).Get(ctx, 1)
		done <- err
	}()
	select {
	case child := <-started:
		_, hasDeadline := child.Deadline()
		require.True(t, hasDeadline, "outgoing lookup needs a batch deadline")
	case <-time.After(time.Second):
		t.Fatal("product call did not start")
	}
	cancel()
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("Get did not respect cancellation")
	}
}

func TestServiceAdd(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name       string
		productErr error
		wantErr    error
	}{
		{name: "success"},
		{name: "missing", productErr: ErrProductNotFound, wantErr: ErrProductNotFound},
		{name: "overflow", wantErr: ErrOverflow},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			store := NewStore()
			if test.name == "overflow" {
				require.NoError(t, store.Add(1, Product{SKU: 2}, math.MaxUint32))
			}
			service := NewService(productFunc(func(_ context.Context, sku int64) (Product, error) {
				return Product{SKU: sku, Price: 3}, test.productErr
			}), store)
			err := service.Add(context.Background(), 1, 2, 1)
			if test.wantErr != nil {
				require.ErrorIs(t, err, test.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
