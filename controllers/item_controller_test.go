package controllers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bhindley/Kitch/clients"
	"github.com/bhindley/Kitch/controllers"
	"github.com/bhindley/Kitch/models"
	"github.com/bhindley/Kitch/services"
	"github.com/gin-gonic/gin"
)

// --- Mocks ---

type controllerTestRepo struct {
	findResult *models.Item
	findErr    error
	createErr  error
}

func (r *controllerTestRepo) FindByBarcode(_ context.Context, _ string) (*models.Item, error) {
	return r.findResult, r.findErr
}

func (r *controllerTestRepo) Create(_ context.Context, _ *models.Item) error {
	return r.createErr
}

type controllerTestOFF struct {
	result *models.Item
	err    error
}

func (c *controllerTestOFF) GetProductByBarcode(_ context.Context, _ string) (*models.Item, error) {
	return c.result, c.err
}

// helper to build a gin test context with a JSON request body.
func performRequest(router *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var reqBody []byte
	if body != nil {
		reqBody, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func setupRouter(svc *services.ItemService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	ctrl := controllers.NewItemController(svc)
	router.POST("/api/items", ctrl.CreateItem)
	return router
}

// --- Tests ---

func TestCreateItem_Success(t *testing.T) {
	repo := &controllerTestRepo{
		findResult: &models.Item{
			ID:            "abc-123",
			Name:          "Test Product",
			Barcode:       "5000159484695",
			Brand:         "TestBrand",
			ContainerSize: "330ml",
			ImageURL:      "https://example.com/img.jpg",
		},
	}
	offClient := &controllerTestOFF{}
	svc := services.NewItemService(repo, offClient)
	router := setupRouter(svc)

	w := performRequest(router, "POST", "/api/items", map[string]string{
		"barcode": "5000159484695",
	})

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	var item models.Item
	if err := json.Unmarshal(w.Body.Bytes(), &item); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if item.Name != "Test Product" {
		t.Errorf("expected name 'Test Product', got '%s'", item.Name)
	}
	if item.ID != "abc-123" {
		t.Errorf("expected id 'abc-123', got '%s'", item.ID)
	}
}

func TestCreateItem_NotFound(t *testing.T) {
	repo := &controllerTestRepo{
		findResult: nil,
	}
	offClient := &controllerTestOFF{
		err: clients.ErrProductNotFound,
	}
	svc := services.NewItemService(repo, offClient)
	router := setupRouter(svc)

	w := performRequest(router, "POST", "/api/items", map[string]string{
		"barcode": "0000000000000",
	})

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}

	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["error"] != "product not found for barcode" {
		t.Errorf("unexpected error message: %s", body["error"])
	}
}

func TestCreateItem_MissingBarcode(t *testing.T) {
	repo := &controllerTestRepo{}
	offClient := &controllerTestOFF{}
	svc := services.NewItemService(repo, offClient)
	router := setupRouter(svc)

	w := performRequest(router, "POST", "/api/items", map[string]string{})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestCreateItem_EmptyBody(t *testing.T) {
	repo := &controllerTestRepo{}
	offClient := &controllerTestOFF{}
	svc := services.NewItemService(repo, offClient)
	router := setupRouter(svc)

	w := performRequest(router, "POST", "/api/items", nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestCreateItem_InternalError(t *testing.T) {
	repo := &controllerTestRepo{
		findErr: errors.New("database connection failed"),
	}
	offClient := &controllerTestOFF{}
	svc := services.NewItemService(repo, offClient)
	router := setupRouter(svc)

	w := performRequest(router, "POST", "/api/items", map[string]string{
		"barcode": "1234567890123",
	})

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}
