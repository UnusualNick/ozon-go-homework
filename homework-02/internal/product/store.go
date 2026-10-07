package product

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
)

var ErrNotFound = errors.New("product not found")

type Product struct {
	SKU   int64
	Name  string
	Price uint32
}

type Store struct {
	products map[int64]Product
	ordered  []int64
}

type jsonProduct struct {
	Name  string `json:"name"`
	Price int64  `json:"price"`
}

func Load(path string) (*Store, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open products file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var raw map[string]jsonProduct
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode products file: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return nil, err
	}

	products := make(map[int64]Product, len(raw))
	ordered := make([]int64, 0, len(raw))
	for rawSKU, item := range raw {
		sku, err := strconv.ParseInt(rawSKU, 10, 64)
		if err != nil || sku <= 0 {
			return nil, fmt.Errorf("invalid sku %q", rawSKU)
		}
		if item.Name == "" {
			return nil, fmt.Errorf("product %d has empty name", sku)
		}
		if item.Price < 0 || item.Price > math.MaxInt32 {
			return nil, fmt.Errorf("product %d price is outside int32 range", sku)
		}
		products[sku] = Product{SKU: sku, Name: item.Name, Price: uint32(item.Price)}
		ordered = append(ordered, sku)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })

	return &Store{products: products, ordered: ordered}, nil
}

func New(products []Product) *Store {
	bySKU := make(map[int64]Product, len(products))
	ordered := make([]int64, 0, len(products))
	for _, item := range products {
		bySKU[item.SKU] = item
		ordered = append(ordered, item.SKU)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	return &Store{products: bySKU, ordered: ordered}
}

func (s *Store) Get(sku int64) (Product, error) {
	item, ok := s.products[sku]
	if !ok {
		return Product{}, ErrNotFound
	}
	return item, nil
}

func (s *Store) List(offset, limit int) ([]Product, int) {
	if offset >= len(s.ordered) {
		return []Product{}, len(s.ordered)
	}
	end := min(offset+limit, len(s.ordered))
	result := make([]Product, 0, end-offset)
	for _, sku := range s.ordered[offset:end] {
		result = append(result, s.products[sku])
	}
	return result, len(s.ordered)
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("products file contains multiple JSON values")
		}
		return fmt.Errorf("read products file tail: %w", err)
	}
	return nil
}
