package inventory

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRESTInventoryCheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		body          string
		wantStatus    int
		wantAvailable bool
		wantError     string
	}{
		{
			name:          "available",
			body:          `{"item_id":"item-1","quantity":2}`,
			wantStatus:    http.StatusOK,
			wantAvailable: true,
		},
		{
			name:       "insufficient inventory",
			body:       `{"item_id":"item-2","quantity":4}`,
			wantStatus: http.StatusConflict,
			wantError:  ErrInsufficientStock.Error(),
		},
		{
			name:       "unknown item",
			body:       `{"item_id":"item-99","quantity":1}`,
			wantStatus: http.StatusNotFound,
			wantError:  ErrItemNotFound.Error(),
		},
		{
			name:       "invalid quantity",
			body:       `{"item_id":"item-1","quantity":0}`,
			wantStatus: http.StatusBadRequest,
			wantError:  ErrInvalidQuantity.Error(),
		},
	}

	handler := NewRESTHandler(NewDefaultStore())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(http.MethodPost, "/v1/inventory/check", strings.NewReader(tt.body))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, tt.wantStatus, response.Body.String())
			}
			if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", contentType)
			}

			var got checkInventoryResponse
			if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if got.Available != tt.wantAvailable {
				t.Errorf("available = %v, want %v", got.Available, tt.wantAvailable)
			}
			if got.Error != tt.wantError {
				t.Errorf("error = %q, want %q", got.Error, tt.wantError)
			}
		})
	}
}

func TestRESTInventoryCheckRejectsMalformedRequests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{name: "empty body", body: ""},
		{name: "malformed JSON", body: `{"item_id":`},
		{name: "unknown field", body: `{"item_id":"item-1","quantity":1,"extra":true}`},
		{name: "multiple objects", body: `{"item_id":"item-1","quantity":1}{}`},
	}

	handler := NewRESTHandler(NewDefaultStore())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(http.MethodPost, "/v1/inventory/check", strings.NewReader(tt.body))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusBadRequest, response.Body.String())
			}
		})
	}
}

func TestRESTInventoryRoutes(t *testing.T) {
	t.Parallel()

	handler := NewRESTHandler(NewDefaultStore())

	t.Run("health check", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
		}
	})

	t.Run("wrong method", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/v1/inventory/check", nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		if response.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
		}
	})
}
