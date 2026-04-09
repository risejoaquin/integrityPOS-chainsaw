package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/intigritypos/integritypos/internal/domain"
)

type SaleRepo struct {
	db *sql.DB
}

func NewSaleRepo(db *sql.DB) *SaleRepo {
	return &SaleRepo{db: db}
}

func (r *SaleRepo) Save(ctx context.Context, tx domain.TxPort, sale domain.Sale) error {
	err := tx.Exec(`
        INSERT INTO local_receipts
        (id, session_id, cajero_id, terminal_id, subtotal_cents, iva_cents, total_cents,
         payment_method, paid_cents, change_cents, status, fiscal_status, signature_hash, created_at)
        VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,datetime('now'))`,
		sale.ID, sale.SessionID, sale.CajeroID, sale.TerminalID,
		sale.SubtotalCents, sale.IvaCents, sale.TotalCents,
		sale.PaymentMethod, sale.PaidCents, sale.ChangeCents,
		sale.Status, sale.FiscalStatus, sale.SignatureHash)
	if err != nil {
		return fmt.Errorf("error guardando venta: %w", err)
	}

	for i := range sale.Items {
		if sale.Items[i].ID == "" {
			sale.Items[i].ID = uuid.New().String()
		}
		item := sale.Items[i]
		err := tx.Exec(`
            INSERT INTO receipt_items
            (id, receipt_id, sku, name, quantity, price_cents, subtotal_cents, iva_cents, total_cents, cost_cents)
            VALUES (?,?,?,?,?,?,?,?,?,?)`,
			item.ID, sale.ID, item.SKU, item.Name, item.Quantity,
			item.PriceCents, item.SubtotalCents, item.IvaCents, item.TotalCents, item.CostCents)
		if err != nil {
			return fmt.Errorf("error guardando ítem %s: %w", item.SKU, err)
		}
	}

	payload, _ := json.Marshal(sale)
	err = tx.Exec(`
        INSERT INTO sync_outbox (id, entity_type, entity_id, payload, priority)
        VALUES (?,?,?,?,1)`,
		"OBX-"+sale.ID, "SALE", sale.ID, string(payload))
	if err != nil {
		return fmt.Errorf("error encolando en outbox: %w", err)
	}

	return nil
}

func (r *SaleRepo) FindByID(ctx context.Context, id string) (*domain.Sale, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, session_id, cajero_id, terminal_id, subtotal_cents, iva_cents, total_cents, payment_method, paid_cents, change_cents, status, fiscal_status, signature_hash, created_at FROM local_receipts WHERE id = ?`, id)
	var s domain.Sale
	var paymentMethod string
	if err := row.Scan(&s.ID, &s.SessionID, &s.CajeroID, &s.TerminalID, &s.SubtotalCents, &s.IvaCents, &s.TotalCents, &paymentMethod, &s.PaidCents, &s.ChangeCents, &s.Status, &s.FiscalStatus, &s.SignatureHash, &s.CreatedAt); err != nil {
		return nil, err
	}
	s.PaymentMethod = domain.PaymentMethod(paymentMethod)
	return &s, nil
}

func (r *SaleRepo) SaveSuspendedSale(ctx context.Context, tx domain.TxPort, sale domain.SuspendedSale) error {
	err := tx.Exec(`
        INSERT INTO suspended_sales
        (id, cajero_id, session_id, items_json, total_cents, suspended_at, expires_at, status)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sale.ID, sale.CajeroID, sale.SessionID, sale.ItemsJSON, sale.TotalCents, sale.SuspendedAt, sale.ExpiresAt, sale.Status)
	return err
}

func (r *SaleRepo) FindSuspendedSaleByID(ctx context.Context, id string) (*domain.SuspendedSale, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, cajero_id, session_id, items_json, total_cents, suspended_at, expires_at, status FROM suspended_sales WHERE id = ?`, id)
	var s domain.SuspendedSale
	if err := row.Scan(&s.ID, &s.CajeroID, &s.SessionID, &s.ItemsJSON, &s.TotalCents, &s.SuspendedAt, &s.ExpiresAt, &s.Status); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SaleRepo) UpdateSuspendedSaleStatus(ctx context.Context, tx domain.TxPort, id, status string) error {
	err := tx.Exec(`UPDATE suspended_sales SET status = ? WHERE id = ?`, status, id)
	return err
}

func (r *SaleRepo) ReserveStock(ctx context.Context, tx domain.TxPort, res domain.StockReservation) error {
	err := tx.Exec(`
        INSERT INTO stock_reservations
        (id, sale_id, sku, quantity, expires_at, created_at)
        VALUES (?, ?, ?, ?, ?, datetime('now'))`,
		res.ID, res.SaleID, res.SKU, res.Quantity, res.ExpiresAt)
	return err
}

func (r *SaleRepo) ReleaseReservationsBySaleID(ctx context.Context, tx domain.TxPort, saleID string) error {
	err := tx.Exec(`DELETE FROM stock_reservations WHERE sale_id = ?`, saleID)
	return err
}

func (r *SaleRepo) ExpireOldReservations(ctx context.Context, tx domain.TxPort) error {
	err := tx.Exec(`DELETE FROM stock_reservations WHERE expires_at <= datetime('now')`)
	return err
}
