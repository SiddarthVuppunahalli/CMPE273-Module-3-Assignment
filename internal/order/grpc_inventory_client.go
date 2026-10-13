package order

import (
	"context"
	"errors"
	"fmt"
	"time"

	servicesv1 "github.com/SiddarthVuppunahalli/CMPE273-Module-3-Assignment/gen/services/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCInventoryClient checks inventory using the generated gRPC client.
type GRPCInventoryClient struct {
	client   servicesv1.InventoryServiceClient
	deadline time.Duration
}

// NewGRPCInventoryClient creates an Inventory client with a per-call deadline.
func NewGRPCInventoryClient(client servicesv1.InventoryServiceClient, deadline time.Duration) *GRPCInventoryClient {
	return &GRPCInventoryClient{client: client, deadline: deadline}
}

// Check calls InventoryService.CheckInventory with a bounded context.
func (c *GRPCInventoryClient) Check(ctx context.Context, itemID string, quantity int32) (InventoryResult, error) {
	callContext, cancel := context.WithTimeout(ctx, c.deadline)
	defer cancel()

	response, err := c.client.CheckInventory(callContext, &servicesv1.CheckInventoryRequest{
		ItemId:   itemID,
		Quantity: quantity,
	})
	if err != nil {
		return InventoryResult{}, grpcInventoryError(ctx, err)
	}
	if response == nil {
		return InventoryResult{}, fmt.Errorf("%w: empty gRPC response", ErrInventoryUnavailable)
	}

	return InventoryResult{
		ItemID:            response.GetItemId(),
		RequestedQuantity: response.GetRequestedQuantity(),
		AvailableQuantity: response.GetAvailableQuantity(),
		Available:         response.GetAvailable(),
	}, nil
}

func grpcInventoryError(parentContext context.Context, err error) error {
	if errors.Is(parentContext.Err(), context.Canceled) {
		return context.Canceled
	}

	switch status.Code(err) {
	case codes.NotFound:
		return responseError(ErrItemNotFound, status.Convert(err).Message())
	case codes.FailedPrecondition:
		return responseError(ErrInsufficientInventory, status.Convert(err).Message())
	case codes.DeadlineExceeded:
		return fmt.Errorf("%w: %v", ErrInventoryTimeout, err)
	case codes.Canceled:
		return context.Canceled
	case codes.Unavailable:
		return fmt.Errorf("%w: %v", ErrInventoryUnavailable, err)
	default:
		return fmt.Errorf("%w: gRPC status %s", ErrInventoryUnavailable, status.Code(err))
	}
}

var _ InventoryChecker = (*GRPCInventoryClient)(nil)
