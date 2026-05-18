package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"integritypos-backend/internal/core/domain"
)

// SaleRepository implements ports.SaleRepository
type SaleRepository struct {
	db *sql.DB
}

// NewSaleRepository creates a new SaleRepository
func NewSaleRepository(db *sql.DB) *SaleRepository {
	return &SaleRepository{db: db}
}

// CreateWithItems creates a sale and its items atomically and returns the sale ID
func (r *SaleRepository) CreateWithItems(ctx context.Context, sale *domain.Sale, items []*domain.SaleItem) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Insert sale
	now := time.Now().UTC()
	res, err := tx.ExecContext(ctx, `INSERT INTO sales (shift_id, user_id, customer_id, total, tax, subtotal, payment_method, payment_reference, notes, voided, created_at, sync_status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending')`,
		sale.ShiftID, sale.UserID, sale.CustomerID, sale.Total, sale.Tax, sale.Subtotal, sale.PaymentMethod, sale.PaymentReference, sale.Notes, 0, now)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to insert sale: %w", err)
	}

	saleID, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to get sale id: %w", err)
	}

	// Insert items and update inventory
	for _, item := range items {
		// Check product exists and quantity
		var currentQty int64
		if err := tx.QueryRowContext(ctx, `SELECT quantity FROM products WHERE id = ?`, item.ProductID).Scan(&currentQty); err != nil {
			tx.Rollback()
			if err == sql.ErrNoRows {
				return 0, fmt.Errorf("product not found: %d", item.ProductID)
			}
			return 0, fmt.Errorf("failed to query product quantity: %w", err)
		}

		if currentQty < item.Quantity {
			tx.Rollback()
			return 0, fmt.Errorf("insufficient inventory for product %d", item.ProductID)
		}

		// Insert sale item
		_, err := tx.ExecContext(ctx, `INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, total, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
			saleID, item.ProductID, item.Quantity, item.UnitPrice, item.Total, now)
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to insert sale item: %w", err)
		}

		// Update product quantity
		newQty := currentQty - item.Quantity
		_, err = tx.ExecContext(ctx, `UPDATE products SET quantity = ?, updated_at = ?, sync_status = 'pending' WHERE id = ?`, newQty, time.Now().UTC(), item.ProductID)
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to update product quantity: %w", err)
		}
	}

	// Insert sync log as pending (keep for backward compatibility)
	_, err = tx.ExecContext(ctx, `INSERT INTO sync_logs (sale_id, status, created_at, updated_at) VALUES (?, ?, ?, ?)`, saleID, "pending", now, now)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to insert sync log: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit tx: %w", err)
	}

	return saleID, nil
}

// Create creates a sale record (single insert)
func (r *SaleRepository) Create(ctx context.Context, sale *domain.Sale) error {
	res, err := r.db.ExecContext(ctx, `INSERT INTO sales (shift_id, user_id, total, tax, subtotal, payment_method, notes, voided, created_at, sync_status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending')`,
		sale.ShiftID, sale.UserID, sale.Total, sale.Tax, sale.Subtotal, sale.PaymentMethod, sale.Notes, 0, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to insert sale: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	sale.ID = id
	return nil
}

// Get retrieves a sale by ID
func (r *SaleRepository) Get(ctx context.Context, id int64) (*domain.Sale, error) {
	s := &domain.Sale{}
	err := r.db.QueryRowContext(ctx, `SELECT id, shift_id, user_id, customer_id, total, tax, subtotal, payment_method, payment_reference, notes, voided, void_reason, sync_status, created_at FROM sales WHERE id = ?`, id).Scan(
		&s.ID, &s.ShiftID, &s.UserID, &s.CustomerID, &s.Total, &s.Tax, &s.Subtotal, &s.PaymentMethod, &s.PaymentReference, &s.Notes, &s.Voided, &s.VoidReason, &s.SyncStatus, &s.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("sale not found")
		}
		return nil, fmt.Errorf("failed to get sale: %w", err)
	}
	return s, nil
}

// List lists sales with optional filters
func (r *SaleRepository) List(ctx context.Context, filters map[string]interface{}) ([]*domain.Sale, error) {
	query := `SELECT id, shift_id, user_id, total, tax, subtotal, payment_method, notes, voided, void_reason, sync_status, created_at FROM sales WHERE 1=1`
	var args []interface{}
	if shiftID, ok := filters["shift_id"]; ok {
		query += " AND shift_id = ?"
		args = append(args, shiftID)
	}
	if userID, ok := filters["user_id"]; ok {
		query += " AND user_id = ?"
		args = append(args, userID)
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query sales: %w", err)
	}
	defer rows.Close()
	var sales []*domain.Sale
	for rows.Next() {
		s := &domain.Sale{}
		if err := rows.Scan(&s.ID, &s.ShiftID, &s.UserID, &s.Total, &s.Tax, &s.Subtotal, &s.PaymentMethod, &s.Notes, &s.Voided, &s.VoidReason, &s.SyncStatus, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan sale: %w", err)
		}
		sales = append(sales, s)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sales: %w", err)
	}
	return sales, nil
}

// GetItems retrieves line items for a sale
func (r *SaleRepository) GetItems(ctx context.Context, saleID int64) ([]*domain.SaleItem, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, sale_id, product_id, quantity, unit_price, total, created_at FROM sale_items WHERE sale_id = ?`, saleID)
	if err != nil {
		return nil, fmt.Errorf("failed to query sale items: %w", err)
	}
	defer rows.Close()
	var items []*domain.SaleItem
	for rows.Next() {
		it := &domain.SaleItem{}
		if err := rows.Scan(&it.ID, &it.SaleID, &it.ProductID, &it.Quantity, &it.UnitPrice, &it.Total, &it.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan sale item: %w", err)
		}
		items = append(items, it)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sale items: %w", err)
	}
	return items, nil
}

// VoidSale marks sale as voided
func (r *SaleRepository) VoidSale(ctx context.Context, saleID int64, reason string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE sales SET voided = 1, void_reason = ?, updated_at = ?, sync_status = 'pending' WHERE id = ?`, reason, time.Now().UTC(), saleID)
	if err != nil {
		return fmt.Errorf("failed to void sale: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("sale not found")
	}
	return nil
}

// GetPendingUnsafe returns pending sales for sync (no transaction safety needed)
func (r *SaleRepository) GetPendingUnsafe(ctx context.Context) ([]*domain.Sale, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, shift_id, user_id, total, tax, subtotal, payment_method, notes, voided, void_reason, sync_status, created_at
		 FROM sales WHERE sync_status = 'pending' ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending sales: %w", err)
	}
	defer rows.Close()
	var sales []*domain.Sale
	for rows.Next() {
		s := &domain.Sale{}
		if err := rows.Scan(&s.ID, &s.ShiftID, &s.UserID, &s.Total, &s.Tax, &s.Subtotal, &s.PaymentMethod, &s.Notes, &s.Voided, &s.VoidReason, &s.SyncStatus, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan sale: %w", err)
		}
		sales = append(sales, s)
	}
	return sales, rows.Err()
}

