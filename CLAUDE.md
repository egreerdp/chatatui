# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Chatatui is a terminal-based real-time chat application built in Go. It consists of two components:
- **Server**: WebSocket-based chat server with room support
- **Client**: Terminal UI client using Bubble Tea

## Commands

```bash
# Run the TUI client
mise run dev

# Start the chat server (listens on :8080)
mise run dev serve

# Build
mise run build

# Run with debug logging (creates debug.log)
DEBUG=1 mise run dev
```

## Architecture

### Server (`internal/server/`)
- **server.go**: HTTP server wrapper with graceful shutdown
- **api/handler.go**: Chi router setup with middleware (Logger, Recoverer, Heartbeat)
- **api/ws_handler.go**: WebSocket endpoint handler at `/ws/{roomID}`
- **hub/**: Real-time messaging core
  - `hub.go`: Manages all rooms, thread-safe room creation/lookup
  - `room.go`: Manages clients within a room, handles message broadcasting
  - `client.go`: WebSocket client with read/write pumps and send channel

### Client (`internal/client/`)
- **ui/index.go**: Bubble Tea model (currently minimal scaffolding)

### Repository (`internal/repository/`)
- **sqlite.go**: GORM-based SQLite connection
- **room.go**, **user.go**, **message.go**: Database models (not yet integrated with hub)

### CLI (`cmd/`)
Uses Cobra for commands:
- Root command runs the TUI client
- `serve` subcommand starts the server

## Key Dependencies

- **github.com/charmbracelet/bubbletea**: TUI framework
- **github.com/go-chi/chi/v5**: HTTP router
- **github.com/coder/websocket**: WebSocket library
- **gorm.io/gorm** + **gorm.io/driver/sqlite**: ORM and database
- **github.com/spf13/cobra** + **github.com/spf13/viper**: CLI and config

## Configuration

Config file: `$HOME/.chatatui.toml` (TOML format, loaded via Viper)
