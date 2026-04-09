@echo off
echo ====================================
echo   p2pollo - Modo desarrollo
echo ====================================
echo.

cd /d "%~dp0"

REM Compilar backend
echo Compilando backend...
set CGO_ENABLED=0
go build -o dist\p2pollo-portable\p2pollo-backend.exe .\cmd\api\
if %errorlevel% neq 0 (
    echo ERROR: Fallo la compilacion del backend
    pause
    exit /b 1
)

REM Iniciar backend en background
echo Iniciando backend API...
start "" /B dist\p2pollo-portable\p2pollo-backend.exe

timeout /t 2 /nobreak >nul

REM Iniciar frontend (usa el exe ya compilado)
echo Iniciando frontend...
dist\p2pollo-portable\p2pollo.exe

REM Al cerrar frontend, matar backend
taskkill /F /IM p2pollo-backend.exe >nul 2>&1
