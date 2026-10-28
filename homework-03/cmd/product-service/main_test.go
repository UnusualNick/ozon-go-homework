package main

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/UnusualNick/ozon-go-homework/homework-03/internal/product"
	productv1 "github.com/UnusualNick/ozon-go-homework/homework-03/pkg/product/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestServeProductOnListener(t *testing.T) {
	t.Parallel()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	done := make(chan error, 1)
	go func() {
		done <- serveProduct(listener, product.New([]product.Product{{SKU: 7, Name: "item", Price: 19}}))
	}()
	t.Cleanup(func() {
		_ = listener.Close()
		<-done
	})

	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	item, err := productv1.NewProductServiceClient(conn).GetProduct(ctx, &productv1.GetProductRequest{Sku: 7})
	require.NoError(t, err)
	require.Equal(t, "item", item.GetName())
	require.Equal(t, int32(19), item.GetPrice())
}
