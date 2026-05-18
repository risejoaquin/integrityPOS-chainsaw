package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"integritypos-backend/internal/core/domain"
)

// AuditLogRepository manages audit_logs table.
type AuditLogRepository struct {
	db *sql.DB
}

// NewAuditLogRepository creates a new AuditLogRepository.
func NewAuditLogRepository(db *sql.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// Create records a new audit log entry.
func (r *AuditLogRepository) Create(ctx context.Context, al *domain.AuditLog) error {
	now := time.Now().UTC()
	al.CreatedAt = now

	res, err := r.db.ExecContext(ctx,
		`INSERT INTO audit_logs (user_id, action, description, created_at) VALUES (?, ?, ?, ?)`,
		al.UserID, al.Action, al.Description, now)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	al.ID = id
	return nil
}

// List returns audit logs ordered by most recent.
func (r *AuditLogRepository) List(ctx context.Context, limit int) ([]*domain.AuditLog, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, action, description, created_at FROM audit_logs ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*domain.AuditLog
	for rows.Next() {
		al := &domain.AuditLog{}
		if err := rows.Scan(&al.ID, &al.UserID, &al.Action, &al.Description, &al.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, al)
	}
	return logs, rows.Err()
}
