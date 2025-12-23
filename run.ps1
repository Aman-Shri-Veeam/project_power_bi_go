#!/usr/bin/env pwsh

# PowerBI Backup Server Startup Script
# Builds and runs the complete Go server with web UI

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Power BI Backup & Restore Server" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check if Go is installed
$goPath = Get-Command go -ErrorAction SilentlyContinue
if (-not $goPath) {
    Write-Host "[ERROR] Go is not installed or not in PATH" -ForegroundColor Red
    Write-Host "Download from: https://go.dev/dl" -ForegroundColor Yellow
    Read-Host "Press Enter to exit"
    exit 1
}

# Show Go version
Write-Host "[INFO] Go version:" -ForegroundColor Blue
go version
Write-Host ""

# Build the server
Write-Host "[INFO] Building server..." -ForegroundColor Blue
go build -o powerbi-backup-server.exe ./cmd/server
if ($LASTEXITCODE -ne 0) {
    Write-Host "[ERROR] Build failed" -ForegroundColor Red
    Read-Host "Press Enter to exit"
    exit 1
}
Write-Host "[OK] Build successful" -ForegroundColor Green
Write-Host ""

# Check .env file
if (-not (Test-Path ".env")) {
    Write-Host "[WARNING] .env file not found" -ForegroundColor Yellow
    Write-Host "Creating from .env.example..." -ForegroundColor Yellow
    Copy-Item ".env.example" ".env"
    Write-Host "[INFO] Created .env - Please edit with your credentials" -ForegroundColor Blue
    Write-Host ""
}

# Run the server
Write-Host "[INFO] Starting server..." -ForegroundColor Blue
Write-Host "[INFO] Web UI:  http://localhost:8060" -ForegroundColor Green
Write-Host "[INFO] API:     http://localhost:8060/api" -ForegroundColor Green
Write-Host "[INFO] Press Ctrl+C to stop" -ForegroundColor Yellow
Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

.\powerbi-backup-server.exe

Read-Host "Press Enter to exit"
