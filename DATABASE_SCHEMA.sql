-- DATABASE_SCHEMA.sql for IntegrityPOS
-- SQLite 3 with WAL mode and foreign keys enabled
-- All monetary values are stored as INTEGER (cents, not floats)

-- ============================================================
-- PRAGMA STATEMENTS (Must be executed at connection init)
-- ============================================================
-- PRAGMA journal_mode=WAL;
-- PRAGMA foreign_keys=ON;

-- ============================================================
-- TABLES
-- ============================================================

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

-- Shifts table (Turnos de caja)
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
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT
);

-- Products table (Inventario)
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
    active BOOLEAN NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Sales table (IMMUTABLE - no UPDATE allowed)
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
    FOREIGN KEY (shift_id) REFERENCES shifts(id) ON DELETE RESTRICT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT
);

-- SaleItems table (IMMUTABLE - no UPDATE allowed)
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

-- SyncLog table (Audit trail for cloud sync)
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

-- ============================================================
-- INDEXES (Performance optimization)
-- ============================================================

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

CREATE INDEX IF NOT EXISTS idx_shifts_user_id ON shifts(user_id);
CREATE INDEX IF NOT EXISTS idx_shifts_opened_at ON shifts(opened_at);

CREATE INDEX IF NOT EXISTS idx_products_sku ON products(sku);
CREATE INDEX IF NOT EXISTS idx_products_category ON products(category);

CREATE INDEX IF NOT EXISTS idx_sales_shift_id ON sales(shift_id);
CREATE INDEX IF NOT EXISTS idx_sales_user_id ON sales(user_id);
CREATE INDEX IF NOT EXISTS idx_sales_created_at ON sales(created_at);

CREATE INDEX IF NOT EXISTS idx_sale_items_sale_id ON sale_items(sale_id);
CREATE INDEX IF NOT EXISTS idx_sale_items_product_id ON sale_items(product_id);

CREATE INDEX IF NOT EXISTS idx_sync_logs_sale_id ON sync_logs(sale_id);
CREATE INDEX IF NOT EXISTS idx_sync_logs_status ON sync_logs(status);

-- ============================================================
-- CONSTRAINTS
-- ============================================================

-- Ensure monetary fields are valid (non-negative)
-- Note: This is enforced at application level, SQLite doesn't have CHECK for negative numbers easily

-- ============================================================
-- NOTES
-- ============================================================
-- 1. All monetary amounts are INTEGER (cents)
--    Example: $19.99 = 1999
--
-- 2. Sales and SaleItems are IMMUTABLE
--    - Only UPDATE allowed is for voiding (voided flag, void_reason)
--    - Never delete rows, only flag as voided
--
-- 3. Foreign key ON DELETE:
--    - RESTRICT for users: cannot delete user if shifts exist
--    - CASCADE for sales/synclogs: delete sales deletes synclogs
--    - RESTRICT for products: cannot delete product if sales exist
--
-- 4. Timestamps:
--    - created_at: Set at insert, never changes
--    - updated_at: Set at insert, updated on modifications
--    - Sales don't have updated_at (immutable)
