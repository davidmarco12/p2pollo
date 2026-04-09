@echo off
echo ====================================
echo   Recompilando p2pollo
echo ====================================
echo.

cd /d "%~dp0"

set "QT_PATH=D:\Qt\6.10.2\mingw_64\bin"
set "MINGW_PATH=D:\Qt\Tools\mingw1310_64\bin"
set "CMAKE_PATH=D:\Qt\Tools\CMake_64\bin"
set "NINJA_PATH=D:\Qt\Tools\Ninja"
set "PATH=%QT_PATH%;%MINGW_PATH%;%CMAKE_PATH%;%NINJA_PATH%;%PATH%"

REM ---- Backend Go ----
echo [1/2] Compilando backend Go...
set CGO_ENABLED=0
go build -o dist\p2pollo-portable\p2pollo-backend.exe .\cmd\api\
if %errorlevel% neq 0 (
    echo ERROR: Fallo la compilacion del backend
    pause
    exit /b 1
)
echo Backend OK

REM ---- Frontend Qt ----
echo.
echo [2/2] Compilando frontend Qt...
cmake --build frontend\build --config Release
if %errorlevel% neq 0 (
    echo ERROR: Fallo la compilacion del frontend
    pause
    exit /b 1
)

REM Copiar nuevo exe al portable
copy /Y frontend\build\p2pollo.exe dist\p2pollo-portable\p2pollo.exe >nul
echo Frontend OK

echo.
echo ====================================
echo   Compilacion exitosa!
echo ====================================
echo.
echo Para ejecutar: dist\p2pollo-portable\p2pollo.bat
echo.
pause