// VoidSaleWithInventory atomically voids a sale and reverses inventory quantities.
// It uses a transaction to: update sale as voided, then add back stock for each item.
func (r *SaleRepository) VoidSaleWithInventory(ctx context.Context, saleID int64, reason string) error {
	// Fetch items first
	items, err := r.GetItems(ctx, saleID)
	if err != nil {
		return fmt.Errorf("failed to fetch sale items for void: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Mark sale as voided
	result, err := tx.ExecContext(ctx,
		`UPDATE sales SET voided = 1, void_reason = ?, sync_status = 'pending' WHERE id = ? AND voided = 0`,
		reason, saleID)
	if err != nil {
		return fmt.Errorf("failed to void sale: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("sale not found or already voided")
	}

	// Reverse inventory: add quantities back
	for _, item := range items {
		_, err = tx.ExecContext(ctx,
			`UPDATE products SET quantity = quantity + ?, updated_at = ?, sync_status = 'pending' WHERE id = ?`,
			item.Quantity, time.Now().UTC(), item.ProductID)
		if err != nil {
			return fmt.Errorf("failed to restore stock for product %d: %w", item.ProductID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit void transaction: %w", err)
	}
	return nil
}

// GetCashSalesTotalForShift returns the sum of 'total' for cash sales in a given shift (excluding voided)
func (r *SaleRepository) GetCashSalesTotalForShift(ctx context.Context, shiftID int64) (int64, error) {
	var total int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(total), 0) FROM sales WHERE shift_id = ? AND payment_method = 'cash' AND voided = 0`, shiftID).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to get cash sales total: %w", err)
	}
	return total, nil
}

// MarkSaleSynced marks a sale as synced
func (r *SaleRepository) MarkSaleSynced(ctx context.Context, id int64) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `UPDATE sales SET sync_status = 'synced' WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to mark sale synced: %w", err)
	}
	// Also update the sync_logs table for backward compatibility
	_, _ = r.db.ExecContext(ctx, `UPDATE sync_logs SET status = 'synced', synced_at = ?, updated_at = ? WHERE sale_id = ?`, now, now, id)
	return nil
}
