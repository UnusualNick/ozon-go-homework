package cart

import (
	"context"
	"errors"
	"fmt"

	productv1 "github.com/UnusualNick/ozon-go-homework/homework-02/pkg/product/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrProductNotFound = errors.New("product not found")

type ProductClient interface {
	GetProduct(ctx context.Context, sku int64) (Product, error)
}

type Service struct {
	products ProductClient
	store    *Store
}

func NewService(products ProductClient, store *Store) *Service {
	return &Service{products: products, store: store}
}

func (s *Service) Add(ctx context.Context, userID, sku int64, count uint32) error {
	product, err := s.products.GetProduct(ctx, sku)
	if err != nil {
		return err
	}
	if err := s.store.Add(userID, product, count); err != nil {
		return fmt.Errorf("add item to cart: %w", err)
	}
	return nil
}

func (s *Service) Get(userID int64) (Cart, error) {
	return s.store.Get(userID)
}

type GRPCProductClient struct {
	client productv1.ProductServiceClient
}

func NewGRPCProductClient(client productv1.ProductServiceClient) *GRPCProductClient {
	return &GRPCProductClient{client: client}
}

func (c *GRPCProductClient) GetProduct(ctx context.Context, sku int64) (Product, error) {
	response, err := c.client.GetProduct(ctx, &productv1.GetProductRequest{Sku: sku})
	if status.Code(err) == codes.NotFound {
		return Product{}, ErrProductNotFound
	}
	if err != nil {
		return Product{}, fmt.Errorf("get product over gRPC: %w", err)
	}
	return Product{SKU: response.GetSku(), Name: response.GetName(), Price: uint32(response.GetPrice())}, nil
}
