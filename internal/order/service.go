package order

import (
	"context"
	"strings"
)

// Result describes an accepted order.
type Result struct {
	Status   string
	ItemID   string
	Quantity int32
}

// Service implements the order workflow independently of its transport.
type Service struct {
	inventory InventoryChecker
}

// NewService creates an Order Service using the supplied Inventory client.
func NewService(inventory InventoryChecker) *Service {
	return &Service{inventory: inventory}
}

// Create validates an order and checks its inventory before accepting it.
func (s *Service) Create(ctx context.Context, itemID string, quantity int32) (Result, error) {
	itemID = strings.TrimSpace(itemID)
	if itemID == "" {
		return Result{}, ErrInvalidItemID
	}
	if quantity <= 0 {
		return Result{}, ErrInvalidQuantity
	}

	if _, err := s.inventory.Check(ctx, itemID, quantity); err != nil {
		return Result{}, err
	}

	return Result{
		Status:   "accepted",
		ItemID:   itemID,
		Quantity: quantity,
	}, nil
}
