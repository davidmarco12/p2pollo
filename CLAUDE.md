# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

p2pollo is a P2P streaming desktop application written in Go with Wails (Go backend + Svelte frontend). It uses BitTorrent (via anacrolix/torrent) to download and stream media content, with mpv as the playback engine for instant subtitle support. The project includes a YTS catalog scraper for browsing movies with posters.

### Architecture Philosophy

**Current (Feb 2025):** mpv-based playback with IPC control from Wails
- Svelte UI provides custom controls and movie catalog navigation
- mpv renders video + embedded subtitles directly (instant subtitle switching)
- Go backend manages torrent downloads and controls mpv via JSON IPC
- No ffmpeg extraction needed for subtitles (mpv reads them natively from containers)

### Roadmap
- **Catalog:** Expandir soporte a más fuentes (actualmente solo YTS)
- **Player:** Mejorar controles custom de mpv en Svelte UI
- **Scraper:** Implementar el parseo HTTP real de 1337x, Nyaa.si y rarg.to (actualmente stub)

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
  streaming/             Servicio de streaming — combina torrent client + mpv player
  player/                mpv IPC controller — JSON IPC for playback control
  catalog/               Movie catalog abstraction
    yts/                 YTS scraper implementation (AJAX + HTML parsing)
  scraper/               Orquestador de búsqueda en sitios de torrents
    providers/           Implementaciones por sitio (1337x, Nyaa.si, rarg.to)
  torrent/               Cliente BitTorrent (wrapper anacrolix/torrent, storage sin mmap, file selection)
  stream/                Stream manager — buffer, piece prioritization, descarga secuencial
  config/                YAML config desde ~/.config/p2pollo/config.yaml, defaults via Viper
```

**Data flow:**
```
Movie browsing: Svelte UI → GetPopularMovies/SearchMovies → YTS scraper → grilla de posters
Movie detail:   Click poster → GetMovieDetails → YTS AJAX → lista de torrents
Playback:       Click torrent → PlayMagnet → Stream service (torrent + mpv IPC) → mpv window
Subtitle:       User clicks subtitle → mpv IPC command → instant switch (no extraction)
```

### Wails Bindings (app.go)

Los métodos exportados de `App` se exponen automáticamente al frontend:

**Catalog:**
- `GetPopularMovies()` → peliculas populares del catalogo (YTS)
- `SearchMovies(query)` → busqueda de peliculas por titulo
- `GetMovieDetails(movieJSON)` → detalle completo con torrents disponibles

**Torrent Search (legacy, para búsqueda avanzada):**
- `Search(query)` → busca torrents en scrapers (rargb, thepiratebay)

**Playback:**
- `PlayMagnet(magnetLink)` → inicia descarga torrent y playback con mpv
- `GetStreamProgress()` → progreso de descarga (bytes, velocidad, peers)
- `StopStream()` → detiene streaming y mpv

**mpv Control:**
- `MPVCommand(cmd, args)` → envia comando JSON IPC a mpv (play/pause, seek, subtitles, etc)
- `GetMPVProperty(prop)` → obtiene propiedad de mpv via IPC

## Key Dependencies

- `wailsapp/wails/v2` — Desktop GUI framework (Go + webview)
- `anacrolix/torrent` — BitTorrent protocol implementation
- `svelte` + `vite` — Frontend framework and build tool
- `spf13/viper` — Configuration management
- `sirupsen/logrus` — Structured logging
- `stretchr/testify` — Test assertions
- **mpv** — Media player (external binary, controlled via IPC)

## Configuration

Default config location: `~/.config/p2pollo/config.yaml`
Template: `config/config.yaml`
Test config: `config/config.test.yaml`

### Required External Dependencies

- **mpv**: Must be installed and available in PATH. Download from https://mpv.io
  - Windows: Install mpv and add to PATH, or configure `player.mpv_path` in config
  - Linux/macOS: `brew install mpv` or `apt install mpv`

## mpv IPC Integration

p2pollo controls mpv via JSON IPC (Unix socket or named pipe on Windows):

**Startup:** Go spawns mpv with `--input-ipc-server=\\.\pipe\mpv-p2pollo-{pid}` (Windows) or `/tmp/mpv-p2pollo-{pid}` (Unix)

**Control:** Send JSON commands via pipe:
```json
{"command": ["loadfile", "http://127.0.0.1:8080/stream"]}
{"command": ["set_property", "pause", false]}
{"command": ["cycle", "sub"]}
```

**State:** Poll properties:
```json
{"command": ["get_property", "time-pos"]}
{"command": ["get_property", "duration"]}
```

See `internal/player/` for full implementation.

## Language

Code comments, CLI output, and documentation are written in Spanish.
