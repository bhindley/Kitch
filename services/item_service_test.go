package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bhindley/Kitch/clients"
	"github.com/bhindley/Kitch/models"
	"github.com/bhindley/Kitch/services"
	"github.com/jackc/pgx/v5"
)

// --- Mocks ---

type mockItemRepository struct {
	findByBarcodeFunc func(ctx context.Context, barcode string) ([]models.Item, error)
	findByIDFunc      func(ctx context.Context, id string) (*models.Item, error)
	createFunc        func(ctx context.Context, item *models.Item) error
	updateFunc        func(ctx context.Context, item *models.Item) error
	deleteFunc        func(ctx context.Context, id string) error
}

func (m *mockItemRepository) FindByBarcode(ctx context.Context, barcode string) ([]models.Item, error) {
	return m.findByBarcodeFunc(ctx, barcode)
}

func (m *mockItemRepository) FindByID(ctx context.Context, id string) (*models.Item, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockItemRepository) Create(ctx context.Context, item *models.Item) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, item)
	}
	return nil
}

func (m *mockItemRepository) Update(ctx context.Context, item *models.Item) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, item)
	}
	return nil
}

func (m *mockItemRepository) Delete(ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

type mockOFFClient struct {
	getProductFunc func(ctx context.Context, barcode string) (*models.Item, error)
}

func (m *mockOFFClient) GetProductByBarcode(ctx context.Context, barcode string) (*models.Item, error) {
	return m.getProductFunc(ctx, barcode)
}

// --- LookupBarcode Tests ---

func TestLookupBarcode_NewBarcode(t *testing.T) {
	var persisted *models.Item
	repo := &mockItemRepository{
		findByBarcodeFunc: func(_ context.Context, _ string) ([]models.Item, error) {
			return nil, nil
		},
		createFunc: func(_ context.Context, item *models.Item) error {
			persisted = item
			return nil
		},
	}
	offClient := &mockOFFClient{
		getProductFunc: func(_ context.Context, barcode string) (*models.Item, error) {
			return &models.Item{
				Name:          "Coca-Cola",
				Barcode:       barcode,
				Brand:         "Coca-Cola",
				ContainerSize: "330ml",
				ImageURL:      "https://example.com/img.jpg",
			}, nil
		},
	}

	svc := services.NewItemService(repo, offClient)
	result, err := svc.LookupBarcode(context.Background(), "5000159484695")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsNew {
		t.Error("expected IsNew to be true")
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].Name != "Coca-Cola" {
		t.Errorf("expected name 'Coca-Cola', got '%s'", result.Items[0].Name)
	}
	if result.Items[0].ID == "" {
		t.Error("expected a generated UUID on persisted item")
	}
	if persisted == nil {
		t.Fatal("expected Create to be called on repo")
	}
	if persisted.ID != result.Items[0].ID {
		t.Error("persisted ID should match returned item ID")
	}
}

func TestLookupBarcode_WithExistingItems(t *testing.T) {
	repo := &mockItemRepository{
		findByBarcodeFunc: func(_ context.Context, _ string) ([]models.Item, error) {
			return []models.Item{
				{ID: "old-1", Name: "First Coke", Barcode: "5000159484695"},
				{ID: "old-2", Name: "Second Coke", Barcode: "5000159484695"},
			}, nil
		},
		createFunc: func(_ context.Context, _ *models.Item) error {
			t.Fatal("Create should not be called when existing items found")
			return nil
		},
	}
	offClient := &mockOFFClient{
		getProductFunc: func(_ context.Context, barcode string) (*models.Item, error) {
			t.Fatal("OFF should not be called when existing items found")
			return nil, nil
		},
	}

	svc := services.NewItemService(repo, offClient)
	result, err := svc.LookupBarcode(context.Background(), "5000159484695")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsNew {
		t.Error("expected IsNew to be false")
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Items))
	}
	if result.Items[0].ID != "old-1" {
		t.Errorf("expected first item ID 'old-1', got '%s'", result.Items[0].ID)
	}
}

func TestLookupBarcode_PersistError(t *testing.T) {
	repo := &mockItemRepository{
		findByBarcodeFunc: func(_ context.Context, _ string) ([]models.Item, error) {
			return nil, nil
		},
		createFunc: func(_ context.Context, _ *models.Item) error {
			return errors.New("insert failed")
		},
	}
	offClient := &mockOFFClient{
		getProductFunc: func(_ context.Context, barcode string) (*models.Item, error) {
			return &models.Item{Name: "Coca-Cola", Barcode: barcode}, nil
		},
	}

	svc := services.NewItemService(repo, offClient)
	result, err := svc.LookupBarcode(context.Background(), "5000159484695")

	if err == nil {
		t.Fatal("expected error from persist failure")
	}
	if result != nil {
		t.Errorf("expected nil result, got: %+v", result)
	}
}

