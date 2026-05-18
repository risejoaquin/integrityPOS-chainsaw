package hardware

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"integritypos-backend/internal/core/domain"
)

// ESC/POS command bytes
var (
	escInit        = []byte{0x1B, 0x40}                   // Initialize printer
	escCut         = []byte{0x1B, 0x6D}                   // Partial cut (GS V m)
	escFeed        = []byte{0x1B, 0x64, 0x03}             // Feed 3 lines
	escFeed6       = []byte{0x1B, 0x64, 0x06}             // Feed 6 lines
	escAlignCenter = []byte{0x1B, 0x61, 0x01}             // Center alignment
	escAlignLeft   = []byte{0x1B, 0x61, 0x00}             // Left alignment
	escAlignRight  = []byte{0x1B, 0x61, 0x02}             // Right alignment
	escBoldOn      = []byte{0x1B, 0x45, 0x01}             // Bold on
	escBoldOff     = []byte{0x1B, 0x45, 0x00}             // Bold off
	escDoubleOn    = []byte{0x1B, 0x21, 0x30}             // Double width + height
	escDoubleOff   = []byte{0x1B, 0x21, 0x00}             // Normal text
	escDrawerKick  = []byte{0x1B, 0x70, 0x00, 0x19, 0xFA} // Kick drawer (RJ11 port)
)

// ESCPOSPrinter handles thermal printer operations via ESC/POS protocol
type ESCPOSPrinter struct {
	devicePath string // e.g., "/dev/usb/lp0" on Linux, "LPT1" on Windows
	mockMode   bool
}

// NewESCPOSPrinter creates a new ESC/POS printer instance
func NewESCPOSPrinter() *ESCPOSPrinter {
	devicePath := os.Getenv("PRINTER_DEVICE")
	mockMode := os.Getenv("MOCK_PRINTER") == "true" || devicePath == ""

	return &ESCPOSPrinter{
		devicePath: devicePath,
		mockMode:   mockMode,
	}
}

// BuildTicket assembles a complete ESC/POS ticket from a sale with items and metadata.
func (p *ESCPOSPrinter) BuildTicket(ctx context.Context, sale *domain.Sale, items []*domain.SaleItem, businessName string) []byte {
	var buf []byte

	// Initialize
	buf = append(buf, escInit...)

	// Header: business name
	buf = append(buf, escAlignCenter...)
	buf = append(buf, escDoubleOn...)
	buf = append(buf, []byte(businessName+"\n")...)
	buf = append(buf, escDoubleOff...)
	buf = append(buf, []byte("================================")...)
	buf = append(buf, []byte{0x0A}...) // LF

	// Ticket metadata
	buf = append(buf, escAlignLeft...)
	buf = append(buf, []byte(fmt.Sprintf("Ticket #%d\n", sale.ID))...)
	buf = append(buf, []byte(fmt.Sprintf("Fecha: %s\n", sale.CreatedAt.Format("2006-01-02 15:04:05")))...)
	buf = append(buf, []byte{0x0A}...)

	// Items header
	buf = append(buf, escBoldOn...)
	buf = append(buf, []byte("CANT  DESCRIPCION       IMPORTE\n")...)
	buf = append(buf, escBoldOff...)
	buf = append(buf, []byte("--------------------------------\n")...)

	// Items
	for _, item := range items {
		line := fmt.Sprintf("%-4d %-18s $%6.2f\n",
			item.Quantity,
			truncateString(fmt.Sprintf("#%d", item.ProductID), 18),
			float64(item.Total)/100)
		buf = append(buf, []byte(line)...)
	}

	// Separator
	buf = append(buf, []byte("--------------------------------\n")...)

	// Totals
	buf = append(buf, escAlignRight...)
	buf = append(buf, []byte(fmt.Sprintf("Subtotal: $%7.2f\n", float64(sale.Subtotal)/100))...)
	buf = append(buf, []byte(fmt.Sprintf("IVA:      $%7.2f\n", float64(sale.Tax)/100))...)
	buf = append(buf, escBoldOn...)
	buf = append(buf, []byte(fmt.Sprintf("TOTAL:    $%7.2f\n", float64(sale.Total)/100))...)
	buf = append(buf, escBoldOff...)
	buf = append(buf, []byte(fmt.Sprintf("Pago: %s\n", sale.PaymentMethod))...)

	if sale.Notes != "" {
		buf = append(buf, []byte(fmt.Sprintf("Notas: %s\n", sale.Notes))...)
	}

	// Footer
	buf = append(buf, escAlignCenter...)
	buf = append(buf, []byte{0x0A}...)
	buf = append(buf, []byte("Gracias por su compra\n")...)
	buf = append(buf, []byte("Vuelva pronto\n")...)
	buf = append(buf, escFeed6...)
	buf = append(buf, escCut...)

	return buf
}

