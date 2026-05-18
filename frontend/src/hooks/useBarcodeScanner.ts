import { useEffect, useRef, useCallback } from 'react'

interface BarcodeOptions {
  onBarcode: (barcode: string) => void
  minLength?: number
  timeout?: number
}

/**
 * Hook that listens for rapid keyboard input (barcode scanner)
 * that ends with Enter. Triggers onBarcode callback.
 */
export function useBarcodeScanner({ onBarcode, minLength = 3, timeout = 100 }: BarcodeOptions) {
  const buffer = useRef('')
  const lastTime = useRef(0)

  const handler = useCallback(
    (e: KeyboardEvent) => {
      // Skip if user is in an input element (except search input)
      const tag = (e.target as HTMLElement).tagName
      if (tag === 'INPUT' || tag === 'TEXTAREA') {
        // Still allow if it's the search input with id "pos-search"
        if ((e.target as HTMLInputElement).id !== 'pos-search') {
          return
        }
      }

      const now = Date.now()

      // If time since last key > timeout, reset buffer
      if (now - lastTime.current > timeout && buffer.current.length > 0) {
        buffer.current = ''
      }

      lastTime.current = now

      if (e.key === 'Enter') {
        e.preventDefault()
        const code = buffer.current.trim()
        if (code.length >= minLength) {
          onBarcode(code)
        }
        buffer.current = ''
        return
      }

      // Ignore special keys
      if (e.key.length === 1) {
        buffer.current += e.key
      }
    },
    [onBarcode, minLength, timeout]
  )

  useEffect(() => {
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [handler])
}