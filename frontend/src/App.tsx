import { useState, useEffect } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import LoginScreen from './components/LoginScreen'
import ShiftOpenScreen from './components/ShiftOpenScreen'
import AdminLayout from './components/AdminLayout'
import PosScreen from './components/PosScreen'
import InventoryScreen from './components/InventoryScreen'
import DashboardScreen from './components/DashboardScreen'
import { SessionProvider } from './components/SessionContext'
import { getCurrentShift, clearAuth, getStoredUser } from './api/endpoints'
import type { Shift } from './types'

type AppPhase = 'loading' | 'login' | 'shift-open' | 'authenticated'

export default function App() {
  const [phase, setPhase] = useState<AppPhase>('loading')
  const [shift, setShift] = useState<Shift | null>(null)

  useEffect(() => {
    checkAuthState()
  }, [])

  async function checkAuthState() {
    const token = sessionStorage.getItem('access_token')
    if (!token) {
      setPhase('login')
      return
    }

    try {
      const currentShift = await getCurrentShift()
      setShift(currentShift)
      setPhase('authenticated')
    } catch (err: unknown) {
      if (err && typeof err === 'object') {
        const axiosErr = err as { response?: { status?: number } }
        if (axiosErr.response?.status === 401) {
          clearAuth()
          setPhase('login')
          return
        }
      }
      // No open shift → go to shift-open
      setPhase('shift-open')
    }
  }

  function handleLogin() {
    setPhase('loading')
    getCurrentShift()
      .then((s) => {
        setShift(s)
        setPhase('authenticated')
      })
      .catch(() => {
        setPhase('shift-open')
      })
  }

  async function handleShiftOpened() {
    setPhase('loading')
    try {
      const s = await getCurrentShift()
      setShift(s)
      setPhase('authenticated')
    } catch {
      setPhase('shift-open')
    }
  }

  function handleShiftClosed() {
    setShift(null)
    setPhase('shift-open')
  }

  function handleLogout() {
    clearAuth()
    setShift(null)
    setPhase('login')
  }

  const user = getStoredUser()

  // Loading screen
  if (phase === 'loading') {
    return (
      <div className="min-h-screen flex items-center justify-center bg-zinc-950">
        <div className="text-zinc-100 text-xl">Cargando...</div>
      </div>
    )
  }

  // Login screen (no layout)
  if (phase === 'login') {
    return <LoginScreen onLogin={handleLogin} />
  }

  // Shift open screen (no layout)
  if (phase === 'shift-open') {
    return <ShiftOpenScreen onShiftOpened={handleShiftOpened} />
  }

  // Authenticated: nested routes inside SessionProvider
  if (phase === 'authenticated' && shift) {
    return (
      <BrowserRouter>
        <SessionProvider
          shift={shift}
          onShiftClosed={handleShiftClosed}
          onLogout={handleLogout}
          userName={user?.username}
        >
          <Routes>
            <Route element={<AdminLayout />}>
              <Route path="/" element={<PosScreen />} />
              <Route path="/inventory" element={<InventoryScreen />} />
              <Route path="/dashboard" element={<DashboardScreen />} />
            </Route>
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </SessionProvider>
      </BrowserRouter>
    )
  }

  // Fallback
  return <LoginScreen onLogin={handleLogin} />
}