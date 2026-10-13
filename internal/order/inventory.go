package order

import (
	"context"
	"errors"
)

var (
	ErrInvalidItemID         = errors.New("item_id is required")
	ErrInvalidQuantity       = errors.New("quantity must be greater than zero")
	ErrItemNotFound          = errors.New("item not found")
	ErrInsufficientInventory = errors.New("insufficient inventory")
	ErrInventoryTimeout      = errors.New("inventory request timed out")
	ErrInventoryUnavailable  = errors.New("inventory service unavailable")
)

// InventoryResult is the transport-independent result of an inventory check.
type InventoryResult struct {
	ItemID            string
	RequestedQuantity int32
	AvailableQuantity int32
	Available         bool
}

// InventoryChecker is implemented by REST and gRPC Inventory clients.
type InventoryChecker interface {
	Check(ctx context.Context, itemID string, quantity int32) (InventoryResult, error)
}
