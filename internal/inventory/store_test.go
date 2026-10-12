package inventory

import (
	"context"
	"errors"
	"testing"
)

func TestStoreCheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		itemID           string
		quantity         int32
		wantAvailable    bool
		wantAvailableQty int32
		wantErr          error
	}{
		{
			name:             "available",
			itemID:           "item-1",
			quantity:         2,
			wantAvailable:    true,
			wantAvailableQty: 10,
		},
		{
			name:             "exact quantity available",
			itemID:           "item-2",
			quantity:         3,
			wantAvailable:    true,
			wantAvailableQty: 3,
		},
		{
			name:             "item ID is trimmed",
			itemID:           "  item-1  ",
			quantity:         1,
			wantAvailable:    true,
			wantAvailableQty: 10,
		},
		{
			name:     "missing item ID",
			itemID:   "   ",
			quantity: 1,
			wantErr:  ErrInvalidItemID,
		},
		{
			name:     "zero quantity",
			itemID:   "item-1",
			quantity: 0,
			wantErr:  ErrInvalidQuantity,
		},
		{
			name:     "negative quantity",
			itemID:   "item-1",
			quantity: -1,
			wantErr:  ErrInvalidQuantity,
		},
		{
			name:     "unknown item",
			itemID:   "item-99",
			quantity: 1,
			wantErr:  ErrItemNotFound,
		},
		{
			name:             "insufficient inventory",
			itemID:           "item-2",
			quantity:         4,
			wantAvailableQty: 3,
			wantErr:          ErrInsufficientStock,
		},
		{
			name:             "zero-stock item",
			itemID:           "item-3",
			quantity:         1,
			wantAvailableQty: 0,
			wantErr:          ErrInsufficientStock,
		},
	}

	store := NewDefaultStore()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := store.Check(context.Background(), tt.itemID, tt.quantity)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Check() error = %v, want %v", err, tt.wantErr)
			}
			if got.Available != tt.wantAvailable {
				t.Errorf("Check() available = %v, want %v", got.Available, tt.wantAvailable)
			}
			if got.AvailableQuantity != tt.wantAvailableQty {
				t.Errorf("Check() available quantity = %d, want %d", got.AvailableQuantity, tt.wantAvailableQty)
			}
		})
	}
}

func TestStoreCheckHonorsCanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := NewDefaultStore().Check(ctx, "item-1", 1)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Check() error = %v, want %v", err, context.Canceled)
	}
}

func TestNewStoreCopiesStock(t *testing.T) {
	t.Parallel()

	stock := map[string]int32{"item-1": 2}
	store := NewStore(stock)
	stock["item-1"] = 0

	result, err := store.Check(context.Background(), "item-1", 2)
	if err != nil {
		t.Fatalf("Check() unexpected error: %v", err)
	}
	if !result.Available {
		t.Fatal("Check() available = false, want true")
	}
}
