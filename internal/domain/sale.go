package domain

type SaleItem struct {
	ID           string
	SKU          string
	Name         string
	Quantity     Quantity
	PriceCents   Money
	SubtotalCents Money
	IvaCents     Money
	TotalCents   Money
	CostCents    Money
}

type Sale struct {
	ID            string
	SessionID     string
	CajeroID      string
	TerminalID    string
	Items         []SaleItem
	SubtotalCents Money
	IvaCents      Money
	TotalCents    Money
	PaymentMethod PaymentMethod
	PaidCents     Money
	ChangeCents   Money
	Status        string
	FiscalStatus  string
	FiscalUUID    string
	SignatureHash string
	CreatedAt     string
}

// SuspendedSale representa una venta en suspensión (F4) con sus datos propios.
type SuspendedSale struct {
	ID          string
	CajeroID    string
	SessionID   string
	ItemsJSON   string // JSON canónico del carrito
	TotalCents  Money
	SuspendedAt string
	ExpiresAt   string
	Status      string // active | recovered | expired
}

// StockReservation representa la reserva de stock asociada a una venta en espera.
type StockReservation struct {
	ID         string
	SaleID     string
	SKU        string
	Quantity   Quantity
	ExpiresAt  string
	CreatedAt  string
}

// Métodos de pago estándar
type PaymentMethod string

const (
	PaymentCash     PaymentMethod = "CASH"
	PaymentCard     PaymentMethod = "CARD"
	PaymentTransfer PaymentMethod = "TRANSFER"
)

// Estados de venta
const (
	SaleCompleted = "completed"
	SalePending   = "pending"
	SaleCancelled = "cancelled"
)

// KardexEntry representa un movimiento contable de inventario.
type KardexEntry struct {
	ID             string
	SKU            string
	MovementType   string
	Quantity       Quantity
	CostCents      Money
	BalanceCents   Money
	ReferenceID    string
	Notes          string
	CreatedAt      string
}

// OutboxEntry representa eventos no enviados para sincronización.
type OutboxEntry struct {
	ID         string
	EntityType string
	EntityID   string
	Payload    string
	Priority   int
	CreatedAt  string
}
