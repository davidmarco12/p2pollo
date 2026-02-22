@echo off
echo ====================================
echo Creando release portable de p2pollo
echo ====================================
echo.

cd /d "%~dp0"

REM Verificar que existe la carpeta portable
if not exist "dist\p2pollo-portable" (
    echo ERROR: No existe dist\p2pollo-portable
    echo Ejecuta primero la compilacion.
    pause
    exit /b 1
)

REM Obtener version o usar fecha
for /f "tokens=2 delims==" %%a in ('wmic OS Get localdatetime /value') do set "dt=%%a"
set "YY=%dt:~2,2%" & set "MM=%dt:~4,2%" & set "DD=%dt:~6,2%"
set "version=v1.0.0-%YY%%MM%%DD%"

echo Version: %version%
echo.

REM Nombre del archivo ZIP
set "zipfile=dist\p2pollo-portable-%version%.zip"

echo Comprimiendo en: %zipfile%
echo.

REM Usar PowerShell para crear ZIP
powershell -Command "Compress-Archive -Path 'dist\p2pollo-portable\*' -DestinationPath '%zipfile%' -Force"

if %errorlevel% equ 0 (
    echo.
    echo ====================================
    echo Release creado exitosamente!
    echo ====================================
    echo.
    echo Archivo: %zipfile%
    for %%A in ("%zipfile%") do echo Tamano: %%~zA bytes
    echo.
    echo Puedes distribuir este archivo ZIP.
    echo Los usuarios solo deben:
    echo   1. Descomprimirlo
    echo   2. Ejecutar p2pollo.bat
    echo.
) else (
    echo.
    echo ERROR: Fallo la compresion
)

pause
