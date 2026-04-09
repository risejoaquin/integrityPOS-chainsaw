package hardware

import (
	"testing"
	"time"

	"github.com/intigritypos/integritypos/internal/domain"
)

// TestPrinterStdoutMode prueba impresión en stdout (para CI/pruebas)
func TestPrinterStdoutMode(t *testing.T) {
	printer := NewPrinterAdapter(ModeStdout, "")
	if err := printer.Open(); err != nil {
		t.Fatalf("No pudo abrir impresora stdout: %v", err)
	}
	defer printer.Close()

	if !printer.IsOnline() {
		t.Fatal("Impresora debería estar online")
	}

	// Imprimir recibo de prueba
	if err := printer.PrintTestReceipt(); err != nil {
		t.Fatalf("Error imprimiendo recibo test: %v", err)
	}
}

// TestPrinterReceiptCompleto prueba impresión con todos los campos
func TestPrinterReceiptCompleto(t *testing.T) {
	printer := NewPrinterAdapter(ModeStdout, "")
	if err := printer.Open(); err != nil {
		t.Fatalf("No pudo abrir impresora: %v", err)
	}
	defer printer.Close()

	sale := &domain.Sale{
		ID:         "VENTA-2026-001",
		TerminalID: "TERMINAL-01",
		CajeroID:   "JUAN",
		SessionID:  "SES-001",
		Items: []domain.SaleItem{
			{
				SKU:        "PROD-001",
				Name:       "Manzana Roja",
				Quantity:   3,
				PriceCents: 1500, // $15.00
				TotalCents: 4500,
			},
			{
				SKU:        "PROD-002",
				Name:       "Leche Entera 1L",
				Quantity:   2,
				PriceCents: 3900, // $39.00
				TotalCents: 7800,
			},
		},
		SubtotalCents: 10561,
		IvaCents:      1689,
		TotalCents:    12250,
		PaymentMethod: domain.PaymentCash,
		PaidCents:     15000,
		ChangeCents:   2750,
		SignatureHash: "SHA256-HMAC-ABC123DEF456789",
		CreatedAt:     time.Now(),
	}

	if err := printer.PrintReceipt(sale); err != nil {
		t.Fatalf("Error imprimiendo recibo: %v", err)
	}

	stats := printer.GetStats()
	if stats["receipts_printed"].(int64) != 1 {
		t.Fatalf("Esperaba 1 recibo impreso, got %v", stats["receipts_printed"])
	}
}

// TestPrinterOpenDrawer prueba apertura de caja
func TestPrinterOpenDrawer(t *testing.T) {
	printer := NewPrinterAdapter(ModeStdout, "")
	if err := printer.Open(); err != nil {
		t.Fatalf("No pudo abrir impresora: %v", err)
	}
	defer printer.Close()

	if err := printer.OpenDrawer(); err != nil {
		t.Fatalf("Error abriendo caja: %v", err)
	}
}

// TestPrinterStats prueba estadísticas
func TestPrinterStats(t *testing.T) {
	printer := NewPrinterAdapter(ModeStdout, "")
	if err := printer.Open(); err != nil {
		t.Fatalf("No pudo abrir impresora: %v", err)
	}
	defer printer.Close()

	// Imprimir varios recibos
	for i := 0; i < 3; i++ {
		if err := printer.PrintTestReceipt(); err != nil {
			t.Fatalf("Error en iteración %d: %v", i, err)
		}
	}

	stats := printer.GetStats()
	if stats["receipts_printed"].(int64) != 3 {
		t.Fatalf("Esperaba 3 recibos, got %v", stats["receipts_printed"])
	}

	if stats["online"] != true {
		t.Fatal("Impresora debería estar online")
	}

	if stats["mode"] != ModeStdout {
		t.Fatalf("Modo incorrecto: %v", stats["mode"])
	}
}

// TestPrinterOffline prueba comportamiento cuando la impresora está offline
func TestPrinterOffline(t *testing.T) {
	printer := NewPrinterAdapter(ModeStdout, "")
	// NO abrir la impresora

	if printer.IsOnline() {
		t.Fatal("Impresora no debería estar online")
	}

	if err := printer.PrintTestReceipt(); err == nil {
		t.Fatal("Debería retornar error cuando está offline")
	}
}

// TestPrinterReceiptWithCardPayment prueba recibo con pago en tarjeta
func TestPrinterReceiptWithCardPayment(t *testing.T) {
	printer := NewPrinterAdapter(ModeStdout, "")
	if err := printer.Open(); err != nil {
		t.Fatalf("No pudo abrir impresora: %v", err)
	}
	defer printer.Close()

	sale := &domain.Sale{
		ID:         "VENTA-CARD-001",
		TerminalID: "TERMINAL-02",
		CajeroID:   "MARIA",
		Items: []domain.SaleItem{
			{
				SKU:        "SKU-ELECTRONICS",
				Name:       "Laptop ThinkPad X1",
				Quantity:   1,
				PriceCents: 1450000, // $14,500.00
				TotalCents: 1450000,
			},
		},
		SubtotalCents: 1250000,
		IvaCents:      200000,
		TotalCents:    1450000,
		PaymentMethod: domain.PaymentCard,
		PaidCents:     1450000,
		ChangeCents:   0,
		SignatureHash: "HMAC-CARD-123",
		CreatedAt:     time.Now(),
	}

	if err := printer.PrintReceipt(sale); err != nil {
		t.Fatalf("Error imprimiendo recibo tarjeta: %v", err)
	}
}
