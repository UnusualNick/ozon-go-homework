package cart

import (
	"errors"
	"math"
	"sort"
	"sync"
)

var (
	ErrCartNotFound = errors.New("cart not found")
	ErrOverflow     = errors.New("cart value overflow")
)

type Product struct {
	SKU   int64
	Name  string
	Price uint32
}

type Item struct {
	Product
	Count uint32
}

type Cart struct {
	Items      []Item
	TotalPrice uint32
}

type Store struct {
	mu    sync.RWMutex
	carts map[int64]map[int64]Item
}

func NewStore() *Store {
	return &Store{carts: make(map[int64]map[int64]Item)}
}

func (s *Store) Add(userID int64, product Product, count uint32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	userCart := s.carts[userID]
	if userCart == nil {
		userCart = make(map[int64]Item)
		s.carts[userID] = userCart
	}
	item := userCart[product.SKU]
	if uint64(item.Count)+uint64(count) > math.MaxUint32 {
		return ErrOverflow
	}
	item.Product = product
	item.Count += count
	userCart[product.SKU] = item
	return nil
}

func (s *Store) Get(userID int64) (Cart, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userCart := s.carts[userID]
	if len(userCart) == 0 {
		return Cart{}, ErrCartNotFound
	}

	result := Cart{Items: make([]Item, 0, len(userCart))}
	var total uint64
	for _, item := range userCart {
		result.Items = append(result.Items, item)
		total += uint64(item.Price) * uint64(item.Count)
		if total > math.MaxUint32 {
			return Cart{}, ErrOverflow
		}
	}
	sort.Slice(result.Items, func(i, j int) bool {
		return result.Items[i].SKU < result.Items[j].SKU
	})
	result.TotalPrice = uint32(total)
	return result, nil
}
