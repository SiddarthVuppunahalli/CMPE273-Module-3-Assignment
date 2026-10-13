package order

import (
	"context"
	"fmt"
	"testing"

	servicesv1 "github.com/SiddarthVuppunahalli/CMPE273-Module-3-Assignment/gen/services/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGRPCCreateOrder(t *testing.T) {
	t.Parallel()

	checker := checkerFunc(func(context.Context, string, int32) (InventoryResult, error) {
		return InventoryResult{Available: true}, nil
	})
	server := NewGRPCServer(NewService(checker))

	response, err := server.CreateOrder(context.Background(), &servicesv1.CreateOrderRequest{
		ItemId:   "item-1",
		Quantity: 2,
	})
	if err != nil {
		t.Fatalf("CreateOrder() unexpected error: %v", err)
	}
	if response.GetStatus() != "accepted" || response.GetItemId() != "item-1" || response.GetQuantity() != 2 {
		t.Errorf("CreateOrder() response = %+v, want accepted item-1 order", response)
	}
}

func TestGRPCCreateOrderMapsErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		checkerErr error
		request    *servicesv1.CreateOrderRequest
		wantCode   codes.Code
	}{
		{
			name:     "nil request",
			request:  nil,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "missing item ID",
			request:  &servicesv1.CreateOrderRequest{Quantity: 1},
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "invalid quantity",
			request:  &servicesv1.CreateOrderRequest{ItemId: "item-1"},
			wantCode: codes.InvalidArgument,
		},
		{
			name:       "unknown item",
			checkerErr: ErrItemNotFound,
			request:    &servicesv1.CreateOrderRequest{ItemId: "item-99", Quantity: 1},
			wantCode:   codes.NotFound,
		},
		{
			name:       "insufficient inventory",
			checkerErr: ErrInsufficientInventory,
			request:    &servicesv1.CreateOrderRequest{ItemId: "item-2", Quantity: 4},
			wantCode:   codes.FailedPrecondition,
		},
		{
			name:       "inventory deadline exceeded",
			checkerErr: fmt.Errorf("wrapped: %w", ErrInventoryTimeout),
			request:    &servicesv1.CreateOrderRequest{ItemId: "item-1", Quantity: 1},
			wantCode:   codes.DeadlineExceeded,
		},
		{
			name:       "inventory unavailable",
			checkerErr: ErrInventoryUnavailable,
			request:    &servicesv1.CreateOrderRequest{ItemId: "item-1", Quantity: 1},
			wantCode:   codes.Unavailable,
		},
		{
			name:       "request canceled",
			checkerErr: context.Canceled,
			request:    &servicesv1.CreateOrderRequest{ItemId: "item-1", Quantity: 1},
			wantCode:   codes.Canceled,
		},
		{
			name:       "unexpected error",
			checkerErr: fmt.Errorf("unexpected"),
			request:    &servicesv1.CreateOrderRequest{ItemId: "item-1", Quantity: 1},
			wantCode:   codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			checker := checkerFunc(func(context.Context, string, int32) (InventoryResult, error) {
				return InventoryResult{}, tt.checkerErr
			})
			server := NewGRPCServer(NewService(checker))

			_, err := server.CreateOrder(context.Background(), tt.request)
			if gotCode := status.Code(err); gotCode != tt.wantCode {
				t.Fatalf("status code = %v, want %v; error = %v", gotCode, tt.wantCode, err)
			}
		})
	}
}
