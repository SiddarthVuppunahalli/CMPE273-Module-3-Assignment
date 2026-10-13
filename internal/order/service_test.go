package order

import (
	"context"
	"errors"
	"testing"
)

type checkerFunc func(context.Context, string, int32) (InventoryResult, error)

func (f checkerFunc) Check(ctx context.Context, itemID string, quantity int32) (InventoryResult, error) {
	return f(ctx, itemID, quantity)
}

func TestServiceCreate(t *testing.T) {
	t.Parallel()

	checker := checkerFunc(func(_ context.Context, itemID string, quantity int32) (InventoryResult, error) {
		if itemID != "item-1" {
			t.Errorf("item ID = %q, want item-1", itemID)
		}
		if quantity != 2 {
			t.Errorf("quantity = %d, want 2", quantity)
		}
		return InventoryResult{Available: true}, nil
	})

	result, err := NewService(checker).Create(context.Background(), "  item-1  ", 2)
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}
	if result.Status != "accepted" || result.ItemID != "item-1" || result.Quantity != 2 {
		t.Errorf("Create() result = %+v, want accepted item-1 order", result)
	}
}

func TestServiceCreateValidatesBeforeInventoryCheck(t *testing.T) {
	t.Parallel()

	called := false
	checker := checkerFunc(func(context.Context, string, int32) (InventoryResult, error) {
		called = true
		return InventoryResult{}, nil
	})
	service := NewService(checker)

	tests := []struct {
		name     string
		itemID   string
		quantity int32
		wantErr  error
	}{
		{name: "missing item ID", itemID: " ", quantity: 1, wantErr: ErrInvalidItemID},
		{name: "zero quantity", itemID: "item-1", quantity: 0, wantErr: ErrInvalidQuantity},
		{name: "negative quantity", itemID: "item-1", quantity: -1, wantErr: ErrInvalidQuantity},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Create(context.Background(), tt.itemID, tt.quantity)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Create() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
	if called {
		t.Fatal("inventory checker was called for an invalid order")
	}
}

func TestServiceCreatePropagatesInventoryError(t *testing.T) {
	t.Parallel()

	checker := checkerFunc(func(context.Context, string, int32) (InventoryResult, error) {
		return InventoryResult{}, ErrInventoryTimeout
	})

	_, err := NewService(checker).Create(context.Background(), "item-1", 1)
	if !errors.Is(err, ErrInventoryTimeout) {
		t.Fatalf("Create() error = %v, want %v", err, ErrInventoryTimeout)
	}
}
