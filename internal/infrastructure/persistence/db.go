package persistence

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// PoolConfig holds SQLite connection pool configuration.
// SQLite doesn't support concurrent writes, so pool must be conservative.
type PoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	QueryTimeout    time.Duration
}

// DefaultPoolConfig returns the recommended SQLite pool configuration for IntegrityPOS
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxOpenConns:    4, // SQLite serializes writes; keep low
		MaxIdleConns:    2, // Keep a few connections ready
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
		QueryTimeout:    10 * time.Second, // Default query timeout
	}
}

// Open opens a SQLite database at the given path with WAL mode, foreign keys,
// and a properly configured connection pool.
func Open(path string) (*sql.DB, error) {
	return OpenWithOptions(path, DefaultPoolConfig())
}

// OpenWithOptions opens a SQLite database with custom pool configuration.
func OpenWithOptions(path string, cfg PoolConfig) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	// ─── SQLite PRAGMAS ──────────────────────────────────────────
	// WAL mode: allows concurrent reads, better write performance
	// Synchronous NORMAL: good balance of safety and speed
	// Foreign keys ON: ensure referential integrity
	_, err = db.Exec(`
		PRAGMA journal_mode=WAL;
		PRAGMA synchronous=NORMAL;
		PRAGMA foreign_keys=ON;
		PRAGMA busy_timeout=5000;
	`)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to set SQLite PRAGMAs: %w", err)
	}

	// ─── Connection Pool ─────────────────────────────────────────
	// SQLite serializes writes, so high concurrency causes lock contention.
	// These settings optimize for the POS workload pattern (bursts during sales).
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	return db, nil
}

// Migrate runs SQL migration statements against the database.
func Migrate(db *sql.DB, sqlStatements string) error {
	_, err := db.Exec(sqlStatements)
	return err
}
