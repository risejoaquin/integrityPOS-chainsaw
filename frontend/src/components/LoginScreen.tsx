import { useState } from 'react'
import { login, storeAuth } from '../api/endpoints'

interface Props {
  onLogin: () => void
}

export default function LoginScreen({ onLogin }: Props) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    if (!username || !password) {
      setError('Ingrese usuario y PIN')
      return
    }
    setLoading(true)
    try {
      const data = await login(username, password)
      storeAuth(data)
      onLogin()
    } catch (err: unknown) {
      if (err && typeof err === 'object' && 'response' in err) {
        const axiosErr = err as { response?: { data?: { message?: string } } }
        setError(axiosErr.response?.data?.message || 'Credenciales inválidas')
      } else {
        setError('Error de conexión')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-zinc-950">
      <div className="bg-zinc-900 rounded-2xl shadow-2xl p-8 w-full max-w-sm border border-zinc-800">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold text-zinc-100 tracking-tight">IntegrityPOS</h1>
          <p className="text-zinc-500 mt-2">Sistema de Punto de Venta</p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-5">
          <div>
            <label className="block text-sm font-medium text-zinc-400 mb-1">Usuario</label>
            <input
              type="text"
              autoFocus
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="w-full px-4 py-3 bg-zinc-800 border border-zinc-700 rounded-lg text-zinc-100
                text-lg placeholder-zinc-600
                focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 focus-visible:border-emerald-500
                transition-colors duration-150"
              placeholder="usuario"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-zinc-400 mb-1">PIN</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full px-4 py-3 bg-zinc-800 border border-zinc-700 rounded-lg text-zinc-100
                text-lg placeholder-zinc-600
                focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 focus-visible:border-emerald-500
                transition-colors duration-150"
              placeholder="••••"
              inputMode="numeric"
            />
          </div>

          {error && (
            <div className="bg-red-900/30 border border-red-800 text-red-400 px-4 py-2 rounded-lg text-sm">
              {error}
            </div>
          )}

          <button
            type="submit"
            disabled={loading}
            className="w-full py-3 bg-emerald-600 hover:bg-emerald-500 disabled:bg-zinc-800 disabled:text-zinc-600
              disabled:cursor-not-allowed text-white font-bold text-lg rounded-lg
              transition-all duration-150 active:scale-[0.98]
              focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400 focus-visible:ring-offset-2 focus-visible:ring-offset-zinc-950"
          >
            {loading ? 'Ingresando...' : 'Ingresar'}
          </button>
        </form>
      </div>
    </div>
  )
}