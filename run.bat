@echo off
REM Run script para p2pollo - Inicia backend Go API + frontend Qt/QML
REM ====================================================================

echo.
echo ============================================================
echo   p2pollo - Starting Application
echo ============================================================
echo.

REM Verificar que los binarios existan
if not exist "build\bin\api.exe" (
    echo ERROR: API backend not found. Run build-all.bat first.
    pause
    exit /b 1
)

if not exist "frontend\build\p2pollo.exe" (
    echo WARNING: Qt frontend not found
    echo Only starting API backend...
    echo.
    goto :start_api_only
)

REM Agregar libmpv.dll al PATH
set "PATH=%CD%;%CD%\lib;%PATH%"

echo [1/2] Starting Go API backend in background...
start "p2pollo API" /MIN cmd /c "build\bin\api.exe"

REM Esperar a que el API esté listo
timeout /t 2 /nobreak >nul

echo [2/2] Starting Qt frontend...
echo.

cd frontend\build
start "p2pollo" p2pollo.exe
cd ..\..

echo.
echo ============================================================
echo   Application started successfully!
echo ============================================================
echo.
echo - API backend running on http://127.0.0.1:9876
echo - Qt frontend opened in new window
echo.
echo Press any key to stop all processes...
pause >nul

REM Detener procesos
echo.
echo Stopping all processes...
taskkill /F /IM api.exe 2>nul
taskkill /F /IM p2pollo.exe 2>nul
echo Done.
exit /b 0

:start_api_only
echo Starting API backend only...
build\bin\api.exe
pause
