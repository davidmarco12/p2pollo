# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

p2pollo is a P2P streaming desktop application written in Go with Wails (Go backend + Svelte frontend). It uses BitTorrent (via anacrolix/torrent) to download and stream media content, with a built-in web scraper for torrent search. The project is primarily developed on Windows.

### Roadmap
- **Scraper:** Implementar el parseo HTTP real de 1337x, Nyaa.si y rarg.to (actualmente stub).
- **Frontend:** Construir la UI de búsqueda, resultados y reproductor en Svelte.

## Build & Development Commands

```bash
# Wails development (hot-reload)
wails dev                # Runs app in dev mode with hot-reload

# Wails build (production)
wails build              # Compiles to build/bin/p2pollo.exe

# Test
go test -v ./...         # Run all tests
go test -short ./...     # Skip network-dependent tests

# Lint & Format
make lint                # Runs golangci-lint (installs if missing)
make fmt                 # gofmt -s -w
make vet                 # go vet ./...

# Dependencies
go mod tidy              # Tidy Go modules
cd frontend && npm install  # Install frontend dependencies
```

## Architecture

```
main.go                  Wails entry point — embeds frontend, configures window, binds App
app.go                   App struct — puente entre frontend y backend (Wails bindings)
wails.json               Wails project configuration
frontend/                Svelte + Vite frontend
  src/                   Svelte components and pages
  wailsjs/               Auto-generated JS bindings to call Go methods
  dist/                  Built frontend assets (embedded into binary)
build/                   Build assets (icons, manifests)
internal/
  streaming/             Servicio de streaming — combina torrent client + HTTP server local
  scraper/               Orquestador de búsqueda en sitios de torrents
    providers/           Implementaciones por sitio (1337x, Nyaa.si, rarg.to)
  torrent/               Cliente BitTorrent (wrapper anacrolix/torrent, storage sin mmap, file selection)
  stream/                Stream manager — buffer, piece prioritization, descarga secuencial
  config/                YAML config desde ~/.config/p2pollo/config.yaml, defaults via Viper
  player/                (legacy) mpv integration
  search/                (legacy) Búsqueda por trackers
cmd/                     (legacy) Cobra CLI commands
```

**Data flow:** Svelte UI → Wails binding (app.go) → Scraper (1337x/Nyaa.si/rarg.to) → resultados al frontend → usuario elige → Streaming Service (torrent client + HTTP server) → HTML5 `<video>` player

### Wails Bindings (app.go)

Los métodos exportados de `App` se exponen automáticamente al frontend:
- `Search(query)` → busca en todos los providers del scraper
- `PlayMagnet(magnetLink)` → inicia streaming y retorna URL local del video
- `GetStreamProgress()` → retorna progreso de descarga (bytes, velocidad, peers)
- `StopStream()` → detiene el streaming actual

## Key Dependencies

- `wailsapp/wails/v2` — Desktop GUI framework (Go + webview)
- `anacrolix/torrent` — BitTorrent protocol implementation
- `svelte` + `vite` — Frontend framework and build tool
- `spf13/viper` — Configuration management
- `sirupsen/logrus` — Structured logging
- `stretchr/testify` — Test assertions

## Configuration

Default config location: `~/.config/p2pollo/config.yaml`
Template: `config/config.yaml`
Test config: `config/config.test.yaml`

## Language

Code comments, CLI output, and documentation are written in Spanish.
