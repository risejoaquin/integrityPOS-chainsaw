package sqlite

import (
	"context"
	"fmt"
	"time"
)

// MarkPendingAsSynced updates all pending sync log entries for the given sale IDs to "synced" status
func (r *SyncLogRepository) MarkPendingAsSynced(ctx context.Context, saleIDs []int64) error {
	if len(saleIDs) == 0 {
		return nil
	}

	now := time.Now().UTC()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	for _, saleID := range saleIDs {
		_, err = tx.ExecContext(ctx, `UPDATE sync_logs SET status = 'synced', synced_at = ?, updated_at = ? WHERE sale_id = ? AND status = 'pending'`,
			now, now, saleID)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to mark sale %d as synced: %w", saleID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit sync update: %w", err)
	}
	return nil
}

// GetPendingSaleIDs retrieves sale IDs of all pending sync log entries
func (r *SyncLogRepository) GetPendingSaleIDs(ctx context.Context) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT sale_id FROM sync_logs WHERE status = 'pending'`)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending sync logs: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan pending sale id: %w", err)
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pending logs: %w", err)
	}
	return ids, nil
}
