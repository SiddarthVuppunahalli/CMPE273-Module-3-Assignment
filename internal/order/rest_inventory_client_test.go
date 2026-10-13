package order

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRESTInventoryClientCheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		status     int
		response   string
		wantResult InventoryResult
		wantErr    error
	}{
		{
			name:     "available",
			status:   http.StatusOK,
			response: `{"item_id":"item-1","requested_quantity":2,"available_quantity":10,"available":true}`,
			wantResult: InventoryResult{
				ItemID:            "item-1",
				RequestedQuantity: 2,
				AvailableQuantity: 10,
				Available:         true,
			},
		},
		{
			name:     "unknown item",
			status:   http.StatusNotFound,
			response: `{"item_id":"item-99","requested_quantity":1,"available":false,"error":"item not found"}`,
			wantErr:  ErrItemNotFound,
		},
		{
			name:     "insufficient inventory",
			status:   http.StatusConflict,
			response: `{"item_id":"item-2","requested_quantity":4,"available_quantity":3,"available":false,"error":"insufficient inventory"}`,
			wantErr:  ErrInsufficientInventory,
		},
		{
			name:     "unexpected upstream status",
			status:   http.StatusInternalServerError,
			response: `{"error":"internal error"}`,
			wantErr:  ErrInventoryUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("method = %s, want POST", r.Method)
				}
				if r.URL.Path != "/v1/inventory/check" {
					t.Errorf("path = %s, want /v1/inventory/check", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewRESTInventoryClient(server.URL, time.Second)
			got, err := client.Check(context.Background(), "item-1", 2)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Check() error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.wantResult && tt.wantErr == nil {
				t.Errorf("Check() result = %+v, want %+v", got, tt.wantResult)
			}
		})
	}
}

func TestRESTInventoryClientTimeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"available":true}`))
	}))
	defer server.Close()

	client := NewRESTInventoryClient(server.URL, 10*time.Millisecond)
	_, err := client.Check(context.Background(), "item-1", 1)
	if !errors.Is(err, ErrInventoryTimeout) {
		t.Fatalf("Check() error = %v, want %v", err, ErrInventoryTimeout)
	}
}

func TestRESTInventoryClientConnectionFailure(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	baseURL := server.URL
	server.Close()

	client := NewRESTInventoryClient(baseURL, time.Second)
	_, err := client.Check(context.Background(), "item-1", 1)
	if !errors.Is(err, ErrInventoryUnavailable) {
		t.Fatalf("Check() error = %v, want %v", err, ErrInventoryUnavailable)
	}
}

func TestRESTInventoryClientRejectsInvalidResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not JSON"))
	}))
	defer server.Close()

	client := NewRESTInventoryClient(server.URL, time.Second)
	_, err := client.Check(context.Background(), "item-1", 1)
	if !errors.Is(err, ErrInventoryUnavailable) {
		t.Fatalf("Check() error = %v, want %v", err, ErrInventoryUnavailable)
	}
}
