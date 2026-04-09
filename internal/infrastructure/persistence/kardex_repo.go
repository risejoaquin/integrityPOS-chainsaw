package persistence

import (
	"context"
	"fmt"

	"github.com/intigritypos/integritypos/internal/domain"
)

type KardexRepo struct {
	// Será usada a través del puerto TxPort en transacciones
}

func NewKardexRepo() *KardexRepo {
	return &KardexRepo{}
}

func (r *KardexRepo) Record(ctx context.Context, tx domain.TxPort, entry domain.KardexEntry) error {
	err := tx.Exec(`
        INSERT INTO inventory_kardex
        (id, sku, movement_type, quantity, cost_cents, balance_cents, reference_id, notes, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))`,
		entry.ID, entry.SKU, entry.MovementType, int64(entry.Quantity),
		entry.CostCents, int64(entry.BalanceCents), entry.ReferenceID, entry.Notes)
	if err != nil {
		return fmt.Errorf("error registrando kardex: %w", err)
	}
	return nil
}

func (r *KardexRepo) GetBySKU(ctx context.Context, sku string, limit int) ([]*domain.KardexEntry, error) {
	// Este método requiere acceso a la DB directamente, no via tx.
	// Para implementar esto, necesítamos pasar db al repo o usar un contexto diferente.
	// Por ahora, retorna error: usa Record() para escribir, no para leer en el mismo flujo.
	return nil, fmt.Errorf("GetBySKU no implementado en v1 (lee-después-de-escribir requiere flush)")
}

// RecordMultiple permite registrar múltiples movimientos de kardex en una sola transacción.
func (r *KardexRepo) RecordMultiple(ctx context.Context, tx domain.TxPort, entries []domain.KardexEntry) error {
	for _, entry := range entries {
		if err := r.Record(ctx, tx, entry); err != nil {
			return err
		}
	}
	return nil
}
