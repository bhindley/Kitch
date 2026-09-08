package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/bhindley/Kitch/clients"
	"github.com/bhindley/Kitch/models"
	"github.com/bhindley/Kitch/repositories"
	"github.com/google/uuid"
)

// ErrItemNotFound is returned when an item cannot be found in the datastore or Open Food Facts.
var ErrItemNotFound = errors.New("item not found")

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

// CreateByBarcode looks up an item by barcode. It checks the datastore first;
// if not found, it fetches from Open Food Facts, persists it, and returns it.
func (s *ItemService) CreateByBarcode(ctx context.Context, barcode string) (*models.Item, error) {
	// Check the datastore first.
	existing, err := s.repo.FindByBarcode(ctx, barcode)
	if err != nil {
		return nil, fmt.Errorf("failed to check datastore: %w", err)
	}
	if existing != nil {
		return existing, nil
	}

	// Not in datastore — fetch from Open Food Facts.
	item, err := s.offClient.GetProductByBarcode(ctx, barcode)
	if err != nil {
		if errors.Is(err, clients.ErrProductNotFound) {
			return nil, ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to fetch from Open Food Facts: %w", err)
	}

	// Assign an ID and persist.
	item.ID = uuid.New().String()

	if err := s.repo.Create(ctx, item); err != nil {
		return nil, fmt.Errorf("failed to persist item: %w", err)
	}

	return item, nil
}
