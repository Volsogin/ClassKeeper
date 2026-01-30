@echo off
cls

echo ========================================
echo   ClassKeeper 2.0 - Starting Server
echo ========================================
echo.

cd backend\cmd\server

echo Checking Go installation...
where go >nul 2>&1
if %errorlevel% neq 0 (
    echo ERROR: Go is not installed!
    echo.
    echo Please install Go from: https://go.dev/dl/
    echo.
    pause
    exit /b 1
)

echo Go found! Version:
go version
echo.

echo Initializing Go modules...
cd ..\..

echo Cleaning old dependencies...
if exist go.sum del go.sum

echo Downloading dependencies (this may take a minute)...
go mod download
if %errorlevel% neq 0 (
    echo ERROR: Failed to download dependencies!
    echo.
    echo Try running: go mod tidy
    pause
    exit /b 1
)

echo Running go mod tidy...
go mod tidy
if %errorlevel% neq 0 (
    echo Warning: go mod tidy failed, but continuing...
)

echo.
echo Starting server...
cd cmd\server
echo.
echo Server will be available at: http://localhost:8080
echo.
echo Press Ctrl+C to stop
echo ========================================
echo.

go run main.go

pause
