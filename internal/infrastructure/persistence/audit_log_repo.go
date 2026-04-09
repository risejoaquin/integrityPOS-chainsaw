package persistence

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/intigritypos/integritypos/internal/domain"
)

type AuditLogRepo struct{}

func NewAuditLogRepo() *AuditLogRepo {
	return &AuditLogRepo{}
}

func (r *AuditLogRepo) Record(ctx context.Context, tx domain.TxPort, userID, action, description, metadata string) error {
	err := tx.Exec(`
        INSERT INTO audit_logs (id, user_id, action, description, metadata, created_at)
        VALUES (?, ?, ?, ?, ?, datetime('now'))`,
		uuid.New().String(),
		userID, action, description, metadata)
	if err != nil {
		return fmt.Errorf("error registrando auditoría: %w", err)
	}
	return nil
}

// RecordSaleCreated registra la creación de una venta.
func (r *AuditLogRepo) RecordSaleCreated(ctx context.Context, tx domain.TxPort, cajeroID, saleID string, totalCents int64) error {
	metadata := fmt.Sprintf(`{"sale_id":"%s","total_cents":%d}`, saleID, totalCents)
	return r.Record(ctx, tx, cajeroID, "SALE_CREATED", fmt.Sprintf("Venta registrada: %s", saleID), metadata)
}

// RecordSaleCancelled registra la cancelación de una venta.
func (r *AuditLogRepo) RecordSaleCancelled(ctx context.Context, tx domain.TxPort, cajeroID, saleID, reason string) error {
	metadata := fmt.Sprintf(`{"sale_id":"%s","reason":"%s"}`, saleID, reason)
	return r.Record(ctx, tx, cajeroID, "SALE_CANCELLED", fmt.Sprintf("Venta cancelada: %s", saleID), metadata)
}

// RecordSessionOpened registra la apertura de sesión de caja.
func (r *AuditLogRepo) RecordSessionOpened(ctx context.Context, tx domain.TxPort, cajeroID, sessionID string, initialCash int64) error {
	metadata := fmt.Sprintf(`{"session_id":"%s","initial_cash":%d}`, sessionID, initialCash)
	return r.Record(ctx, tx, cajeroID, "SESSION_OPENED", fmt.Sprintf("Sesión abierta: %s", sessionID), metadata)
}

// RecordSessionClosed registra el cierre de sesión de caja.
func (r *AuditLogRepo) RecordSessionClosed(ctx context.Context, tx domain.TxPort, cajeroID, sessionID string, expectedCash, realCash, difference int64) error {
	metadata := fmt.Sprintf(`{"session_id":"%s","expected_cash":%d,"real_cash":%d,"difference":%d}`, sessionID, expectedCash, realCash, difference)
	return r.Record(ctx, tx, cajeroID, "SESSION_CLOSED", fmt.Sprintf("Sesión cerrada: %s", sessionID), metadata)
}
