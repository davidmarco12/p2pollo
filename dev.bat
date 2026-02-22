@echo off
REM Script para ejecutar wails dev con libmpv.dll en el PATH

REM Agregar directorio actual y lib/ al PATH
set "PATH=%CD%;%CD%\lib;%PATH%"

REM Verificar que libmpv.dll existe
if not exist "libmpv.dll" (
    if not exist "lib\libmpv.dll" (
        echo ERROR: libmpv.dll no encontrado
        echo Copia lib/libmpv-2.dll como libmpv.dll
        exit /b 1
    )
)

echo Ejecutando wails dev con libmpv.dll en PATH...
wails dev
