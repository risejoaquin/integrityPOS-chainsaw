package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/intigritypos/integritypos/internal/domain"
)

// FiscalEntry representa una entrada de CFDI timbrado en la DB
type FiscalEntry struct {
	ID              string    `db:"id"`
	ReceiptID       string    `db:"receipt_id"`
	CFDIVersion     string    `db:"cfdi_version"`
	SerieAndFolio   string    `db:"serie_folio"`
	CFDIXML         string    `db:"cfdi_xml"`
	Signature       string    `db:"signature_b64"`
	SATUuid         string    `db:"sat_uuid"`
	TimbradoAt      *time.Time `db:"timbrado_at"`
	Status          string    `db:"status"` // pending | timbrado | cancelado | error
	ErrorMsg        *string   `db:"error_msg"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

// FiscalRepository maneja la persistencia de CFDI
type FiscalRepository struct {
	db *sql.DB
}

// NewFiscalRepository crea instancia del repo
func NewFiscalRepository(db *sql.DB) *FiscalRepository {
	return &FiscalRepository{db: db}
}

// SaveCFDI guarda un CFDI generado (antes de timbrado)
func (r *FiscalRepository) SaveCFDI(ctx context.Context, tx domain.TxPort, cfdi *FiscalEntry) error {
	if cfdi.ID == "" {
		cfdi.ID = uuid.New().String()
	}
	if cfdi.CreatedAt.IsZero() {
		cfdi.CreatedAt = time.Now()
	}
	cfdi.UpdatedAt = time.Now()

	query := `INSERT INTO fiscal_timbres (
		id, receipt_id, cfdi_version, serie_folio, 
		cfdi_xml, signature_b64, status, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	return tx.Exec(query,
		cfdi.ID, cfdi.ReceiptID, cfdi.CFDIVersion, cfdi.SerieAndFolio,
		cfdi.CFDIXML, cfdi.Signature, cfdi.Status, cfdi.CreatedAt, cfdi.UpdatedAt)
}

// MarkTimbrado marca un CFDI como timbrado (con UUID del SAT)
func (r *FiscalRepository) MarkTimbrado(ctx context.Context, receiptID, satUUID string) error {
	query := `UPDATE fiscal_timbres 
		SET status = 'timbrado', sat_uuid = ?, timbrado_at = ?, updated_at = ?
		WHERE receipt_id = ? AND status = 'pending'`

	err := r.db.QueryRowContext(ctx, query, satUUID, time.Now(), time.Now(), receiptID).Err()
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("error marking timbrado: %w", err)
	}
	return nil
}

// MarkError marca un CFDI como error
func (r *FiscalRepository) MarkError(ctx context.Context, receiptID, errMsg string) error {
	query := `UPDATE fiscal_timbres 
		SET status = 'error', error_msg = ?, updated_at = ?
		WHERE receipt_id = ? AND status IN ('pending', 'timbrado')`

	err := r.db.QueryRowContext(ctx, query, errMsg, time.Now(), receiptID).Err()
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("error marking error: %w", err)
	}
	return nil
}

// GetByReceiptID retorna CFDI por receipt ID
func (r *FiscalRepository) GetByReceiptID(ctx context.Context, receiptID string) (*FiscalEntry, error) {
	query := `SELECT 
		id, receipt_id, cfdi_version, serie_folio,
		cfdi_xml, signature_b64, sat_uuid, timbrado_at,
		status, error_msg, created_at, updated_at
	FROM fiscal_timbres WHERE receipt_id = ?`

	var entry FiscalEntry
	err := r.db.QueryRowContext(ctx, query, receiptID).Scan(
		&entry.ID, &entry.ReceiptID, &entry.CFDIVersion, &entry.SerieAndFolio,
		&entry.CFDIXML, &entry.Signature, &entry.SATUuid, &entry.TimbradoAt,
		&entry.Status, &entry.ErrorMsg, &entry.CreatedAt, &entry.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error querying CFDI: %w", err)
	}

	return &entry, nil
}

// GetPendingTimbrados retorna CFDI pendientes de timbrado (para batch processing)
func (r *FiscalRepository) GetPendingTimbrados(ctx context.Context, limit int) ([]*FiscalEntry, error) {
	query := `SELECT 
		id, receipt_id, cfdi_version, serie_folio,
		cfdi_xml, signature_b64, sat_uuid, timbrado_at,
		status, error_msg, created_at, updated_at
	FROM fiscal_timbres 
	WHERE status = 'pending'
	ORDER BY created_at ASC
	LIMIT ?`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("error querying pending: %w", err)
	}
	defer rows.Close()

	var entries []*FiscalEntry
	for rows.Next() {
		var entry FiscalEntry
		if err := rows.Scan(
			&entry.ID, &entry.ReceiptID, &entry.CFDIVersion, &entry.SerieAndFolio,
			&entry.CFDIXML, &entry.Signature, &entry.SATUuid, &entry.TimbradoAt,
			&entry.Status, &entry.ErrorMsg, &entry.CreatedAt, &entry.UpdatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, &entry)
	}

	return entries, rows.Err()
}

// GetStats retorna estadísticas de timbrados
func (r *FiscalRepository) GetStats(ctx context.Context) (map[string]int64, error) {
	stats := make(map[string]int64)

	statuses := []string{"pending", "timbrado", "cancelado", "error"}
	for _, status := range statuses {
		query := fmt.Sprintf(`SELECT COUNT(*) FROM fiscal_timbres WHERE status = '%s'`, status)
		var count int64
		if err := r.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
			return nil, fmt.Errorf("error counting %s: %w", status, err)
		}
		stats[status] = count
	}

	return stats, nil
}

// ExportCFDI retorna el XML canónico para almacenamiento/auditoría
func (r *FiscalRepository) ExportCFDI(ctx context.Context, receiptID string) (xmlBytes []byte, err error) {
	entry, err := r.GetByReceiptID(ctx, receiptID)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, fmt.Errorf("CFDI no encontrado para receipt %s", receiptID)
	}

	return []byte(entry.CFDIXML), nil
}

// ExportCFDIWithMetadata retorna CFDI + metadata como JSON
func (r *FiscalRepository) ExportCFDIWithMetadata(ctx context.Context, receiptID string) (jsonBytes []byte, err error) {
	entry, err := r.GetByReceiptID(ctx, receiptID)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, fmt.Errorf("CFDI no encontrado para receipt %s", receiptID)
	}

	return json.MarshalIndent(entry, "", "  ")
}
