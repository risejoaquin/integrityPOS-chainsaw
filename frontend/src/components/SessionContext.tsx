import { createContext, useContext, type ReactNode } from 'react'
import type { Shift } from '../types'

interface SessionContextValue {
  shift: Shift
  onShiftClosed: () => void
  onLogout: () => void
  userName?: string
}

const SessionContext = createContext<SessionContextValue | null>(null)

export function SessionProvider({
  shift,
  onShiftClosed,
  onLogout,
  userName,
  children,
}: SessionContextValue & { children: ReactNode }) {
  return (
    <SessionContext.Provider value={{ shift, onShiftClosed, onLogout, userName }}>
      {children}
    </SessionContext.Provider>
  )
}

export function useSession(): SessionContextValue {
  const ctx = useContext(SessionContext)
  if (!ctx) throw new Error('useSession must be used within SessionProvider')
  return ctx
}