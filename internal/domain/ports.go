package domain

import "context"

type SaleRepository interface {
	Save(ctx context.Context, tx TxPort, sale Sale) error
	FindByID(ctx context.Context, id string) (*Sale, error)
}

type SuspendedSaleRepository interface {
	Save(ctx context.Context, tx TxPort, sale SuspendedSale) error
	FindByID(ctx context.Context, id string) (*SuspendedSale, error)
	UpdateStatus(ctx context.Context, tx TxPort, id, status string) error
}

type StockReservationRepository interface {
	Reserve(ctx context.Context, tx TxPort, reservation StockReservation) error
	ReleaseBySaleID(ctx context.Context, tx TxPort, saleID string) error
	ExpireOld(ctx context.Context, tx TxPort) error
}

type ProductRepository interface {
	FindBySKU(ctx context.Context, sku string) (*Product, error)
	UpdateStock(ctx context.Context, tx TxPort, sku string, delta Quantity) error
	GetStockDisponible(ctx context.Context, sku string) (Quantity, error)
}

type SessionRepository interface {
	Save(ctx context.Context, session CashSession) error
	FindByID(ctx context.Context, id string) (*CashSession, error)
	CloseSession(ctx context.Context, sessionID string, realCash Money, difference Money) error
	AddMovement(ctx context.Context, movement CashMovement) error
}

// TxPort — Abstracción de transacción SQL (para no importar database/sql en el dominio)
type TxPort interface {
	Exec(query string, args ...any) error
	QueryRow(query string, args ...any) RowScanner
}

type RowScanner interface {
	Scan(dest ...any) error
}

