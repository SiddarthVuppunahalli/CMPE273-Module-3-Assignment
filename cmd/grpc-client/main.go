package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	servicesv1 "github.com/SiddarthVuppunahalli/CMPE273-Module-3-Assignment/gen/services/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

type orderOutput struct {
	Status   string `json:"status"`
	ItemID   string `json:"item_id"`
	Quantity int32  `json:"quantity"`
}

type healthOutput struct {
	Status string `json:"status"`
}

func main() {
	address := flag.String("address", "localhost:50050", "Order Service gRPC address")
	itemID := flag.String("item", "item-1", "item ID to order")
	quantity := flag.Int("quantity", 2, "quantity to order")
	checkHealth := flag.Bool("health", false, "check Order Service health instead of creating an order")
	requestTimeout := flag.Duration("timeout", 5*time.Second, "client-to-Order request timeout")
	flag.Parse()

	if *quantity < 0 || *quantity > int(^uint32(0)>>1) {
		fmt.Fprintln(os.Stderr, "quantity must fit in a non-negative int32")
		os.Exit(2)
	}

	connection, err := grpc.NewClient(
		*address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create gRPC client: %v\n", err)
		os.Exit(1)
	}
	defer connection.Close()

	ctx, cancel := context.WithTimeout(context.Background(), *requestTimeout)
	defer cancel()

	if *checkHealth {
		response, err := healthv1.NewHealthClient(connection).Check(ctx, &healthv1.HealthCheckRequest{})
		if err != nil {
			printRPCError(err)
		}
		printJSON(healthOutput{Status: response.GetStatus().String()})
		return
	}

	response, err := servicesv1.NewOrderServiceClient(connection).CreateOrder(ctx, &servicesv1.CreateOrderRequest{
		ItemId:   *itemID,
		Quantity: int32(*quantity),
	})
	if err != nil {
		printRPCError(err)
	}
	printJSON(orderOutput{
		Status:   response.GetStatus(),
		ItemID:   response.GetItemId(),
		Quantity: response.GetQuantity(),
	})
}

func printJSON(value any) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintf(os.Stderr, "encode output: %v\n", err)
		os.Exit(1)
	}
}

func printRPCError(err error) {
	grpcStatus := status.Convert(err)
	fmt.Fprintf(os.Stderr, "gRPC request failed: code=%s message=%q\n", grpcStatus.Code(), grpcStatus.Message())
	os.Exit(1)
}
