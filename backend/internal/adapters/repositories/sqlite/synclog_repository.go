package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"integritypos-backend/internal/core/domain"
)

// SyncLogRepository implements ports.SyncLogRepository
type SyncLogRepository struct {
	db *sql.DB
}

// NewSyncLogRepository creates a new SyncLogRepository
func NewSyncLogRepository(db *sql.DB) *SyncLogRepository {
	return &SyncLogRepository{db: db}
}

// Create creates a new sync log entry
func (r *SyncLogRepository) Create(ctx context.Context, logEntry *domain.SyncLog) error {
	now := time.Now().UTC()
	res, err := r.db.ExecContext(ctx, `INSERT INTO sync_logs (sale_id, status, error_message, synced_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		logEntry.SaleID, logEntry.Status, logEntry.ErrorMessage, logEntry.SyncedAt, now, now)
	if err != nil {
		return fmt.Errorf("failed to create sync log: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	logEntry.ID = id
	logEntry.CreatedAt = now
	logEntry.UpdatedAt = now
	return nil
}

// Update updates a sync log entry
func (r *SyncLogRepository) Update(ctx context.Context, logEntry *domain.SyncLog) error {
	logEntry.UpdatedAt = time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `UPDATE sync_logs SET status = ?, error_message = ?, synced_at = ?, updated_at = ? WHERE id = ?`,
		logEntry.Status, logEntry.ErrorMessage, logEntry.SyncedAt, logEntry.UpdatedAt, logEntry.ID)
	if err != nil {
		return fmt.Errorf("failed to update sync log: %w", err)
	}
	return nil
}

// GetPending retrieves all pending sync entries
func (r *SyncLogRepository) GetPending(ctx context.Context) ([]*domain.SyncLog, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, sale_id, status, error_message, synced_at, created_at, updated_at FROM sync_logs WHERE status = 'pending'`)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending sync logs: %w", err)
	}
	defer rows.Close()
	var logs []*domain.SyncLog
	for rows.Next() {
		lg := &domain.SyncLog{}
		if err := rows.Scan(&lg.ID, &lg.SaleID, &lg.Status, &lg.ErrorMessage, &lg.SyncedAt, &lg.CreatedAt, &lg.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan sync log: %w", err)
		}
		logs = append(logs, lg)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sync logs: %w", err)
	}
	return logs, nil
}

// PurgeSyncedBefore deletes all sync_log records that are 'synced' and older than the given cutoff time.
// Returns the number of deleted rows.
func (r *SyncLogRepository) PurgeSyncedBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM sync_logs WHERE status = 'synced' AND synced_at < ?`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to purge old sync logs: %w", err)
	}
	return res.RowsAffected()
}

// List lists sync logs with optional filters
func (r *SyncLogRepository) List(ctx context.Context, filters map[string]interface{}) ([]*domain.SyncLog, error) {
	query := `SELECT id, sale_id, status, error_message, synced_at, created_at, updated_at FROM sync_logs WHERE 1=1`
	var args []interface{}
	if status, ok := filters["status"]; ok {
		query += " AND status = ?"
		args = append(args, status)
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query sync logs: %w", err)
	}
	defer rows.Close()
	var logs []*domain.SyncLog
	for rows.Next() {
		lg := &domain.SyncLog{}
		if err := rows.Scan(&lg.ID, &lg.SaleID, &lg.Status, &lg.ErrorMessage, &lg.SyncedAt, &lg.CreatedAt, &lg.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan sync log: %w", err)
		}
		logs = append(logs, lg)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sync logs: %w", err)
	}
	return logs, nil
}
