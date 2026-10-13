package order

import (
	"context"
	"errors"
	"testing"
	"time"

	servicesv1 "github.com/SiddarthVuppunahalli/CMPE273-Module-3-Assignment/gen/services/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type inventoryGRPCClientFunc func(context.Context, *servicesv1.CheckInventoryRequest, ...grpc.CallOption) (*servicesv1.CheckInventoryResponse, error)

func (f inventoryGRPCClientFunc) CheckInventory(ctx context.Context, request *servicesv1.CheckInventoryRequest, options ...grpc.CallOption) (*servicesv1.CheckInventoryResponse, error) {
	return f(ctx, request, options...)
}

func TestGRPCInventoryClientCheck(t *testing.T) {
	t.Parallel()

	client := NewGRPCInventoryClient(inventoryGRPCClientFunc(
		func(_ context.Context, request *servicesv1.CheckInventoryRequest, _ ...grpc.CallOption) (*servicesv1.CheckInventoryResponse, error) {
			if request.GetItemId() != "item-1" || request.GetQuantity() != 2 {
				t.Errorf("request = %+v, want item-1 quantity 2", request)
			}
			return &servicesv1.CheckInventoryResponse{
				ItemId:            "item-1",
				RequestedQuantity: 2,
				AvailableQuantity: 10,
				Available:         true,
			}, nil
		}), time.Second)

	got, err := client.Check(context.Background(), "item-1", 2)
	if err != nil {
		t.Fatalf("Check() unexpected error: %v", err)
	}
	want := InventoryResult{
		ItemID:            "item-1",
		RequestedQuantity: 2,
		AvailableQuantity: 10,
		Available:         true,
	}
	if got != want {
		t.Errorf("Check() result = %+v, want %+v", got, want)
	}
}

func TestGRPCInventoryClientMapsStatuses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		grpcCode codes.Code
		wantErr  error
	}{
		{name: "unknown item", grpcCode: codes.NotFound, wantErr: ErrItemNotFound},
		{name: "insufficient inventory", grpcCode: codes.FailedPrecondition, wantErr: ErrInsufficientInventory},
		{name: "deadline exceeded", grpcCode: codes.DeadlineExceeded, wantErr: ErrInventoryTimeout},
		{name: "service unavailable", grpcCode: codes.Unavailable, wantErr: ErrInventoryUnavailable},
		{name: "unexpected status", grpcCode: codes.Internal, wantErr: ErrInventoryUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			generatedClient := inventoryGRPCClientFunc(
				func(context.Context, *servicesv1.CheckInventoryRequest, ...grpc.CallOption) (*servicesv1.CheckInventoryResponse, error) {
					return nil, status.Error(tt.grpcCode, "inventory error")
				})
			client := NewGRPCInventoryClient(generatedClient, time.Second)

			_, err := client.Check(context.Background(), "item-1", 1)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Check() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestGRPCInventoryClientAppliesDeadline(t *testing.T) {
	t.Parallel()

	generatedClient := inventoryGRPCClientFunc(
		func(ctx context.Context, _ *servicesv1.CheckInventoryRequest, _ ...grpc.CallOption) (*servicesv1.CheckInventoryResponse, error) {
			if _, ok := ctx.Deadline(); !ok {
				t.Error("gRPC call context has no deadline")
			}
			<-ctx.Done()
			return nil, status.FromContextError(ctx.Err()).Err()
		})
	client := NewGRPCInventoryClient(generatedClient, 20*time.Millisecond)

	started := time.Now()
	_, err := client.Check(context.Background(), "item-1", 1)
	if !errors.Is(err, ErrInventoryTimeout) {
		t.Fatalf("Check() error = %v, want %v", err, ErrInventoryTimeout)
	}
	if elapsed := time.Since(started); elapsed >= time.Second {
		t.Errorf("Check() exceeded expected deadline and returned after %v", elapsed)
	}
}

func TestGRPCInventoryClientPropagatesCancellation(t *testing.T) {
	t.Parallel()

	generatedClient := inventoryGRPCClientFunc(
		func(ctx context.Context, _ *servicesv1.CheckInventoryRequest, _ ...grpc.CallOption) (*servicesv1.CheckInventoryResponse, error) {
			<-ctx.Done()
			return nil, status.FromContextError(ctx.Err()).Err()
		})
	client := NewGRPCInventoryClient(generatedClient, time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Check(ctx, "item-1", 1)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Check() error = %v, want %v", err, context.Canceled)
	}
}
