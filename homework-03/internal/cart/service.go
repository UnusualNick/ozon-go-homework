package cart

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	productv1 "github.com/UnusualNick/ozon-go-homework/homework-03/pkg/product/v1"
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
	lookupCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	product, err := s.products.GetProduct(lookupCtx, sku)
	if err != nil {
		return err
	}
	if err := s.store.Add(userID, product, count); err != nil {
		return fmt.Errorf("add item to cart: %w", err)
	}
	return nil
}

func (s *Service) Get(ctx context.Context, userID int64) (Cart, error) {
	if err := ctx.Err(); err != nil {
		return Cart{}, err
	}
	items, err := s.store.snapshot(userID)
	if err != nil {
		return Cart{}, err
	}

	batchCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	results := make([]Item, len(items))
	errorsByIndex := make([]error, len(items))
	var group sync.WaitGroup
	for index, item := range items {
		group.Add(1)
		go func() {
			defer group.Done()
			product, err := s.products.GetProduct(batchCtx, item.SKU)
			if err != nil {
				errorsByIndex[index] = err
				return
			}
			product.SKU = item.SKU
			results[index] = Item{Product: product, Count: item.Count}
		}()
	}
	group.Wait()
	for _, err := range errorsByIndex {
		if errors.Is(err, ErrProductNotFound) || status.Code(err) == codes.NotFound {
			return Cart{}, ErrProductNotFound
		}
	}
	if err := batchCtx.Err(); err != nil {
		return Cart{}, err
	}
	for _, err := range errorsByIndex {
		if err != nil {
			return Cart{}, fmt.Errorf("get product: %w", err)
		}
	}

	cart := Cart{Items: results}
	var total uint64
	for _, item := range results {
		total += uint64(item.Price) * uint64(item.Count)
		if total > math.MaxUint32 {
			return Cart{}, ErrOverflow
		}
	}
	cart.TotalPrice = uint32(total)
	return cart, nil
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
