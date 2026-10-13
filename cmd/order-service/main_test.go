package main

import (
	"testing"
	"time"
)

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("ORDER_REST_ADDR", "")
	t.Setenv("ORDER_GRPC_ADDR", "")
	t.Setenv("INVENTORY_REST_URL", "")
	t.Setenv("INVENTORY_GRPC_ADDR", "")
	t.Setenv("ORDER_INVENTORY_TIMEOUT", "")

	got, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() unexpected error: %v", err)
	}

	if got.restAddress != ":8080" {
		t.Errorf("REST address = %q, want :8080", got.restAddress)
	}
	if got.grpcAddress != ":50050" {
		t.Errorf("gRPC address = %q, want :50050", got.grpcAddress)
	}
	if got.inventoryRESTURL != "http://localhost:8081" {
		t.Errorf("Inventory REST URL = %q, want http://localhost:8081", got.inventoryRESTURL)
	}
	if got.inventoryGRPCAddress != "localhost:50051" {
		t.Errorf("Inventory gRPC address = %q, want localhost:50051", got.inventoryGRPCAddress)
	}
	if got.inventoryTimeout != time.Second {
		t.Errorf("Inventory timeout = %v, want %v", got.inventoryTimeout, time.Second)
	}
}

func TestLoadConfigOverrides(t *testing.T) {
	t.Setenv("ORDER_REST_ADDR", ":9080")
	t.Setenv("ORDER_GRPC_ADDR", ":55050")
	t.Setenv("INVENTORY_REST_URL", "http://inventory:9081")
	t.Setenv("INVENTORY_GRPC_ADDR", "inventory:55051")
	t.Setenv("ORDER_INVENTORY_TIMEOUT", "250ms")

	got, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() unexpected error: %v", err)
	}

	if got.restAddress != ":9080" || got.grpcAddress != ":55050" {
		t.Errorf("listener addresses = %q, %q, want :9080, :55050", got.restAddress, got.grpcAddress)
	}
	if got.inventoryRESTURL != "http://inventory:9081" || got.inventoryGRPCAddress != "inventory:55051" {
		t.Errorf("Inventory targets = %q, %q, want configured targets", got.inventoryRESTURL, got.inventoryGRPCAddress)
	}
	if got.inventoryTimeout != 250*time.Millisecond {
		t.Errorf("Inventory timeout = %v, want %v", got.inventoryTimeout, 250*time.Millisecond)
	}
}

func TestLoadConfigRejectsInvalidTimeout(t *testing.T) {
	tests := []string{"slow", "0s", "-1s"}
	for _, value := range tests {
		t.Run(value, func(t *testing.T) {
			t.Setenv("ORDER_INVENTORY_TIMEOUT", value)
			if _, err := loadConfig(); err == nil {
				t.Fatalf("loadConfig() error = nil for %q, want error", value)
			}
		})
	}
}
