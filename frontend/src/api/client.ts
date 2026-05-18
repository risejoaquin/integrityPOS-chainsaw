import axios from 'axios'

// Detect if running inside Electron
const isElectron = !!(window as Window & typeof globalThis & { electronAPI?: { isElectron?: boolean } }).electronAPI?.isElectron

// In production Electron, Vite proxy is not available, so call backend directly
// In development, Vite proxy forwards /api to localhost:8080
const BASE_URL = isElectron ? 'http://localhost:8080/api/v1' : '/api/v1'

const api = axios.create({
  baseURL: BASE_URL,
  timeout: 15000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor: inject X-Hardware-Id and Authorization headers
api.interceptors.request.use((config) => {
  // Hardware ID from localStorage (generated on first run)
  let hwid = localStorage.getItem('X-Hardware-Id')
  if (!hwid) {
    hwid = 'POS-' + Math.random().toString(36).substring(2, 10).toUpperCase()
    localStorage.setItem('X-Hardware-Id', hwid)
  }
  config.headers['X-Hardware-Id'] = hwid

  // JWT token from session storage
  const token = sessionStorage.getItem('access_token')
  if (token) {
    config.headers['Authorization'] = `Bearer ${token}`
  }

  return config
})

// Response interceptor: handle 401 globally
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (axios.isAxiosError(error) && error.response?.status === 401) {
      sessionStorage.removeItem('access_token')
      sessionStorage.removeItem('user')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

export default api