import api from './client'
import type { LoginResponse, Shift, Product, CreateSalePayload, User, SalesSummary } from '../types'

// ─── Auth ───────────────────────────────────────────────
export async function login(username: string, password: string): Promise<LoginResponse> {
  const res = await api.post('/auth/login', { username, password })
  return res.data
}

export async function refreshToken(token: string): Promise<LoginResponse> {
  const res = await api.post('/auth/refresh', { token })
  return res.data
}

// ─── Shifts ─────────────────────────────────────────────
export async function getCurrentShift(): Promise<Shift> {
  const res = await api.get('/shifts/current')
  return res.data
}

export async function openShift(openBalance: number): Promise<Shift> {
  const res = await api.post('/shifts/open', { open_balance: openBalance })
  return res.data
}

export async function closeShift(closeBalance: number): Promise<Shift> {
  const res = await api.post('/shifts/close', { close_balance: closeBalance })
  return res.data
}

export async function getShift(id: number): Promise<Shift> {
  const res = await api.get(`/shifts/${id}`)
  return res.data
}

// ─── Products ───────────────────────────────────────────
export async function listProducts(filters?: Record<string, string>): Promise<Product[]> {
  const params = new URLSearchParams(filters)
  const res = await api.get(`/products?${params.toString()}`)
  return res.data
}

export async function getProduct(id: number): Promise<Product> {
  const res = await api.get(`/products/${id}`)
  return res.data
}

export async function getProductByBarcode(barcode: string): Promise<Product> {
  const res = await api.get(`/products/barcode/${encodeURIComponent(barcode)}`)
  return res.data
}

export async function createProduct(product: Partial<Product>): Promise<Product> {
  const res = await api.post('/products', product)
  return res.data
}

// ─── Sales ──────────────────────────────────────────────
export async function getSalesSummary(shiftId: number): Promise<SalesSummary> {
  const res = await api.get(`/shifts/${shiftId}/summary`)
  return res.data
}

export async function createSale(payload: CreateSalePayload) {
  const res = await api.post('/sales', payload)
  return res.data
}

export async function voidSale(id: number, reason: string) {
  const res = await api.post(`/sales/${id}/void`, { reason })
  return res.data
}

// ─── Hardware ───────────────────────────────────────────
export async function printTicket(saleId: number) {
  const res = await api.post(`/hardware/print-ticket/${saleId}`)
  return res.data
}

export async function openDrawer() {
  const res = await api.post('/hardware/open-drawer')
  return res.data
}

// ─── User (stub for profile) ────────────────────────────
export function getStoredUser(): User | null {
  const raw = sessionStorage.getItem('user')
  if (!raw) return null
  try {
    return JSON.parse(raw) as User
  } catch {
    return null
  }
}

export function storeAuth(data: LoginResponse) {
  sessionStorage.setItem('access_token', data.access_token)
  sessionStorage.setItem('user', JSON.stringify(data.user))
}

export function clearAuth() {
  sessionStorage.removeItem('access_token')
  sessionStorage.removeItem('user')
}