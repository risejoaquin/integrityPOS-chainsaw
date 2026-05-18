import { useState, useRef, useCallback, useEffect } from 'react'
import { toast } from 'sonner'
import { getProductByBarcode, listProducts, createSale, printTicket, openDrawer, closeShift } from '../api/endpoints'
import { formatCurrency } from '../utils/currency'
import { useBarcodeScanner } from '../hooks/useBarcodeScanner'
import { useSession } from './SessionContext'
import CheckoutModal from './CheckoutModal'
import type { CartItem, Product } from '../types'

const TAX_RATE = 0.16 // 16% IVA

// ─── Keyboard Hint Badge ─────────────────────────────
function Kbd({ children }: { children: string }) {
  return (
    <kbd className="ml-1.5 inline-flex items-center px-1.5 py-0.5 text-[10px] font-bold
      bg-zinc-700 text-zinc-300 rounded border border-zinc-600 leading-none">
      {children}
    </kbd>
  )
}

export default function PosScreen() {
  const { shift, onShiftClosed } = useSession()

  const [cart, setCart] = useState<CartItem[]>([])
  const [searchQuery, setSearchQuery] = useState('')
  const [searchResults, setSearchResults] = useState<Product[]>([])
  const [showCheckout, setShowCheckout] = useState(false)
  const [loading, setLoading] = useState(false)
  const searchInputRef = useRef<HTMLInputElement>(null)

  const user = JSON.parse(sessionStorage.getItem('user') || '{}')

  // Calculate totals (ALL in cents = int64)
  const subtotal = cart.reduce((sum, item) => sum + item.total, 0)
  const tax = Math.round(subtotal * TAX_RATE)
  const total = subtotal + tax

  // ─── Barcode Scanner ────────────────────────────────
  const handleBarcode = useCallback(async (barcode: string) => {
    try {
      const product = await getProductByBarcode(barcode)
      addToCart(product)
      toast.success(`${product.name} agregado`, { duration: 2000 })
    } catch {
      toast.error(`Producto no encontrado: ${barcode}`, { duration: 3000 })
    }
  }, [])

  useBarcodeScanner({ onBarcode: handleBarcode })

  // ─── Keyboard Shortcuts ─────────────────────────────
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (showCheckout) return

      const tag = (e.target as HTMLElement).tagName
      const isInput = tag === 'INPUT' || tag === 'TEXTAREA'

      switch (e.key) {
        case 'F1':
          e.preventDefault()
          searchInputRef.current?.focus()
          break
        case 'F8':
          e.preventDefault()
          if (cart.length > 0) setShowCheckout(true)
          break
        case 'Escape':
          if (!isInput && cart.length > 0) {
            e.preventDefault()
            setCart([])
            toast.info('Carrito limpiado')
          }
          break
        case 'Delete':
          if (!isInput && cart.length > 0) {
            e.preventDefault()
            const removed = cart[cart.length - 1]
            setCart((prev) => prev.slice(0, -1))
            toast.info(`${removed.product_name} eliminado`)
          }
          break
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [cart.length, showCheckout])

  // ─── Search Products ────────────────────────────────
  useEffect(() => {
    if (searchQuery.length < 2) {
      setSearchResults([])
      return
    }
    const timer = setTimeout(async () => {
      try {
        const products = await listProducts({ name: searchQuery })
        setSearchResults(products)
      } catch {
        setSearchResults([])
      }
    }, 300)
    return () => clearTimeout(timer)
  }, [searchQuery])

  // ─── Cart Operations ────────────────────────────────
  function addToCart(product: Product) {
    setCart((prev) => {
      const existing = prev.find((item) => item.product_id === product.id)
      if (existing) {
        return prev.map((item) =>
          item.product_id === product.id
            ? { ...item, quantity: item.quantity + 1, total: (item.quantity + 1) * item.unit_price }
            : item
        )
      }
      return [
        ...prev,
        {
          product_id: product.id,
          product_name: product.name,
          quantity: 1,
          unit_price: product.price,
          total: product.price,
        },
      ]
    })
    setSearchQuery('')
    setSearchResults([])
  }

  function selectProduct(product: Product) {
    addToCart(product)
    searchInputRef.current?.focus()
  }

  function removeItem(index: number) {
    const removed = cart[index]
    setCart((prev) => prev.filter((_, i) => i !== index))
    toast.info(`${removed.product_name} eliminado`)
  }

  function updateQuantity(index: number, delta: number) {
    setCart((prev) =>
      prev.map((item, i) =>
        i === index
          ? {
              ...item,
              quantity: Math.max(1, item.quantity + delta),
              total: Math.max(1, item.quantity + delta) * item.unit_price,
            }
          : item
      )
    )
  }

  // ─── Checkout Process ───────────────────────────────
  async function handleCheckout(amountReceived: number, paymentMethod: string, paymentReference?: string) {
    if (!user?.id) return

    setLoading(true)
    try {
      const payload: any = {
        sale: {
          shift_id: shift.id,
          user_id: user.id,
          total,
          tax,
          subtotal,
          payment_method: paymentMethod,
          payment_reference: paymentReference || null,
          notes: '',
        },
        items: cart.map((item) => ({
          product_id: item.product_id,
          quantity: item.quantity,
          unit_price: item.unit_price,
          total: item.total,
        })),
      }

      const sale = await createSale(payload)

      // Fire-and-forget: print ticket and open drawer in parallel
      printTicket(sale.id).catch(() => {})
      openDrawer().catch(() => {})

      setCart([])
      setShowCheckout(false)
      toast.success(`Venta #${sale.id} completada`, { duration: 4000 })
    } catch (err: unknown) {
      const axiosErr = err as { response?: { data?: { message?: string } } }
      toast.error(axiosErr.response?.data?.message || 'Error al procesar venta')
    } finally {
      setLoading(false)
    }
  }

  // ─── Close Shift ────────────────────────────────────
  async function handleCloseShift() {
    if (!confirm('¿Cerrar turno? El carrito se perderá si tiene artículos.')) return
    try {
      await closeShift(total)
      toast.success('Turno cerrado exitosamente')
      onShiftClosed()
    } catch (err: unknown) {
      const axiosErr = err as { response?: { data?: { message?: string } } }
      toast.error(axiosErr.response?.data?.message || 'Error al cerrar turno')
    }
  }

  return (
    <div className="h-full flex flex-col bg-zinc-950 text-zinc-100">
      {/* Sub-header bar: cart summary + keyboard hints + close shift */}
      <div className="bg-zinc-900 px-4 py-1.5 border-b border-zinc-800 flex items-center gap-4 shrink-0">
        <span className="text-sm font-medium text-zinc-500">
          Artículos: <span className="text-zinc-300 font-bold">{cart.length}</span>
        </span>
        <span className="text-sm font-medium text-zinc-500">
          Total: <span className="text-zinc-300 font-bold font-mono">{formatCurrency(total)}</span>
        </span>
        <div className="ml-auto flex items-center gap-3 text-[11px] text-zinc-600">
          <span><Kbd>F1</Kbd> Buscar</span>
          <span><Kbd>F8</Kbd> Cobrar</span>
          <span><Kbd>Esc</Kbd> Limpiar</span>
          <span><Kbd>Supr</Kbd> Quitar</span>
        </div>
        <button
          onClick={handleCloseShift}
          className="ml-4 text-xs text-amber-400 hover:text-amber-300 transition-colors duration-150
            focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 rounded px-2 py-1"
        >
          Cerrar Turno
        </button>
      </div>

      {/* Main Content */}
      <div className="flex flex-1 overflow-hidden">
        {/* Left: Cart Items */}
        <div className="flex-1 flex flex-col border-r border-zinc-800 min-w-0">
          {/* Cart list */}
          <div className="flex-1 overflow-y-auto">
            {cart.length === 0 ? (
              <div className="flex flex-col items-center justify-center h-full text-zinc-600">
                <span className="text-5xl mb-4">🛒</span>
                <span className="text-lg">Escanee un código de barras</span>
                <span className="text-sm mt-2">o presione <Kbd>F1</Kbd> para buscar</span>
              </div>
            ) : (
              <table className="w-full">
                <thead>
                  <tr className="text-zinc-500 text-xs uppercase tracking-wider border-b border-zinc-800">
                    <th className="text-left px-4 py-3 font-medium">Producto</th>
                    <th className="text-center px-2 py-3 font-medium">Cant</th>
                    <th className="text-right px-2 py-3 font-medium">Precio</th>
                    <th className="text-right px-4 py-3 font-medium">Total</th>
                    <th className="px-2 py-3"></th>
                  </tr>
                </thead>
                <tbody>
                  {cart.map((item, index) => (
                    <tr
                      key={index}
                      className="border-b border-zinc-900 hover:bg-zinc-900/60 transition-colors duration-100"
                    >
                      <td className="px-4 py-3 font-medium text-zinc-200">{item.product_name}</td>
                      <td className="px-2 py-3 text-center">
                        <div className="flex items-center justify-center gap-1">
                          <button
                            onClick={() => updateQuantity(index, -1)}
                            className="w-7 h-7 rounded-md bg-zinc-800 hover:bg-zinc-700 text-xs font-bold
                              transition-colors duration-100
                              focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500"
                          >−</button>
                          <span className="w-8 text-center font-mono text-zinc-200">{item.quantity}</span>
                          <button
                            onClick={() => updateQuantity(index, 1)}
                            className="w-7 h-7 rounded-md bg-zinc-800 hover:bg-zinc-700 text-xs font-bold
                              transition-colors duration-100
                              focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500"
                          >+</button>
                        </div>
                      </td>
                      <td className="px-2 py-3 text-right font-mono text-zinc-400">{formatCurrency(item.unit_price)}</td>
                      <td className="px-4 py-3 text-right font-mono font-bold text-zinc-100">{formatCurrency(item.total)}</td>
                      <td className="px-2 py-3">
                        <button
                          onClick={() => removeItem(index)}
                          className="text-red-500 hover:text-red-400 text-sm transition-colors duration-100
                            focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500 rounded"
                        >✕</button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>

        {/* Right: Search & Totals */}
        <div className="w-80 flex flex-col bg-zinc-950 shrink-0">
          {/* Search */}
          <div className="p-3 border-b border-zinc-800">
            <div className="relative">
              <input
                id="pos-search"
                ref={searchInputRef}
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="Buscar producto..."
                className="w-full px-3 py-2.5 bg-zinc-900 border border-zinc-700 rounded-lg text-zinc-100
                  placeholder-zinc-600 font-medium
                  focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 focus-visible:border-emerald-500
                  transition-colors duration-150"
              />
              <span className="absolute right-2.5 top-1/2 -translate-y-1/2">
                <Kbd>F1</Kbd>
              </span>
            </div>
            {searchResults.length > 0 && (
              <div className="mt-2 max-h-60 overflow-y-auto bg-zinc-900 rounded-lg border border-zinc-700 shadow-xl">
                {searchResults.map((p) => (
                  <button
                    key={p.id}
                    onClick={() => selectProduct(p)}
                    className="w-full text-left px-3 py-2.5 hover:bg-zinc-800 border-b border-zinc-800 last:border-0
                      transition-colors duration-100
                      focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 focus-visible:ring-inset"
                  >
                    <div className="font-medium text-zinc-200">{p.name}</div>
                    <div className="text-sm font-mono text-emerald-400">{formatCurrency(p.price)}</div>
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Totals panel — massive TOTAL display */}
          <div className="p-4 space-y-3 border-b border-zinc-800">
            <div className="flex justify-between text-zinc-400 text-sm">
              <span>Subtotal</span>
              <span className="font-mono">{formatCurrency(subtotal)}</span>
            </div>
            <div className="flex justify-between text-zinc-400 text-sm">
              <span>IVA (16%)</span>
              <span className="font-mono">{formatCurrency(tax)}</span>
            </div>
            <div className="pt-3 border-t border-zinc-700">
              <div className="text-zinc-500 text-xs uppercase tracking-wider mb-1">Total a Pagar</div>
              <div className="font-mono font-black text-5xl text-emerald-400 leading-tight tracking-tight">
                {formatCurrency(total)}
              </div>
            </div>
          </div>

          {/* Actions */}
          <div className="p-3 space-y-2 mt-auto">
            <button
              onClick={() => cart.length > 0 && setShowCheckout(true)}
              disabled={cart.length === 0}
              className="w-full py-4 bg-emerald-600 hover:bg-emerald-500 disabled:bg-zinc-800 disabled:text-zinc-600
                disabled:cursor-not-allowed text-white font-bold text-xl rounded-xl
                transition-all duration-150 active:scale-[0.98]
                focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400 focus-visible:ring-offset-2 focus-visible:ring-offset-zinc-950"
            >
              COBRAR <Kbd>F8</Kbd>
            </button>
            <button
              onClick={() => {
                setCart([])
                toast.info('Carrito limpiado')
              }}
              disabled={cart.length === 0}
              className="w-full py-2.5 bg-zinc-900 hover:bg-zinc-800 disabled:opacity-40 disabled:cursor-not-allowed
                text-zinc-400 rounded-lg text-sm
                transition-all duration-150
                focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-zinc-500 focus-visible:ring-offset-2 focus-visible:ring-offset-zinc-950"
            >
              Limpiar carrito <Kbd>Esc</Kbd>
            </button>
          </div>
        </div>
      </div>

      {/* Checkout Modal */}
      {showCheckout && (
        <CheckoutModal
          items={cart}
          subtotal={subtotal}
          tax={tax}
          total={total}
          onConfirm={handleCheckout}
          onCancel={() => setShowCheckout(false)}
        />
      )}

      {/* Loading overlay */}
      {loading && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 backdrop-blur-sm">
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-8 text-zinc-100 text-lg shadow-2xl">
            <div className="animate-pulse flex items-center gap-3">
              <div className="w-3 h-3 rounded-full bg-emerald-500"></div>
              Procesando venta...
            </div>
          </div>
        </div>
      )}
    </div>
  )
}