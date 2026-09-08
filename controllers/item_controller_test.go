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
	"github.com/jackc/pgx/v5"
)

// --- Mocks ---

type controllerTestRepo struct {
	findByBarcodeResult []models.Item
	findByBarcodeErr    error
	findByIDResult      *models.Item
	findByIDErr         error
	createErr           error
	updateErr           error
	deleteErr           error
}

func (r *controllerTestRepo) FindByBarcode(_ context.Context, _ string) ([]models.Item, error) {
	return r.findByBarcodeResult, r.findByBarcodeErr
}

func (r *controllerTestRepo) FindByID(_ context.Context, _ string) (*models.Item, error) {
	return r.findByIDResult, r.findByIDErr
}

func (r *controllerTestRepo) Create(_ context.Context, _ *models.Item) error {
	return r.createErr
}

func (r *controllerTestRepo) Update(_ context.Context, _ *models.Item) error {
	return r.updateErr
}

func (r *controllerTestRepo) Delete(_ context.Context, _ string) error {
	return r.deleteErr
}

type controllerTestOFF struct {
	result *models.Item
	err    error
}

func (c *controllerTestOFF) GetProductByBarcode(_ context.Context, barcode string) (*models.Item, error) {
	return c.result, c.err
}

// --- Helpers ---

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
	router.GET("/api/barcode/:barcode", ctrl.LookupBarcode)
	router.POST("/api/items", ctrl.CreateItem)
	router.PUT("/api/items/:id", ctrl.UpdateItem)
	router.DELETE("/api/items/:id", ctrl.DeleteItem)
	return router
}

// --- LookupBarcode Tests ---

