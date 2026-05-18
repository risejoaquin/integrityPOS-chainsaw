# Guía de Instalación y Compilación — IntegrityPOS

## Requisitos Previos

| Herramienta | Versión Mínima | Propósito |
|-------------|---------------|-----------|
| **Go** | 1.21+ | Compilación del backend |
| **Node.js** | 18 LTS | Compilación del frontend (React + Vite) |
| **npm** | 9+ | Gestión de paquetes del frontend |
| **GCC/MinGW** (CGO) | Cualquiera | SQLite requiere CGO_ENABLED |
| **Git** | Cualquiera | Control de versiones |

### Instalación de GCC/MinGW (Windows)

SQLite usa `mattn/go-sqlite3` que requiere CGO. En Windows instala **TDM-GCC** o **MinGW-w64**:

1. Descarga desde: https://jmeubank.github.io/tdm-gcc/
2. Ejecuta el instalador, selecciona "Create" → "MinGW-w64" → arquitectura `x86_64`
3. Asegúrate de que `gcc` esté disponible en tu PATH:
   ```
   gcc --version
   ```

### Linux
```bash
sudo apt install gcc
# o en Fedora/RHEL:
sudo dnf install gcc
```

### macOS
```bash
xcode-select --install
```

---

## 1. Clonar el Repositorio

```bash
git clone <URL_DEL_REPOSITORIO> integritypos
cd integritypos
```

---

## 2. Configuración del Entorno

Crea el archivo `.env` en la raíz del proyecto basado en `.env.example`:

```bash
cp .env.example .env
```

### Variables de Entorno

| Variable | Descripción | Obligatoria |
|----------|-------------|-------------|
| `SUPABASE_URL` | URL de tu proyecto Supabase (ej. `https://xyz.supabase.co`) | Sí (si usas sincronización) |
| `SUPABASE_KEY` | API Key anon/public de Supabase | Sí (si usas sincronización) |
| `JWT_SECRET` | Secreto para firmar tokens JWT | No (default: `dev_integrity_secret_2026`) |
| `PORT` | Puerto del backend (default: `8080`) | No |
| `DB_PATH` | Ruta del archivo SQLite (default: `integritypos.db`) | No |
| `PRINTER_DEVICE` | Ruta del dispositivo de impresora (Linux) | No |
| `PRINTER_NAME` | Nombre de la impresora (Windows, default: `EPSON`) | No |
| `MOCK_PRINTER` | `"true"` para simular impresora sin hardware | No |
| `BUSINESS_NAME` | Nombre del negocio en tickets | No |

Ejemplo mínimo `.env`:
```
SUPABASE_URL=https://tu-proyecto.supabase.co
SUPABASE_KEY=eyJhbGciOiJIUzI1NiIs...
JWT_SECRET=mi_secreto_personal_2026
PORT=8080
MOCK_PRINTER=true
BUSINESS_NAME=Mi Ferretería
```

---

## 3. Compilación del Backend (Go)

```bash
cd backend

# Descargar dependencias
go mod tidy

# Verificar compilación (sin errores)
go vet ./...

# Compilar binario
go build -o integritypos-api.exe ./cmd/api

# (Opcional) Compilar para Linux desde Windows:
# GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o integritypos-api ./cmd/api
```

### Ejecutar el servidor en modo desarrollo

```bash
# Desde la raíz del proyecto (necesita el .env)
go run ./backend/cmd/api

# O desde la carpeta backend
cd backend && go run ../backend/cmd/api
```

El servidor iniciará en `http://localhost:8080`. Verás los logs:
```
IntegrityPOS Backend - Initializing...
✓ Database initialized
✓ Default users seeded
✓ Repositories initialized
...
✓ Router configured
Starting IntegrityPOS Backend on port 8080...
```

---

## 4. Compilación del Frontend (React + Vite)

```bash
cd frontend

# Instalar dependencias
npm install

# Compilar para producción (genera carpeta dist/)
npm run build:vite
```

### Ejecutar frontend en modo desarrollo (Vite)

```bash
npm run dev:vite
```

Esto levanta el servidor de desarrollo en `http://localhost:5173`.

### Compilar y empaquetar (Electron)

**Importante:** Primero compila el backend y el frontend:

```bash
cd frontend
npm run build:vite          # Compila React
npm run build:backend       # Compila el backend Go
```

Luego genera el instalador:

```bash
# Build completo (genera .exe installer)
npm run build:all

# O build portable (carpeta autocontenida)
npm run build:portable
```

