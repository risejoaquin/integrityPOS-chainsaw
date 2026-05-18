/**
 * Formats a cents amount (int64) to display currency string.
 * Example: 1999 -> "$19.99"
 */
export function formatCurrency(cents: number): string {
  const abs = Math.abs(cents)
  const dollars = Math.floor(abs / 100)
  const centavos = abs % 100
  const sign = cents < 0 ? '-' : ''
  return `${sign}$${dollars}.${centavos.toString().padStart(2, '0')}`
}

/**
 * Parses a dollar amount string to cents (int64).
 * Example: "19.99" -> 1999
 */
export function parseCurrency(dollars: string): number {
  const cleaned = dollars.replace(/[^0-9.]/g, '')
  const float = parseFloat(cleaned)
  if (isNaN(float)) return 0
  return Math.round(float * 100)
}