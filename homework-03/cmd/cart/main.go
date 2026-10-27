package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/UnusualNick/ozon-go-homework/homework-03/internal/cart"
	productv1 "github.com/UnusualNick/ozon-go-homework/homework-03/pkg/product/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	productAddress := env("PRODUCT_SERVICE_ADDR", "localhost:50051")
	httpAddress := env("CART_HTTP_ADDR", ":8080")

	connection, err := grpc.NewClient(productAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("create product-service client: %v", err)
	}
	defer connection.Close()

	service := cart.NewService(
		cart.NewGRPCProductClient(productv1.NewProductServiceClient(connection)),
		cart.NewStore(),
	)
	listener, err := net.Listen("tcp", httpAddress)
	if err != nil {
		log.Fatalf("listen on %s: %v", httpAddress, err)
	}
	server, stopped := serveCart(listener, service)
	log.Printf("cart listens on %s", httpAddress)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(stop)
	select {
	case <-stop:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("shutdown cart: %v", err)
		}
		if err := <-stopped; err != nil && err != http.ErrServerClosed {
			log.Printf("serve HTTP: %v", err)
		}
	case err := <-stopped:
		if err != nil && err != http.ErrServerClosed {
			log.Printf("serve HTTP: %v", err)
		}
	}
}

func serveCart(listener net.Listener, service *cart.Service) (*http.Server, <-chan error) {
	server := &http.Server{
		Handler:           cart.NewHTTPHandler(service),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	stopped := make(chan error, 1)
	go func() { stopped <- server.Serve(listener) }()
	return server, stopped
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
