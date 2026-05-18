#!/bin/bash
# Build verification script for Phase 2

echo "========================================"
echo "IntegrityPOS Phase 2 - Compilation Check"
echo "========================================"
echo

cd backend || { echo "Error: backend directory not found"; exit 1; }

echo "[1/3] Running go mod tidy..."
go mod tidy
if [ $? -ne 0 ]; then
    echo "✗ go mod tidy failed"
    exit 1
fi
echo "✓ Dependencies resolved"

echo
echo "[2/3] Running go vet..."
go vet ./...
if [ $? -ne 0 ]; then
    echo "✗ go vet failed"
    exit 1
fi
echo "✓ No vet errors"

echo
echo "[3/3] Building..."
go build -v ./cmd/api
if [ $? -ne 0 ]; then
    echo "✗ Build failed"
    exit 1
fi
echo "✓ Build successful"

echo
echo "========================================"
echo "✓ ALL CHECKS PASSED - Phase 2 Ready"
echo "========================================"
