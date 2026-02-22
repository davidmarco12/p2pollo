@echo off
echo ====================================
echo  p2pollo Backend API
echo ====================================
echo.
echo Iniciando servidor en http://127.0.0.1:9876
echo.

cd /d "%~dp0"
go run cmd/api/*.go

echo.
echo ====================================
echo El servidor se detuvo.
echo Presiona cualquier tecla para salir...
pause >nul
