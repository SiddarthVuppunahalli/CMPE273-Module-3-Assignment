package order

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

type restInventoryRequest struct {
	ItemID   string `json:"item_id"`
	Quantity int32  `json:"quantity"`
}

type restInventoryResponse struct {
	ItemID            string `json:"item_id"`
	RequestedQuantity int32  `json:"requested_quantity"`
	AvailableQuantity int32  `json:"available_quantity"`
	Available         bool   `json:"available"`
	Error             string `json:"error"`
}

// RESTInventoryClient checks inventory over HTTP and JSON.
type RESTInventoryClient struct {
	endpoint string
	client   *http.Client
}

// NewRESTInventoryClient creates a client with a timeout covering the entire HTTP exchange.
func NewRESTInventoryClient(baseURL string, timeout time.Duration) *RESTInventoryClient {
	return &RESTInventoryClient{
		endpoint: strings.TrimRight(baseURL, "/") + "/v1/inventory/check",
		client:   &http.Client{Timeout: timeout},
	}
}

// Check calls the Inventory Service REST endpoint.
func (c *RESTInventoryClient) Check(ctx context.Context, itemID string, quantity int32) (InventoryResult, error) {
	body, err := json.Marshal(restInventoryRequest{ItemID: itemID, Quantity: quantity})
	if err != nil {
		return InventoryResult{}, fmt.Errorf("encode inventory request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return InventoryResult{}, fmt.Errorf("create inventory request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.client.Do(request)
	if err != nil {
		if isTimeout(err) {
			return InventoryResult{}, fmt.Errorf("%w: %v", ErrInventoryTimeout, err)
		}
		return InventoryResult{}, fmt.Errorf("%w: %v", ErrInventoryUnavailable, err)
	}
	defer response.Body.Close()

	var inventoryResponse restInventoryResponse
	if err := json.NewDecoder(response.Body).Decode(&inventoryResponse); err != nil {
		return InventoryResult{}, fmt.Errorf("%w: decode response: %v", ErrInventoryUnavailable, err)
	}

	result := InventoryResult{
		ItemID:            inventoryResponse.ItemID,
		RequestedQuantity: inventoryResponse.RequestedQuantity,
		AvailableQuantity: inventoryResponse.AvailableQuantity,
		Available:         inventoryResponse.Available,
	}

	switch response.StatusCode {
	case http.StatusOK:
		return result, nil
	case http.StatusNotFound:
		return result, responseError(ErrItemNotFound, inventoryResponse.Error)
	case http.StatusConflict:
		return result, responseError(ErrInsufficientInventory, inventoryResponse.Error)
	default:
		return result, fmt.Errorf("%w: inventory returned HTTP %d", ErrInventoryUnavailable, response.StatusCode)
	}
}

func responseError(kind error, message string) error {
	if message == "" {
		return kind
	}
	return fmt.Errorf("%w: %s", kind, message)
}

func isTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

var _ InventoryChecker = (*RESTInventoryClient)(nil)
