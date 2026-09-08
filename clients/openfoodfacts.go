package clients

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/bhindley/Kitch/models"
)

// ErrProductNotFound is returned when a barcode does not match any product in Open Food Facts.
var ErrProductNotFound = errors.New("product not found in Open Food Facts")

// OFFClient defines the contract for fetching product data from Open Food Facts.
type OFFClient interface {
	GetProductByBarcode(ctx context.Context, barcode string) (*models.Item, error)
}

// offProduct maps the fields we care about from the OFF API product object.
type offProduct struct {
	ProductName string `json:"product_name"`
	Brands      string `json:"brands"`
	Quantity    string `json:"quantity"`
	ImageURL    string `json:"image_url"`
}

// offResponse maps the top-level OFF API response.
type offResponse struct {
	Status  int        `json:"status"`
	Product offProduct `json:"product"`
}

// OpenFoodFactsClient is the default implementation of OFFClient.
type OpenFoodFactsClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewOpenFoodFactsClient creates a client with default settings.
func NewOpenFoodFactsClient() *OpenFoodFactsClient {
	return &OpenFoodFactsClient{
		baseURL: "https://world.openfoodfacts.org",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// NewOpenFoodFactsClientWithHTTP creates a client with a custom base URL and HTTP client.
// Useful for testing.
func NewOpenFoodFactsClientWithHTTP(baseURL string, httpClient *http.Client) *OpenFoodFactsClient {
	return &OpenFoodFactsClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// GetProductByBarcode fetches a product from the Open Food Facts API by its barcode.
// Returns ErrProductNotFound if the barcode does not match any product.
func (c *OpenFoodFactsClient) GetProductByBarcode(ctx context.Context, barcode string) (*models.Item, error) {
	url := fmt.Sprintf("%s/api/v2/product/%s.json", c.baseURL, barcode)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create OFF request: %w", err)
	}
	req.Header.Set("User-Agent", "Kitch/1.0 (github.com/bhindley/Kitch)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OFF request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OFF returned status %d", resp.StatusCode)
	}

	var offResp offResponse
	if err := json.NewDecoder(resp.Body).Decode(&offResp); err != nil {
		return nil, fmt.Errorf("failed to decode OFF response: %w", err)
	}

	if offResp.Status != 1 {
		return nil, ErrProductNotFound
	}

	item := &models.Item{
		Name:          offResp.Product.ProductName,
		Barcode:       barcode,
		Brand:         offResp.Product.Brands,
		ContainerSize: offResp.Product.Quantity,
		ImageURL:      offResp.Product.ImageURL,
	}

	return item, nil
}
