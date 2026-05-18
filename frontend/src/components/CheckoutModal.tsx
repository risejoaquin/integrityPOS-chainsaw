import { useState, useEffect, useRef } from 'react'
import { formatCurrency } from '../utils/currency'
import type { CartItem } from '../types'

interface Props {
  items: CartItem[]
  subtotal: number
  tax: number
  total: number
  onConfirm: (amountReceived: number, paymentMethod: string, paymentReference?: string) => void
  onCancel: () => void
}

export default function CheckoutModal({ items, subtotal, tax, total, onConfirm, onCancel }: Props) {
  const [method, setMethod] = useState<'cash' | 'card'>('cash')
  const [received, setReceived] = useState('')
  const [reference, setReference] = useState('')
  const [error, setError] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    inputRef.current?.focus()
  }, [method])

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onCancel()
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [onCancel])

  const receivedCents = Math.round(parseFloat(received.replace(/[^0-9.]/g, '') || '0') * 100)
  const change = receivedCents - total
  const sufficient = receivedCents >= total

  const handleConfirm = () => {
    setError('')

    if (method === 'cash') {
      if (!received || !sufficient) {
        setError('El monto recibido es insuficiente')
        return
      }
      onConfirm(receivedCents, 'cash')
    } else {
      // Card — no cash calculation, total is always charged
      onConfirm(total, 'card', reference || undefined)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50 backdrop-blur-sm">
      <div className="bg-zinc-900 rounded-2xl shadow-2xl w-full max-w-lg border border-zinc-800 p-6">
        <h2 className="text-xl font-bold text-zinc-100 mb-4">Cobrar</h2>

        {/* Summary */}
        <div className="bg-zinc-950 rounded-xl p-4 mb-4 space-y-1">
          <div className="flex justify-between text-zinc-500">
            <span>Artículos</span>
            <span className="font-mono">{items.length}</span>
          </div>
          <div className="flex justify-between text-zinc-400">
            <span>Subtotal</span>
            <span className="font-mono">{formatCurrency(subtotal)}</span>
          </div>
          <div className="flex justify-between text-zinc-400">
            <span>IVA</span>
            <span className="font-mono">{formatCurrency(tax)}</span>
          </div>
          <div className="flex justify-between text-zinc-100 text-xl font-bold border-t border-zinc-800 pt-2 mt-2">
            <span>TOTAL</span>
            <span className="font-mono">{formatCurrency(total)}</span>
          </div>
        </div>

        {/* Payment method — big clear buttons */}
        <div className="flex gap-3 mb-5">
          <button
            type="button"
            onClick={() => { setMethod('cash'); setError('') }}
            className={`flex-1 py-4 rounded-xl font-bold text-lg transition-all duration-150
              focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500
              ${method === 'cash'
                ? 'bg-emerald-600 text-white ring-2 ring-emerald-400 scale-[1.02]'
                : 'bg-zinc-800 text-zinc-300 hover:bg-zinc-700'
              }`}
          >
            <span className="block text-2xl mb-1">💵</span>
            EFECTIVO
          </button>
          <button
            type="button"
            onClick={() => { setMethod('card'); setError('') }}
            className={`flex-1 py-4 rounded-xl font-bold text-lg transition-all duration-150
              focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500
              ${method === 'card'
                ? 'bg-blue-600 text-white ring-2 ring-blue-400 scale-[1.02]'
                : 'bg-zinc-800 text-zinc-300 hover:bg-zinc-700'
              }`}
          >
            <span className="block text-2xl mb-1">💳</span>
            TARJETA
          </button>
        </div>

        {/* Cash flow: amount received + change */}
        {method === 'cash' && (
          <div className="mb-4">
            <label className="block text-sm font-medium text-zinc-400 mb-1">
              Monto recibido
            </label>
            <div className="relative">
              <span className="absolute left-4 top-1/2 -translate-y-1/2 text-zinc-500 text-lg font-bold font-mono">$</span>
              <input
                ref={inputRef}
                type="text"
                value={received}
                onChange={(e) => setReceived(e.target.value.replace(/[^0-9.]/g, ''))}
                className="w-full pl-8 pr-4 py-3 bg-zinc-800 border border-zinc-700 rounded-lg text-zinc-100
                  text-xl font-bold font-mono text-right
                  focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 focus-visible:border-emerald-500
                  transition-colors duration-150"
                placeholder="0.00"
                inputMode="decimal"
                onKeyDown={(e) => { if (e.key === 'Enter') handleConfirm() }}
              />
            </div>
            {receivedCents > 0 && (
              <div className={`text-right mt-2 text-lg font-bold font-mono ${change >= 0 ? 'text-emerald-400' : 'text-red-400'}`}>
                Cambio: {formatCurrency(Math.max(0, change))}
              </div>
            )}
          </div>
        )}

        {/* Card flow: optional reference */}
        {method === 'card' && (
          <div className="mb-4">
            <div className="bg-zinc-800/50 border border-zinc-700 rounded-xl p-4 mb-3">
              <p className="text-zinc-400 text-sm mb-1">Total a cobrar</p>
              <p className="text-zinc-100 text-2xl font-bold font-mono">{formatCurrency(total)}</p>
              <p className="text-zinc-500 text-xs mt-1">Pago con tarjeta — no genera cambio</p>
            </div>
            <label className="block text-sm font-medium text-zinc-400 mb-1">
              Referencia / Voucher <span className="text-zinc-600">(opcional)</span>
            </label>
            <input
              ref={inputRef}
              type="text"
              value={reference}
              onChange={(e) => setReference(e.target.value)}
              className="w-full px-4 py-3 bg-zinc-800 border border-zinc-700 rounded-lg text-zinc-100
                text-base font-mono
                focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:border-blue-500
                transition-colors duration-150"
              placeholder="Ej: Voucher #123456"
              onKeyDown={(e) => { if (e.key === 'Enter') handleConfirm() }}
            />
          </div>
        )}

        {error && (
          <div className="bg-red-900/30 border border-red-800 text-red-400 px-4 py-2 rounded-lg text-sm mb-4">{error}</div>
        )}

        <div className="flex gap-3">
          <button type="button" onClick={onCancel}
            className="flex-1 py-3 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 font-medium rounded-lg
              transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-zinc-500">
            Cancelar (Esc)
          </button>
          <button type="button" onClick={handleConfirm}
            className="flex-1 py-3 bg-emerald-600 hover:bg-emerald-500 text-white font-bold text-lg rounded-lg
              transition-all duration-150 active:scale-[0.98]
              focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400 focus-visible:ring-offset-2 focus-visible:ring-offset-zinc-900">
            Confirmar (Enter)
          </button>
        </div>
      </div>
    </div>
  )
}