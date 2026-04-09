# Script de PowerShell para ejecutar IntegrityPOS
# Uso: .\run.ps1

param(
    [switch]$Build,
    [switch]$Run,
    [switch]$Clean
)

$ErrorActionPreference = "Stop"

# Cambiar al directorio del script
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $scriptDir

Write-Host "=== IntegrityPOS Build Script ===" -ForegroundColor Cyan

if ($Clean) {
    Write-Host "Limpiando archivos generados..." -ForegroundColor Yellow
    if (Test-Path "integritypos.exe") {
        Remove-Item "integritypos.exe"
    }
    if (Test-Path "integritypos.db") {
        Remove-Item "integritypos.db"
    }
    Write-Host "Limpieza completada." -ForegroundColor Green
    exit 0
}

if ($Build) {
    Write-Host "Descargando dependencias..." -ForegroundColor Yellow
    & go mod tidy

    Write-Host "Compilando proyecto..." -ForegroundColor Yellow
    & go build -o integritypos.exe ./cmd/server

    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ Compilación exitosa: integritypos.exe" -ForegroundColor Green
        Write-Host "Para ejecutar: .\integritypos.exe" -ForegroundColor Cyan
    } else {
        Write-Host "✗ Error en compilación" -ForegroundColor Red
        exit 1
    }
}

if ($Run) {
    Write-Host "Descargando dependencias..." -ForegroundColor Yellow
    & go mod tidy

    Write-Host "Ejecutando servidor..." -ForegroundColor Yellow
    & go run ./cmd/server
}

if (-not $Build -and -not $Run -and -not $Clean) {
    Write-Host "Uso del script:" -ForegroundColor Cyan
    Write-Host "  .\run.ps1 -Build    # Compilar ejecutable" -ForegroundColor White
    Write-Host "  .\run.ps1 -Run      # Ejecutar directamente" -ForegroundColor White
    Write-Host "  .\run.ps1 -Clean    # Limpiar archivos generados" -ForegroundColor White
    Write-Host "" -ForegroundColor White
    Write-Host "Ejemplos:" -ForegroundColor Cyan
    Write-Host "  .\run.ps1 -Build -Run  # Compilar y ejecutar" -ForegroundColor White
}