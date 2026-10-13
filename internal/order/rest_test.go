package order

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRESTCreateOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		checkerErr error
		body       string
		wantStatus int
		wantError  string
	}{
		{
			name:       "accepted",
			body:       `{"item_id":"item-1","quantity":2}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid order",
			body:       `{"item_id":"item-1","quantity":0}`,
			wantStatus: http.StatusBadRequest,
			wantError:  ErrInvalidQuantity.Error(),
		},
		{
			name:       "item unavailable",
			checkerErr: ErrInsufficientInventory,
			body:       `{"item_id":"item-2","quantity":4}`,
			wantStatus: http.StatusConflict,
			wantError:  ErrInsufficientInventory.Error(),
		},
		{
			name:       "inventory timeout",
			checkerErr: ErrInventoryTimeout,
			body:       `{"item_id":"item-1","quantity":1}`,
			wantStatus: http.StatusGatewayTimeout,
			wantError:  ErrInventoryTimeout.Error(),
		},
		{
			name:       "inventory unavailable",
			checkerErr: ErrInventoryUnavailable,
			body:       `{"item_id":"item-1","quantity":1}`,
			wantStatus: http.StatusServiceUnavailable,
			wantError:  ErrInventoryUnavailable.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			checker := checkerFunc(func(context.Context, string, int32) (InventoryResult, error) {
				return InventoryResult{Available: tt.checkerErr == nil}, tt.checkerErr
			})
			handler := NewRESTHandler(NewService(checker))
			request := httptest.NewRequest(http.MethodPost, "/v1/orders", strings.NewReader(tt.body))
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)
			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, tt.wantStatus, response.Body.String())
			}

			var got createOrderResponse
			if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if got.Error != tt.wantError {
				t.Errorf("error = %q, want %q", got.Error, tt.wantError)
			}
			if tt.wantStatus == http.StatusCreated && got.Status != "accepted" {
				t.Errorf("status body = %q, want accepted", got.Status)
			}
		})
	}
}

func TestRESTCreateOrderRejectsMalformedRequest(t *testing.T) {
	t.Parallel()

	checker := checkerFunc(func(context.Context, string, int32) (InventoryResult, error) {
		t.Fatal("inventory checker called for malformed request")
		return InventoryResult{}, nil
	})
	handler := NewRESTHandler(NewService(checker))
	request := httptest.NewRequest(http.MethodPost, "/v1/orders", strings.NewReader(`{"item_id":`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestRESTOrderRoutes(t *testing.T) {
	t.Parallel()

	checker := checkerFunc(func(context.Context, string, int32) (InventoryResult, error) {
		return InventoryResult{}, errors.New("not used")
	})
	handler := NewRESTHandler(NewService(checker))

	t.Run("health check", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
		}
	})

	t.Run("wrong method", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/v1/orders", nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
		}
	})
}
