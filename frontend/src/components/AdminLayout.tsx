import { useNavigate, useLocation, Outlet } from 'react-router-dom'
import SyncStatusBar from './SyncStatusBar'
import { useSession } from './SessionContext'

const tabs = [
  { path: '/', label: 'Terminal POS', icon: '💰' },
  { path: '/inventory', label: 'Inventario', icon: '📦' },
  { path: '/dashboard', label: 'Dashboard', icon: '📊' },
]

export default function AdminLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const { onLogout, userName, shift } = useSession()

  return (
    <div className="h-screen flex flex-col bg-zinc-950 text-zinc-100">
      {/* Top Header */}
      <header className="bg-zinc-900 border-b border-zinc-800 px-4 py-2 flex items-center justify-between shrink-0">
        <div className="flex items-center gap-4">
          <span className="font-bold text-lg tracking-tight">IntegrityPOS</span>
          {location.pathname === '/' && (
            <>
              <span className="text-zinc-500 text-sm">Caja #{shift.id}</span>
              <span className="text-zinc-600">|</span>
            </>
          )}
          <span className="text-zinc-500 text-sm">{userName || 'Admin'}</span>
        </div>
        <div className="flex items-center gap-4">
          <SyncStatusBar />
          <button
            onClick={onLogout}
            className="text-sm text-zinc-500 hover:text-zinc-300 transition-colors duration-150
              focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 rounded px-2 py-1"
          >
            Salir
          </button>
        </div>
      </header>

      {/* Navigation Tabs */}
      <nav className="bg-zinc-950 border-b border-zinc-800 flex shrink-0 gap-1 px-2">
        {tabs.map((tab) => (
          <button
            key={tab.path}
            onClick={() => navigate(tab.path)}
            className={`flex items-center gap-2 px-5 py-3 text-sm font-medium border-b-2 transition-all duration-150
              focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 focus-visible:ring-inset
              ${location.pathname === tab.path
                ? 'border-emerald-500 text-emerald-400'
                : 'border-transparent text-zinc-500 hover:text-zinc-300 hover:border-zinc-600'
              }`}
          >
            <span>{tab.icon}</span>
            <span>{tab.label}</span>
          </button>
        ))}
      </nav>

      {/* Main Content — dynamic via Outlet */}
      <main className="flex-1 overflow-hidden">
        <Outlet />
      </main>
    </div>
  )
}