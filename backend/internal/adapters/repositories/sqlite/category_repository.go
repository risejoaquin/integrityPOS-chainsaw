package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"integritypos-backend/internal/core/domain"
)

// CategoryRepository manages categories table.
type CategoryRepository struct {
	db *sql.DB
}

// NewCategoryRepository creates a new CategoryRepository.
func NewCategoryRepository(db *sql.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

// UpsertFromCloud inserts or updates a category received from cloud sync.
func (r *CategoryRepository) UpsertFromCloud(ctx context.Context, c *domain.Category) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO categories (id, name, description, created_at, updated_at, sync_status)
		VALUES (?, ?, ?, ?, ?, 'synced')
		ON CONFLICT(id) DO UPDATE SET
			name        = excluded.name,
			description = excluded.description,
			updated_at   = excluded.updated_at,
			sync_status  = 'synced'
	`, c.ID, c.Name, c.Description, c.CreatedAt, c.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to upsert category from cloud: %w", err)
	}
	return nil
}

// GetByName retrieves a category by name.
func (r *CategoryRepository) GetByName(ctx context.Context, name string) (*domain.Category, error) {
	c := &domain.Category{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, description, created_at, updated_at FROM categories WHERE name = ?`, name,
	).Scan(&c.ID, &c.Name, &c.Description, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get category by name: %w", err)
	}
	return c, nil
}
