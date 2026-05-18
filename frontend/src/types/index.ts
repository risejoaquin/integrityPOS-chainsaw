export interface User {
  id: number
  username: string
  email: string
  role: 'cashier' | 'admin' | 'manager'
}

export interface LoginResponse {
  access_token: string
  token_type: string
  expires_in: number
  user: User
}

export interface Shift {
  id: number
  user_id: number
  opened_at: string
  closed_at: string | null
  open_balance: number
  close_balance: number | null
  notes: string
  created_at: string
  updated_at: string
  is_active?: boolean
}

export interface Product {
  id: number
  sku: string
  name: string
  description: string
  barcode: string
  price: number
  cost: number
  quantity: number
  category: string
  active: boolean
  sync_status: 'pending' | 'synced' | 'failed'
  created_at: string
  updated_at: string
}

export interface SalesSummary {
  total_sales: number
  total_revenue: number
  total_tax: number
  payment_breakdown: Record<string, number>
  shift_id: number
}

export interface SaleItem {
  product_id: number
  quantity: number
  unit_price: number
  total: number
  product_name?: string
}

export interface CartItem {
  product_id: number
  product_name: string
  quantity: number
  unit_price: number
  total: number
}

export interface Sale {
  id?: number
  shift_id: number
  user_id: number
  total: number
  tax: number
  subtotal: number
  payment_method: string
  notes: string
  sync_status?: 'pending' | 'synced' | 'failed'
  created_at?: string
}

export interface CreateSalePayload {
  sale: {
    shift_id: number
    user_id: number
    total: number
    tax: number
    subtotal: number
    payment_method: string
    notes: string
  }
  items: Array<{
    product_id: number
    quantity: number
    unit_price: number
    total: number
  }>
}