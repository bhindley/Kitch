package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/bhindley/Kitch/clients"
	"github.com/bhindley/Kitch/models"
	"github.com/bhindley/Kitch/repositories"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ErrItemNotFound is returned when an item cannot be found.
var ErrItemNotFound = errors.New("item not found")

// LookupResult is returned by LookupBarcode. It contains all items matching
// the barcode and whether this is the first time the barcode has been seen.
type LookupResult struct {
	Items []models.Item `json:"items"`
	IsNew bool          `json:"is_new"`
}

// ItemService handles business logic for items.
type ItemService struct {
	repo      repositories.ItemRepository
	offClient clients.OFFClient
}

// NewItemService creates a new ItemService.
func NewItemService(repo repositories.ItemRepository, offClient clients.OFFClient) *ItemService {
	return &ItemService{
		repo:      repo,
		offClient: offClient,
	}
}

// LookupBarcode checks the datastore for existing items with the given barcode,
// then fetches product data from Open Food Facts and persists it.
func (s *ItemService) LookupBarcode(ctx context.Context, barcode string) (*LookupResult, error) {
	// Check the datastore for existing items with this barcode.
	existing, err := s.repo.FindByBarcode(ctx, barcode)
	if err != nil {
		return nil, fmt.Errorf("failed to check datastore: %w", err)
	}

	// If items already exist in the datastore, return them without calling OFF.
	if len(existing) > 0 {
		return &LookupResult{
			Items: existing,
			IsNew: false,
		}, nil
	}

	// Not in datastore — fetch from Open Food Facts.
	product, err := s.offClient.GetProductByBarcode(ctx, barcode)
	if err != nil {
		if errors.Is(err, clients.ErrProductNotFound) {
			return nil, ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to fetch from Open Food Facts: %w", err)
	}

	// Persist the new item.
	product.ID = uuid.New().String()

	if err := s.repo.Create(ctx, product); err != nil {
		return nil, fmt.Errorf("failed to persist item: %w", err)
	}

	return &LookupResult{
		Items: []models.Item{*product},
		IsNew: true,
	}, nil
}

// List returns all items, optionally filtered by a search query.
func (s *ItemService) List(ctx context.Context, query string) ([]models.Item, error) {
	if query != "" {
		items, err := s.repo.Search(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("failed to search items: %w", err)
		}
		return items, nil
	}

	items, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list items: %w", err)
	}
	return items, nil
}

// Create persists a new item with a generated UUID.
func (s *ItemService) Create(ctx context.Context, item models.Item) (*models.Item, error) {
	item.ID = uuid.New().String()

	if err := s.repo.Create(ctx, &item); err != nil {
		return nil, fmt.Errorf("failed to persist item: %w", err)
	}

	return &item, nil
}

// Update modifies an existing item by ID.
func (s *ItemService) Update(ctx context.Context, id string, update models.Item) (*models.Item, error) {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find item: %w", err)
	}
	if existing == nil {
		return nil, ErrItemNotFound
	}

	update.ID = id
	if err := s.repo.Update(ctx, &update); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to update item: %w", err)
	}

	return &update, nil
}

// Delete removes an item by ID.
func (s *ItemService) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrItemNotFound
		}
		return fmt.Errorf("failed to delete item: %w", err)
	}

	return nil
}
