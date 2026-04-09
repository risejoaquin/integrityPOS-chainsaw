@echo off
cd /d "C:\Users\Joaquin\OneDrive\Documentos\intigrityPOS"
echo Descargando dependencias...
go mod tidy
if %errorlevel% neq 0 (
    echo Error en go mod tidy
    pause
    exit /b 1
)

echo Compilando proyecto...
go build -o integritypos.exe ./cmd/server
if %errorlevel% neq 0 (
    echo Error en compilacion
    pause
    exit /b 1
)

echo Ejecutable creado: integritypos.exe
echo Para ejecutar: integritypos.exe
pause