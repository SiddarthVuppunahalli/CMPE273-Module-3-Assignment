package inventory

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type checkInventoryRequest struct {
	ItemID   string `json:"item_id"`
	Quantity int32  `json:"quantity"`
}

type checkInventoryResponse struct {
	ItemID            string `json:"item_id"`
	RequestedQuantity int32  `json:"requested_quantity"`
	AvailableQuantity int32  `json:"available_quantity"`
	Available         bool   `json:"available"`
	Error             string `json:"error,omitempty"`
}

// NewRESTHandler creates the Inventory Service HTTP routes.
func NewRESTHandler(checker Checker) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealth)
	mux.HandleFunc("POST /v1/inventory/check", handleInventoryCheck(checker))
	return mux
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleInventoryCheck(checker Checker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request checkInventoryRequest
		if err := decodeJSON(r, &request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		result, err := checker.Check(r.Context(), request.ItemID, request.Quantity)
		response := checkInventoryResponse{
			ItemID:            result.ItemID,
			RequestedQuantity: result.RequestedQuantity,
			AvailableQuantity: result.AvailableQuantity,
			Available:         result.Available,
		}

		status := http.StatusOK
		switch {
		case err == nil:
		case errors.Is(err, ErrInvalidItemID), errors.Is(err, ErrInvalidQuantity):
			status = http.StatusBadRequest
		case errors.Is(err, ErrItemNotFound):
			status = http.StatusNotFound
		case errors.Is(err, ErrInsufficientStock):
			status = http.StatusConflict
		default:
			status = http.StatusInternalServerError
		}

		if err != nil {
			response.Error = err.Error()
		}
		writeJSON(w, status, response)
	}
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