// PrintTicket prints a formatted ticket for a complete sale.
func (p *ESCPOSPrinter) PrintTicket(ctx context.Context, sale *domain.Sale, items []*domain.SaleItem) error {
	businessName := os.Getenv("BUSINESS_NAME")
	if businessName == "" {
		businessName = "INTEGRITY POS"
	}
	data := p.BuildTicket(ctx, sale, items, businessName)
	return p.sendToPrinter(data)
}

// PrintRaw sends raw bytes directly to the printer
func (p *ESCPOSPrinter) PrintRaw(ctx context.Context, data []byte) error {
	return p.sendToPrinter(data)
}

// KickDrawer sends the ESC/POS command to open the cash drawer via RJ11
func (p *ESCPOSPrinter) KickDrawer(ctx context.Context) error {
	return p.sendToPrinter(escDrawerKick)
}

// sendToPrinter writes bytes to the physical printer or falls back to mock
func (p *ESCPOSPrinter) sendToPrinter(data []byte) error {
	if p.mockMode {
		log.Printf("[printer-mock] Would print %d bytes (device: %q, content preview):", len(data), p.devicePath)
		var preview strings.Builder
		for _, b := range data {
			if b >= 32 && b <= 126 {
				preview.WriteByte(b)
			} else if b == 0x0A {
				preview.WriteByte('\n')
			}
		}
		log.Printf("[printer-mock] %s", preview.String())
		return nil
	}

	switch runtime.GOOS {
	case "linux":
		return p.writeToFile(data)
	case "windows":
		return p.writeToWindowsPrinter(data)
	default:
		log.Printf("[printer] Unknown OS %s, falling back to mock", runtime.GOOS)
		return nil
	}
}

func (p *ESCPOSPrinter) writeToFile(data []byte) error {
	f, err := os.OpenFile(p.devicePath, os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("failed to open printer device %s: %w", p.devicePath, err)
	}
	defer f.Close()

	n, err := f.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to printer: %w", err)
	}
	if n != len(data) {
		return fmt.Errorf("short write to printer: %d of %d bytes", n, len(data))
	}
	return nil
}

func (p *ESCPOSPrinter) writeToWindowsPrinter(data []byte) error {
	printerName := os.Getenv("PRINTER_NAME")
	if printerName == "" {
		printerName = "EPSON"
	}

	tmpFile, err := os.CreateTemp("", "pos_print_*.bin")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpPath)

	cmd := exec.Command("cmd", "/c", fmt.Sprintf("print /D:\"%s\" \"%s\"", printerName, tmpPath))
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[printer] Windows print command failed: %v, output: %s", err, string(output))
		return fmt.Errorf("windows print failed: %w", err)
	}
	return nil
}

// Ensure io.Writer compatibility
var _ io.Writer = (*ESCPOSPrinter)(nil)

func (p *ESCPOSPrinter) Write(data []byte) (int, error) {
	err := p.sendToPrinter(data)
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
