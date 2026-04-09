# PrinterAdapter — Impresora Térmica ESC/POS

**Estado:** ✅ Implementado (Fase 7)  
**Protocolo:** ESC/POS (Epson Standard Thermal Printer)  
**Soportado:** 80mm (32 caracteres), corte parcial y completo, apertura de caja  

---

## 📋 Características

### Formatos
- **Alineación**: Izquierda, centro, derecha
- **Énfasis**: Negrita, doble altura, doble ancho
- **Encabezados**: Empresa, terminal, recibo
- **Desglose**: Subtotal, IVA 16%, Total
- **Pago**: Método, cantidad pagada, cambio

### Acciones
- 🖨️ Impresión de recibos venta
- 🗂️ Corte de papel (parcial / completo)
- 💰 Apertura de caja (RJ-11)
- 📊 Estadísticas en tiempo real

---

## 🔧 Configuración

### Variables de Entorno

```bash
# Modo de conexión: direct | network | stdout
PRINTER_MODE=direct

# Dispositivo (Linux/USB)
PRINTER_DEVICE=/dev/usb/lp0

# Alternativas compatibles:
# /dev/ttyUSB0 (RS-232)
# /dev/lp0 (printer port)
```

### Inicialización en main.go

```go
printer := hardware.NewPrinterAdapter(hardware.PrinterMode(printerMode), printerDevice)
if err := printer.Open(); err != nil {
    log.Printf("ADVERTENCIA: No se pudo inicializar impresora: %v", err)
    printer = nil // Seguir sin impresora
}
defer printer.Close()
```

### Inyección en Server

```go
server := web.NewServer(":8080", db, secret, syncWorker, printer)
```

---

## 📡 REST API

### POST /printer/test
Imprime un recibo de prueba para validar conexión.

**Response (200 OK):**
```json
{
  "status": "ok",
  "message": "test receipt printed",
  "stats": {
    "online": true,
    "mode": "direct",
    "device": "/dev/usb/lp0",
    "receipts_printed": 5,
    "paper_width_chars": 32
  }
}
```

**Response (503 Service Unavailable):**
```json
{
  "status": "offline",
  "error": "printer disconnected"
}
```

---

### POST /printer/drawer
Abre la caja de dinero (RJ-11 estándar, ~100ms).

**Response (200 OK):**
```json
{
  "status": "ok",
  "message": "drawer opened"
}
```

---

## 🎯 Integración Automática

### Flujo en POST /sale

1. **Validación de stock** (en tx)
2. **Escritura db** (receipts + items + kardex + outbox)
3. **Commit tx**
4. **Impresión async** (background goroutine, no bloquea HTTP)
   - Si impresora offline: log error, venta continúa
   - Si error de escritura USB: marca printer offline

```go
go func() {
    if err := s.printer.PrintReceipt(sale); err != nil {
        fmt.Printf("[PRINTER ERROR] %s: %v\n", receiptID, err)
    }
}()
```

---

## 🧪 Tests

### Unit Tests (printer_adapter_test.go)

```bash
go test ./internal/infrastructure/hardware -v
```

**Casos:**
- ✅ `TestPrinterStdoutMode` — Impresión en stdout (CI)
- ✅ `TestPrinterReceiptCompleto` — Todos los campos
- ✅ `TestPrinterOpenDrawer` — Apertura de caja
- ✅ `TestPrinterStats` — Estadísticas
- ✅ `TestPrinterOffline` — Comportamiento offline
- ✅ `TestPrinterReceiptWithCardPayment` — Pago tarjeta

---

## 🐧 Instalación en Raspberry Pi

### 1. Drivers USB
```bash
sudo apt update && sudo apt install cups-client libcups2-dev
```

### 2. Permisos
```bash
# Detectar puerto impresora
ls -la /dev/usb/lp*
lsusb

# Permisos (si es necesario)
sudo usermod -aG lpadmin $USER
sudo chmod 666 /dev/usb/lp0
```

### 3. Test Direct
```bash
# Imprimir caracteres de prueba
echo -e "HOLA\n" > /dev/usb/lp0
```

