package inventory

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrInvalidItemID     = errors.New("item_id is required")
	ErrInvalidQuantity   = errors.New("quantity must be greater than zero")
	ErrItemNotFound      = errors.New("item not found")
	ErrInsufficientStock = errors.New("insufficient inventory")
)

// Result describes the inventory available for a request.
type Result struct {
	ItemID            string
	RequestedQuantity int32
	AvailableQuantity int32
	Available         bool
}

// Checker is implemented by anything that can check inventory availability.
type Checker interface {
	Check(ctx context.Context, itemID string, quantity int32) (Result, error)
}

// Store is a read-only, in-memory inventory store.
type Store struct {
	stock map[string]int32
}

// NewStore copies stock so callers cannot change the store after construction.
func NewStore(stock map[string]int32) *Store {
	stockCopy := make(map[string]int32, len(stock))
	for itemID, quantity := range stock {
		stockCopy[itemID] = quantity
	}

	return &Store{stock: stockCopy}
}

// NewDefaultStore returns the deterministic catalog used by both transports.
func NewDefaultStore() *Store {
	return NewStore(map[string]int32{
		"item-1": 10,
		"item-2": 3,
		"item-3": 0,
	})
}

// Check validates a request and reports whether the requested stock is available.
// It does not reserve or reduce inventory.
func (s *Store) Check(ctx context.Context, itemID string, quantity int32) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	itemID = strings.TrimSpace(itemID)
	result := Result{
		ItemID:            itemID,
		RequestedQuantity: quantity,
	}

	if itemID == "" {
		return result, ErrInvalidItemID
	}
	if quantity <= 0 {
		return result, ErrInvalidQuantity
	}

	availableQuantity, found := s.stock[itemID]
	if !found {
		return result, ErrItemNotFound
	}

	result.AvailableQuantity = availableQuantity
	result.Available = availableQuantity >= quantity
	if !result.Available {
		return result, ErrInsufficientStock
	}

	return result, nil
}
