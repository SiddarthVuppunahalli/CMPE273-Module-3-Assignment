package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	servicesv1 "github.com/SiddarthVuppunahalli/CMPE273-Module-3-Assignment/gen/services/v1"
	"github.com/SiddarthVuppunahalli/CMPE273-Module-3-Assignment/internal/inventory"
	"github.com/SiddarthVuppunahalli/CMPE273-Module-3-Assignment/internal/order"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const bufferSize = 1024 * 1024

func TestRESTOrderToInventorySuccess(t *testing.T) {
	inventoryServer := httptest.NewServer(inventory.NewRESTHandler(inventory.NewDefaultStore()))
	defer inventoryServer.Close()

	orderService := order.NewService(order.NewRESTInventoryClient(inventoryServer.URL, time.Second))
	orderServer := httptest.NewServer(order.NewRESTHandler(orderService))
	defer orderServer.Close()

	response := postOrder(t, orderServer.URL, "item-1", 2)
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusCreated)
	}

	var body struct {
		Status string `json:"status"`
		ItemID string `json:"item_id"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != "accepted" || body.ItemID != "item-1" {
		t.Errorf("response = %+v, want accepted item-1 order", body)
	}
}

func TestRESTTimeoutLeavesOrderRunning(t *testing.T) {
	delayedInventory := inventory.WithDelay(inventory.NewDefaultStore(), 200*time.Millisecond)
	inventoryServer := httptest.NewServer(inventory.NewRESTHandler(delayedInventory))
	defer inventoryServer.Close()

	orderService := order.NewService(order.NewRESTInventoryClient(inventoryServer.URL, 20*time.Millisecond))
	orderServer := httptest.NewServer(order.NewRESTHandler(orderService))
	defer orderServer.Close()

	response := postOrder(t, orderServer.URL, "item-1", 1)
	defer response.Body.Close()
	if response.StatusCode != http.StatusGatewayTimeout {
		t.Fatalf("timeout status = %d, want %d", response.StatusCode, http.StatusGatewayTimeout)
	}

	healthResponse, err := http.Get(orderServer.URL + "/healthz")
	if err != nil {
		t.Fatalf("health request after timeout: %v", err)
	}
	defer healthResponse.Body.Close()
	if healthResponse.StatusCode != http.StatusOK {
		t.Fatalf("health status after timeout = %d, want %d", healthResponse.StatusCode, http.StatusOK)
	}
}

func TestGRPCOrderToInventorySuccess(t *testing.T) {
	inventoryConnection := startInventoryGRPC(t, inventory.NewDefaultStore())
	orderConnection := startOrderGRPC(t, order.NewGRPCInventoryClient(
		servicesv1.NewInventoryServiceClient(inventoryConnection),
		time.Second,
	))

	client := servicesv1.NewOrderServiceClient(orderConnection)
	response, err := client.CreateOrder(context.Background(), &servicesv1.CreateOrderRequest{
		ItemId:   "item-1",
		Quantity: 2,
	})
	if err != nil {
		t.Fatalf("CreateOrder() unexpected error: %v", err)
	}
	if response.GetStatus() != "accepted" || response.GetItemId() != "item-1" {
		t.Errorf("CreateOrder() response = %+v, want accepted item-1 order", response)
	}
}

func TestGRPCDeadlineLeavesOrderRunning(t *testing.T) {
	delayedInventory := inventory.WithDelay(inventory.NewDefaultStore(), 200*time.Millisecond)
	inventoryConnection := startInventoryGRPC(t, delayedInventory)
	orderConnection := startOrderGRPC(t, order.NewGRPCInventoryClient(
		servicesv1.NewInventoryServiceClient(inventoryConnection),
		20*time.Millisecond,
	))

	client := servicesv1.NewOrderServiceClient(orderConnection)
	_, err := client.CreateOrder(context.Background(), &servicesv1.CreateOrderRequest{
		ItemId:   "item-1",
		Quantity: 1,
	})
	if gotCode := status.Code(err); gotCode != codes.DeadlineExceeded {
		t.Fatalf("CreateOrder() status = %v, want %v; error = %v", gotCode, codes.DeadlineExceeded, err)
	}

	healthClient := healthv1.NewHealthClient(orderConnection)
	healthResponse, err := healthClient.Check(context.Background(), &healthv1.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("gRPC health check after deadline: %v", err)
	}
	if healthResponse.GetStatus() != healthv1.HealthCheckResponse_SERVING {
		t.Fatalf("gRPC health after deadline = %v, want SERVING", healthResponse.GetStatus())
	}
}

func postOrder(t *testing.T, baseURL, itemID string, quantity int32) *http.Response {
	t.Helper()

	body, err := json.Marshal(map[string]any{"item_id": itemID, "quantity": quantity})
	if err != nil {
		t.Fatalf("encode order: %v", err)
	}
	response, err := http.Post(baseURL+"/v1/orders", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post order: %v", err)
	}
	return response
}

func startInventoryGRPC(t *testing.T, checker inventory.Checker) *grpc.ClientConn {
	t.Helper()

	listener := bufconn.Listen(bufferSize)
	server := grpc.NewServer()
	servicesv1.RegisterInventoryServiceServer(server, inventory.NewGRPCServer(checker))
	go func() {
		_ = server.Serve(listener)
	}()

	connection := dialBuffer(t, listener)
	t.Cleanup(func() {
		_ = connection.Close()
		server.Stop()
		_ = listener.Close()
	})
	return connection
}

func startOrderGRPC(t *testing.T, inventoryClient order.InventoryChecker) *grpc.ClientConn {
	t.Helper()

	listener := bufconn.Listen(bufferSize)
	server := grpc.NewServer()
	servicesv1.RegisterOrderServiceServer(server, order.NewGRPCServer(order.NewService(inventoryClient)))
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthv1.HealthCheckResponse_SERVING)
	healthv1.RegisterHealthServer(server, healthServer)
	go func() {
		_ = server.Serve(listener)
	}()

	connection := dialBuffer(t, listener)
	t.Cleanup(func() {
		_ = connection.Close()
		server.Stop()
		_ = listener.Close()
	})
	return connection
}

func dialBuffer(t *testing.T, listener *bufconn.Listener) *grpc.ClientConn {
	t.Helper()

	connection, err := grpc.NewClient(
		"passthrough:///buffer",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
	)
	if err != nil {
		t.Fatalf("create buffered gRPC client: %v", err)
	}
	return connection
}
