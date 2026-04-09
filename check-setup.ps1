# Script para verificar y preparar IntegrityPOS
# Uso: .\check-setup.ps1

param(
    [switch]$Install
)

$ErrorActionPreference = "Stop"

Write-Host "=== IntegrityPOS Setup Checker ===" -ForegroundColor Cyan
Write-Host ""

# Función para verificar si un comando existe
function Test-Command {
    param($Command)
    try {
        Get-Command $Command -ErrorAction Stop | Out-Null
        return $true
    } catch {
        return $false
    }
}

# Verificar Go
Write-Host "1. Verificando Go..." -ForegroundColor Yellow
if (Test-Command "go") {
    $goVersion = & go version
    Write-Host "   ✓ Go encontrado: $goVersion" -ForegroundColor Green
} else {
    Write-Host "   ✗ Go NO encontrado" -ForegroundColor Red
    Write-Host ""
    Write-Host "   Para instalar Go:" -ForegroundColor Yellow
    Write-Host "   1. Ve a: https://golang.org/dl/" -ForegroundColor White
    Write-Host "   2. Descarga go1.21.x.windows-amd64.msi" -ForegroundColor White
    Write-Host "   3. Ejecuta el instalador" -ForegroundColor White
    Write-Host "   4. Marca 'Add to PATH'" -ForegroundColor White
    Write-Host "   5. Reinicia la terminal" -ForegroundColor White
    Write-Host ""
    if (-not $Install) {
        exit 1
    }
}

# Verificar GCC
Write-Host "2. Verificando GCC..." -ForegroundColor Yellow
if (Test-Command "gcc") {
    $gccVersion = & gcc --version | Select-Object -First 1
    Write-Host "   ✓ GCC encontrado: $gccVersion" -ForegroundColor Green
} else {
    Write-Host "   ✗ GCC NO encontrado (requerido para SQLite)" -ForegroundColor Red
    Write-Host ""
    Write-Host "   Para instalar GCC:" -ForegroundColor Yellow
    Write-Host "   Opción 1 - Chocolatey:" -ForegroundColor White
    Write-Host "   choco install mingw" -ForegroundColor White
    Write-Host ""
    Write-Host "   Opción 2 - Manual:" -ForegroundColor White
    Write-Host "   1. Ve a: https://www.mingw-w64.org/downloads/" -ForegroundColor White
    Write-Host "   2. Descarga e instala MinGW-w64" -ForegroundColor White
    Write-Host "   3. Agrega 'C:\mingw64\bin\' al PATH" -ForegroundColor White
    Write-Host ""
    if (-not $Install) {
        exit 1
    }
}

# Verificar Node.js (opcional para frontend)
Write-Host "3. Verificando Node.js (opcional)..." -ForegroundColor Yellow
if (Test-Command "node") {
    $nodeVersion = & node --version
    Write-Host "   ✓ Node.js encontrado: $nodeVersion" -ForegroundColor Green
} else {
    Write-Host "   ⚠ Node.js NO encontrado (opcional para frontend)" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "=== Configuración del Proyecto ===" -ForegroundColor Cyan

# Verificar/Crear archivo .env
Write-Host "4. Verificando configuración..." -ForegroundColor Yellow
if (Test-Path ".env") {
    Write-Host "   ✓ Archivo .env encontrado" -ForegroundColor Green
} else {
    if (Test-Path ".env.example") {
        Copy-Item ".env.example" ".env"
        Write-Host "   ✓ Archivo .env creado desde .env.example" -ForegroundColor Green
        Write-Host "   ⚠ RECUERDA editar .env con tus configuraciones!" -ForegroundColor Yellow
    } else {
        Write-Host "   ✗ Archivo .env.example no encontrado" -ForegroundColor Red
    }
}

# Instalar dependencias si Go está disponible
if (Test-Command "go") {
    Write-Host ""
    Write-Host "5. Instalando dependencias de Go..." -ForegroundColor Yellow
    try {
        & go mod tidy
        Write-Host "   ✓ Dependencias instaladas" -ForegroundColor Green
    } catch {
        Write-Host "   ✗ Error instalando dependencias: $($_.Exception.Message)" -ForegroundColor Red
    }

    Write-Host ""
    Write-Host "6. Verificando compilación..." -ForegroundColor Yellow
    try {
        & go build -o integritypos.exe ./cmd/server
        Write-Host "   ✓ Compilación exitosa" -ForegroundColor Green
        Write-Host ""
        Write-Host "🎉 ¡Proyecto listo!" -ForegroundColor Green
        Write-Host ""
        Write-Host "Para ejecutar:" -ForegroundColor Cyan
        Write-Host "  .\integritypos.exe" -ForegroundColor White
        Write-Host ""
        Write-Host "Para desarrollo:" -ForegroundColor Cyan
        Write-Host "  go run ./cmd/server" -ForegroundColor White
    } catch {
        Write-Host "   ✗ Error de compilación: $($_.Exception.Message)" -ForegroundColor Red
        Write-Host "   Revisa las dependencias y configuración" -ForegroundColor Yellow
    }
}

Write-Host ""
Write-Host "=== Próximos Pasos ===" -ForegroundColor Cyan
Write-Host "1. Si no tienes Go: Instálalo desde https://golang.org/dl/" -ForegroundColor White
Write-Host "2. Si no tienes GCC: Instala MinGW desde https://www.mingw-w64.org/" -ForegroundColor White
Write-Host "3. Edita el archivo .env con tus configuraciones" -ForegroundColor White
Write-Host "4. Ejecuta: .\check-setup.ps1 (este script)" -ForegroundColor White
Write-Host "5. Una vez listo: go run ./cmd/server" -ForegroundColor White
Write-Host ""
Write-Host "📚 Documentación:" -ForegroundColor Cyan
Write-Host "  INSTALACION_GO.md - Guía detallada de instalación" -ForegroundColor White
Write-Host "  INSTRUCCIONES_EJECUCION.md - Cómo ejecutar el proyecto" -ForegroundColor White
Write-Host "  README.md - Documentación completa" -ForegroundColor White