package hardware

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/intigritypos/integritypos/internal/domain"
)

// ESC/POS Command Set (Epson Standard Thermal Printer)
const (
	ESC           = "\u001b"
	GS            = "\u001d"
	EOL           = "\n"
	
	// Alignment
	ESC_ALIGN_LEFT   = ESC + "[0 @"   // \x1b [ 0 @
	ESC_ALIGN_CENTER = ESC + "[1 @"   // \x1b [ 1 @
	ESC_ALIGN_RIGHT  = ESC + "[2 @"   // \x1b [ 2 @
	
	// Text emphasis
	ESC_BOLD_ON      = ESC + "E" + string(1)
	ESC_BOLD_OFF     = ESC + "E" + string(0)
	ESC_DOUBLE_HEIGHT = ESC + "!" + string(0x10)
	ESC_DOUBLE_WIDTH  = ESC + "!" + string(0x20)
	ESC_NORMAL       = ESC + "!" + string(0)
	
	// Drawer (RJ-11 pin configuration)
	ESC_DRAWER       = GS + "p" + string(0) + string(25)
	
	// Cut paper
	ESC_CUT_FULL     = GS + "V" + string(65) + string(0)  // Full cut
	ESC_CUT_PARTIAL  = GS + "V" + string(66) + string(0)  // Partial cut
	
	// Initialization
	ESC_INIT         = ESC + "@"
	
	// Line spacing (1/8" units)
	ESC_LINE_SPACING = ESC + "3" + string(40)  // ~5mm line height
)

// PrinterMode indica el tipo de conexión
type PrinterMode string

const (
	ModeDirect  PrinterMode = "direct"  // /dev/usb/lp0 o /dev/ttyUSB0 (directo)
	ModeNetwork PrinterMode = "network" // TCP/IP (futuro)
	ModeStdout  PrinterMode = "stdout"  // Pruebas: stdout
)

// PrinterAdapter encapsula toda la lógica de impresión térmica ESC/POS
type PrinterAdapter struct {
	mode       PrinterMode
	devicePath string
	netAddr    string
	file       *os.File
	mu         sync.Mutex
	isOnline   bool
	timeout    time.Duration

	// Estadísticas
	receiptsPrinted int64
	paperWidth      int // caracteres
	charWidth       int // ancho en mm (aprox)
}

// NewPrinterAdapter crea un adaptador para la impresora térmica
// mode: "direct" (ej: /dev/usb/lp0) | "network" (ej: 192.168.1.100:9100) | "stdout" (pruebas)
func NewPrinterAdapter(mode PrinterMode, devicePath string) *PrinterAdapter {
	return &PrinterAdapter{
		mode:       mode,
		devicePath: devicePath,
		paperWidth: 32,     // Estándar 80mm: ~32 caracteres
		charWidth:  3,      // ~3mm por carácter
		timeout:    5 * time.Second,
		isOnline:   false,
	}
}

// Open conecta a la impresora y verifica que esté lista
func (p *PrinterAdapter) Open() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.mode {
	case ModeDirect:
		f, err := os.OpenFile(p.devicePath, os.O_WRONLY, 0600)
		if err != nil {
			return fmt.Errorf("no se pudo abrir puerto %s: %w", p.devicePath, err)
		}
		p.file = f

	case ModeStdout:
		p.file = os.Stdout

	case ModeNetwork:
		// Futuro: implementar conexión TCP
		return fmt.Errorf("modo network no soportado en v1")

	default:
		return fmt.Errorf("modo desconocido: %s", p.mode)
	}

	// Inicializar impresora
	if err := p.init(); err != nil {
		return fmt.Errorf("init fallido: %w", err)
	}

	p.isOnline = true
	return nil
}

// Close desconecta la impresora
func (p *PrinterAdapter) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.file == nil || p.mode == ModeStdout {
		return nil
	}

	if err := p.file.Close(); err != nil {
		return fmt.Errorf("error cerrando puerto: %w", err)
	}

	p.isOnline = false
	return nil
}

