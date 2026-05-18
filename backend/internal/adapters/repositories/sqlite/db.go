package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Database represents a SQLite database connection pool
type Database struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewDatabase creates a new database connection with WAL mode and foreign keys enabled
func NewDatabase(dbPath string) (*Database, error) {
	// Open connection with journal_mode=WAL for concurrency
	// uri=file: enables URI filename support
	conn := fmt.Sprintf("file:%s?cache=shared&mode=rwc&_journal_mode=WAL&_foreign_keys=ON", dbPath)

	db, err := sql.Open("sqlite3", conn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings for local development
	// SQLite is single-writer, so keep pool small
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Enable WAL mode and foreign keys at application level (extra safety)
	if _, err := db.ExecContext(ctx, "PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;"); err != nil {
		return nil, fmt.Errorf("failed to set pragmas: %w", err)
	}

	database := &Database{db: db}

	// Initialize schema
	if err := database.initSchema(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return database, nil
}

// initSchema creates all tables if they don't exist
func (d *Database) initSchema(ctx context.Context) error {
	schema := `
	-- Users table
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		role TEXT NOT NULL CHECK(role IN ('cashier', 'admin', 'manager')),
		active BOOLEAN NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	-- Shifts table
	CREATE TABLE IF NOT EXISTS shifts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		opened_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		closed_at DATETIME,
		open_balance INTEGER NOT NULL,
		close_balance INTEGER,
		notes TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		sync_status TEXT NOT NULL DEFAULT 'pending' CHECK(sync_status IN ('pending', 'synced', 'failed')),
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT
	);

	-- Categories table
	CREATE TABLE IF NOT EXISTS categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		description TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		sync_status TEXT NOT NULL DEFAULT 'pending' CHECK(sync_status IN ('pending', 'synced', 'failed'))
	);

	-- Products table
	CREATE TABLE IF NOT EXISTS products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		sku TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		barcode TEXT UNIQUE,
		price INTEGER NOT NULL,
		cost INTEGER NOT NULL,
		quantity INTEGER NOT NULL DEFAULT 0,
		category TEXT,
		category_id INTEGER,
		active BOOLEAN NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		sync_status TEXT NOT NULL DEFAULT 'pending' CHECK(sync_status IN ('pending', 'synced', 'failed')),
		FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL
	);

	-- Sales table (IMMUTABLE)
	CREATE TABLE IF NOT EXISTS sales (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		shift_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		total INTEGER NOT NULL,
		tax INTEGER NOT NULL,
		subtotal INTEGER NOT NULL,
		payment_method TEXT NOT NULL CHECK(payment_method IN ('cash', 'card', 'check')),
		notes TEXT,
		voided BOOLEAN NOT NULL DEFAULT 0,
		void_reason TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		sync_status TEXT NOT NULL DEFAULT 'pending' CHECK(sync_status IN ('pending', 'synced', 'failed')),
		FOREIGN KEY (shift_id) REFERENCES shifts(id) ON DELETE RESTRICT,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT
	);

	-- SaleItems table (IMMUTABLE)
	CREATE TABLE IF NOT EXISTS sale_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		sale_id INTEGER NOT NULL,
		product_id INTEGER NOT NULL,
		quantity INTEGER NOT NULL,
		unit_price INTEGER NOT NULL,
		total INTEGER NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (sale_id) REFERENCES sales(id) ON DELETE CASCADE,
		FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE RESTRICT
	);

	-- SyncLog table
	CREATE TABLE IF NOT EXISTS sync_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		sale_id INTEGER NOT NULL,
		status TEXT NOT NULL CHECK(status IN ('pending', 'synced', 'failed')) DEFAULT 'pending',
		error_message TEXT,
		synced_at DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (sale_id) REFERENCES sales(id) ON DELETE CASCADE
	);

	-- Customers table
	CREATE TABLE IF NOT EXISTS customers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT,
		phone TEXT,
		address TEXT,
		notes TEXT,
		active BOOLEAN NOT NULL DEFAULT 1,
		sync_status TEXT NOT NULL DEFAULT 'pending' CHECK(sync_status IN ('pending', 'synced', 'failed')),
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	-- Indexes for performance
	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	CREATE INDEX IF NOT EXISTS idx_shifts_user_id ON shifts(user_id);
	CREATE INDEX IF NOT EXISTS idx_shifts_opened_at ON shifts(opened_at);
	CREATE INDEX IF NOT EXISTS idx_shifts_sync_status ON shifts(sync_status);
	CREATE INDEX IF NOT EXISTS idx_products_sku ON products(sku);
	CREATE INDEX IF NOT EXISTS idx_products_sync_status ON products(sync_status);
	CREATE INDEX IF NOT EXISTS idx_sales_shift_id ON sales(shift_id);
	CREATE INDEX IF NOT EXISTS idx_sales_user_id ON sales(user_id);
	CREATE INDEX IF NOT EXISTS idx_sales_sync_status ON sales(sync_status);
	CREATE INDEX IF NOT EXISTS idx_sync_logs_sale_id ON sync_logs(sale_id);
	CREATE INDEX IF NOT EXISTS idx_sync_logs_status ON sync_logs(status);
	`

	_, err := d.db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Run migrations for existing databases
	if err := d.runMigrations(ctx); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Ensure sync_config table exists for downstream sync tracking
	if _, err := d.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS sync_config (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)
	`); err != nil {
		return fmt.Errorf("failed to create sync_config table: %w", err)
	}

	return nil
}

// runMigrations handles schema changes for existing databases
func (d *Database) runMigrations(ctx context.Context) error {
	migrations := []string{
		`ALTER TABLE products ADD COLUMN barcode TEXT UNIQUE;`,
		`ALTER TABLE shifts ADD COLUMN sync_status TEXT NOT NULL DEFAULT 'pending' CHECK(sync_status IN ('pending', 'synced', 'failed'));`,
		`ALTER TABLE products ADD COLUMN sync_status TEXT NOT NULL DEFAULT 'pending' CHECK(sync_status IN ('pending', 'synced', 'failed'));`,
		`ALTER TABLE sales ADD COLUMN sync_status TEXT NOT NULL DEFAULT 'pending' CHECK(sync_status IN ('pending', 'synced', 'failed'));`,
		`ALTER TABLE products ADD COLUMN category_id INTEGER REFERENCES categories(id);`,
		`CREATE TABLE IF NOT EXISTS categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			description TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			sync_status TEXT NOT NULL DEFAULT 'pending' CHECK(sync_status IN ('pending', 'synced', 'failed'))
		);`,
		`CREATE TABLE IF NOT EXISTS customers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT,
			phone TEXT,
			address TEXT,
			notes TEXT,
			active BOOLEAN NOT NULL DEFAULT 1,
			sync_status TEXT NOT NULL DEFAULT 'pending' CHECK(sync_status IN ('pending', 'synced', 'failed')),
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`ALTER TABLE sales ADD COLUMN customer_id INTEGER REFERENCES customers(id);`,
		`ALTER TABLE sales ADD COLUMN payment_reference TEXT;`,
		`ALTER TABLE shifts ADD COLUMN declared_cash INTEGER;`,
		`ALTER TABLE shifts ADD COLUMN expected_cash INTEGER;`,
		`ALTER TABLE shifts ADD COLUMN difference INTEGER;`,
		`ALTER TABLE cash_movements ADD COLUMN type TEXT NOT NULL DEFAULT 'out';`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			action TEXT NOT NULL,
			description TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT
		);`,
	}

	for _, m := range migrations {
		// Ignore errors if column already exists (SQLite error "duplicate column name")
		if _, err := d.db.ExecContext(ctx, m); err != nil {
			// SQLite returns error code 1 for "duplicate column name"
			// We just continue silently if the column already exists
		}
	}

	return nil
}

// GetDB returns the underlying *sql.DB connection
func (d *Database) GetDB() *sql.DB {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.db
}

// Close closes the database connection
func (d *Database) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// HealthCheck verifies the database is accessible
func (d *Database) HealthCheck(ctx context.Context) error {
	return d.db.PingContext(ctx)
}
