@echo off
setlocal

echo ============================================
echo   Dotachi LAN Gaming - Installer Builder
echo ============================================
echo.

:: Step 1: Build the Wails application
echo [1/3] Building Dotachi client with Wails...
cd /d "%~dp0.."
wails build
if %ERRORLEVEL% neq 0 (
    echo ERROR: Wails build failed!
    exit /b 1
)

:: Step 2: Copy the built binary to the installer directory
echo [2/3] Copying built binary...
copy /Y "build\bin\dotachi.exe" "installer\dotachi.exe"
if %ERRORLEVEL% neq 0 (
    echo ERROR: Could not copy dotachi.exe!
    echo Make sure the Wails build succeeded.
    exit /b 1
)

:: Step 3: Build the NSIS installer
echo [3/3] Building NSIS installer...
cd /d "%~dp0"
makensis dotachi.nsi
if %ERRORLEVEL% neq 0 (
    echo ERROR: NSIS build failed!
    echo Make sure NSIS is installed and makensis is in your PATH.
    exit /b 1
)

echo.
echo ============================================
echo   Build complete!
echo   Installer: %~dp0DotachiSetup.exe
echo ============================================
endlocal
