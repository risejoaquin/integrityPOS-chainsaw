package domain

type UnitType string

const (
	UnitPiece UnitType = "PIEZA"
	UnitGram  UnitType = "GRAMO"
	UnitMeter UnitType = "METRO"
)

type Product struct {
	SKU            string
	Name           string
	Barcode        string
	PriceCents     Money
	CostCents      Money
	CostTotalCents Money
	StockActual    int64
	StockMinimo    int64
	UnitType       UnitType
	UnitFactor     int64
	Active         bool
}