func TestLookupBarcode_NotFoundInOFF(t *testing.T) {
	repo := &mockItemRepository{
		findByBarcodeFunc: func(_ context.Context, _ string) ([]models.Item, error) {
			return nil, nil
		},
	}
	offClient := &mockOFFClient{
		getProductFunc: func(_ context.Context, _ string) (*models.Item, error) {
			return nil, clients.ErrProductNotFound
		},
	}

	svc := services.NewItemService(repo, offClient)
	result, err := svc.LookupBarcode(context.Background(), "0000000000000")

	if !errors.Is(err, services.ErrItemNotFound) {
		t.Fatalf("expected ErrItemNotFound, got: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got: %+v", result)
	}
}

func TestLookupBarcode_RepoError(t *testing.T) {
	repo := &mockItemRepository{
		findByBarcodeFunc: func(_ context.Context, _ string) ([]models.Item, error) {
			return nil, errors.New("database connection failed")
		},
	}
	offClient := &mockOFFClient{
		getProductFunc: func(_ context.Context, _ string) (*models.Item, error) {
			t.Fatal("OFF should not be called when repo errors")
			return nil, nil
		},
	}

	svc := services.NewItemService(repo, offClient)
	result, err := svc.LookupBarcode(context.Background(), "1234567890123")

	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Errorf("expected nil result, got: %+v", result)
	}
}

func TestLookupBarcode_OFFError(t *testing.T) {
	repo := &mockItemRepository{
		findByBarcodeFunc: func(_ context.Context, _ string) ([]models.Item, error) {
			return nil, nil
		},
	}
	offClient := &mockOFFClient{
		getProductFunc: func(_ context.Context, _ string) (*models.Item, error) {
			return nil, errors.New("OFF server timeout")
		},
	}

	svc := services.NewItemService(repo, offClient)
	result, err := svc.LookupBarcode(context.Background(), "1234567890123")

	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Errorf("expected nil result, got: %+v", result)
	}
}

// --- Create Tests ---

