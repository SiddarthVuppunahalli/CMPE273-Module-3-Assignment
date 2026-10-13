package inventory

import (
	"context"
	"errors"

	servicesv1 "github.com/SiddarthVuppunahalli/CMPE273-Module-3-Assignment/gen/services/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCServer implements the generated InventoryService contract.
type GRPCServer struct {
	servicesv1.UnimplementedInventoryServiceServer
	checker Checker
}

// NewGRPCServer creates an InventoryService gRPC server.
func NewGRPCServer(checker Checker) *GRPCServer {
	return &GRPCServer{checker: checker}
}

// CheckInventory reports whether an item can satisfy a requested quantity.
func (s *GRPCServer) CheckInventory(ctx context.Context, request *servicesv1.CheckInventoryRequest) (*servicesv1.CheckInventoryResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	result, err := s.checker.Check(ctx, request.GetItemId(), request.GetQuantity())
	if err != nil {
		return nil, inventoryStatusError(err)
	}

	return &servicesv1.CheckInventoryResponse{
		ItemId:            result.ItemID,
		RequestedQuantity: result.RequestedQuantity,
		AvailableQuantity: result.AvailableQuantity,
		Available:         result.Available,
	}, nil
}

func inventoryStatusError(err error) error {
	switch {
	case errors.Is(err, ErrInvalidItemID), errors.Is(err, ErrInvalidQuantity):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, ErrItemNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, ErrInsufficientStock):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return status.FromContextError(err).Err()
	default:
		return status.Error(codes.Internal, "inventory check failed")
	}
}

var _ servicesv1.InventoryServiceServer = (*GRPCServer)(nil)
