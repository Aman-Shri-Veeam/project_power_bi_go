@echo off
REM PowerBI Backup & Restore Server Startup Script
REM 
REM This script builds and runs the Go server with integrated frontend
REM Access UI at: http://localhost:8060

echo.
echo Power BI Backup & Restore Server
echo ----------------------------------------
echo.

REM Check if Go is installed
where go >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Go is not installed or not in PATH
    echo Please install Go from https://go.dev/dl
    pause
    exit /b 1
)

REM Show Go version
echo [INFO] Go version:
go version
echo.

REM Build the server
echo [INFO] Building server...
go build -o powerbi-backup-server.exe ./cmd/server
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Build failed
    pause
    exit /b 1
)
echo [OK] Build successful
echo.

REM Check .env file
if not exist ".env" (
    echo WARNING: .env file not found
    echo Creating from .env.example...
    copy .env.example .env
    echo INFO: Created .env - Please edit with your credentials
    echo.
)

REM Run the server
echo Starting Power BI Backup & Restore Server...
echo Web UI: http://localhost:8060
echo API:    http://localhost:8060/api
echo Press Ctrl+C to stop
echo.
echo ----------------------------------------
echo.

.\powerbi-backup-server.exe

pause
