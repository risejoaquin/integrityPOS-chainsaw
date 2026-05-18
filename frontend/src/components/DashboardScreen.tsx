import { useState, useEffect } from 'react'
import { listProducts, getCurrentShift } from '../api/endpoints'
import { formatCurrency } from '../utils/currency'
import type { Product, Shift } from '../types'

interface KPICard {
  label: string
  value: string
  subvalue?: string
  icon: string
  color: string
}

export default function DashboardScreen() {
  const [products, setProducts] = useState<Product[]>([])
  const [shift, setShift] = useState<Shift | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchData()
  }, [])

  async function fetchData() {
    setLoading(true)
    try {
      const [prods, currentShift] = await Promise.allSettled([
        listProducts(),
        getCurrentShift(),
      ])
      if (prods.status === 'fulfilled') setProducts(prods.value)
      if (currentShift.status === 'fulfilled') setShift(currentShift.value)
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
  }

  const totalProducts = products.length
  const totalStock = products.reduce((sum, p) => sum + p.quantity, 0)
  const lowStockItems = products.filter((p) => p.quantity > 0 && p.quantity <= 5)
  const outOfStockItems = products.filter((p) => p.quantity <= 0)
  const totalInventoryValue = products.reduce((sum, p) => sum + p.price * p.quantity, 0)

  const kpis: KPICard[] = [
    {
      label: 'Productos en Catálogo',
      value: totalProducts.toString(),
      subvalue: `${totalStock} unidades totales`,
      icon: '📦',
      color: 'border-l-blue-500',
    },
    {
      label: 'Valor del Inventario',
      value: formatCurrency(totalInventoryValue),
      subvalue: 'Precio de venta total',
      icon: '💰',
      color: 'border-l-emerald-500',
    },
    {
      label: 'Stock Bajo',
      value: lowStockItems.length.toString(),
      subvalue: `${outOfStockItems.length} agotados`,
      icon: '⚠️',
      color: 'border-l-yellow-500',
    },
    {
      label: 'Estado de Caja',
      value: shift ? `#${shift.id}` : 'Sin turno',
      subvalue: shift
        ? `Abierto: ${new Date(shift.opened_at).toLocaleString('es-MX')}`
        : 'Ningún turno activo',
      icon: '🛒',
      color: shift ? 'border-l-emerald-500' : 'border-l-zinc-600',
    },
  ]

  if (loading) {
    return (
      <div className="h-full flex items-center justify-center bg-zinc-950">
        <div className="text-zinc-500">Cargando dashboard...</div>
      </div>
    )
  }

  return (
    <div className="h-full overflow-y-auto p-4 bg-zinc-950">
      <h1 className="text-xl font-bold text-zinc-100 mb-6">Dashboard</h1>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        {kpis.map((kpi, i) => (
          <div
            key={i}
            className={`bg-zinc-900 ${kpi.color} rounded-xl p-4 border border-zinc-800`}
          >
            <div className="flex items-start justify-between">
              <div>
                <p className="text-zinc-500 text-sm">{kpi.label}</p>
                <p className="text-2xl font-bold text-zinc-100 mt-1">{kpi.value}</p>
                {kpi.subvalue && (
                  <p className="text-zinc-600 text-xs mt-1">{kpi.subvalue}</p>
                )}
              </div>
              <span className="text-2xl">{kpi.icon}</span>
            </div>
          </div>
        ))}
      </div>

      {/* Product Stats */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-6">
        {/* Productos con stock bajo */}
        {lowStockItems.length > 0 && (
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4">
            <h2 className="text-lg font-semibold text-zinc-100 mb-3 flex items-center gap-2">
              <span>⚠️</span> Productos con stock bajo
            </h2>
            <table className="w-full text-sm">
              <thead>
                <tr className="text-zinc-500 text-xs uppercase border-b border-zinc-800">
                  <th className="text-left py-2 font-medium">Producto</th>
                  <th className="text-right py-2 font-medium">Stock</th>
                  <th className="text-right py-2 font-medium">Precio</th>
                </tr>
              </thead>
              <tbody>
                {lowStockItems.slice(0, 10).map((p) => (
                  <tr key={p.id} className="border-b border-zinc-800/50">
                    <td className="py-2 text-zinc-100">{p.name}</td>
                    <td className={`py-2 text-right font-mono ${p.quantity <= 0 ? 'text-red-400' : 'text-yellow-400'}`}>
                      {p.quantity}
                    </td>
                    <td className="py-2 text-right font-mono text-zinc-400">
                      {formatCurrency(p.price)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* Productos agotados */}
        {outOfStockItems.length > 0 && (
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4">
            <h2 className="text-lg font-semibold text-zinc-100 mb-3 flex items-center gap-2">
              <span>🚫</span> Productos agotados
            </h2>
            <table className="w-full text-sm">
              <thead>
                <tr className="text-zinc-500 text-xs uppercase border-b border-zinc-800">
                  <th className="text-left py-2 font-medium">Producto</th>
                  <th className="text-right py-2 font-medium">SKU</th>
                  <th className="text-right py-2 font-medium">Precio</th>
                </tr>
              </thead>
              <tbody>
                {outOfStockItems.slice(0, 10).map((p) => (
                  <tr key={p.id} className="border-b border-zinc-800/50">
                    <td className="py-2 text-zinc-100">{p.name}</td>
                    <td className="py-2 text-right font-mono text-zinc-500">{p.sku}</td>
                    <td className="py-2 text-right font-mono text-zinc-400">
                      {formatCurrency(p.price)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* Top productos por valor */}
        {products.length > 0 && (
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4">
            <h2 className="text-lg font-semibold text-zinc-100 mb-3 flex items-center gap-2">
              <span>🏆</span> Productos de mayor precio
            </h2>
            <table className="w-full text-sm">
              <thead>
                <tr className="text-zinc-500 text-xs uppercase border-b border-zinc-800">
                  <th className="text-left py-2 font-medium">Producto</th>
                  <th className="text-right py-2 font-medium">Stock</th>
                  <th className="text-right py-2 font-medium">Precio</th>
                </tr>
              </thead>
              <tbody>
                {[...products]
                  .sort((a, b) => b.price - a.price)
                  .slice(0, 5)
                  .map((p) => (
                    <tr key={p.id} className="border-b border-zinc-800/50">
                      <td className="py-2 text-zinc-100">{p.name}</td>
                      <td className="py-2 text-right font-mono text-zinc-500">{p.quantity}</td>
                      <td className="py-2 text-right font-mono text-emerald-400">
                        {formatCurrency(p.price)}
                      </td>
                    </tr>
                  ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* System info */}
      <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4">
        <h2 className="text-lg font-semibold text-zinc-100 mb-3">Información del Sistema</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3 text-sm">
          <div>
            <span className="text-zinc-500">Total SKUs</span>
            <p className="text-zinc-100 font-medium">{totalProducts}</p>
          </div>
          <div>
            <span className="text-zinc-500">Unidades en stock</span>
            <p className="text-zinc-100 font-medium">{totalStock}</p>
          </div>
          <div>
            <span className="text-zinc-500">Productos agotados</span>
            <p className="text-red-400 font-medium">{outOfStockItems.length}</p>
          </div>
          <div>
            <span className="text-zinc-500">Valor inventario</span>
            <p className="text-emerald-400 font-medium">{formatCurrency(totalInventoryValue)}</p>
          </div>
        </div>
      </div>
    </div>
  )
}