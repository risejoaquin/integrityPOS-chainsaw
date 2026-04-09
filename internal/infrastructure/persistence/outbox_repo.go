package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/intigritypos/integritypos/internal/domain"
)

type OutboxRepo struct {
	db *sql.DB
}

func NewOutboxRepo(db *sql.DB) *OutboxRepo {
	return &OutboxRepo{db: db}
}

func (r *OutboxRepo) Enqueue(ctx context.Context, tx domain.TxPort, entry domain.OutboxEntry) error {
	err := tx.Exec(`
        INSERT INTO sync_outbox (id, entity_type, entity_id, payload, priority, status, attempts, created_at)
        VALUES (?, ?, ?, ?, ?, 'pending', 0, datetime('now'))`,
		entry.ID, entry.EntityType, entry.EntityID, entry.Payload, entry.Priority)
	if err != nil {
		return fmt.Errorf("error encolando en outbox: %w", err)
	}
	return nil
}

type OutboxEntryRow struct {
	ID         string
	EntityType string
	EntityID   string
	Payload    string
	Priority   int
	Attempts   int
	Status     string
}

func (r *OutboxRepo) GetPending(ctx context.Context, limit int) ([]*OutboxEntryRow, error) {
	rows, err := r.db.QueryContext(ctx, `
        SELECT id, entity_type, entity_id, payload, priority, attempts, status
        FROM sync_outbox
        WHERE status = 'pending'
        ORDER BY priority ASC, created_at ASC
        LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("error leyendo outbox pendiente: %w", err)
	}
	defer rows.Close()

	var entries []*OutboxEntryRow
	for rows.Next() {
		var e OutboxEntryRow
		if err := rows.Scan(&e.ID, &e.EntityType, &e.EntityID, &e.Payload, &e.Priority, &e.Attempts, &e.Status); err != nil {
			return nil, err
		}
		entries = append(entries, &e)
	}
	return entries, rows.Err()
}

func (r *OutboxRepo) MarkSynced(ctx context.Context, id, cloudID string) error {
	_, err := r.db.ExecContext(ctx, `
        UPDATE sync_outbox
        SET status = 'synced', cloud_id = ?, synced_at = datetime('now'), attempts = attempts + 1
        WHERE id = ?`, cloudID, id)
	return err
}

func (r *OutboxRepo) MarkFailed(ctx context.Context, id, errMsg string) error {
	_, err := r.db.ExecContext(ctx, `
        UPDATE sync_outbox
        SET status = 'failed', last_error = ?, attempts = attempts + 1
        WHERE id = ?`, errMsg, id)
	return err
}

func (r *OutboxRepo) MarkCorrupted(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `
        UPDATE sync_outbox
        SET status = 'corrupted'
        WHERE id = ?`, id)
	return err
}

func (r *OutboxRepo) GetStats(ctx context.Context) (map[string]interface{}, error) {
	var pending, synced, failed, corrupted int64
	row := r.db.QueryRowContext(ctx, `
        SELECT
            SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END),
            SUM(CASE WHEN status = 'synced' THEN 1 ELSE 0 END),
            SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END),
            SUM(CASE WHEN status = 'corrupted' THEN 1 ELSE 0 END)
        FROM sync_outbox`)
	if err := row.Scan(&pending, &synced, &failed, &corrupted); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"pending":    pending,
		"synced":     synced,
		"failed":     failed,
		"corrupted":  corrupted,
		"total":      pending + synced + failed + corrupted,
	}, nil
}

// EnqueueSale es un helper para enqueuer una venta recién creada.
func (r *OutboxRepo) EnqueueSale(ctx context.Context, tx domain.TxPort, saleID string, payload string) error {
	return r.Enqueue(ctx, tx, domain.OutboxEntry{
		ID:         "OBX-SALE-" + saleID,
		EntityType: "SALE",
		EntityID:   saleID,
		Payload:    payload,
		Priority:   1, // Ventas: máxima prioridad
	})
}

// EnqueueCashClose es un helper para enqueuer un cierre de caja.
func (r *OutboxRepo) EnqueueCashClose(ctx context.Context, tx domain.TxPort, sessionID string, payload string) error {
	return r.Enqueue(ctx, tx, domain.OutboxEntry{
		ID:         "OBX-CLOSE-" + sessionID,
		EntityType: "CASH_CLOSE",
		EntityID:   sessionID,
		Payload:    payload,
		Priority:   2, // Cierres: prioridad media
	})
}
