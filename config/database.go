package config

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ConnectDB creates a connection pool to PostgreSQL.
// It reads DATABASE_URL from the environment, defaulting to a local dev connection.
func ConnectDB() (*pgxpool.Pool, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/kitch?sslmode=disable"
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return pool, nil
}

// RunMigrations creates the required tables if they don't exist.
func RunMigrations(pool *pgxpool.Pool) error {
	query := `
	CREATE TABLE IF NOT EXISTS items (
		id             TEXT PRIMARY KEY,
		name           TEXT NOT NULL DEFAULT '',
		barcode        TEXT NOT NULL DEFAULT '',
		brand          TEXT NOT NULL DEFAULT '',
		container_size TEXT NOT NULL DEFAULT '',
		image_url      TEXT NOT NULL DEFAULT ''
	);

	CREATE UNIQUE INDEX IF NOT EXISTS idx_items_barcode ON items (barcode)
		WHERE barcode != '';
	`

	_, err := pool.Exec(context.Background(), query)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
