# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

p2pollo is a P2P streaming client written in Go. It uses BitTorrent (via anacrolix/torrent) to download and stream media content. The project is primarily developed on Windows.

**Current state:** CLI-based with mpv as external player.
**Direction:** Migrating to a GUI desktop application with an embedded video player (libVLC) and a built-in web scraper for torrent search.

### Roadmap
- **GUI:** Replace the CLI with a Wails desktop application (Go backend + web frontend via native webview).
- **Embedded player:** Replace mpv with an HTML5 `<video>` element fed by a local HTTP server that streams torrent content — no external player dependencies.
- **Scraper:** Built-in web scraper to search torrents directly from sites like 1337x, Nyaa.si, and RARBG (rarg.to), replacing the current tracker-based search.

## Build & Development Commands

```bash
# Build
make build              # Compiles to bin/p2pollo (NOTE: Makefile targets ./cmd/main.go but entry point is ./main.go — may need fixing)
go build -o bin/p2pollo .  # Direct build from root

# Run
make run                # go run ./cmd/main.go
go run .                # Run from root main.go

# Test
make test               # go test -v ./...
go test -v ./...        # Run all tests
go test -v ./internal/client/  # Run tests for a specific package
go test -short ./...    # Skip network-dependent tests

# Coverage
make coverage           # Generates coverage.txt and coverage.html

# Lint & Format
make lint               # Runs golangci-lint (installs if missing)
make fmt                # gofmt -s -w
make vet                # go vet ./...

# Dependencies
make deps               # go mod download && go mod tidy

# Cross-compile
make build-all          # Builds for linux/amd64, darwin/amd64, darwin/arm64, windows/amd64
```

## Architecture

```
main.go                  Entry point — delegates to cmd.Execute()
cmd/                     Cobra CLI commands
  root.go                Root command, global flags (--verbose, --profile, --config), pprof setup
  play.go                `p2pollo play <magnet>` — streams a torrent via mpv
  search.go              `p2pollo search <query>` — searches torrents across trackers
  config.go              `p2pollo config` — configuration management
internal/
  client/                Wrapper around anacrolix/torrent — manages P2P connections, readers, piece prioritization
  config/                YAML config loading from ~/.config/p2pollo/config.yaml, defaults via Viper
  player/                mpv integration — supports file-based and stdin pipe streaming
  search/                Torrent search across multiple trackers with caching and filtering
  stream/                Stream manager — buffer monitoring, piece prioritization for sequential playback
```

**Data flow (current):** CLI command → Client (BitTorrent) → Torrent reader → temp file or stdin pipe → mpv player
**Data flow (planned):** Wails GUI → Scraper (1337x/Nyaa.si/rarg.to) → Client (BitTorrent) → Torrent reader → Local HTTP server → HTML5 `<video>` player

The `play` command has two modes:
- **Default:** Downloads to a temp file in `~/.cache/p2pollo/temp/`, launches mpv once buffer threshold is met
- **Pipe mode** (`--pipe`): Streams directly from torrent reader to mpv's stdin (experimental, no disk usage)

## Key Dependencies

- `anacrolix/torrent` — BitTorrent protocol implementation
- `spf13/cobra` + `spf13/viper` — CLI framework and configuration
- `sirupsen/logrus` — Structured logging
- `fatih/color` + `schollz/progressbar` — Terminal UI
- `stretchr/testify` — Test assertions

## Configuration

Default config location: `~/.config/p2pollo/config.yaml`
Template: `config/config.yaml`
Test config: `config/config.test.yaml`

## Language

Code comments, CLI output, and documentation are written in Spanish.
