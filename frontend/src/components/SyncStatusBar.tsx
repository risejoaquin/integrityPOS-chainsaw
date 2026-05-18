import { useState, useEffect } from 'react'
import api from '../api/client'

interface SyncInfo {
  online: boolean
  pendingProducts: number
  pendingSales: number
  lastPullTimestamp: string
}

export default function SyncStatusBar() {
  const [info, setInfo] = useState<SyncInfo>({
    online: navigator.onLine,
    pendingProducts: 0,
    pendingSales: 0,
    lastPullTimestamp: '',
  })
  const [forcing, setForcing] = useState(false)

  useEffect(() => {
    const goOnline = () => { setInfo((p) => ({ ...p, online: true })); fetchSyncInfo() }
    const goOffline = () => { setInfo((p) => ({ ...p, online: false })) }

    window.addEventListener('online', goOnline)
    window.addEventListener('offline', goOffline)
    return () => {
      window.removeEventListener('online', goOnline)
      window.removeEventListener('offline', goOffline)
    }
  }, [])

  useEffect(() => {
    fetchSyncInfo()
    const interval = setInterval(fetchSyncInfo, 30000)
    return () => clearInterval(interval)
  }, [])

  async function fetchSyncInfo() {
    if (!navigator.onLine) return
    try {
      const res = await api.get('/sync/status')
      setInfo((p) => ({
        ...p,
        online: true,
        pendingProducts: res.data.pending_products ?? 0,
        pendingSales: res.data.pending_sales ?? 0,
        lastPullTimestamp: res.data.last_product_pull ?? '',
      }))
    } catch {
      // offline or server down — keep previous state
    }
  }

  async function forceSync() {
    if (!navigator.onLine || forcing) return
    setForcing(true)
    try {
      await api.post('/sync/force')
      // Wait a little for the worker to process
      await new Promise((r) => setTimeout(r, 2000))
      await fetchSyncInfo()
    } catch {
      // ignore
    } finally {
      setForcing(false)
    }
  }

  const isSynced = info.pendingProducts === 0 && info.pendingSales === 0

  return (
    <div className="flex items-center gap-3 text-xs">
      {/* Online/Offline indicator */}
      <div className="flex items-center gap-1.5">
        <span
          className={`w-2 h-2 rounded-full ${
            info.online ? 'bg-green-400 shadow-sm shadow-green-400/50' : 'bg-red-400 shadow-sm shadow-red-400/50'
          }`}
        />
        <span className={info.online ? 'text-green-400' : 'text-red-400'}>
          {info.online ? 'Online' : 'Offline'}
        </span>
      </div>

      {/* Pending counters */}
      {info.online && (
        <>
          <div className="flex items-center gap-1.5 text-zinc-500">
            <span className="text-zinc-600">|</span>
            <span>Pendientes:</span>
            <span className={info.pendingProducts > 0 ? 'text-yellow-400 font-medium' : 'text-zinc-600'}>
              {info.pendingProducts} prod.
            </span>
            <span className={info.pendingSales > 0 ? 'text-yellow-400 font-medium' : 'text-zinc-600'}>
              {info.pendingSales} ventas
            </span>
          </div>

          {/* Force sync button */}
          <button
            onClick={forceSync}
            disabled={forcing}
            className="px-2 py-1 bg-zinc-800 hover:bg-zinc-700 disabled:opacity-50 rounded text-zinc-400"
            title="Forzar sincronización"
          >
            {forcing ? '⏳' : '🔄'}
          </button>
        </>
      )}

      {/* Cloud sync indicator */}
      <div className="flex items-center gap-1.5" title={
        isSynced ? 'Sincronizado con la nube' :
        'Pendiente de sincronización'
      }>
        <svg className={`w-4 h-4 ${
          isSynced ? 'text-green-400' :
          'text-yellow-400 animate-pulse'
        }`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
            d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
          />
        </svg>
        <span className={
          isSynced ? 'text-green-400' :
          'text-yellow-400'
        }>
          {isSynced ? 'Sincronizado' : 'Pendiente...'}
        </span>
      </div>

      {/* Last pull timestamp */}
      {info.lastPullTimestamp && (
        <span className="text-zinc-600 hidden md:inline">
          Última descarga: {new Date(info.lastPullTimestamp).toLocaleString('es-MX')}
        </span>
      )}
    </div>
  )
}