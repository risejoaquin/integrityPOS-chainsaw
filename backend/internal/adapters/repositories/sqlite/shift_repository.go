package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"integritypos-backend/internal/core/domain"
)

// ShiftRepository implements the domain.ShiftRepository interface
type ShiftRepository struct {
	db *sql.DB
}

// NewShiftRepository creates a new shift repository
func NewShiftRepository(db *sql.DB) *ShiftRepository {
	return &ShiftRepository{db: db}
}

// Create creates a new shift
func (r *ShiftRepository) Create(ctx context.Context, shift *domain.Shift) error {
	now := time.Now().UTC()
	shift.CreatedAt = now
	shift.UpdatedAt = now
	shift.OpenedAt = now

	result, err := r.db.ExecContext(
		ctx,
		`INSERT INTO shifts (user_id, opened_at, open_balance, notes, created_at, updated_at, sync_status)
		 VALUES (?, ?, ?, ?, ?, ?, 'pending')`,
		shift.UserID,
		shift.OpenedAt,
		shift.OpenBalance,
		shift.Notes,
		shift.CreatedAt,
		shift.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create shift: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	shift.ID = id
	return nil
}

// Get retrieves a shift by ID
func (r *ShiftRepository) Get(ctx context.Context, id int64) (*domain.Shift, error) {
	shift := &domain.Shift{}

	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, user_id, opened_at, closed_at, open_balance, close_balance,
		        declared_cash, expected_cash, difference,
		        notes, created_at, updated_at, sync_status
		 FROM shifts WHERE id = ?`,
		id,
	).Scan(
		&shift.ID, &shift.UserID, &shift.OpenedAt, &shift.ClosedAt,
		&shift.OpenBalance, &shift.CloseBalance,
		&shift.DeclaredCash, &shift.ExpectedCash, &shift.Difference,
		&shift.Notes, &shift.CreatedAt, &shift.UpdatedAt, &shift.SyncStatus,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("shift not found")
		}
		return nil, fmt.Errorf("failed to get shift: %w", err)
	}

	return shift, nil
}

// GetActiveByUser gets the active (unclosed) shift for a user
func (r *ShiftRepository) GetActiveByUser(ctx context.Context, userID int64) (*domain.Shift, error) {
	shift := &domain.Shift{}

	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, user_id, opened_at, closed_at, open_balance, close_balance,
		        declared_cash, expected_cash, difference,
		        notes, created_at, updated_at, sync_status
		 FROM shifts WHERE user_id = ? AND closed_at IS NULL
		 ORDER BY opened_at DESC LIMIT 1`,
		userID,
	).Scan(
		&shift.ID, &shift.UserID, &shift.OpenedAt, &shift.ClosedAt,
		&shift.OpenBalance, &shift.CloseBalance,
		&shift.DeclaredCash, &shift.ExpectedCash, &shift.Difference,
		&shift.Notes, &shift.CreatedAt, &shift.UpdatedAt, &shift.SyncStatus,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("no active shift found")
		}
		return nil, fmt.Errorf("failed to get active shift: %w", err)
	}

	return shift, nil
}

// Update updates a shift (including arqueo fields)
func (r *ShiftRepository) Update(ctx context.Context, shift *domain.Shift) error {
	shift.UpdatedAt = time.Now().UTC()

	result, err := r.db.ExecContext(
		ctx,
		`UPDATE shifts
		 SET user_id = ?, opened_at = ?, closed_at = ?, open_balance = ?, close_balance = ?,
		     declared_cash = ?, expected_cash = ?, difference = ?,
		     notes = ?, updated_at = ?, sync_status = 'pending'
		 WHERE id = ?`,
		shift.UserID, shift.OpenedAt, shift.ClosedAt, shift.OpenBalance, shift.CloseBalance,
		shift.DeclaredCash, shift.ExpectedCash, shift.Difference,
		shift.Notes, shift.UpdatedAt, shift.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update shift: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("shift not found")
	}

	return nil
}

// List lists shifts with optional filters
func (r *ShiftRepository) List(ctx context.Context, filters map[string]interface{}) ([]*domain.Shift, error) {
	query := `SELECT id, user_id, opened_at, closed_at, open_balance, close_balance,
	                 declared_cash, expected_cash, difference,
	                 notes, created_at, updated_at, sync_status
	          FROM shifts WHERE 1=1`
	var args []interface{}

	if userID, ok := filters["user_id"]; ok {
		query += " AND user_id = ?"
		args = append(args, userID)
	}

	if closed, ok := filters["closed"]; ok {
		if closed.(bool) {
			query += " AND closed_at IS NOT NULL"
		} else {
			query += " AND closed_at IS NULL"
		}
	}

	query += " ORDER BY opened_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query shifts: %w", err)
	}
	defer rows.Close()

	var shifts []*domain.Shift
	for rows.Next() {
		shift := &domain.Shift{}
		if err := rows.Scan(
			&shift.ID, &shift.UserID, &shift.OpenedAt, &shift.ClosedAt,
			&shift.OpenBalance, &shift.CloseBalance,
			&shift.DeclaredCash, &shift.ExpectedCash, &shift.Difference,
			&shift.Notes, &shift.CreatedAt, &shift.UpdatedAt, &shift.SyncStatus,
		); err != nil {
			return nil, fmt.Errorf("failed to scan shift: %w", err)
		}
		shifts = append(shifts, shift)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating shifts: %w", err)
	}

	return shifts, nil
}

// GetPendingUnsafe returns pending shifts for sync
func (r *ShiftRepository) GetPendingUnsafe(ctx context.Context) ([]*domain.Shift, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, opened_at, closed_at, open_balance, close_balance,
		        declared_cash, expected_cash, difference,
		        notes, created_at, updated_at, sync_status
		 FROM shifts WHERE sync_status = 'pending' ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending shifts: %w", err)
	}
	defer rows.Close()

	var shifts []*domain.Shift
	for rows.Next() {
		s := &domain.Shift{}
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.OpenedAt, &s.ClosedAt,
			&s.OpenBalance, &s.CloseBalance,
			&s.DeclaredCash, &s.ExpectedCash, &s.Difference,
			&s.Notes, &s.CreatedAt, &s.UpdatedAt, &s.SyncStatus,
		); err != nil {
			return nil, fmt.Errorf("failed to scan shift: %w", err)
		}
		shifts = append(shifts, s)
	}
	return shifts, rows.Err()
}

// MarkShiftSynced marks a shift as synced
func (r *ShiftRepository) MarkShiftSynced(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE shifts SET sync_status = 'synced', updated_at = ? WHERE id = ?`,
		time.Now().UTC(), id)
	return err
}
