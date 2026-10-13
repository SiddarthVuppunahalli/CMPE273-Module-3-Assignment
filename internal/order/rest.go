package order

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type createOrderRequest struct {
	ItemID   string `json:"item_id"`
	Quantity int32  `json:"quantity"`
}

type createOrderResponse struct {
	Status   string `json:"status,omitempty"`
	ItemID   string `json:"item_id,omitempty"`
	Quantity int32  `json:"quantity,omitempty"`
	Error    string `json:"error,omitempty"`
}

// NewRESTHandler creates the Order Service HTTP routes.
func NewRESTHandler(service *Service) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealth)
	mux.HandleFunc("POST /v1/orders", handleCreateOrder(service))
	return mux
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleCreateOrder(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request createOrderRequest
		if err := decodeJSON(r, &request); err != nil {
			writeJSON(w, http.StatusBadRequest, createOrderResponse{Error: err.Error()})
			return
		}

		result, err := service.Create(r.Context(), request.ItemID, request.Quantity)
		if err != nil {
			writeJSON(w, orderHTTPStatus(err), createOrderResponse{Error: orderErrorMessage(err)})
			return
		}

		writeJSON(w, http.StatusCreated, createOrderResponse{
			Status:   result.Status,
			ItemID:   result.ItemID,
			Quantity: result.Quantity,
		})
	}
}

func orderHTTPStatus(err error) int {
	switch {
	case errors.Is(err, ErrInvalidItemID), errors.Is(err, ErrInvalidQuantity):
		return http.StatusBadRequest
	case errors.Is(err, ErrItemNotFound), errors.Is(err, ErrInsufficientInventory):
		return http.StatusConflict
	case errors.Is(err, ErrInventoryTimeout):
		return http.StatusGatewayTimeout
	case errors.Is(err, ErrInventoryUnavailable):
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func orderErrorMessage(err error) string {
	knownErrors := []error{
		ErrInvalidItemID,
		ErrInvalidQuantity,
		ErrItemNotFound,
		ErrInsufficientInventory,
		ErrInventoryTimeout,
		ErrInventoryUnavailable,
	}
	for _, knownErr := range knownErrors {
		if errors.Is(err, knownErr) {
			return knownErr.Error()
		}
	}
	return "order could not be processed"
}

func decodeJSON(r *http.Request, destination any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("request body is required")
		}
		return errors.New("request body must contain valid JSON")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
