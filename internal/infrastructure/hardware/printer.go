package hardware

// PrintReceipt is DEPRECATED — Use PrinterAdapter instead
//
// Deprecated: Use NewPrinterAdapter() and PrintReceipt(sale) method
// This stub is kept for backward compatibility only.
//
// Migration path:
//   // Old (deprecated)
//   hardware.PrintReceipt(data)
//
//   // New (use this)
//   printer := hardware.NewPrinterAdapter(hardware.ModeDirect, "/dev/usb/lp0")
//   printer.Open()
//   defer printer.Close()
//   printer.PrintReceipt(sale)
//
// See: PRINTER_ADAPTER.md for complete documentation

