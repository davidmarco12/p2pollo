# Frontend Qt/QML de p2pollo

Frontend Qt/QML con mpv embebido para p2pollo. Se comunica con el backend Go via HTTP REST API.

## Requisitos previos

### 1. Qt 6.5 o superior

Descargar e instalar Qt desde: https://www.qt.io/download-qt-installer

Componentes necesarios:
- Qt 6.10 (MinGW 64-bit para Windows, o GCC/Clang para Linux/macOS)
- Qt Creator (IDE opcional pero recomendado)
- Qt Quick (QML)
- Qt Network
- Qt Multimedia

### 2. CMake 3.21 o superior

```bash
# Windows (con Chocolatey)
choco install cmake

# Linux
sudo apt install cmake  # Debian/Ubuntu
sudo dnf install cmake  # Fedora

# macOS
brew install cmake
```

### 3. libmpv

#### Windows

Ya tienes `libmpv-2.dll` en `../lib/`. El CMakeLists.txt se encarga de copiarlo automáticamente.

#### Linux

```bash
# Debian/Ubuntu
sudo apt install libmpv-dev

# Fedora
sudo dnf install mpv-libs-devel

# Arch
sudo pacman -S mpv
```

#### macOS

```bash
brew install mpv
```

### 4. MpvQt (binding de libmpv para Qt)

MpvQt es el wrapper oficial de KDE para usar libmpv en aplicaciones Qt/QML.

#### Compilar desde fuente

```bash
# Clonar repositorio
git clone https://invent.kde.org/libraries/mpvqt.git
cd mpvqt

# Crear directorio de build
mkdir build && cd build

# Configurar con CMake
cmake .. -DCMAKE_INSTALL_PREFIX=/ruta/a/qt  # Ajustar según tu instalación de Qt
# Ejemplo Windows: -DCMAKE_INSTALL_PREFIX=C:/Qt/6.10.0/mingw_64
# Ejemplo Linux: -DCMAKE_INSTALL_PREFIX=/usr/local

# Compilar
cmake --build .

# Instalar
cmake --install .
```

#### Alternativa: Usar paquetes del sistema (Linux)

```bash
# Arch Linux (AUR)
yay -S mpvqt

# Otras distribuciones: compilar desde fuente
```

---

## Compilación

### Opción 1: Qt Creator (Recomendado para desarrollo)

1. Abrir Qt Creator
2. File → Open File or Project
3. Seleccionar `frontend/CMakeLists.txt`
4. Configurar el kit de compilación (MinGW 64-bit o GCC)
5. Presionar el botón "Build" (Ctrl+B)
6. Ejecutar con el botón "Run" (Ctrl+R)

### Opción 2: Línea de comandos

#### Windows (MinGW)

```bash
cd frontend
mkdir build
cd build

# Configurar (ajustar ruta de Qt según tu instalación)
cmake .. -G "MinGW Makefiles" -DCMAKE_PREFIX_PATH=C:/Qt/6.10.0/mingw_64

# Compilar
cmake --build .

# Ejecutar
./p2pollo.exe
```

#### Linux

```bash
cd frontend
mkdir build
cd build

# Configurar
cmake .. -DCMAKE_PREFIX_PATH=/path/to/qt

# Compilar
make -j$(nproc)

# Ejecutar
./p2pollo
```

#### macOS

```bash
cd frontend
mkdir build
cd build

# Configurar
cmake .. -DCMAKE_PREFIX_PATH=/usr/local/opt/qt

# Compilar
make -j$(sysctl -n hw.ncpu)

# Ejecutar
./p2pollo
```

---

## Ejecución

### 1. Iniciar el backend Go

Primero, asegúrate de que el servidor API Go esté ejecutándose:

```bash
# Desde la raíz del proyecto
cd ..
./api.bat  # Windows
# o
./build/bin/api  # Linux/macOS
```

El servidor debe estar escuchando en `http://127.0.0.1:9876`

### 2. Ejecutar el frontend Qt

```bash
cd frontend/build
./p2pollo  # Linux/macOS
./p2pollo.exe  # Windows
```

---

## Estructura del proyecto

```
frontend/
├── CMakeLists.txt          # Configuración de build
├── main.cpp                # Entry point C++
├── backend.h               # Backend C++ (header)
├── backend.cpp             # Backend C++ (implementación)
├── resources.qrc           # Qt resources
├── qml/
│   ├── main.qml           # QML root - navegación
│   ├── HomePage.qml       # Página de inicio (películas populares)
│   ├── SearchPage.qml     # Búsqueda de películas
│   ├── MovieDetail.qml    # Detalles de película + torrents
│   ├── PlayerPage.qml     # Reproductor con mpv embebido
│   └── components/
│       ├── MovieCard.qml      # Tarjeta de película
│       ├── MovieGrid.qml      # Grilla de películas
│       ├── SearchBar.qml      # Barra de búsqueda
│       └── ProgressBar.qml    # Barra de progreso
└── build/                  # Directorio de build (generado)
```

---

## Arquitectura

