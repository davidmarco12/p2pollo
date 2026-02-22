@echo off
REM Build script para p2pollo (Backend Go + Frontend Qt/QML)
REM ============================================================

echo.
echo ============================================================
echo   p2pollo - Build All
echo ============================================================
echo.

REM --- Build Backend Go API ---
echo [1/2] Building Go API backend...
echo.

go build -o build\bin\api.exe .\cmd\api
if %errorlevel% neq 0 (
    echo ERROR: Failed to build Go API backend
    exit /b 1
)

echo ✓ Go API backend built successfully: build\bin\api.exe
echo.

REM --- Build Frontend Qt/QML ---
echo [2/2] Building Qt/QML frontend...
echo.

REM Check if Qt is installed
where /q qmake
if %errorlevel% neq 0 (
    echo WARNING: Qt not found in PATH
    echo Please install Qt 6.5+ and add it to PATH
    echo Example: set PATH=C:\Qt\6.10.0\mingw_64\bin;%PATH%
    echo.
    echo Skipping Qt frontend build...
    goto :skip_qt
)

REM Create build directory
if not exist "frontend\build\" mkdir "frontend\build"
cd frontend\build

REM Run CMake
cmake .. -G "MinGW Makefiles"
if %errorlevel% neq 0 (
    echo ERROR: CMake configuration failed
    echo Make sure MpvQt is installed
    cd ..\..
    exit /b 1
)

REM Build
cmake --build .
if %errorlevel% neq 0 (
    echo ERROR: Qt frontend build failed
    cd ..\..
    exit /b 1
)

REM Deploy Qt dependencies
echo Deploying Qt dependencies...
D:\Qt\6.10.2\mingw_64\bin\windeployqt.exe p2pollo.exe --qmldir ..\qml >nul 2>&1

cd ..\..
echo ✓ Qt frontend built successfully: frontend\build\p2pollo.exe
echo.

:skip_qt

echo.
echo ============================================================
echo   Build Complete!
echo ============================================================
echo.
echo To run the application:
echo   1. Start the API backend: .\api.bat
echo   2. Start the Qt frontend: cd frontend\build ^&^& p2pollo.exe
echo.
echo Or use the combined launcher: .\run.bat
echo.

pause