### 4. Configurar IntegrityPOS
```bash
export PRINTER_MODE=direct
export PRINTER_DEVICE=/dev/usb/lp0
./integritypos_server
```

---

## 🎨 Formato ESC/POS Generado

```
┌────────────────────────────────┐
│       INTEGRITYPOS             │
│      RECIBO DE VENTA           │
├────────────────────────────────┤
│ Terminal: TERMINAL-01          │
│ Cajero: JUAN                   │
│ Recibo: UUID-12345             │
│ Fecha: 02/01/2026 15:04:05     │
├────────────────────────────────┤
│ DESC.                QTY PRICEE│
├────────────────────────────────┤
│ Manzana Roja                   │
│   x3 @ $15.00 = $45.00         │
│ Leche Entera 1L                │
│   x2 @ $39.00 = $78.00         │
├────────────────────────────────┤
│                   SUBTOTAL:... │
│                   IVA 16%:   $ │
│                       TOTAL: $ │
├────────────────────────────────┤
│ Pagado con: CASH               │
│ Cantidad pagada: $123.00       │
│ CAMBIO: $2.75                  │
├────────────────────────────────┤
│ FIRMA:                         │
│ SHA256-HMAC-ABC123DEF456...    │
├────────────────────────────────┤
│    Gracias por su compra        │
│      www.integritypos.com       │
└────────────────────────────────┘
[CUT]
```

---

## ⚙️ Comandos ESC/POS Implementados

| Comando | Descripción | Bytes |
|---------|-------------|-------|
| `ESC @` | Inicialización | `0x1B 0x40` |
| `ESC [ 0 @` | Alineación izquierda | `0x1B 0x5B 0x30 0x40` |
| `ESC [ 1 @` | Alineación centro | `0x1B 0x5B 0x31 0x40` |
| `ESC [ 2 @` | Alineación derecha | `0x1B 0x5B 0x32 0x40` |
| `ESC E 1` | Negrita ON | `0x1B 0x45 0x01` |
| `ESC E 0` | Negrita OFF | `0x1B 0x45 0x00` |
| `GS p 0 25` | Apertura caja | `0x1D 0x70 0x00 0x19` |
| `GS V 65` | Corte completo | `0x1D 0x56 0x41` |
| `GS V 66` | Corte parcial | `0x1D 0x56 0x42` |

---

## 🔍 Troubleshooting

### Impresora no responde
```bash
# Test de puerto
cat > /dev/usb/lp0 <<< "TEST"

# Si /dev/usb/lp0 no existe
lsusb  # Identificar VID:PID
```

### Permiso denegado
```bash
sudo chmod 666 /dev/usb/lp0
# O permanente:
sudo usermod -aG lpadmin $(whoami)
```

### Impresora en modo "stdout" (desarrollo)
```bash
export PRINTER_MODE=stdout
# Recibos se imprimen en consola, útil para testing sin hardware
```

### Performance
- **Latencia de impresión:** ~2-3 segundos (80mm x 10 líneas)
- **No bloquea HTTP:** Impresión async en background goroutine
- **Reintentos:** Si falla, se loguea y venta se marca como exitosa

---

## 📊 Estadísticas en Vivo

```bash
curl -s http://localhost:8080/printer/test | jq '.stats'
```

**Output:**
```json
{
  "online": true,
  "mode": "direct",
  "device": "/dev/usb/lp0",
  "receipts_printed": 42,
  "paper_width_chars": 32
}
```

---

## 🚀 Próximos Pasos (v2.0+)

- [ ] Soporte red (TCP/IP, Windows Print Spooler)
- [ ] Logging de errores a audit_logs
- [ ] Retry con exponential backoff (como SyncWorker)
- [ ] Soporte ESCPOS+ (colores, códigos QR)
- [ ] Monitoreo de nivel de papel
- [ ] Impresora térmica de etiquetas (ZPL)

---

**Última actualización:** 2026-03-27  
**Versión:** 1.0 (ESC/POS)  
**Autor:** IntegrityPOS Team
