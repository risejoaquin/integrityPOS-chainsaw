# IntegrityPOS

Sistema POS completo con arquitectura hexagonal, SQLite, autenticación JWT, backups encriptados y interfaz Electron.

##  Requisitos Previos

### Windows
- **Go 1.21+** - [Descargar e instalar](https://golang.org/dl/)
- **GCC (MinGW-w64)** - Para SQLite: `choco install mingw` o [instalación manual](https://www.mingw-w64.org/)
- **Node.js 16+** - Opcional, para frontend: [Descargar](https://nodejs.org/)

### Verificar Instalación
```bash
go version    # Debería mostrar go1.21.x
gcc --version # Debería mostrar GCC
```

###  Problema Común: "go: command not found"
Si ves este error, **Go no está instalado**. Sigue estos pasos:

1. Ve a https://golang.org/dl/
2. Descarga `go1.21.x.windows-amd64.msi`
3. Ejecuta el instalador
4. **IMPORTANTE**: Marca "Add to PATH"
5. Reinicia la terminal
6. Verifica con `go version`

##  Inicio Rápido (Windows)

### 1. Instalar Dependencias
```bash
go mod tidy
```

### 2. Configurar Variables de Entorno
Copia el archivo de ejemplo y configura tus valores:
```bash
copy .env.example .env
# Edita .env con tus configuraciones
```

### 3. Ejecutar el Servidor
```bash
# Opción A: Ejecutar directamente
go run ./cmd/server

# Opción B: Usar script batch
run.bat
```

### 4. Construir Ejecutable
```bash
# Opción A: Build directo
go build -o integritypos.exe ./cmd/server

# Opción B: Usar script batch
build.bat
```

##  Interfaz de Usuario

### Ejecutar Frontend Electron
```bash
cd frontend
npm install
npm start
```

##  Requisitos

- **Go 1.21+** - Lenguaje principal
- **GCC** - Para CGO (SQLite)
- **Node.js 16+** - Para frontend (opcional)
- **Git** - Para dependencias

##  Configuración

### Variables de Entorno Requeridas
- `HMAC_SECRET` - Clave para firmas HMAC
- `JWT_SECRET` - Clave para tokens JWT

### Variables Opcionales
- `DATABASE_PATH` - Ruta de base de datos (default: ./integritypos.db)
- `BACKUP_DIR` - Directorio de backups (default: ./backups)
- `PRINTER_MODE` - Modo de impresora (default: stdout)

##  Arquitectura

- **Hexagonal Architecture** - Separación clara de capas
- **SQLite + WAL** - Base de datos con optimización
- **JWT Authentication** - Seguridad de API
- **Rate Limiting** - Protección anti-abuso
- **Encrypted Backups** - Seguridad de datos
- **Database Sharding** - Preparado para escalabilidad

##  Testing

```bash
# Ejecutar todos los tests
go test ./...

# Tests con coverage
go test -cover ./...
```

##  Estructura del Proyecto

```
integritypos/
├── cmd/server/           # Punto de entrada
├── internal/
│   ├── application/      # Casos de uso
│   ├── domain/          # Entidades y lógica de negocio
│   ├── infrastructure/  # Adaptadores externos
│   └── infrastructure/
│       ├── persistence/ # Base de datos
│       ├── web/        # HTTP handlers
│       └── backup/     # Sistema de backups
├── frontend/            # Interfaz Electron + React
├── migrations/          # Scripts de base de datos
└── docs/               # Documentación
```

##  Desarrollo

### Agregar Nuevas Dependencias
```bash
go get github.com/nueva/dependencia
go mod tidy
```

### Ejecutar con Hot Reload
```bash
# Usar air o similar para hot reload
go install github.com/cosmtrek/air@latest
air
```

## � Solución de Problemas

### "go: command not found"
- Go no está instalado o no está en PATH
- Sigue la guía en `INSTALACION_GO.md`
- Reinicia la terminal después de instalar

### "gcc: command not found"
- GCC es requerido para SQLite (CGO)
- Instala MinGW: `choco install mingw`
- O descarga manualmente de https://www.mingw-w64.org/

### Error de compilación
```bash
# Limpiar módulos
go clean -modcache
go mod tidy
go mod download
```

### Puerto 8080 ocupado
```bash
# Cambiar puerto (modificar main.go)
go run ./cmd/server
```

### Base de datos corrupta
```bash
# Borrar y recrear
del integritypos.db
go run ./cmd/server
```

##  Documentación Adicional

- `INSTALACION_GO.md` - Instalación de Go paso a paso
- `INSTRUCCIONES_EJECUCION.md` - Guía completa de ejecución
- `MANUAL_CONFIGURATION.md` - Configuración avanzada
- `ADVANCED_PHASE10.md` - Características avanzadas
- `FRONTEND_PHASE9.md` - Documentación del frontend
- `FISCAL_PHASE8.md` - Integración fiscal

## Contribución

1. Fork el proyecto
2. Crea una rama para tu feature (`git checkout -b feature/AmazingFeature`)
3. Commit tus cambios (`git commit -m 'Add some AmazingFeature'`)
4. Push a la rama (`git push origin feature/AmazingFeature`)
5. Abre un Pull Request

##  Licencia

Este proyecto está bajo la Licencia MIT - ver el archivo [LICENSE](LICENSE) para más detalles.
