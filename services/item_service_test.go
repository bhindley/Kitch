package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bhindley/Kitch/clients"
	"github.com/bhindley/Kitch/models"
	"github.com/bhindley/Kitch/services"
)

// --- Mocks ---

type mockItemRepository struct {
	findByBarcodeFunc func(ctx context.Context, barcode string) (*models.Item, error)
	createFunc        func(ctx context.Context, item *models.Item) error
}

func (m *mockItemRepository) FindByBarcode(ctx context.Context, barcode string) (*models.Item, error) {
	return m.findByBarcodeFunc(ctx, barcode)
}

func (m *mockItemRepository) Create(ctx context.Context, item *models.Item) error {
	return m.createFunc(ctx, item)
}

type mockOFFClient struct {
	getProductFunc func(ctx context.Context, barcode string) (*models.Item, error)
}

func (m *mockOFFClient) GetProductByBarcode(ctx context.Context, barcode string) (*models.Item, error) {
	return m.getProductFunc(ctx, barcode)
}

// --- Tests ---

func TestCreateByBarcode_FoundInDatastore(t *testing.T) {
	existing := &models.Item{
		ID:      "existing-id",
		Name:    "Cached Product",
		Barcode: "1234567890123",
		Brand:   "TestBrand",
	}

	repo := &mockItemRepository{
		findByBarcodeFunc: func(ctx context.Context, barcode string) (*models.Item, error) {
			return existing, nil
		},
		createFunc: func(ctx context.Context, item *models.Item) error {
			t.Fatal("Create should not be called when item exists in datastore")
			return nil
		},
	}

	offClient := &mockOFFClient{
		getProductFunc: func(ctx context.Context, barcode string) (*models.Item, error) {
			t.Fatal("OFF client should not be called when item exists in datastore")
			return nil, nil
		},
	}

	svc := services.NewItemService(repo, offClient)
	item, err := svc.CreateByBarcode(context.Background(), "1234567890123")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.ID != "existing-id" {
		t.Errorf("expected existing ID, got '%s'", item.ID)
	}
	if item.Name != "Cached Product" {
		t.Errorf("expected 'Cached Product', got '%s'", item.Name)
	}
}

func TestCreateByBarcode_FetchedFromOFF(t *testing.T) {
	var createdItem *models.Item

	repo := &mockItemRepository{
		findByBarcodeFunc: func(ctx context.Context, barcode string) (*models.Item, error) {
			return nil, nil // not found
		},
		createFunc: func(ctx context.Context, item *models.Item) error {
			createdItem = item
			return nil
		},
	}

	offClient := &mockOFFClient{
		getProductFunc: func(ctx context.Context, barcode string) (*models.Item, error) {
			return &models.Item{
				Name:          "OFF Product",
				Barcode:       barcode,
				Brand:         "OFF Brand",
				ContainerSize: "500ml",
				ImageURL:      "https://example.com/image.jpg",
			}, nil
		},
	}

	svc := services.NewItemService(repo, offClient)
	item, err := svc.CreateByBarcode(context.Background(), "9876543210987")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.ID == "" {
		t.Error("expected a generated UUID, got empty string")
	}
	if item.Name != "OFF Product" {
		t.Errorf("expected 'OFF Product', got '%s'", item.Name)
	}
	if item.Barcode != "9876543210987" {
		t.Errorf("expected barcode '9876543210987', got '%s'", item.Barcode)
	}
	if createdItem == nil {
		t.Fatal("expected Create to be called")
	}
	if createdItem.ID != item.ID {
		t.Error("persisted item ID should match returned item ID")
	}
}

func TestCreateByBarcode_NotFoundAnywhere(t *testing.T) {
	repo := &mockItemRepository{
		findByBarcodeFunc: func(ctx context.Context, barcode string) (*models.Item, error) {
			return nil, nil
		},
		createFunc: func(ctx context.Context, item *models.Item) error {
			t.Fatal("Create should not be called when item is not found")
			return nil
		},
	}

	offClient := &mockOFFClient{
		getProductFunc: func(ctx context.Context, barcode string) (*models.Item, error) {
			return nil, clients.ErrProductNotFound
		},
	}

	svc := services.NewItemService(repo, offClient)
	item, err := svc.CreateByBarcode(context.Background(), "0000000000000")

	if !errors.Is(err, services.ErrItemNotFound) {
		t.Fatalf("expected ErrItemNotFound, got: %v", err)
	}
	if item != nil {
		t.Errorf("expected nil item, got: %+v", item)
	}
}

func TestCreateByBarcode_RepoError(t *testing.T) {
	repo := &mockItemRepository{
		findByBarcodeFunc: func(ctx context.Context, barcode string) (*models.Item, error) {
			return nil, errors.New("database connection failed")
		},
		createFunc: func(ctx context.Context, item *models.Item) error {
			return nil
		},
	}

	offClient := &mockOFFClient{
		getProductFunc: func(ctx context.Context, barcode string) (*models.Item, error) {
			t.Fatal("OFF client should not be called when repo errors")
			return nil, nil
		},
	}

	svc := services.NewItemService(repo, offClient)
	item, err := svc.CreateByBarcode(context.Background(), "1234567890123")

	if err == nil {
		t.Fatal("expected error from repo failure")
	}
	if item != nil {
		t.Errorf("expected nil item on error, got: %+v", item)
	}
}

func TestCreateByBarcode_OFFClientError(t *testing.T) {
	repo := &mockItemRepository{
		findByBarcodeFunc: func(ctx context.Context, barcode string) (*models.Item, error) {
			return nil, nil
		},
		createFunc: func(ctx context.Context, item *models.Item) error {
			return nil
		},
	}

	offClient := &mockOFFClient{
		getProductFunc: func(ctx context.Context, barcode string) (*models.Item, error) {
			return nil, errors.New("OFF server timeout")
		},
	}

	svc := services.NewItemService(repo, offClient)
	item, err := svc.CreateByBarcode(context.Background(), "1234567890123")

	if err == nil {
		t.Fatal("expected error from OFF client failure")
	}
	if item != nil {
		t.Errorf("expected nil item on error, got: %+v", item)
	}
}

func TestCreateByBarcode_PersistError(t *testing.T) {
	repo := &mockItemRepository{
		findByBarcodeFunc: func(ctx context.Context, barcode string) (*models.Item, error) {
			return nil, nil
		},
		createFunc: func(ctx context.Context, item *models.Item) error {
			return errors.New("insert failed")
		},
	}

	offClient := &mockOFFClient{
		getProductFunc: func(ctx context.Context, barcode string) (*models.Item, error) {
			return &models.Item{
				Name:    "OFF Product",
				Barcode: barcode,
			}, nil
		},
	}

	svc := services.NewItemService(repo, offClient)
	item, err := svc.CreateByBarcode(context.Background(), "1234567890123")

	if err == nil {
		t.Fatal("expected error from persist failure")
	}
	if item != nil {
		t.Errorf("expected nil item on error, got: %+v", item)
	}
}
