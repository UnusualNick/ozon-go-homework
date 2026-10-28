package product

import (
	"context"
	"errors"
	"strconv"

	productv1 "github.com/UnusualNick/ozon-go-homework/homework-03/pkg/product/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultPageSize = 50
	maxPageSize     = 100
)

type Service struct {
	productv1.UnimplementedProductServiceServer
	store *Store
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

func (s *Service) GetProduct(_ context.Context, request *productv1.GetProductRequest) (*productv1.GetProductResponse, error) {
	if request.GetSku() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "sku must be greater than zero")
	}

	item, err := s.store.Get(request.GetSku())
	if errors.Is(err, ErrNotFound) {
		return nil, status.Error(codes.NotFound, "product not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "cannot get product")
	}

	return &productv1.GetProductResponse{
		Sku:   item.SKU,
		Name:  item.Name,
		Price: int32(item.Price),
	}, nil
}

func (s *Service) ListProducts(_ context.Context, request *productv1.ListProductsRequest) (*productv1.ListProductsResponse, error) {
	pageSize := int(request.GetPageSize())
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	if pageSize < 0 || pageSize > maxPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "page_size must be between 1 and %d", maxPageSize)
	}

	offset := 0
	if token := request.GetPageToken(); token != "" {
		parsed, err := strconv.Atoi(token)
		if err != nil || parsed < 0 {
			return nil, status.Error(codes.InvalidArgument, "invalid page_token")
		}
		offset = parsed
	}

	items, total := s.store.List(offset, pageSize)
	response := &productv1.ListProductsResponse{Products: make([]*productv1.Product, 0, len(items))}
	for _, item := range items {
		response.Products = append(response.Products, &productv1.Product{
			Sku: item.SKU, Name: item.Name, Price: int32(item.Price),
		})
	}
	if next := offset + len(items); next < total {
		response.NextPageToken = strconv.Itoa(next)
	}
	return response, nil
}
