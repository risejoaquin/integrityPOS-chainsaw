package domain

// Fórmula Fiscal Canónica SAT 4.0: Redondeo simétrico exacto
// subtotal = (total * 100 + 58) / 116
// Donde 58 = 116 / 2 (punto medio para redondeo correcto)
func DesglosaIVAIncluido(amount Money) (subtotal Money, iva Money) {
	subtotal = Money((int64(amount)*100 + 58) / 116)
	iva = amount - subtotal
	return
}
