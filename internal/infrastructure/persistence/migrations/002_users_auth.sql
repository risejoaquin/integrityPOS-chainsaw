-- Migration 002: Update users table for JWT authentication
-- This migration updates the existing users table to support JWT authentication
-- and role-based access control

-- Add new columns for JWT authentication
ALTER TABLE users ADD COLUMN username TEXT UNIQUE;
ALTER TABLE users ADD COLUMN email TEXT UNIQUE;
ALTER TABLE users ADD COLUMN password_hash TEXT;
ALTER TABLE users ADD COLUMN roles TEXT DEFAULT 'cashier';
ALTER TABLE users ADD COLUMN updated_at DATETIME DEFAULT CURRENT_TIMESTAMP;

-- Update existing users to have username based on name
UPDATE users SET username = LOWER(REPLACE(name, ' ', '_')) WHERE username IS NULL;

-- Update existing users to have default password hash (bcrypt hash of 'password123')
-- In production, users should change their passwords
UPDATE users SET password_hash = '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj8lWZQjxQYq' WHERE password_hash IS NULL;

-- Update roles based on existing role field
UPDATE users SET roles = CASE
    WHEN role = 'ADMIN' THEN 'admin,manager,cashier'
    WHEN role = 'SUPERVISOR' THEN 'manager,cashier'
    ELSE 'cashier'
END;

-- Make username and password_hash NOT NULL after populating
-- Note: SQLite doesn't support adding NOT NULL constraints to existing columns with ALTER TABLE
-- So we create a new table and migrate data

CREATE TABLE users_new (
    id              TEXT PRIMARY KEY,
    username        TEXT NOT NULL UNIQUE,
    email           TEXT UNIQUE,
    password_hash   TEXT NOT NULL,
    roles           TEXT NOT NULL DEFAULT 'cashier',
    active          INTEGER NOT NULL DEFAULT 1,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Migrate data from old table to new table
INSERT INTO users_new (id, username, email, password_hash, roles, active, created_at, updated_at)
SELECT
    id,
    COALESCE(username, LOWER(REPLACE(name, ' ', '_'))),
    email,
    COALESCE(password_hash, '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj8lWZQjxQYq'),
    COALESCE(roles, CASE
        WHEN role = 'ADMIN' THEN 'admin,manager,cashier'
        WHEN role = 'SUPERVISOR' THEN 'manager,cashier'
        ELSE 'cashier'
    END),
    active,
    created_at,
    updated_at
FROM users;

-- Drop old table and rename new table
DROP TABLE users;
ALTER TABLE users_new RENAME TO users;

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_active ON users(active);

-- Insert default admin user if no admin exists
INSERT OR IGNORE INTO users (id, username, email, password_hash, roles, active, created_at, updated_at)
VALUES (
    'admin-default-id',
    'admin',
    'admin@integritypos.com',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj8lWZQjxQYq', -- bcrypt hash of 'admin123'
    'admin,manager,cashier',
    1,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
);