const { app, BrowserWindow, dialog } = require('electron')
const { spawn } = require('child_process')
const path = require('path')
const fs = require('fs')

// Keep a global reference to prevent garbage collection
let mainWindow = null
let backendProcess = null

// ─── Determine Paths ─────────────────────────────────────

function getBackendPath() {
  const isPackaged = app.isPackaged
  const binaryName = process.platform === 'win32' ? 'integritypos-api.exe' : 'integritypos-api'

  if (isPackaged) {
    // In production, the binary is in extraResources
    const resourcesPath = process.resourcesPath
    return path.join(resourcesPath, 'backend', binaryName)
  }

  // In development, use the compiled binary from backend/cmd/api
  return path.join(__dirname, '..', 'backend', 'cmd', 'api', binaryName)
}

function getDbPath() {
  const userDataPath = app.getPath('userData')
  return path.join(userDataPath, 'integritypos.db')
}

// ─── Backend Sidecar ─────────────────────────────────────

function startBackend() {
  const backendPath = getBackendPath()
  const dbPath = getDbPath()

  console.log(`[Electron] Starting backend: ${backendPath}`)
  console.log(`[Electron] Database path: ${dbPath}`)

  // Verify backend binary exists
  if (!fs.existsSync(backendPath)) {
    console.error(`[Electron] Backend binary not found at: ${backendPath}`)
    dialog.showErrorBox(
      'Error de Inicialización',
      `No se encontró el backend en:\n${backendPath}\n\nPor favor, ejecute la compilación completa primero.`
    )
    app.quit()
    return
  }

  const env = {
    ...process.env,
    DB_PATH: dbPath,
    PORT: '8080',
    JWT_SECRET: process.env.JWT_SECRET || 'integrity_electron_prod_2026',
  }

  backendProcess = spawn(backendPath, [], {
    env,
    stdio: ['ignore', 'pipe', 'pipe'],
    windowsHide: true,
  })

  backendProcess.stdout.on('data', (data) => {
    console.log(`[Backend] ${data.toString().trim()}`)
  })

  backendProcess.stderr.on('data', (data) => {
    console.error(`[Backend ERR] ${data.toString().trim()}`)
  })

  backendProcess.on('error', (err) => {
    console.error(`[Electron] Failed to start backend:`, err.message)
    dialog.showErrorBox('Error del Backend', `No se pudo iniciar el servidor backend:\n${err.message}`)
  })

  backendProcess.on('exit', (code, signal) => {
    console.log(`[Backend] exited with code ${code}, signal ${signal}`)
    backendProcess = null
  })
}

function stopBackend() {
  if (backendProcess) {
    console.log('[Electron] Stopping backend process...')
    if (process.platform === 'win32') {
      // On Windows, spawn taskkill to ensure child process tree is killed
      spawn('taskkill', ['/pid', String(backendProcess.pid), '/f', '/t'])
    } else {
      backendProcess.kill('SIGTERM')
    }
    backendProcess = null
  }
}

// ─── Create Window ───────────────────────────────────────

function createWindow() {
  const isPackaged = app.isPackaged

  mainWindow = new BrowserWindow({
    width: 1280,
    height: 800,
    minWidth: 1024,
    minHeight: 700,
    fullscreen: true,
    autoHideMenuBar: true,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      nodeIntegration: false,
      contextIsolation: true,
    },
    icon: path.join(__dirname, 'public', 'icon.png'),
    backgroundColor: '#111827', // Tailwind gray-900
  })

  // Remove menu bar for immersive POS look
  mainWindow.setMenuBarVisibility(false)

  if (isPackaged) {
    // Production: load the built Vite output
    const indexPath = path.join(__dirname, 'dist', 'index.html')
    mainWindow.loadFile(indexPath)
  } else {
    // Development: load Vite dev server
    mainWindow.loadURL('http://localhost:5173')
    // Open DevTools in development
    mainWindow.webContents.openDevTools()
  }

  mainWindow.on('closed', () => {
    mainWindow = null
  })
}

// ─── Health Check ────────────────────────────────────────

function waitForBackend(maxRetries = 20, interval = 500) {
  return new Promise((resolve, reject) => {
    let retries = 0
    const check = () => {
      const http = require('http')
      const req = http.get('http://localhost:8080/health', (res) => {
        if (res.statusCode === 200) {
          console.log('[Electron] Backend is ready')
          resolve()
        } else {
          retry()
        }
      })
      req.on('error', () => retry())
      req.end()
    }
    const retry = () => {
      retries++
      if (retries >= maxRetries) {
        reject(new Error(`Backend did not start after ${maxRetries * interval}ms`))
        return
      }
      setTimeout(check, interval)
    }
    check()
  })
}

// ─── App Lifecycle ───────────────────────────────────────

app.whenReady().then(async () => {
  startBackend()

  try {
    await waitForBackend()
    createWindow()
  } catch (err) {
    console.error('[Electron]', err.message)
    dialog.showErrorBox(
      'Error del Backend',
      `El servidor backend no respondió después de varios intentos.\n\n${err.message}`
    )
    app.quit()
  }

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow()
    }
  })
})

app.on('window-all-closed', () => {
  stopBackend()
  if (process.platform !== 'darwin') {
    app.quit()
  }
})

app.on('before-quit', () => {
  stopBackend()
})

// Handle cleanup on unexpected exit
process.on('exit', () => {
  stopBackend()
})

process.on('SIGINT', () => {
  stopBackend()
  process.exit(0)
})

process.on('SIGTERM', () => {
  stopBackend()
  process.exit(0)
})