// IsOnline retorna si la impresora está conectada
func (p *PrinterAdapter) IsOnline() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.isOnline
}

// PrintReceipt imprime un recibo completo (venta)
func (p *PrinterAdapter) PrintReceipt(sale *domain.Sale) error {
	if !p.IsOnline() {
		return fmt.Errorf("impresora fuera de línea")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Buffer de comandos
	var buf []byte

	// ─── ENCABEZADO ───────────────────────────────────────────
	buf = append(buf, []byte(ESC_INIT)...)          // Reset
	buf = append(buf, []byte(ESC_LINE_SPACING)...)  // Line height
	buf = append(buf, []byte(ESC_ALIGN_CENTER)...)  // Center

	// Título
	buf = append(buf, []byte(ESC_BOLD_ON)...)
	buf = append(buf, []byte("INTEGRITYPOS\n")...)
	buf = append(buf, []byte("RECIBO DE VENTA\n")...)
	buf = append(buf, []byte(ESC_BOLD_OFF)...)

	// Información sesión
	buf = append(buf, []byte(ESC_ALIGN_LEFT)...)
	buf = append(buf, []byte(fmt.Sprintf("\nTerminal: %s\n", sale.TerminalID))...)
	buf = append(buf, []byte(fmt.Sprintf("Cajero: %s\n", sale.CajeroID))...)
	buf = append(buf, []byte(fmt.Sprintf("Recibo: %s\n", sale.ID))...)

	dateStr := sale.CreatedAt
	if t, err := time.Parse(time.RFC3339, sale.CreatedAt); err == nil {
		dateStr = t.Format("02/01/2006 15:04:05")
	} else if sale.CreatedAt == "" {
		dateStr = time.Now().Format("02/01/2006 15:04:05")
	}

	buf = append(buf, []byte(fmt.Sprintf("Fecha: %s\n", dateStr))...)

	// Línea separadora
	buf = append(buf, []byte(p.separator())...)

	// ─── ÍTEMS ────────────────────────────────────────────────
	buf = append(buf, []byte("DESC.                CANT. PRECIO TOTAL\n")...)
	buf = append(buf, []byte(p.separator())...)

	for _, item := range sale.Items {
		// Nombre (truncado a 20 chars)
		name := item.Name
		if len(name) > 20 {
			name = name[:20]
		}
		buf = append(buf, []byte(fmt.Sprintf("%-20s\n", name))...)

		// Cantidad × Precio = Subtotal
		qty := float64(item.Quantity)
		price := float64(item.PriceCents) / 100
		total := float64(item.TotalCents) / 100
		buf = append(buf, []byte(fmt.Sprintf("  x%.0f @ $%.2f = $%.2f\n", qty, price, total))...)
	}

	buf = append(buf, []byte(p.separator())...)

	// ─── TOTALES ──────────────────────────────────────────────
	buf = append(buf, []byte(ESC_ALIGN_RIGHT)...)
	subtotal := float64(sale.SubtotalCents) / 100
	iva := float64(sale.IvaCents) / 100
	total := float64(sale.TotalCents) / 100
	paid := float64(sale.PaidCents) / 100
	change := float64(sale.ChangeCents) / 100

	buf = append(buf, []byte(fmt.Sprintf("SUBTOTAL: $%.2f\n", subtotal))...)
	buf = append(buf, []byte(fmt.Sprintf("IVA 16%%:   $%.2f\n", iva))...)
	buf = append(buf, []byte(ESC_BOLD_ON)...)
	buf = append(buf, []byte(fmt.Sprintf("TOTAL:     $%.2f\n", total))...)
	buf = append(buf, []byte(ESC_BOLD_OFF)...)

	buf = append(buf, []byte(p.separator())...)

	// Pago
	buf = append(buf, []byte(fmt.Sprintf("Pagado con: %s\n", sale.PaymentMethod))...)
	buf = append(buf, []byte(fmt.Sprintf("Cantidad pagada: $%.2f\n", paid))...)
	if sale.ChangeCents > 0 {
		buf = append(buf, []byte(fmt.Sprintf("CAMBIO: $%.2f\n", change))...)
	}

	// ─── FIRMA DIGITAL ────────────────────────────────────────
	buf = append(buf, []byte(p.separator())...)
	buf = append(buf, []byte("FIRMA:\n")...)
	buf = append(buf, []byte(p.wrapText(sale.SignatureHash, 28))...)

	// ─── PIE ───────────────────────────────────────────────────
	buf = append(buf, []byte(p.separator())...)
	buf = append(buf, []byte(ESC_ALIGN_CENTER)...)
	buf = append(buf, []byte("Gracias por su compra\n")...)
	buf = append(buf, []byte("www.integritypos.com\n\n\n")...)

	// Cortar papel
	buf = append(buf, []byte(ESC_CUT_FULL)...)

	// Enviar a impresora
	if _, err := p.file.Write(buf); err != nil {
		p.isOnline = false
		return fmt.Errorf("error escribiendo a impresora: %w", err)
	}

	p.receiptsPrinted++
	return nil
}

// OpenDrawer abre la caja de dinero
func (p *PrinterAdapter) OpenDrawer() error {
	if !p.IsOnline() {
		return fmt.Errorf("impresora fuera de línea")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Comando ESC/POS estándar (RJ-11, 100ms)
	cmd := []byte(ESC_DRAWER)
	if _, err := p.file.Write(cmd); err != nil {
		p.isOnline = false
		return fmt.Errorf("error abriendo caja: %w", err)
	}

	// Esperar a que se abra (RJ-11 suele tomar ~100ms)
	time.Sleep(100 * time.Millisecond)
	return nil
}

// PrintTestReceipt imprime un recibo de prueba para validar impresora
func (p *PrinterAdapter) PrintTestReceipt() error {
	testSale := &domain.Sale{
		ID:            "TEST-001",
		TerminalID:    "TERM-01",
		CajeroID:      "PRUEBA",
		Items: []domain.SaleItem{
			{
				SKU:        "SKU001",
				Name:       "Producto Test",
				Quantity:   2,
				PriceCents: 5000, // $50.00 con IVA
				TotalCents: 10000,
			},
		},
		SubtotalCents: 8620,
		IvaCents:      1380,
		TotalCents:    10000,
		PaymentMethod: domain.PaymentCash,
		PaidCents:     10000,
		ChangeCents:   0,
		SignatureHash: "ABC123DEF456",
		CreatedAt:     time.Now().Format(time.RFC3339),
	}

	return p.PrintReceipt(testSale)
}

// GetStats retorna estadísticas de la impresora
func (p *PrinterAdapter) GetStats() map[string]interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()

	return map[string]interface{}{
		"online":             p.isOnline,
		"mode":               p.mode,
		"device":             p.devicePath,
		"receipts_printed":   p.receiptsPrinted,
		"paper_width_chars":  p.paperWidth,
	}
}

// ─────────────────────────────────────────────────────────────────────────

// init envía comandos de inicialización a la impresora
func (p *PrinterAdapter) init() error {
	cmd := []byte(ESC_INIT)
	_, err := p.file.Write(cmd)
	return err
}

// separator retorna una línea divisora de guiones
func (p *PrinterAdapter) separator() string {
	sep := ""
	for i := 0; i < p.paperWidth; i++ {
		sep += "-"
	}
	return sep + "\n"
}

// wrapText envuelve un texto a N caracteres por línea
func (p *PrinterAdapter) wrapText(text string, width int) string {
	if len(text) <= width {
		return text
	}

	var result string
	for i := 0; i < len(text); i += width {
		end := i + width
		if end > len(text) {
			end = len(text)
		}
		result += text[i:end] + "\n"
	}
	return result
}
