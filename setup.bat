@echo off
echo ====================================
echo  ClassKeeper 2.0 - Manual Setup
echo ====================================
echo.
echo This script will set up dependencies manually.
echo.

cd backend

echo Step 1: Cleaning old files...
if exist go.sum del go.sum
echo Done!
echo.

echo Step 2: Downloading dependencies...
echo This may take 1-2 minutes on first run...
echo.
go mod download
if %errorlevel% neq 0 (
    echo ERROR during download!
    pause
    exit /b 1
)
echo.

echo Step 3: Tidying modules...
go mod tidy
if %errorlevel% neq 0 (
    echo ERROR during tidy!
    pause
    exit /b 1
)
echo.

echo ====================================
echo SUCCESS! Dependencies are ready.
echo ====================================
echo.
echo Now you can run: start.bat
echo.
pause
