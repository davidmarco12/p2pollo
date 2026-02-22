@echo off
REM Ejecutar el servidor HTTP API de p2pollo
REM El servidor escucha en http://127.0.0.1:9876

REM Agregar libmpv.dll al PATH
set "PATH=%CD%;%CD%\lib;%PATH%"

echo Iniciando servidor HTTP API de p2pollo en http://127.0.0.1:9876
echo.
echo Endpoints disponibles:
echo   GET  /api/health             - Health check
echo   GET  /api/popular            - Peliculas populares
echo   GET  /api/search?q=query     - Buscar peliculas
echo   POST /api/movie/details      - Detalles de pelicula
echo   POST /api/play               - Iniciar streaming
echo   GET  /api/stream-path        - Obtener path del archivo
echo   GET  /api/progress           - Progreso de descarga
echo   POST /api/stop               - Detener streaming
echo   GET  /api/mpv/state          - Estado de mpv
echo   GET  /api/mpv/tracks         - Tracks de mpv
echo   POST /api/mpv/command        - Comando a mpv
echo   POST /api/mpv/property       - Establecer propiedad
echo.
echo Presiona Ctrl+C para detener el servidor
echo.

build\bin\api.exe
