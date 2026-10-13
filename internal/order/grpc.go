package order

import (
	"context"
	"errors"

	servicesv1 "github.com/SiddarthVuppunahalli/CMPE273-Module-3-Assignment/gen/services/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCServer implements the generated OrderService contract.
type GRPCServer struct {
	servicesv1.UnimplementedOrderServiceServer
	service *Service
}

// NewGRPCServer creates an OrderService gRPC server.
func NewGRPCServer(service *Service) *GRPCServer {
	return &GRPCServer{service: service}
}

// CreateOrder validates an order and checks Inventory over gRPC.
func (s *GRPCServer) CreateOrder(ctx context.Context, request *servicesv1.CreateOrderRequest) (*servicesv1.CreateOrderResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	result, err := s.service.Create(ctx, request.GetItemId(), request.GetQuantity())
	if err != nil {
		return nil, orderStatusError(err)
	}

	return &servicesv1.CreateOrderResponse{
		Status:   result.Status,
		ItemId:   result.ItemID,
		Quantity: result.Quantity,
	}, nil
}

func orderStatusError(err error) error {
	switch {
	case errors.Is(err, ErrInvalidItemID), errors.Is(err, ErrInvalidQuantity):
		return status.Error(codes.InvalidArgument, orderErrorMessage(err))
	case errors.Is(err, ErrItemNotFound):
		return status.Error(codes.NotFound, ErrItemNotFound.Error())
	case errors.Is(err, ErrInsufficientInventory):
		return status.Error(codes.FailedPrecondition, ErrInsufficientInventory.Error())
	case errors.Is(err, ErrInventoryTimeout), errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, ErrInventoryTimeout.Error())
	case errors.Is(err, ErrInventoryUnavailable):
		return status.Error(codes.Unavailable, ErrInventoryUnavailable.Error())
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, context.Canceled.Error())
	default:
		return status.Error(codes.Internal, "order could not be processed")
	}
}

var _ servicesv1.OrderServiceServer = (*GRPCServer)(nil)
