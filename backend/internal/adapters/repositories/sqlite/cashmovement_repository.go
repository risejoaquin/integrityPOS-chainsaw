package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"integritypos-backend/internal/core/domain"
)

// CashMovementRepository manages cash_movements table.
type CashMovementRepository struct {
	db *sql.DB
}

// NewCashMovementRepository creates a new CashMovementRepository.
func NewCashMovementRepository(db *sql.DB) *CashMovementRepository {
	return &CashMovementRepository{db: db}
}

// Create records a new cash movement (in = injection, out = expense).
func (r *CashMovementRepository) Create(ctx context.Context, cm *domain.CashMovement) error {
	now := time.Now().UTC()
	cm.CreatedAt = now

	res, err := r.db.ExecContext(ctx,
		`INSERT INTO cash_movements (shift_id, user_id, type, amount, reason, sync_status, created_at)
		 VALUES (?, ?, ?, ?, ?, 'pending', ?)`,
		cm.ShiftID, cm.UserID, cm.Type, cm.Amount, cm.Reason, now)
	if err != nil {
		return fmt.Errorf("failed to create cash movement: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	cm.ID = id
	return nil
}

// GetNetCashForShift returns net cash impact: SUM('in') - SUM('out') for a shift.
func (r *CashMovementRepository) GetNetCashForShift(ctx context.Context, shiftID int64) (int64, error) {
	var net int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(CASE WHEN type = 'in' THEN amount ELSE 0 END), 0) -
		        COALESCE(SUM(CASE WHEN type = 'out' THEN amount ELSE 0 END), 0)
		 FROM cash_movements WHERE shift_id = ?`, shiftID).Scan(&net)
	if err != nil {
		return 0, fmt.Errorf("failed to get net cash: %w", err)
	}
	return net, nil
}

// ListByShift returns all cash movements for a given shift.
func (r *CashMovementRepository) ListByShift(ctx context.Context, shiftID int64) ([]*domain.CashMovement, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, shift_id, user_id, type, amount, reason, sync_status, created_at
		 FROM cash_movements WHERE shift_id = ? ORDER BY created_at ASC`, shiftID)
	if err != nil {
		return nil, fmt.Errorf("failed to query cash movements: %w", err)
	}
	defer rows.Close()

	var items []*domain.CashMovement
	for rows.Next() {
		cm := &domain.CashMovement{}
		if err := rows.Scan(&cm.ID, &cm.ShiftID, &cm.UserID, &cm.Type, &cm.Amount, &cm.Reason, &cm.SyncStatus, &cm.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan cash movement: %w", err)
		}
		items = append(items, cm)
	}
	return items, rows.Err()
}

// GetPendingUnsafe returns pending cash movements for upstream sync.
func (r *CashMovementRepository) GetPendingUnsafe(ctx context.Context) ([]*domain.CashMovement, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, shift_id, user_id, type, amount, reason, sync_status, created_at
		 FROM cash_movements WHERE sync_status = 'pending' ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending cash movements: %w", err)
	}
	defer rows.Close()

	var items []*domain.CashMovement
	for rows.Next() {
		cm := &domain.CashMovement{}
		if err := rows.Scan(&cm.ID, &cm.ShiftID, &cm.UserID, &cm.Type, &cm.Amount, &cm.Reason, &cm.SyncStatus, &cm.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan cash movement: %w", err)
		}
		items = append(items, cm)
	}
	return items, rows.Err()
}

// MarkSynced marks a cash movement as synced.
func (r *CashMovementRepository) MarkSynced(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE cash_movements SET sync_status = 'synced' WHERE id = ?`, id)
	return err
}
