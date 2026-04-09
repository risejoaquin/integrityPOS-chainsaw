package services

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"sync"
	"time"

	"github.com/intigritypos/integritypos/internal/domain"
)

// FiscalContext contiene la información fiscal de una venta
type FiscalContext struct {
	ReceiptID     string
	Sale          *domain.Sale
	CFDIVersion   string
	SerieAndFolio string // Ej: "A1" (SAT-assigned)
	CertificateSN string
	Timestamp     time.Time
	CFDIXML       string
	Signature     string
	SATSeal       string // UUID returned by SAT
}

// FiscalWorker coordina la generación y timbrado de CFDI 4.0
type FiscalWorker struct {
	mu              sync.RWMutex
	privateKey      *rsa.PrivateKey
	certificate     *x509.Certificate
	tsaURL          string
	satURL          string
	emitterRFC      string // RFC del emisor (empresa)
	emitterName     string
	emitterAddress  string
	timedOut        bool
	cfdiGenerations int64
	timbroCalls     int64
	failedTimbros   int64
}

// NewFiscalWorker crea instancia del worker.
// privateKeyPEM y certPEM son strings en formato PEM
func NewFiscalWorker(privateKeyPEM, certPEM, emitterRFC, emitterName, emitterAddress, tsaURL, satURL string) (*FiscalWorker, error) {
	// Parse private key
	keyBlock, _ := pem.Decode([]byte(privateKeyPEM))
	if keyBlock == nil {
		return nil, fmt.Errorf("invalid PEM private key")
	}

	privKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("error parsing private key: %w", err)
	}

	// Parse certificate
	certBlock, _ := pem.Decode([]byte(certPEM))
	if certBlock == nil {
		return nil, fmt.Errorf("invalid PEM certificate")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("error parsing certificate: %w", err)
	}

	return &FiscalWorker{
		privateKey:  privKey,
		certificate: cert,
		tsaURL:      tsaURL,
		satURL:      satURL,
		emitterRFC:  emitterRFC,
		emitterName: emitterName,
		emitterAddress: emitterAddress,
	}, nil
}

// GenerateCFDI crea el XML CFDI 4.0 canónico para una venta
func (fw *FiscalWorker) GenerateCFDI(ctx context.Context, receiptID string, sale *domain.Sale, serieAndFolio string) (*FiscalContext, error) {
	fw.mu.Lock()
	fw.cfdiGenerations++
	fw.mu.Unlock()

	if fw.timedOut {
		return nil, fmt.Errorf("fiscal worker timed out")
	}

	cfdiCtx := &FiscalContext{
		ReceiptID:     receiptID,
		Sale:          sale,
		CFDIVersion:   "4.0",
		SerieAndFolio: serieAndFolio,
		Timestamp:     time.Now().UTC(),
	}

	// Extraer SN del certificado
	cfdiCtx.CertificateSN = fmt.Sprintf("%X", fw.certificate.SerialNumber)

	// Generar XML canónico
	cfdi := fw.buildCFDIXML(cfdiCtx)
	cfdiBytes, err := xml.MarshalIndent(cfdi, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error marshaling CFDI XML: %w", err)
	}

	cfdiCtx.CFDIXML = string(cfdiBytes)

	// Firmar el CFDI
	h := sha256.New()
	h.Write(cfdiBytes)
	digest := h.Sum(nil)

	signature, err := rsa.SignPKCS1v15(rand.Reader, fw.privateKey, crypto.SHA256, digest)
	if err != nil {
		return nil, fmt.Errorf("error signing CFDI: %w", err)
	}

	cfdiCtx.Signature = base64.StdEncoding.EncodeToString(signature)

	return cfdiCtx, nil
}

// TimbraCFDI envía el CFDI firmado al SAT para obtener timestamp y sello
func (fw *FiscalWorker) TimbraCFDI(ctx context.Context, cfdiCtx *FiscalContext) (satUUID string, err error) {
	fw.mu.Lock()
	fw.timbroCalls++
	fw.mu.Unlock()

	if fw.timedOut {
		fw.mu.Lock()
		fw.failedTimbros++
		fw.mu.Unlock()
		return "", fmt.Errorf("fiscal worker timed out")
	}

	// En produción: enviar a SAT endpoint con SOAP/HTTP
	// Por ahora: generar UUID mock (RFC 4122 v4)
	uuid := generateUUID()

	// Simular timestamp del SAT (en produción: obtener de TSA)
	cfdiCtx.SATSeal = uuid
	cfdiCtx.Timestamp = time.Now().UTC()

	return uuid, nil
}

// GetStats retorna estadísticas del worker
func (fw *FiscalWorker) GetStats() map[string]interface{} {
	fw.mu.RLock()
	defer fw.mu.RUnlock()

	return map[string]interface{}{
		"cfdi_generations":  fw.cfdiGenerations,
		"timbro_calls":      fw.timbroCalls,
		"failed_timbros":    fw.failedTimbros,
		"is_online":         !fw.timedOut,
		"emitter_rfc":       fw.emitterRFC,
		"certificate_sn":    fmt.Sprintf("%X", fw.certificate.SerialNumber),
		"cfdi_version":      "4.0",
	}
}

// ─────────────────────────────────────────────────────────────────────────────

// buildCFDIXML construye la estructura XML CFDI 4.0
// Referencia: https://www.sat.gob.mx/especificaciones/Paginas/preguntas_generales.aspx
func (fw *FiscalWorker) buildCFDIXML(cfdiCtx *FiscalContext) interface{} {
	// Simplificado para esta fase (CFDI completo en v2.0)
	// Estructura mínima: Comprobante + emisor + receptor + concepto + timbre
	return map[string]interface{}{
		"Comprobante": map[string]interface{}{
			"Version":     "4.0",
			"Folio":       cfdiCtx.SerieAndFolio,
			"Fecha":       cfdiCtx.Timestamp.Format("2006-01-02T15:04:05"),
			"Subtotal":    float64(cfdiCtx.Sale.SubtotalCents) / 100,
			"Impuestos":   float64(cfdiCtx.Sale.IvaCents) / 100,
			"Total":       float64(cfdiCtx.Sale.TotalCents) / 100,
			"TipoDeComprobante": "I", // Ingreso
			"LugarExpedicion":    fw.emitterAddress,
		},
		"Emisor": map[string]interface{}{
			"RFC":  fw.emitterRFC,
			"Nombre": fw.emitterName,
		},
		"Receptor": map[string]interface{}{
			"RFC":  "XAXX010101000", // RFC genérico para consumidor final
			"Nombre": "Público en General",
		},
		"Conceptos": map[string]interface{}{
			"CantidadItems": len(cfdiCtx.Sale.Items),
		},
	}
}

// generateUUID genera un UUID v4 canónico
func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// ─────────────────────────────────────────────────────────────────────────────

// TimbroCFA es simplificación de TimbreDAtosCertificados (SAT)
type TimbroCFA struct {
	XMLNamespace   string
	UUID           string
	FechaTimbrado  time.Time
	SelloCFD       string
	NoCertificado  string
	SelloSAT       string
	Version        string
}

// EstadoTimbro enum para estados de timbrado
type EstadoTimbro string

const (
	EstadoPending   EstadoTimbro = "pending"
	EstadoTimbrado  EstadoTimbro = "timbrado"
	EstadoCancelado EstadoTimbro = "cancelado"
	EstadoError     EstadoTimbro = "error"
)
