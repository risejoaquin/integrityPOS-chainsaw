package sqlite

import (
	"context"
	"database/sql"
	"fmt"
)

// ConfigRepository manages key-value configuration in the sync_config table.
type ConfigRepository struct {
	db *sql.DB
}

// NewConfigRepository creates a new config repository.
func NewConfigRepository(db *sql.DB) *ConfigRepository {
	return &ConfigRepository{db: db}
}

// EnsureConfigTable creates the config table if it doesn't exist.
func (r *ConfigRepository) EnsureConfigTable(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS sync_config (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create sync_config table: %w", err)
	}
	return nil
}

// Get returns a config value by key. Returns empty string if not found.
func (r *ConfigRepository) Get(ctx context.Context, key string) (string, error) {
	var value string
	err := r.db.QueryRowContext(ctx, `SELECT value FROM sync_config WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get config %s: %w", key, err)
	}
	return value, nil
}

// Set upserts a config value.
func (r *ConfigRepository) Set(ctx context.Context, key, value string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sync_config (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, value)
	if err != nil {
		return fmt.Errorf("failed to set config %s: %w", key, err)
	}
	return nil
}
