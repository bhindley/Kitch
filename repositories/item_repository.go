package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/bhindley/Kitch/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ItemRepository defines the data access contract for items.
type ItemRepository interface {
	FindByBarcode(ctx context.Context, barcode string) (*models.Item, error)
	Create(ctx context.Context, item *models.Item) error
}

// PostgresItemRepository implements ItemRepository using PostgreSQL.
type PostgresItemRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresItemRepository creates a new PostgresItemRepository.
func NewPostgresItemRepository(pool *pgxpool.Pool) *PostgresItemRepository {
	return &PostgresItemRepository{pool: pool}
}

// FindByBarcode looks up an item by its barcode. Returns nil, nil if not found.
func (r *PostgresItemRepository) FindByBarcode(ctx context.Context, barcode string) (*models.Item, error) {
	query := `SELECT id, name, barcode, brand, container_size, image_url
	          FROM items WHERE barcode = $1`

	var item models.Item
	err := r.pool.QueryRow(ctx, query, barcode).Scan(
		&item.ID,
		&item.Name,
		&item.Barcode,
		&item.Brand,
		&item.ContainerSize,
		&item.ImageURL,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query item by barcode: %w", err)
	}

	return &item, nil
}

// Create inserts a new item into the database.
func (r *PostgresItemRepository) Create(ctx context.Context, item *models.Item) error {
	query := `INSERT INTO items (id, name, barcode, brand, container_size, image_url)
	          VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.pool.Exec(ctx, query,
		item.ID,
		item.Name,
		item.Barcode,
		item.Brand,
		item.ContainerSize,
		item.ImageURL,
	)
	if err != nil {
		return fmt.Errorf("failed to insert item: %w", err)
	}

	return nil
}