```
┌─────────────────────────────────────────────────────────┐
│                    Frontend Qt/QML                      │
│                                                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐            │
│  │ HomePage │→ │MovieDetail│→ │PlayerPage│            │
│  │  (QML)   │  │   (QML)   │  │  (QML)   │            │
│  └────┬─────┘  └─────┬─────┘  └────┬─────┘            │
│       │              │              │                   │
│       └──────────────┴──────────────┘                   │
│                      │                                   │
│              ┌───────▼────────┐                         │
│              │   Backend C++   │                         │
│              │ (HTTP requests) │                         │
│              └───────┬────────┘                         │
└──────────────────────┼──────────────────────────────────┘
                       │ HTTP REST
                       │
┌──────────────────────▼──────────────────────────────────┐
│                  Backend Go (API)                        │
│              http://127.0.0.1:9876                       │
│                                                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐             │
│  │ Catalog  │  │Streaming │  │   mpv    │             │
│  │  (YTS)   │  │(torrent) │  │(libmpv)  │             │
│  └──────────┘  └──────────┘  └──────────┘             │
└──────────────────────────────────────────────────────────┘
```

### Flujo de datos:

1. **HomePage** carga películas populares via `backend.getPopularMovies()`
2. Usuario busca → **SearchPage** llama `backend.searchMovies(query)`
3. Usuario selecciona película → **MovieDetail** carga detalles y torrents
4. Usuario selecciona torrent → **PlayerPage**:
   - Llama `backend.playMagnet(magnetLink)`
   - Backend Go inicia descarga del torrent
   - Polling de `/api/stream-path` cada 1s hasta que `ready == true`
   - Cuando listo, carga archivo en mpv: `mpvPlayer.command(["loadfile", path])`
   - Polling de `/api/mpv/state` para sincronizar UI con mpv

---

## Integración con MpvQt

PlayerPage.qml usa el componente `MpvObject` de MpvQt:

```qml
import org.kde.mpv 1.0

MpvObject {
    id: mpvPlayer
    anchors.fill: parent

    // Cargar archivo cuando stream esté listo
    function loadStream(path) {
        mpvPlayer.command(["loadfile", path, "replace"])
    }
}
```

### Control de mpv

Todos los controles se hacen via comandos enviados al backend Go, que los reenvía a libmpv:

```javascript
// Play/Pause
backend.sendMPVCommand(["cycle", "pause"])

// Seek
backend.sendMPVCommand(["seek", 30, "relative"])

// Volumen
backend.setMPVProperty("volume", 80)

// Subtítulos
backend.setMPVProperty("sid", trackId)
```

---

## Desarrollo

### Hot reload de QML

Qt Creator soporta hot reload de archivos QML sin recompilar C++:
1. Modificar archivo .qml
2. Guardar (Ctrl+S)
3. La UI se actualiza automáticamente

### Debugging

```bash
# Ejecutar con logs de Qt
QT_LOGGING_RULES="*.debug=true" ./p2pollo

# Logs de mpv
export MPV_VERBOSE=1
./p2pollo
```

### Agregar nuevos componentes QML

1. Crear el archivo en `qml/` o `qml/components/`
2. Agregarlo a `resources.qrc`:
   ```xml
   <file>qml/components/MiComponente.qml</file>
   ```
3. Recompilar el proyecto

---

## Troubleshooting

### Error: "module "org.kde.mpv" is not installed"

MpvQt no está instalado correctamente. Verificar:
```bash
# Verificar que MpvQt esté en Qt imports
ls /path/to/qt/qml/org/kde/mpv
```

### Error: "libmpv-2.dll not found" (Windows)

Verificar que `libmpv-2.dll` esté en:
- `../lib/libmpv-2.dll` (copiado automáticamente por CMake)
- O en el PATH del sistema

### mpv no carga el video

1. Verificar que el backend Go esté ejecutándose
2. Verificar `/api/stream-path` retorna `ready: true`
3. Verificar que la ruta del archivo sea válida
4. Revisar logs de mpv

### Error de compilación de MpvQt

Asegurar que las versiones sean compatibles:
- Qt 6.5+
- libmpv 0.35+
- CMake 3.21+

---

## Distribución

### Crear ejecutable standalone

#### Windows

```bash
# Compilar en Release
cmake .. -DCMAKE_BUILD_TYPE=Release
cmake --build . --config Release

# Copiar DLLs de Qt (windeployqt)
C:/Qt/6.10.0/mingw_64/bin/windeployqt.exe ./p2pollo.exe

# Resultado: carpeta con .exe + DLLs
```

#### Linux

```bash
# Compilar estático o usar AppImage
cmake .. -DCMAKE_BUILD_TYPE=Release
make

# Crear AppImage (requiere linuxdeploy)
linuxdeploy --executable=p2pollo --appdir=AppDir --output=appimage
```

#### macOS

```bash
# Crear app bundle
cmake .. -DCMAKE_BUILD_TYPE=Release
make

# Crear .dmg
macdeployqt p2pollo.app -dmg
```

---

## Próximas mejoras

- [ ] Caché de imágenes de posters
- [ ] Paginación en grilla de películas
- [ ] Filtros por género/año
- [ ] Lista de favoritos
- [ ] Historial de reproducción
- [ ] Soporte para múltiples idiomas
- [ ] Temas personalizables
- [ ] Control de subtítulos desde UI

---

## Recursos

- [Qt Documentation](https://doc.qt.io/)
- [QML Documentation](https://doc.qt.io/qt-6/qmlapplications.html)
- [MpvQt GitHub](https://invent.kde.org/libraries/mpvqt)
- [libmpv API](https://mpv.io/manual/stable/#lua-scripting-[,options])
- [p2pollo API Documentation](../cmd/api/README.md)
