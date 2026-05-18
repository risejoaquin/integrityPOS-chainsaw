import { useState, useEffect } from 'react'
import { listProducts, createProduct } from '../api/endpoints'
import { formatCurrency, parseCurrency } from '../utils/currency'
import type { Product } from '../types'

interface ProductForm {
  sku: string
  name: string
  description: string
  barcode: string
  price: string  // en dólares para el input
  cost: string
  category: string
  quantity: string
}

const emptyForm: ProductForm = {
  sku: '', name: '', description: '', barcode: '',
  price: '', cost: '', category: '', quantity: '0',
}

export default function InventoryScreen() {
  const [products, setProducts] = useState<Product[]>([])
  const [loading, setLoading] = useState(true)
  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState<ProductForm>(emptyForm)
  const [formError, setFormError] = useState('')
  const [search, setSearch] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => { fetchProducts() }, [])

  async function fetchProducts() {
    setLoading(true)
    try {
      const data = await listProducts()
      setProducts(data)
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
  }

  const filtered = search
    ? products.filter((p) =>
        p.name.toLowerCase().includes(search.toLowerCase()) ||
        p.sku.toLowerCase().includes(search.toLowerCase()) ||
        p.barcode.toLowerCase().includes(search.toLowerCase())
      )
    : products

  function handleInputChange(e: React.ChangeEvent<HTMLInputElement>) {
    setForm({ ...form, [e.target.name]: e.target.value })
  }

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    setFormError('')

    // Validar
    if (!form.name || !form.sku) {
      setFormError('Nombre y SKU son requeridos')
      return
    }

    const priceCents = parseCurrency(form.price || '0')
    const costCents = parseCurrency(form.cost || '0')
    const quantity = parseInt(form.quantity || '0', 10)

    if (priceCents < 0 || costCents < 0) {
      setFormError('Precio y costo no pueden ser negativos')
      return
    }

    setSubmitting(true)
    try {
      await createProduct({
        sku: form.sku,
        name: form.name,
        description: form.description,
        barcode: form.barcode,
        price: priceCents,
        cost: costCents,
        category: form.category,
        quantity: quantity,
        active: true,
      })
      setForm(emptyForm)
      setShowForm(false)
      await fetchProducts()
    } catch (err: unknown) {
      const axiosErr = err as { response?: { data?: { message?: string } } }
      setFormError(axiosErr.response?.data?.message || 'Error al crear producto')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="h-full flex flex-col p-4 bg-zinc-950">
      {/* Header */}
      <div className="flex items-center justify-between mb-4 shrink-0">
        <h1 className="text-xl font-bold text-zinc-100">Inventario</h1>
        <div className="flex items-center gap-3">
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Buscar por nombre o SKU..."
            className="px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-zinc-100 text-sm w-64 placeholder-zinc-600
              focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 focus-visible:border-emerald-500
              transition-colors duration-150"
          />
          <button
            onClick={() => setShowForm(!showForm)}
            className="px-4 py-2 bg-emerald-600 hover:bg-emerald-500 text-white font-medium rounded-lg text-sm
              transition-colors duration-150
              focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400 focus-visible:ring-offset-2 focus-visible:ring-offset-zinc-950"
          >
            {showForm ? 'Cancelar' : '+ Nuevo Producto'}
          </button>
          <button
            onClick={fetchProducts}
            className="px-3 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-400 rounded-lg text-sm
              transition-colors duration-150
              focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-zinc-500"
          >
            ↻
          </button>
        </div>
      </div>

      {/* Create Form */}
      {showForm && (
        <form onSubmit={handleCreate} className="bg-zinc-900 border border-zinc-800 rounded-xl p-4 mb-4 shrink-0">
          <h2 className="text-lg font-semibold text-zinc-100 mb-3">Nuevo Producto</h2>
          <div className="grid grid-cols-3 gap-3 mb-3">
            <div>
              <label className="block text-xs text-zinc-500 mb-1">SKU</label>
              <input name="sku" value={form.sku} onChange={handleInputChange}
                className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-zinc-100 text-sm
                  focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 focus-visible:border-emerald-500
                  transition-colors duration-150"
                placeholder="SKU-001" />
            </div>
            <div>
              <label className="block text-xs text-zinc-500 mb-1">Nombre *</label>
              <input name="name" value={form.name} onChange={handleInputChange}
                className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-zinc-100 text-sm
                  focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 focus-visible:border-emerald-500
                  transition-colors duration-150"
                placeholder="Nombre del producto" />
            </div>
            <div>
              <label className="block text-xs text-zinc-500 mb-1">Categoría</label>
              <input name="category" value={form.category} onChange={handleInputChange}
                className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-zinc-100 text-sm
                  focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 focus-visible:border-emerald-500
                  transition-colors duration-150"
                placeholder="Ferretería" />
            </div>
            <div>
              <label className="block text-xs text-zinc-500 mb-1">Código de Barras</label>
              <input name="barcode" value={form.barcode} onChange={handleInputChange}
                className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-zinc-100 text-sm
                  focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 focus-visible:border-emerald-500
                  transition-colors duration-150"
                placeholder="7501234567890" inputMode="numeric" />
            </div>
            <div>
              <label className="block text-xs text-zinc-500 mb-1">Precio ($)</label>
              <input name="price" value={form.price} onChange={handleInputChange}
                className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-zinc-100 text-sm
                  focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 focus-visible:border-emerald-500
                  transition-colors duration-150"
                placeholder="15.50" inputMode="decimal" />
            </div>
            <div>
              <label className="block text-xs text-zinc-500 mb-1">Costo ($)</label>
              <input name="cost" value={form.cost} onChange={handleInputChange}
                className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-zinc-100 text-sm
                  focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 focus-visible:border-emerald-500
                  transition-colors duration-150"
                placeholder="10.00" inputMode="decimal" />
            </div>
            <div>
              <label className="block text-xs text-zinc-500 mb-1">Stock</label>
              <input name="quantity" value={form.quantity} onChange={handleInputChange}
                className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-zinc-100 text-sm
                  focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 focus-visible:border-emerald-500
                  transition-colors duration-150"
                placeholder="0" inputMode="numeric" />
            </div>
          </div>
          <div className="mb-3">
            <label className="block text-xs text-zinc-500 mb-1">Descripción</label>
            <input name="description" value={form.description} onChange={handleInputChange}
              className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-zinc-100 text-sm
                focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 focus-visible:border-emerald-500
                transition-colors duration-150"
              placeholder="Descripción opcional" />
          </div>
          {formError && (
            <div className="bg-red-900/30 border border-red-800 text-red-400 px-3 py-2 rounded-lg text-sm mb-3">
              {formError}
            </div>
          )}
          <button type="submit" disabled={submitting}
            className="px-6 py-2 bg-emerald-600 hover:bg-emerald-500 disabled:bg-zinc-800 disabled:text-zinc-600 disabled:cursor-not-allowed text-white font-medium rounded-lg
              transition-colors duration-150
              focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400 focus-visible:ring-offset-2 focus-visible:ring-offset-zinc-950">
            {submitting ? 'Creando...' : 'Crear Producto'}
          </button>
          <p className="text-xs text-zinc-600 mt-2">
            Los precios se envían como enteros (centavos) al backend. Ej: $15.50 → 1550
          </p>
        </form>
      )}

      {/* Table */}
      <div className="flex-1 overflow-y-auto bg-zinc-900 border border-zinc-800 rounded-xl">
        {loading ? (
          <div className="flex items-center justify-center h-32 text-zinc-500">Cargando...</div>
        ) : filtered.length === 0 ? (
          <div className="flex items-center justify-center h-32 text-zinc-600">
            {search ? 'Sin resultados' : 'No hay productos. Cree el primero.'}
          </div>
        ) : (
          <table className="w-full">
            <thead>
              <tr className="text-zinc-500 text-xs uppercase border-b border-zinc-800">
                <th className="text-left px-4 py-3 font-medium">ID</th>
                <th className="text-left px-4 py-3 font-medium">Nombre</th>
                <th className="text-left px-4 py-3 font-medium">SKU</th>
                <th className="text-left px-4 py-3 font-medium">Barcode</th>
                <th className="text-left px-4 py-3 font-medium">Categoría</th>
                <th className="text-right px-4 py-3 font-medium">Stock</th>
                <th className="text-right px-4 py-3 font-medium">Precio</th>
                <th className="text-right px-4 py-3 font-medium">Costo</th>
                <th className="text-center px-4 py-3 font-medium">Sync</th>
                <th className="text-center px-4 py-3 font-medium">Activo</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((p) => (
                <tr key={p.id} className="border-b border-zinc-800/50 hover:bg-zinc-800/30 transition-colors duration-100">
                  <td className="px-4 py-3 text-zinc-500 font-mono text-sm">{p.id}</td>
                  <td className="px-4 py-3 font-medium text-zinc-100">{p.name}</td>
                  <td className="px-4 py-3 text-zinc-500 font-mono text-sm">{p.sku}</td>
                  <td className="px-4 py-3 text-zinc-600 font-mono text-sm">{p.barcode || '-'}</td>
                  <td className="px-4 py-3 text-zinc-500">{p.category}</td>
                  <td className={`px-4 py-3 text-right font-mono ${p.quantity <= 0 ? 'text-red-400' : 'text-zinc-400'}`}>
                    {p.quantity}
                  </td>
                  <td className="px-4 py-3 text-right font-mono text-emerald-400">
                    {formatCurrency(p.price)}
                  </td>
                  <td className="px-4 py-3 text-right font-mono text-zinc-500">
                    {formatCurrency(p.cost)}
                  </td>
                  <td className="px-4 py-3 text-center">
                    {p.sync_status === 'synced' ? (
                      <span className="inline-flex items-center gap-1 text-emerald-400" title="Sincronizado">
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1 text-yellow-400" title="Pendiente de sincronización">
                        <svg className="w-4 h-4 animate-pulse" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-center">
                    <span className={`inline-block w-2 h-2 rounded-full ${p.active ? 'bg-emerald-400' : 'bg-zinc-600'}`} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}