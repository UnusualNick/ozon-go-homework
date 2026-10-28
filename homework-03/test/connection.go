//go:build api

package test

import (
	"os"

	productv1 "github.com/UnusualNick/ozon-go-homework/homework-03/pkg/product/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// These helpers only connect to running services. The tests never start them.
func connectProduct() (productv1.ProductServiceClient, *grpc.ClientConn, error) {
	target := os.Getenv("PRODUCT_SERVICE_ADDR")
	if target == "" {
		target = "127.0.0.1:50051"
	}
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	return productv1.NewProductServiceClient(conn), conn, nil
}

func cartBaseURL() string {
	if value := os.Getenv("CART_BASE_URL"); value != "" {
		return value
	}
	return "http://127.0.0.1:8080"
}
