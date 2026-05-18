package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"integritypos-backend/internal/core/domain"
)

// CustomerRepository manages customers table.
type CustomerRepository struct {
	db *sql.DB
}

// NewCustomerRepository creates a new CustomerRepository.
func NewCustomerRepository(db *sql.DB) *CustomerRepository {
	return &CustomerRepository{db: db}
}

// Create creates a new customer.
func (r *CustomerRepository) Create(ctx context.Context, c *domain.Customer) error {
	now := time.Now().UTC()
	c.CreatedAt = now
	c.UpdatedAt = now

	res, err := r.db.ExecContext(ctx, `INSERT INTO customers (name, email, phone, address, notes, active, created_at, updated_at, sync_status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'pending')`,
		c.Name, c.Email, c.Phone, c.Address, c.Notes, c.Active, c.CreatedAt, c.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create customer: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	c.ID = id
	return nil
}

// Get retrieves a customer by ID.
func (r *CustomerRepository) Get(ctx context.Context, id int64) (*domain.Customer, error) {
	c := &domain.Customer{}
	err := r.db.QueryRowContext(ctx, `SELECT id, name, email, phone, address, notes, active, sync_status, created_at, updated_at FROM customers WHERE id = ?`, id).Scan(
		&c.ID, &c.Name, &c.Email, &c.Phone, &c.Address, &c.Notes, &c.Active, &c.SyncStatus, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("customer not found")
		}
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}
	return c, nil
}

// List lists customers with optional filters.
func (r *CustomerRepository) List(ctx context.Context, filters map[string]interface{}) ([]*domain.Customer, error) {
	query := `SELECT id, name, email, phone, address, notes, active, sync_status, created_at, updated_at FROM customers WHERE 1=1`
	var args []interface{}
	if active, ok := filters["active"]; ok {
		query += " AND active = ?"
		args = append(args, active)
	}
	query += " ORDER BY name ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query customers: %w", err)
	}
	defer rows.Close()

	var customers []*domain.Customer
	for rows.Next() {
		c := &domain.Customer{}
		if err := rows.Scan(&c.ID, &c.Name, &c.Email, &c.Phone, &c.Address, &c.Notes, &c.Active, &c.SyncStatus, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan customer: %w", err)
		}
		customers = append(customers, c)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating customers: %w", err)
	}
	return customers, nil
}

// GetPendingUnsafe returns pending customers for upstream sync.
func (r *CustomerRepository) GetPendingUnsafe(ctx context.Context) ([]*domain.Customer, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, email, phone, address, notes, active, sync_status, created_at, updated_at
		 FROM customers WHERE sync_status = 'pending' ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending customers: %w", err)
	}
	defer rows.Close()
	var customers []*domain.Customer
	for rows.Next() {
		c := &domain.Customer{}
		if err := rows.Scan(&c.ID, &c.Name, &c.Email, &c.Phone, &c.Address, &c.Notes, &c.Active, &c.SyncStatus, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan customer: %w", err)
		}
		customers = append(customers, c)
	}
	return customers, rows.Err()
}

// MarkCustomerSynced marks a customer as synced.
func (r *CustomerRepository) MarkCustomerSynced(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE customers SET sync_status = 'synced', updated_at = ? WHERE id = ?`,
		time.Now().UTC(), id)
	return err
}

// UpsertFromCloud inserts or updates a customer received from cloud sync.
func (r *CustomerRepository) UpsertFromCloud(ctx context.Context, c *domain.Customer) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO customers (id, name, email, phone, address, notes, active, created_at, updated_at, sync_status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'synced')
		ON CONFLICT(id) DO UPDATE SET
			name       = excluded.name,
			email      = excluded.email,
			phone      = excluded.phone,
			address    = excluded.address,
			notes      = excluded.notes,
			active     = excluded.active,
			updated_at = excluded.updated_at,
			sync_status = 'synced'
	`, c.ID, c.Name, c.Email, c.Phone, c.Address, c.Notes, c.Active, c.CreatedAt, c.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to upsert customer from cloud: %w", err)
	}
	return nil
}
