package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"integritypos-backend/internal/core/domain"
)

// ProductRepository implements ports.ProductRepository
type ProductRepository struct {
	db *sql.DB
}

// NewProductRepository creates a new ProductRepository
func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

// Create creates a new product
func (r *ProductRepository) Create(ctx context.Context, product *domain.Product) error {
	now := time.Now().UTC()
	product.CreatedAt = now
	product.UpdatedAt = now

	res, err := r.db.ExecContext(ctx, `INSERT INTO products (sku, name, description, barcode, price, cost, quantity, category, active, created_at, updated_at, sync_status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending')`,
		product.SKU, product.Name, product.Description, product.Barcode, product.Price, product.Cost, product.Quantity, product.Category, product.Active, product.CreatedAt, product.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create product: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	product.ID = id
	return nil
}

// Get retrieves a product by ID
func (r *ProductRepository) Get(ctx context.Context, id int64) (*domain.Product, error) {
	p := &domain.Product{}
	err := r.db.QueryRowContext(ctx, `SELECT id, sku, name, description, barcode, price, cost, quantity, category, active, sync_status, created_at, updated_at FROM products WHERE id = ?`, id).Scan(
		&p.ID, &p.SKU, &p.Name, &p.Description, &p.Barcode, &p.Price, &p.Cost, &p.Quantity, &p.Category, &p.Active, &p.SyncStatus, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("product not found")
		}
		return nil, fmt.Errorf("failed to get product: %w", err)
	}
	return p, nil
}

// GetByBarcode retrieves a product by barcode
func (r *ProductRepository) GetByBarcode(ctx context.Context, barcode string) (*domain.Product, error) {
	p := &domain.Product{}
	err := r.db.QueryRowContext(ctx, `SELECT id, sku, name, description, barcode, price, cost, quantity, category, active, sync_status, created_at, updated_at FROM products WHERE barcode = ?`, barcode).Scan(
		&p.ID, &p.SKU, &p.Name, &p.Description, &p.Barcode, &p.Price, &p.Cost, &p.Quantity, &p.Category, &p.Active, &p.SyncStatus, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("product not found")
		}
		return nil, fmt.Errorf("failed to get product by barcode: %w", err)
	}
	return p, nil
}

// GetBySKU retrieves a product by SKU
func (r *ProductRepository) GetBySKU(ctx context.Context, sku string) (*domain.Product, error) {
	p := &domain.Product{}
	err := r.db.QueryRowContext(ctx, `SELECT id, sku, name, description, barcode, price, cost, quantity, category, active, sync_status, created_at, updated_at FROM products WHERE sku = ?`, sku).Scan(
		&p.ID, &p.SKU, &p.Name, &p.Description, &p.Barcode, &p.Price, &p.Cost, &p.Quantity, &p.Category, &p.Active, &p.SyncStatus, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("product not found")
		}
		return nil, fmt.Errorf("failed to get product by sku: %w", err)
	}
	return p, nil
}

// Update updates a product
func (r *ProductRepository) Update(ctx context.Context, product *domain.Product) error {
	product.UpdatedAt = time.Now().UTC()
	res, err := r.db.ExecContext(ctx, `UPDATE products SET sku = ?, name = ?, description = ?, barcode = ?, price = ?, cost = ?, quantity = ?, category = ?, active = ?, updated_at = ?, sync_status = 'pending' WHERE id = ?`,
		product.SKU, product.Name, product.Description, product.Barcode, product.Price, product.Cost, product.Quantity, product.Category, product.Active, product.UpdatedAt, product.ID)
	if err != nil {
		return fmt.Errorf("failed to update product: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("product not found")
	}
	return nil
}

// List lists products with optional filters
func (r *ProductRepository) List(ctx context.Context, filters map[string]interface{}) ([]*domain.Product, error) {
	query := `SELECT id, sku, name, description, barcode, price, cost, quantity, category, active, sync_status, created_at, updated_at FROM products WHERE 1=1`
	var args []interface{}
	if category, ok := filters["category"]; ok {
		query += " AND category = ?"
		args = append(args, category)
	}
	if active, ok := filters["active"]; ok {
		query += " AND active = ?"
		args = append(args, active)
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query products: %w", err)
	}
	defer rows.Close()
	var products []*domain.Product
	for rows.Next() {
		p := &domain.Product{}
		if err := rows.Scan(&p.ID, &p.SKU, &p.Name, &p.Description, &p.Barcode, &p.Price, &p.Cost, &p.Quantity, &p.Category, &p.Active, &p.SyncStatus, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}
		products = append(products, p)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating products: %w", err)
	}
	return products, nil
}

// UpdateInventory updates product quantity (atomic by checking current quantity)
func (r *ProductRepository) UpdateInventory(ctx context.Context, productID int64, quantityDelta int64) error {
	// Start a transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Get current quantity
	var current int64
	if err := tx.QueryRowContext(ctx, `SELECT quantity FROM products WHERE id = ?`, productID).Scan(&current); err != nil {
		tx.Rollback()
		if err == sql.ErrNoRows {
			return fmt.Errorf("product not found")
		}
		return fmt.Errorf("failed to query current quantity: %w", err)
	}

	newQty := current + quantityDelta
	if newQty < 0 {
		tx.Rollback()
		return fmt.Errorf("insufficient inventory")
	}

	_, err = tx.ExecContext(ctx, `UPDATE products SET quantity = ?, updated_at = ?, sync_status = 'pending' WHERE id = ?`, newQty, time.Now().UTC(), productID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update inventory: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit inventory update: %w", err)
	}
	return nil
}

// GetPendingUnsafe returns pending products for sync (no transaction safety needed)
func (r *ProductRepository) GetPendingUnsafe(ctx context.Context) ([]*domain.Product, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, sku, name, description, barcode, price, cost, quantity, category, active, sync_status, created_at, updated_at
		 FROM products WHERE sync_status = 'pending' ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending products: %w", err)
	}
	defer rows.Close()
	var products []*domain.Product
	for rows.Next() {
		p := &domain.Product{}
		if err := rows.Scan(&p.ID, &p.SKU, &p.Name, &p.Description, &p.Barcode, &p.Price, &p.Cost, &p.Quantity, &p.Category, &p.Active, &p.SyncStatus, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

// MarkProductSynced marks a product as synced
func (r *ProductRepository) MarkProductSynced(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE products SET sync_status = 'synced', updated_at = ? WHERE id = ?`,
		time.Now().UTC(), id)
	return err
}

// UpsertFromCloud inserts or updates a product received from cloud sync.
// It sets sync_status to 'synced' to prevent the upstream worker from re-pushing it.
func (r *ProductRepository) UpsertFromCloud(ctx context.Context, p *domain.Product) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO products (id, sku, name, description, barcode, price, cost, quantity, category, category_id, active, created_at, updated_at, sync_status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'synced')
		ON CONFLICT(id) DO UPDATE SET
			sku          = excluded.sku,
			name         = excluded.name,
			description  = excluded.description,
			barcode      = excluded.barcode,
			price        = excluded.price,
			cost         = excluded.cost,
			quantity     = excluded.quantity,
			category     = excluded.category,
			category_id  = excluded.category_id,
			active       = excluded.active,
			updated_at   = excluded.updated_at,
			sync_status  = 'synced'
	`,
		p.ID, p.SKU, p.Name, p.Description, p.Barcode, p.Price, p.Cost, p.Quantity, p.Category, p.CategoryID, p.Active, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to upsert product from cloud: %w", err)
	}
	return nil
}