func TestLookupBarcode_NewBarcode(t *testing.T) {
	repo := &controllerTestRepo{}
	offClient := &controllerTestOFF{
		result: &models.Item{
			Name:    "Coca-Cola",
			Barcode: "5000159484695",
			Brand:   "Coca-Cola",
		},
	}
	svc := services.NewItemService(repo, offClient)
	router := setupRouter(svc)

	w := performRequest(router, "GET", "/api/barcode/5000159484695", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result services.LookupResult
	json.Unmarshal(w.Body.Bytes(), &result)
	if !result.IsNew {
		t.Error("expected is_new to be true")
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].Name != "Coca-Cola" {
		t.Errorf("expected name 'Coca-Cola', got '%s'", result.Items[0].Name)
	}
}

func TestLookupBarcode_WithExisting(t *testing.T) {
	repo := &controllerTestRepo{
		findByBarcodeResult: []models.Item{
			{ID: "old-1", Name: "Old Coke", Barcode: "5000159484695"},
		},
	}
	offClient := &controllerTestOFF{
		result: &models.Item{Name: "Coca-Cola", Barcode: "5000159484695"},
	}
	svc := services.NewItemService(repo, offClient)
	router := setupRouter(svc)

	w := performRequest(router, "GET", "/api/barcode/5000159484695", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result services.LookupResult
	json.Unmarshal(w.Body.Bytes(), &result)
	if result.IsNew {
		t.Error("expected is_new to be false")
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(result.Items))
	}
}

func TestLookupBarcode_NotFound(t *testing.T) {
	repo := &controllerTestRepo{}
	offClient := &controllerTestOFF{
		err: clients.ErrProductNotFound,
	}
	svc := services.NewItemService(repo, offClient)
	router := setupRouter(svc)

	w := performRequest(router, "GET", "/api/barcode/0000000000000", nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestLookupBarcode_InternalError(t *testing.T) {
	repo := &controllerTestRepo{
		findByBarcodeErr: errors.New("db failed"),
	}
	offClient := &controllerTestOFF{}
	svc := services.NewItemService(repo, offClient)
	router := setupRouter(svc)

	w := performRequest(router, "GET", "/api/barcode/1234567890123", nil)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

// --- CreateItem Tests ---

func TestCreateItem_Success(t *testing.T) {
	repo := &controllerTestRepo{}
	offClient := &controllerTestOFF{}
	svc := services.NewItemService(repo, offClient)
	router := setupRouter(svc)

	w := performRequest(router, "POST", "/api/items", map[string]string{
		"name":           "Coca-Cola",
		"barcode":        "5000159484695",
		"brand":          "Coca-Cola",
		"container_size": "330ml",
	})

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	var item models.Item
	json.Unmarshal(w.Body.Bytes(), &item)
	if item.ID == "" {
		t.Error("expected a generated UUID")
	}
	if item.Name != "Coca-Cola" {
		t.Errorf("expected name 'Coca-Cola', got '%s'", item.Name)
	}
	if item.Barcode != "5000159484695" {
		t.Errorf("expected barcode '5000159484695', got '%s'", item.Barcode)
	}
}

func TestCreateItem_MissingName(t *testing.T) {
	repo := &controllerTestRepo{}
	offClient := &controllerTestOFF{}
	svc := services.NewItemService(repo, offClient)
	router := setupRouter(svc)

	w := performRequest(router, "POST", "/api/items", map[string]string{
		"barcode": "5000159484695",
	})

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
		createErr: errors.New("insert failed"),
	}
	offClient := &controllerTestOFF{}
	svc := services.NewItemService(repo, offClient)
	router := setupRouter(svc)

	w := performRequest(router, "POST", "/api/items", map[string]string{
		"name": "Test",
	})

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

// --- UpdateItem Tests ---

func TestUpdateItem_Success(t *testing.T) {
	repo := &controllerTestRepo{
		findByIDResult: &models.Item{ID: "abc-123", Name: "Old Name"},
	}
	offClient := &controllerTestOFF{}
	svc := services.NewItemService(repo, offClient)
	router := setupRouter(svc)

	w := performRequest(router, "PUT", "/api/items/abc-123", map[string]string{
		"name":  "New Name",
		"brand": "New Brand",
	})

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var item models.Item
	json.Unmarshal(w.Body.Bytes(), &item)
	if item.ID != "abc-123" {
		t.Errorf("expected id 'abc-123', got '%s'", item.ID)
	}
	if item.Name != "New Name" {
		t.Errorf("expected name 'New Name', got '%s'", item.Name)
	}
}

func TestUpdateItem_NotFound(t *testing.T) {
	repo := &controllerTestRepo{findByIDResult: nil}
	offClient := &controllerTestOFF{}
	svc := services.NewItemService(repo, offClient)
	router := setupRouter(svc)

	w := performRequest(router, "PUT", "/api/items/nonexistent", map[string]string{
		"name": "X",
	})

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestUpdateItem_InternalError(t *testing.T) {
	repo := &controllerTestRepo{
		findByIDResult: &models.Item{ID: "abc-123"},
		updateErr:      errors.New("update failed"),
	}
	offClient := &controllerTestOFF{}
	svc := services.NewItemService(repo, offClient)
	router := setupRouter(svc)

	w := performRequest(router, "PUT", "/api/items/abc-123", map[string]string{
		"name": "X",
	})

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

// --- DeleteItem Tests ---

func TestDeleteItem_Success(t *testing.T) {
	repo := &controllerTestRepo{}
	offClient := &controllerTestOFF{}
	svc := services.NewItemService(repo, offClient)
	router := setupRouter(svc)

	w := performRequest(router, "DELETE", "/api/items/abc-123", nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}
}

func TestDeleteItem_NotFound(t *testing.T) {
	repo := &controllerTestRepo{deleteErr: pgx.ErrNoRows}
	offClient := &controllerTestOFF{}
	svc := services.NewItemService(repo, offClient)
	router := setupRouter(svc)

	w := performRequest(router, "DELETE", "/api/items/nonexistent", nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestDeleteItem_InternalError(t *testing.T) {
	repo := &controllerTestRepo{deleteErr: errors.New("delete failed")}
	offClient := &controllerTestOFF{}
	svc := services.NewItemService(repo, offClient)
	router := setupRouter(svc)

	w := performRequest(router, "DELETE", "/api/items/abc-123", nil)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}
