package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/intigritypos/integritypos/internal/domain"
)

type ProductRepo struct {
	db *sql.DB
}

func NewProductRepo(db *sql.DB) *ProductRepo {
	return &ProductRepo{db: db}
}

func (r *ProductRepo) FindBySKU(ctx context.Context, sku string) (*domain.Product, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT sku, name, barcode, price_cents, cost_cents, cost_total_cents,
			stock_actual, stock_minimo, unit_type, unit_factor, active
		 FROM products WHERE sku = ? AND active = 1`, sku)
	var p domain.Product
	var unitType string
	var activeInt int
	if err := row.Scan(&p.SKU, &p.Name, &p.Barcode, &p.PriceCents, &p.CostCents, &p.CostTotalCents,
		&p.StockActual, &p.StockMinimo, &unitType, &p.UnitFactor, &activeInt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("producto no encontrado: %s", sku)
		}
		return nil, err
	}
	p.UnitType = domain.UnitType(unitType)
	p.Active = activeInt == 1
	return &p, nil
}

func (r *ProductRepo) GetStockDisponible(ctx context.Context, sku string) (domain.Quantity, error) {
	var disponible int64
	err := r.db.QueryRowContext(ctx, `
        SELECT i.stock_actual - COALESCE(SUM(r.quantity), 0)
        FROM products i
        LEFT JOIN stock_reservations r
            ON r.sku = i.sku AND r.expires_at > datetime('now')
        WHERE i.sku = ?
        GROUP BY i.sku`, sku).Scan(&disponible)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("producto no encontrado: %s", sku)
	}
	if err != nil {
		return 0, err
	}
	return domain.Quantity(disponible), nil
}

func (r *ProductRepo) UpdateStock(ctx context.Context, tx domain.TxPort, sku string, delta domain.Quantity) error {
	err := tx.Exec(`UPDATE products SET stock_actual = stock_actual + ?, updated_at = datetime('now') WHERE sku = ?`, int64(delta), sku)
	return err
}
