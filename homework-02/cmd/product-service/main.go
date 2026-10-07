package main

import (
	"log"
	"net"
	"os"

	"github.com/UnusualNick/ozon-go-homework/homework-02/internal/product"
	productv1 "github.com/UnusualNick/ozon-go-homework/homework-02/pkg/product/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	productsPath := env("PRODUCT_FILE", "products.json")
	address := env("PRODUCT_GRPC_ADDR", ":50051")

	store, err := product.Load(productsPath)
	if err != nil {
		log.Fatalf("load products: %v", err)
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("listen on %s: %v", address, err)
	}

	server := grpc.NewServer()
	productv1.RegisterProductServiceServer(server, product.NewService(store))
	reflection.Register(server)
	log.Printf("product-service listens on %s", address)
	if err := server.Serve(listener); err != nil {
		log.Fatalf("serve gRPC: %v", err)
	}
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
