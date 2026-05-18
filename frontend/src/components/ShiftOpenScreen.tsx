import { useState } from 'react'
import { openShift } from '../api/endpoints'
import { formatCurrency, parseCurrency } from '../utils/currency'
import { clearAuth } from '../api/endpoints'

interface Props {
  onShiftOpened: () => void
}

export default function ShiftOpenScreen({ onShiftOpened }: Props) {
  const [amount, setAmount] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleOpen = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    const cents = parseCurrency(amount || '0')
    if (cents < 0) {
      setError('El monto no puede ser negativo')
      return
    }
    setLoading(true)
    try {
      await openShift(cents)
      onShiftOpened()
    } catch (err: unknown) {
      if (err && typeof err === 'object') {
        const axiosErr = err as { response?: { data?: { message?: string } } }
        setError(axiosErr.response?.data?.message || 'Error al abrir caja')
      } else {
        setError('Error de conexión')
      }
    } finally {
      setLoading(false)
    }
  }

  const quickAmounts = [0, 500, 1000, 2000, 5000]

  return (
    <div className="min-h-screen flex items-center justify-center bg-zinc-950">
      <div className="bg-zinc-900 rounded-2xl shadow-2xl p-8 w-full max-w-md border border-zinc-800">
        <div className="text-center mb-8">
          <h1 className="text-2xl font-bold text-zinc-100">Apertura de Caja</h1>
          <p className="text-zinc-500 mt-1">Ingrese el fondo inicial</p>
        </div>

        <form onSubmit={handleOpen} className="space-y-5">
          <div>
            <label className="block text-sm font-medium text-zinc-400 mb-1">
              Fondo inicial
            </label>
            <div className="relative">
              <span className="absolute left-4 top-1/2 -translate-y-1/2 text-zinc-500 text-xl font-bold font-mono">
                $
              </span>
              <input
                type="text"
                autoFocus
                value={amount}
                onChange={(e) => setAmount(e.target.value.replace(/[^0-9.]/g, ''))}
                className="w-full pl-8 pr-4 py-4 bg-zinc-800 border border-zinc-700 rounded-lg text-zinc-100 text-2xl font-bold font-mono text-right
                  focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 focus-visible:border-emerald-500
                  transition-colors duration-150"
                placeholder="0.00"
                inputMode="decimal"
              />
            </div>
            <p className="text-sm text-zinc-600 mt-1 text-right font-mono">
              {amount ? formatCurrency(parseCurrency(amount)) : '$0.00'}
            </p>
          </div>

          <div className="grid grid-cols-5 gap-2">
            {quickAmounts.map((val) => (
              <button
                key={val}
                type="button"
                onClick={() => setAmount(formatCurrency(val).replace('$', ''))}
                className="py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-400 rounded-lg text-sm font-medium
                  transition-colors duration-150
                  focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500"
              >
                {val === 0 ? 'Cero' : formatCurrency(val)}
              </button>
            ))}
          </div>

          {error && (
            <div className="bg-red-900/30 border border-red-800 text-red-400 px-4 py-2 rounded-lg text-sm">
              {error}
            </div>
          )}

          <div className="flex gap-3">
            <button
              type="button"
              onClick={clearAuth}
              className="flex-1 py-3 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 font-medium rounded-lg
                transition-colors duration-150
                focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-zinc-500"
            >
              Cerrar sesión
            </button>
            <button
              type="submit"
              disabled={loading}
              className="flex-1 py-3 bg-emerald-600 hover:bg-emerald-500 disabled:bg-zinc-800 disabled:text-zinc-600 disabled:cursor-not-allowed
                text-white font-bold text-lg rounded-lg
                transition-all duration-150 active:scale-[0.98]
                focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400 focus-visible:ring-offset-2 focus-visible:ring-offset-zinc-950"
            >
              {loading ? 'Abriendo...' : 'Abrir Caja'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}