@echo off
echo ====================================
echo  p2pollo Frontend
echo ====================================
echo.
echo IMPORTANTE: El backend debe estar corriendo en http://127.0.0.1:9876
echo.
pause
echo.
echo Iniciando frontend...
echo.

cd /d "%~dp0frontend\build"
p2pollo.exe

echo.
echo ====================================
echo La aplicacion se cerro.
echo Presiona cualquier tecla para salir...
pause >nul
