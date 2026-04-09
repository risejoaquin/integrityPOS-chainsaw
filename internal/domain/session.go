package domain

type CashSession struct {
	ID            string
	CajeroID      string
	TerminalID    string
	InitialCash   Money
	TotalSales    Money
	Withdrawals   Money
	ExpectedCash  Money
	RealCash      Money
	Difference    Money
	OpenedAt      string
	ClosedAt      string
	SignatureHash string
}

type CashMovement struct {
	ID        string
	SessionID string
	Amount    Money
	Type      string // WITHDRAWAL|DEPOSIT
	Reason    string
	CreatedAt string
}