func TestCreate_Success(t *testing.T) {
	var persisted *models.Item
	repo := &mockItemRepository{
		findByBarcodeFunc: func(_ context.Context, _ string) ([]models.Item, error) {
			return nil, nil
		},
		createFunc: func(_ context.Context, item *models.Item) error {
			persisted = item
			return nil
		},
	}
	offClient := &mockOFFClient{
		getProductFunc: func(_ context.Context, _ string) (*models.Item, error) {
			return nil, nil
		},
	}

	svc := services.NewItemService(repo, offClient)
	item, err := svc.Create(context.Background(), models.Item{
		Name:          "Coca-Cola",
		Barcode:       "5000159484695",
		Brand:         "Coca-Cola",
		ContainerSize: "330ml",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.ID == "" {
		t.Error("expected a generated UUID")
	}
	if item.Name != "Coca-Cola" {
		t.Errorf("expected name 'Coca-Cola', got '%s'", item.Name)
	}
	if persisted == nil {
		t.Fatal("expected Create to be called on repo")
	}
	if persisted.ID != item.ID {
		t.Error("persisted ID should match returned ID")
	}
}

func TestCreate_RepoError(t *testing.T) {
	repo := &mockItemRepository{
		findByBarcodeFunc: func(_ context.Context, _ string) ([]models.Item, error) {
			return nil, nil
		},
		createFunc: func(_ context.Context, _ *models.Item) error {
			return errors.New("insert failed")
		},
	}
	offClient := &mockOFFClient{
		getProductFunc: func(_ context.Context, _ string) (*models.Item, error) {
			return nil, nil
		},
	}

	svc := services.NewItemService(repo, offClient)
	item, err := svc.Create(context.Background(), models.Item{Name: "Test"})

	if err == nil {
		t.Fatal("expected error")
	}
	if item != nil {
		t.Errorf("expected nil item, got: %+v", item)
	}
}

// --- Update Tests ---

func TestUpdate_Success(t *testing.T) {
	repo := &mockItemRepository{
		findByBarcodeFunc: func(_ context.Context, _ string) ([]models.Item, error) {
			return nil, nil
		},
		findByIDFunc: func(_ context.Context, id string) (*models.Item, error) {
			return &models.Item{ID: id, Name: "Old Name"}, nil
		},
		updateFunc: func(_ context.Context, _ *models.Item) error {
			return nil
		},
	}
	offClient := &mockOFFClient{
		getProductFunc: func(_ context.Context, _ string) (*models.Item, error) {
			return nil, nil
		},
	}

	svc := services.NewItemService(repo, offClient)
	updated, err := svc.Update(context.Background(), "abc-123", models.Item{
		Name:  "New Name",
		Brand: "New Brand",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.ID != "abc-123" {
		t.Errorf("expected ID 'abc-123', got '%s'", updated.ID)
	}
	if updated.Name != "New Name" {
		t.Errorf("expected name 'New Name', got '%s'", updated.Name)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	repo := &mockItemRepository{
		findByBarcodeFunc: func(_ context.Context, _ string) ([]models.Item, error) {
			return nil, nil
		},
		findByIDFunc: func(_ context.Context, _ string) (*models.Item, error) {
			return nil, nil
		},
	}
	offClient := &mockOFFClient{
		getProductFunc: func(_ context.Context, _ string) (*models.Item, error) {
			return nil, nil
		},
	}

	svc := services.NewItemService(repo, offClient)
	updated, err := svc.Update(context.Background(), "nonexistent", models.Item{Name: "X"})

	if !errors.Is(err, services.ErrItemNotFound) {
		t.Fatalf("expected ErrItemNotFound, got: %v", err)
	}
	if updated != nil {
		t.Errorf("expected nil, got: %+v", updated)
	}
}

func TestUpdate_RepoError(t *testing.T) {
	repo := &mockItemRepository{
		findByBarcodeFunc: func(_ context.Context, _ string) ([]models.Item, error) {
			return nil, nil
		},
		findByIDFunc: func(_ context.Context, id string) (*models.Item, error) {
			return &models.Item{ID: id}, nil
		},
		updateFunc: func(_ context.Context, _ *models.Item) error {
			return errors.New("update failed")
		},
	}
	offClient := &mockOFFClient{
		getProductFunc: func(_ context.Context, _ string) (*models.Item, error) {
			return nil, nil
		},
	}

	svc := services.NewItemService(repo, offClient)
	updated, err := svc.Update(context.Background(), "abc-123", models.Item{Name: "X"})

	if err == nil {
		t.Fatal("expected error")
	}
	if updated != nil {
		t.Errorf("expected nil, got: %+v", updated)
	}
}

// --- Delete Tests ---

func TestDelete_Success(t *testing.T) {
	repo := &mockItemRepository{
		findByBarcodeFunc: func(_ context.Context, _ string) ([]models.Item, error) {
			return nil, nil
		},
		deleteFunc: func(_ context.Context, _ string) error {
			return nil
		},
	}
	offClient := &mockOFFClient{
		getProductFunc: func(_ context.Context, _ string) (*models.Item, error) {
			return nil, nil
		},
	}

	svc := services.NewItemService(repo, offClient)
	err := svc.Delete(context.Background(), "abc-123")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDelete_NotFound(t *testing.T) {
	repo := &mockItemRepository{
		findByBarcodeFunc: func(_ context.Context, _ string) ([]models.Item, error) {
			return nil, nil
		},
		deleteFunc: func(_ context.Context, _ string) error {
			return pgx.ErrNoRows
		},
	}
	offClient := &mockOFFClient{
		getProductFunc: func(_ context.Context, _ string) (*models.Item, error) {
			return nil, nil
		},
	}

	svc := services.NewItemService(repo, offClient)
	err := svc.Delete(context.Background(), "nonexistent")

	if !errors.Is(err, services.ErrItemNotFound) {
		t.Fatalf("expected ErrItemNotFound, got: %v", err)
	}
}

func TestDelete_RepoError(t *testing.T) {
	repo := &mockItemRepository{
		findByBarcodeFunc: func(_ context.Context, _ string) ([]models.Item, error) {
			return nil, nil
		},
		deleteFunc: func(_ context.Context, _ string) error {
			return errors.New("delete failed")
		},
	}
	offClient := &mockOFFClient{
		getProductFunc: func(_ context.Context, _ string) (*models.Item, error) {
			return nil, nil
		},
	}

	svc := services.NewItemService(repo, offClient)
	err := svc.Delete(context.Background(), "abc-123")

	if err == nil {
		t.Fatal("expected error")
	}
}
