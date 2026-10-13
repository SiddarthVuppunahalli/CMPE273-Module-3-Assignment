package inventory

import (
	"context"
	"testing"
	"time"

	servicesv1 "github.com/SiddarthVuppunahalli/CMPE273-Module-3-Assignment/gen/services/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGRPCCheckInventory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		request  *servicesv1.CheckInventoryRequest
		wantCode codes.Code
	}{
		{
			name:     "available",
			request:  &servicesv1.CheckInventoryRequest{ItemId: "item-1", Quantity: 2},
			wantCode: codes.OK,
		},
		{
			name:     "insufficient inventory",
			request:  &servicesv1.CheckInventoryRequest{ItemId: "item-2", Quantity: 4},
			wantCode: codes.FailedPrecondition,
		},
		{
			name:     "unknown item",
			request:  &servicesv1.CheckInventoryRequest{ItemId: "item-99", Quantity: 1},
			wantCode: codes.NotFound,
		},
		{
			name:     "missing item ID",
			request:  &servicesv1.CheckInventoryRequest{Quantity: 1},
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "invalid quantity",
			request:  &servicesv1.CheckInventoryRequest{ItemId: "item-1"},
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "nil request",
			request:  nil,
			wantCode: codes.InvalidArgument,
		},
	}

	server := NewGRPCServer(NewDefaultStore())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			response, err := server.CheckInventory(context.Background(), tt.request)
			if gotCode := status.Code(err); gotCode != tt.wantCode {
				t.Fatalf("status code = %v, want %v; error = %v", gotCode, tt.wantCode, err)
			}

			if tt.wantCode == codes.OK {
				if response == nil {
					t.Fatal("response = nil, want inventory response")
				}
				if !response.GetAvailable() {
					t.Fatal("available = false, want true")
				}
				if response.GetAvailableQuantity() != 10 {
					t.Errorf("available quantity = %d, want 10", response.GetAvailableQuantity())
				}
			}
		})
	}
}

func TestGRPCCheckInventoryHonorsCanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	server := NewGRPCServer(NewDefaultStore())
	_, err := server.CheckInventory(ctx, &servicesv1.CheckInventoryRequest{
		ItemId:   "item-1",
		Quantity: 1,
	})
	if gotCode := status.Code(err); gotCode != codes.Canceled {
		t.Fatalf("status code = %v, want %v; error = %v", gotCode, codes.Canceled, err)
	}
}

func TestGRPCCheckInventoryHonorsExpiredDeadline(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	server := NewGRPCServer(NewDefaultStore())
	_, err := server.CheckInventory(ctx, &servicesv1.CheckInventoryRequest{
		ItemId:   "item-1",
		Quantity: 1,
	})
	if gotCode := status.Code(err); gotCode != codes.DeadlineExceeded {
		t.Fatalf("status code = %v, want %v; error = %v", gotCode, codes.DeadlineExceeded, err)
	}
}
