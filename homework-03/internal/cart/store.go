package cart

import (
	"errors"
	"math"
	"sort"
	"sync"
	"sync/atomic"
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

type cartEntry struct {
	product Product
	count   atomic.Uint32
}

type Store struct {
	mu    sync.RWMutex
	carts map[int64]map[int64]*cartEntry
}

func NewStore() *Store {
	return &Store{carts: make(map[int64]map[int64]*cartEntry)}
}

func (s *Store) Add(userID int64, product Product, count uint32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	userCart := s.carts[userID]
	if userCart == nil {
		userCart = make(map[int64]*cartEntry)
		s.carts[userID] = userCart
	}
	entry := userCart[product.SKU]
	if entry == nil {
		entry = &cartEntry{}
		userCart[product.SKU] = entry
	}
	previous := entry.count.Load()
	if count > math.MaxUint32-previous {
		return ErrOverflow
	}
	entry.product = product
	entry.count.Add(count)
	return nil
}

// snapshot copies the cart under the read lock so remote calls never hold it.
func (s *Store) snapshot(userID int64) ([]Item, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userCart := s.carts[userID]
	if len(userCart) == 0 {
		return nil, ErrCartNotFound
	}
	items := make([]Item, 0, len(userCart))
	for _, entry := range userCart {
		items = append(items, Item{Product: entry.product, Count: entry.count.Load()})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].SKU < items[j].SKU })
	return items, nil
}

func (s *Store) Get(userID int64) (Cart, error) {
	items, err := s.snapshot(userID)
	if err != nil {
		return Cart{}, err
	}
	result := Cart{Items: items}
	var total uint64
	for _, item := range items {
		total += uint64(item.Price) * uint64(item.Count)
		if total > math.MaxUint32 {
			return Cart{}, ErrOverflow
		}
	}
	result.TotalPrice = uint32(total)
	return result, nil
}
