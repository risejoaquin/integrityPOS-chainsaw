package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/intigritypos/integritypos/internal/domain"
	"github.com/intigritypos/integritypos/internal/infrastructure/persistence"
)

// IntegrationService orquesta el registro coordinado de ventas, kardex, auditoría y outbox.
type IntegrationService struct {
	saleRepo   domain.SaleRepository
	prodRepo   domain.ProductRepository
	kardexRepo *persistence.KardexRepo
	outboxRepo *persistence.OutboxRepo
	auditRepo  *persistence.AuditLogRepo
}

func NewIntegrationService(
	saleRepo domain.SaleRepository,
	prodRepo domain.ProductRepository,
	kardexRepo *persistence.KardexRepo,
	outboxRepo *persistence.OutboxRepo,
	auditRepo *persistence.AuditLogRepo,
) *IntegrationService {
	return &IntegrationService{
		saleRepo:   saleRepo,
		prodRepo:   prodRepo,
		kardexRepo: kardexRepo,
		outboxRepo: outboxRepo,
		auditRepo:  auditRepo,
	}
}

// RecordSaleWithIntegration graba una venta con kardex, auditoría y outbox en una sola transacción.
func (s *IntegrationService) RecordSaleWithIntegration(
	ctx context.Context,
	tx domain.TxPort,
	sale domain.Sale,
	kafexEntries []domain.KardexEntry,
) error {
	// 1. Guardar la venta
	if err := s.saleRepo.Save(ctx, tx, sale); err != nil {
		return fmt.Errorf("error guardando venta: %w", err)
	}

	// 2. Registrar movimientos en kardex
	for _, ke := range kafexEntries {
		if err := s.kardexRepo.Record(ctx, tx, ke); err != nil {
			return fmt.Errorf("error registrando kardex: %w", err)
		}
	}

	// 3. Registrar en auditoría
	if err := s.auditRepo.RecordSaleCreated(ctx, tx, sale.CajeroID, sale.ID, int64(sale.TotalCents)); err != nil {
		return fmt.Errorf("error registrando auditoría: %w", err)
	}

	// 4. Enqueuer en outbox para sincronización
	payload, _ := json.Marshal(map[string]interface{}{
		"sale_id":      sale.ID,
		"session_id":   sale.SessionID,
		"total_cents":  sale.TotalCents,
		"signature":    sale.SignatureHash,
		"created_at":   sale.CreatedAt,
	})
	if err := s.outboxRepo.EnqueueSale(ctx, tx, sale.ID, string(payload)); err != nil {
		return fmt.Errorf("error encolando venta: %w", err)
	}

	return nil
}

// RecordCashCloseWithIntegration graba un cierre de sesión con auditoría y outbox.
func (s *IntegrationService) RecordCashCloseWithIntegration(
	ctx context.Context,
	tx domain.TxPort,
	cajeroID, sessionID string,
	expectedCash, realCash, difference domain.Money,
) error {
	// 1. Registrar en auditoría
	if err := s.auditRepo.RecordSessionClosed(ctx, tx, cajeroID, sessionID,
		int64(expectedCash), int64(realCash), int64(difference)); err != nil {
		return fmt.Errorf("error registrando auditoría de cierre: %w", err)
	}

	// 2. Enqueuer en outbox
	payload, _ := json.Marshal(map[string]interface{}{
		"session_id":    sessionID,
		"expected_cash": expectedCash,
		"real_cash":     realCash,
		"difference":    difference,
	})
	if err := s.outboxRepo.EnqueueCashClose(ctx, tx, sessionID, string(payload)); err != nil {
		return fmt.Errorf("error encolando cierre: %w", err)
	}

	return nil
}

// BuildKardexEntriesForSale construye de entradas de kardex desde los ítems de una venta.
func BuildKardexEntriesForSale(sale domain.Sale) []domain.KardexEntry {
	var entries []domain.KardexEntry
	for _, item := range sale.Items {
		entries = append(entries, domain.KardexEntry{
			ID:           uuid.New().String(),
			SKU:          item.SKU,
			MovementType: "SALE",
			Quantity:     -item.Quantity, // negativo para salida
			CostCents:    item.CostCents,
			BalanceCents: 0, // Se actualizaría desde products.stock_actual
			ReferenceID:  sale.ID,
			Notes:        fmt.Sprintf("Venta recibo #%s", sale.ID),
		})
	}
	return entries
}
