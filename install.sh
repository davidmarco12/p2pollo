#!/bin/bash

# Script de instalación para Streaming CLI
# Uso: curl -sSL https://raw.githubusercontent.com/tuusuario/streaming-cli/main/scripts/install.sh | bash

set -e

echo "==================================="
echo "  Streaming CLI - Instalador"
echo "==================================="
echo ""

# Detectar OS
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
    Linux*)     OS=linux;;
    Darwin*)    OS=darwin;;
    *)          echo "OS no soportado: $OS"; exit 1;;
esac

case "$ARCH" in
    x86_64)     ARCH=amd64;;
    arm64)      ARCH=arm64;;
    aarch64)    ARCH=arm64;;
    *)          echo "Arquitectura no soportada: $ARCH"; exit 1;;
esac

echo "Detectado: $OS $ARCH"
echo ""

# Verificar mpv
if ! command -v mpv &> /dev/null; then
    echo "⚠️  mpv no está instalado"
    echo ""
    echo "Por favor instala mpv:"
    echo "  Ubuntu/Debian: sudo apt install mpv"
    echo "  macOS: brew install mpv"
    echo ""
    read -p "¿Continuar de todas formas? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Descargar binario
VERSION="latest"
BINARY_NAME="streaming-cli-${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then
    BINARY_NAME="${BINARY_NAME}.exe"
fi

echo "Descargando streaming-cli..."
DOWNLOAD_URL="https://github.com/tuusuario/streaming-cli/releases/${VERSION}/download/${BINARY_NAME}"

INSTALL_DIR="$HOME/.local/bin"
mkdir -p "$INSTALL_DIR"

if command -v curl &> /dev/null; then
    curl -L "$DOWNLOAD_URL" -o "$INSTALL_DIR/streaming-cli"
elif command -v wget &> /dev/null; then
    wget "$DOWNLOAD_URL" -O "$INSTALL_DIR/streaming-cli"
else
    echo "Error: curl o wget requerido"
    exit 1
fi

chmod +x "$INSTALL_DIR/streaming-cli"

# Verificar PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo ""
    echo "⚠️  $INSTALL_DIR no está en tu PATH"
    echo ""
    echo "Añade esto a tu ~/.bashrc o ~/.zshrc:"
    echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
    echo ""
fi

echo ""
echo "✅ Instalación completada!"
echo ""
echo "Comandos de inicio:"
echo "  streaming-cli init              # Inicializar configuración"
echo "  streaming-cli search <query>    # Buscar contenido"
echo "  streaming-cli play <magnet>     # Reproducir"
echo ""
echo "Para más información: streaming-cli --help"