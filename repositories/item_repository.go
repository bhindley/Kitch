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
	FindByBarcode(ctx context.Context, barcode string) ([]models.Item, error)
	FindByID(ctx context.Context, id string) (*models.Item, error)
	Create(ctx context.Context, item *models.Item) error
	Update(ctx context.Context, item *models.Item) error
	Delete(ctx context.Context, id string) error
}

// PostgresItemRepository implements ItemRepository using PostgreSQL.
type PostgresItemRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresItemRepository creates a new PostgresItemRepository.
func NewPostgresItemRepository(pool *pgxpool.Pool) *PostgresItemRepository {
	return &PostgresItemRepository{pool: pool}
}

// FindByBarcode returns all items matching the given barcode.
// Returns an empty slice if none are found.
func (r *PostgresItemRepository) FindByBarcode(ctx context.Context, barcode string) ([]models.Item, error) {
	query := `SELECT id, name, barcode, brand, container_size, image_url
	          FROM items WHERE barcode = $1`

	rows, err := r.pool.Query(ctx, query, barcode)
	if err != nil {
		return nil, fmt.Errorf("failed to query items by barcode: %w", err)
	}
	defer rows.Close()

	var items []models.Item
	for rows.Next() {
		var item models.Item
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Barcode,
			&item.Brand,
			&item.ContainerSize,
			&item.ImageURL,
		); err != nil {
			return nil, fmt.Errorf("failed to scan item row: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating item rows: %w", err)
	}

	return items, nil
}

// FindByID looks up a single item by its ID. Returns nil, nil if not found.
func (r *PostgresItemRepository) FindByID(ctx context.Context, id string) (*models.Item, error) {
	query := `SELECT id, name, barcode, brand, container_size, image_url
	          FROM items WHERE id = $1`

	var item models.Item
	err := r.pool.QueryRow(ctx, query, id).Scan(
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
		return nil, fmt.Errorf("failed to query item by id: %w", err)
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

// Update modifies an existing item. All fields except ID are updated.
func (r *PostgresItemRepository) Update(ctx context.Context, item *models.Item) error {
	query := `UPDATE items
	          SET name = $2, barcode = $3, brand = $4, container_size = $5, image_url = $6
	          WHERE id = $1`

	result, err := r.pool.Exec(ctx, query,
		item.ID,
		item.Name,
		item.Barcode,
		item.Brand,
		item.ContainerSize,
		item.ImageURL,
	)
	if err != nil {
		return fmt.Errorf("failed to update item: %w", err)
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

// Delete removes an item by its ID.
func (r *PostgresItemRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM items WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete item: %w", err)
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}