Los instaladores se generarán en `frontend/release/`.

---

## 5. Base de Datos Local (SQLite)

El sistema **crea la base de datos automáticamente** al iniciar el backend por primera vez.

- Archivo: `integritypos.db` (en la raíz del proyecto o donde apunte `DB_PATH`)
- Los usuarios semilla se insertan automáticamente si la tabla `users` está vacía:

| Usuario | Contraseña | Rol |
|---------|-----------|------|
| **Administrador Principal** | `admin123` | admin |
| **Cajero Turno 1** | `cajero123` | cashier |

Para reiniciar desde cero, solo borra el archivo `integritypos.db` y reinicia el backend.

---

## 6. Sincronización con Supabase

Para habilitar la sincronización en la nube (ventas, turnos, productos, clientes, egresos):

1. Crea un proyecto en [Supabase](https://supabase.com)
2. Ejecuta el schema de `DATABASE_SCHEMA.sql` en el SQL Editor de Supabase
3. Configura las variables `SUPABASE_URL` y `SUPABASE_KEY` en tu `.env`

Sin estas variables, el sistema funciona **completamente offline** sin errores.

---

## 7. Solución de Problemas Comunes

### Error: `CGO_ENABLED=0` no compila

**Problema:** El paquete `mattn/go-sqlite3` requiere CGO.
**Solución:**
```bash
set CGO_ENABLED=1    # Windows (CMD)
$env:CGO_ENABLED=1   # Windows (PowerShell)
export CGO_ENABLED=1 # Linux/macOS

# Verifica que GCC esté instalado:
gcc --version
```

### Error: Puerto en uso

**Problema:** El puerto 8080 ya está ocupado.
**Solución:** Cambia el puerto en `.env`:
```
PORT=3001
```
O mata el proceso que ocupa el puerto:
```bash
# Windows
netstat -ano | findstr :8080
taskkill /PID <PID> /F

# Linux/macOS
lsof -i :8080
kill -9 <PID>
```

### Error: `go mod tidy` falla por conexión

**Problema:** No se pueden descargar dependencias de Go.
**Solución:**
```bash
go env -w GOPROXY=https://proxy.golang.org,direct
# O usar proxy chino si estás en China:
go env -w GOPROXY=https://goproxy.cn,direct
```

### Error: `npm install` falla

**Solución:**
```bash
# Limpiar caché y reintentar
npm cache clean --force
rm -rf node_modules package-lock.json
npm install
```

### Error: `CGO_ENABLED` no está definido en Windows

Si obtienes `cc1.exe: sorry, unimplemented: 64-bit mode not compiled`, asegúrate de instalar **TDM-GCC x86_64** (no la versión de 32 bits).

### Error: La base de datos no se crea

El backend crea `integritypos.db` en el directorio actual de trabajo. Si ejecutas el backend desde una ubicación diferente, establece `DB_PATH` como ruta absoluta:
```
DB_PATH=C:\Users\TuUsuario\integritypos\integritypos.db
```

---

## 8. Flujo Rápido de Inicio

```bash
# 1. Clonar
git clone <URL> integritypos
cd integritypos

# 2. Configurar
cp .env.example .env
# Editar .env con SUPABASE_URL y SUPABASE_KEY si se requiere sincronización

# 3. Backend (terminal 1)
cd backend
go mod tidy
go run ./cmd/api

# 4. Frontend (terminal 2)
cd frontend
npm install
npm run dev:vite

# 5. Abrir http://localhost:5173 en el navegador
# Iniciar sesión con admin123
```

### Compilación completa (standalone + portable)

```bash
cd frontend
npm install
npm run build:vite
npm run build:backend
npm run build:portable
# El .exe portable estará en: frontend/release/IntegrityPOS-*-portable.exe
```

---

## Resumen

Siguiendo esta guía podrás:

1. **Compilar el backend** (API REST en Go) — listo para servir peticiones
2. **Compilar el frontend** (React + Vite) — interfaz de usuario
3. **Empaquetar la aplicación desktop** (Electron) — instalador de Windows
4. **Iniciar sesión** con las credenciales precargadas (`admin123` / `cajero123`)
5. **Usar el POS** completo: ventas, turnos, inventario, clientes, egresos, reportes
6. **Sincronizar con Supabase** si configuraste las credenciales

Todo el sistema funciona **offline-first** — la base de datos SQLite local es la fuente primaria y la sincronización con la nube es opcional.