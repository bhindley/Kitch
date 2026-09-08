package clients_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bhindley/Kitch/clients"
)

func TestGetProductByBarcode_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/product/5000159484695.json" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("User-Agent") != "Kitch/1.0 (github.com/bhindley/Kitch)" {
			t.Errorf("unexpected User-Agent: %s", r.Header.Get("User-Agent"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"status": 1,
			"product": {
				"product_name": "Coca-Cola",
				"brands": "Coca-Cola",
				"quantity": "330ml",
				"image_url": "https://images.openfoodfacts.org/images/products/500/015/948/4695/front.jpg"
			}
		}`))
	}))
	defer server.Close()

	client := clients.NewOpenFoodFactsClientWithHTTP(server.URL, server.Client())

	item, err := client.GetProductByBarcode(context.Background(), "5000159484695")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if item.Name != "Coca-Cola" {
		t.Errorf("expected name 'Coca-Cola', got '%s'", item.Name)
	}
	if item.Brand != "Coca-Cola" {
		t.Errorf("expected brand 'Coca-Cola', got '%s'", item.Brand)
	}
	if item.Barcode != "5000159484695" {
		t.Errorf("expected barcode '5000159484695', got '%s'", item.Barcode)
	}
	if item.ContainerSize != "330ml" {
		t.Errorf("expected container size '330ml', got '%s'", item.ContainerSize)
	}
	if item.ImageURL != "https://images.openfoodfacts.org/images/products/500/015/948/4695/front.jpg" {
		t.Errorf("unexpected image URL: %s", item.ImageURL)
	}
	if item.ID != "" {
		t.Errorf("expected empty ID from OFF client, got '%s'", item.ID)
	}
}

func TestGetProductByBarcode_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"status": 0,
			"status_verbose": "product not found"
		}`))
	}))
	defer server.Close()

	client := clients.NewOpenFoodFactsClientWithHTTP(server.URL, server.Client())

	item, err := client.GetProductByBarcode(context.Background(), "0000000000000")
	if !errors.Is(err, clients.ErrProductNotFound) {
		t.Fatalf("expected ErrProductNotFound, got: %v", err)
	}
	if item != nil {
		t.Errorf("expected nil item, got: %+v", item)
	}
}

func TestGetProductByBarcode_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := clients.NewOpenFoodFactsClientWithHTTP(server.URL, server.Client())

	item, err := client.GetProductByBarcode(context.Background(), "1234567890123")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if item != nil {
		t.Errorf("expected nil item on error, got: %+v", item)
	}
}

func TestGetProductByBarcode_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not valid json`))
	}))
	defer server.Close()

	client := clients.NewOpenFoodFactsClientWithHTTP(server.URL, server.Client())

	item, err := client.GetProductByBarcode(context.Background(), "1234567890123")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if item != nil {
		t.Errorf("expected nil item on error, got: %+v", item)
	}
}

func TestGetProductByBarcode_ServerDown(t *testing.T) {
	// Use a closed server to simulate connection failure.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	client := clients.NewOpenFoodFactsClientWithHTTP(server.URL, server.Client())

	item, err := client.GetProductByBarcode(context.Background(), "1234567890123")
	if err == nil {
		t.Fatal("expected error when server is down")
	}
	if item != nil {
		t.Errorf("expected nil item on error, got: %+v", item)
	}
}
