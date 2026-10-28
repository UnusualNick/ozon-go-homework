package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/UnusualNick/ozon-go-homework/homework-03/internal/cart"
	"github.com/stretchr/testify/require"
)

type testProducts struct{}

func (testProducts) GetProduct(_ context.Context, sku int64) (cart.Product, error) {
	return cart.Product{SKU: sku, Name: "item", Price: 15}, nil
}

func TestServeCartOnListener(t *testing.T) {
	t.Parallel()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	service := cart.NewService(testProducts{}, cart.NewStore())
	server, stopped := serveCart(listener, service)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, server.Shutdown(ctx))
		require.ErrorIs(t, <-stopped, http.ErrServerClosed)
	})

	client := &http.Client{Timeout: 5 * time.Second}
	baseURL := "http://" + listener.Addr().String()
	post, err := client.Post(baseURL+"/user/1/cart/7", "application/json", strings.NewReader(`{"count":2}`))
	require.NoError(t, err)
	defer post.Body.Close()
	require.Equal(t, http.StatusOK, post.StatusCode)

	response, err := client.Get(baseURL + "/user/1/cart")
	require.NoError(t, err)
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Contains(t, string(body), `"total_price":30`)
}
