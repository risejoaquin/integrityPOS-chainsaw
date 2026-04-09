package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/intigritypos/integritypos/internal/domain"
)

// generateTestKeysAndCert genera RSA key + self-signed cert para testing
func generateTestKeysAndCert() (privateKeyPEM, certPEM string, err error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}

	privKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privKeyBytes,
	})

	// Self-signed certificate (SAT no valida en test)
	template := &x509.Certificate{
		SerialNumber:       []byte("TEST-SERIAL-001"),
		NotBefore:          time.Now(),
		NotAfter:           time.Now().AddDate(1, 0, 0),
		BasicConstraintsValid: true,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return "", "", err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	return string(privKeyPEM), string(certPEM), nil
}

func TestFiscalWorkerNew(t *testing.T) {
	privKeyPEM, certPEM, err := generateTestKeysAndCert()
	if err != nil {
		t.Fatalf("generate keys: %v", err)
	}

	worker, err := NewFiscalWorker(
		privKeyPEM, certPEM,
		"ABC123456XYZ",
		"Empresa Test S.A.",
		"Calle Principal 123, CDMX",
		"http://ts.sat.gob.mx",
		"http://cfdi.sat.gob.mx/timbre",
	)
	if err != nil {
		t.Fatalf("new worker: %v", err)
	}

	if worker == nil {
		t.Fatal("worker is nil")
	}

	stats := worker.GetStats()
	if stats["cfdi_version"] != "4.0" {
		t.Fatalf("expected CFDI 4.0, got %v", stats["cfdi_version"])
	}
}

func TestFiscalWorkerGenerateCFDI(t *testing.T) {
	privKeyPEM, certPEM, err := generateTestKeysAndCert()
	if err != nil {
		t.Fatalf("generate keys: %v", err)
	}

	worker, err := NewFiscalWorker(
		privKeyPEM, certPEM,
		"ABC123456XYZ",
		"Empresa Test S.A.",
		"Calle Principal 123",
		"",
		"",
	)
	if err != nil {
		t.Fatalf("new worker: %v", err)
	}

	sale := &domain.Sale{
		ID:            "VENTA-001",
		SessionID:     "SES-001",
		CajeroID:      "CAJERO01",
		TerminalID:    "TERM-01",
		SubtotalCents: domain.Money(86200),
			IvaCents:      domain.Money(13800),
		TotalCents:    domain.Money(100000),
		Items: []domain.SaleItem{
			{
				SKU:        "PROD001",
				Name:       "Producto Test",
				Quantity:   1,
				PriceCents: domain.Money(100000),
			},
		},
		PaymentMethod: domain.PaymentCash,
		CreatedAt:     time.Now(),
	}

	cfdiCtx, err := worker.GenerateCFDI(context.Background(), "VENTA-001", sale, "A1")
	if err != nil {
		t.Fatalf("generate CFDI: %v", err)
	}

	if cfdiCtx == nil {
		t.Fatal("cfdi context is nil")
	}

	if cfdiCtx.ReceiptID != "VENTA-001" {
		t.Fatalf("expected receipt_id VENTA-001, got %s", cfdiCtx.ReceiptID)
	}

	if cfdiCtx.CFDIVersion != "4.0" {
		t.Fatalf("expected CFDI 4.0, got %s", cfdiCtx.CFDIVersion)
	}

	if cfdiCtx.Signature == "" {
		t.Fatal("signature is empty")
	}

	if cfdiCtx.CFDIXML == "" {
		t.Fatal("CFDI XML is empty")
	}
}

func TestFiscalWorkerTimbraCFDI(t *testing.T) {
	privKeyPEM, certPEM, err := generateTestKeysAndCert()
	if err != nil {
		t.Fatalf("generate keys: %v", err)
	}

	worker, err := NewFiscalWorker(
		privKeyPEM, certPEM,
		"ABC123456XYZ",
		"Empresa Test S.A.",
		"Calle Principal 123",
		"",
		"",
	)
	if err != nil {
		t.Fatalf("new worker: %v", err)
	}

	sale := &domain.Sale{
		ID:            "VENTA-002",
		SessionID:     "SES-001",
		CajeroID:      "CAJERO01",
		TerminalID:    "TERM-01",
		SubtotalCents: domain.Money(50000),
			IvaCents:      domain.Money(8000),
		TotalCents:    domain.Money(58000),
		Items:         []domain.SaleItem{},
		PaymentMethod: domain.PaymentCash,
		CreatedAt:     time.Now(),
	}

	cfdiCtx, err := worker.GenerateCFDI(context.Background(), "VENTA-002", sale, "A2")
	if err != nil {
		t.Fatalf("generate CFDI: %v", err)
	}

	satUUID, err := worker.TimbraCFDI(context.Background(), cfdiCtx)
	if err != nil {
		t.Fatalf("timbra CFDI: %v", err)
	}

	if satUUID == "" {
		t.Fatal("SAT UUID is empty")
	}

	if cfdiCtx.SATSeal != satUUID {
		t.Fatalf("expected seal %s, got %s", satUUID, cfdiCtx.SATSeal)
	}
}

func TestFiscalWorkerStats(t *testing.T) {
	privKeyPEM, certPEM, _ := generateTestKeysAndCert()

	worker, _ := NewFiscalWorker(
		privKeyPEM, certPEM,
		"ABC123456XYZ",
		"Test Empresa",
		"Test Address",
		"",
		"",
	)

	sale := &domain.Sale{
		ID:            "TEST-001",
		SubtotalCents: domain.Money(1000),
			IvaCents:      domain.Money(160),
		TotalCents:    domain.Money(1160),
		Items:         []domain.SaleItem{},
		PaymentMethod: domain.PaymentCash,
		CreatedAt:     time.Now(),
	}

	// Generate 3 CFDIs
	for i := 0; i < 3; i++ {
		worker.GenerateCFDI(context.Background(), "TEST-00"+string(rune('1'+i)), sale, "A1")
	}

	stats := worker.GetStats()

	if stats["cfdi_generations"].(int64) != 3 {
		t.Fatalf("expected 3 generations, got %v", stats["cfdi_generations"])
	}

	if stats["is_online"] != true {
		t.Fatal("expected online")
	}

	if stats["cfdi_version"] != "4.0" {
		t.Fatalf("expected 4.0, got %v", stats["cfdi_version"])
	}
}

func TestFiscalWorkerInvalidKeys(t *testing.T) {
	_, err := NewFiscalWorker(
		"INVALID KEY",
		"INVALID CERT",
		"RFC",
		"Name",
		"Address",
		"",
		"",
	)
	if err == nil {
		t.Fatal("expected error with invalid keys")
	}
}
