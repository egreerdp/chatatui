# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Chatatui is a terminal-based real-time chat application built in Go. It consists of two components:
- **Server**: WebSocket-based chat server with room support, backed by PostgreSQL and Redis
- **Client**: Terminal UI client using Bubble Tea

## Commands

```bash
# Build then run the TUI client (reads dev-config.toml by default)
mise run dev

# Build then run the chat server
mise run dev serve

# Build only
mise run build

# Run with debug logging (creates debug.log)
DEBUG=1 mise run dev

# Run all tests
go test ./...

# Run tests for a single package
go test ./internal/service/...

# Lint
golangci-lint run
```

## Architecture

### Request flow

```
Client (TUI) → WebSocket /ws/{roomID}
                   ↓
           api.WSHandler
                   ↓
           hub.Hub  ←→  Broker (Redis pub/sub)
                   ↓
           hub.Room → hub.Client (writePump/readPump goroutines)
```

Messages flow through Redis so that multiple server instances share state. `hub.Room.Broadcast` publishes to Redis; `hub.Hub.CreateRoom` subscribes and calls `room.fanOut` for every incoming message.

### Server (`internal/server/`)

- **api/handler.go**: Routes + middleware wiring. All authenticated routes use `middleware.APIKeyAuth` then `RateLimiter.Middleware`. Interfaces consumed by the handler (`ChatService`, `RoomHub`, `UserStore`, `RoomStore`) are defined here — not in their implementation packages.
- **api/ws_handler.go**: Accepts WebSocket, resolves room via `hub.GetOrCreateRoom`, creates a `hub.Client`, sends message history, then calls `client.Run`.
- **hub/hub.go**: Thread-safe map of active `Room`s. `GetOrCreateRoom` is the safe entry point; `CreateRoom` errors on duplicates.
- **hub/room.go**: Fan-out to connected clients. Activates a `BroadcastPool` (10 workers) once a room reaches 10 clients.
- **hub/client.go**: `readPump` receives raw text, persists via `MessagePersister`, wraps in a `hub.Message` JSON envelope, and calls `room.Broadcast`. `writePump` drains the `send` channel.
- **hub/broker.go**: `Broker` interface + `RedisBroker` implementation using Redis pub/sub. Channel key format: `room:<uuid>`.

### Service layer (`internal/service/`)

`ChatService` sits between the API and the repository. Its store interfaces (`RoomStore`, `MessageStore`) are defined in `service/service.go`; mocks live in `service/_mocks/`.

### Client (`internal/client/ui/`)

Bubble Tea model split across four files: `model.go` (types/state), `update.go` (Msg handlers), `view.go` (rendering), `commands.go` (Cmd factories). The model manages WebSocket lifecycle directly, including reconnect backoff and typing indicators with a TTL.

### Repository (`internal/repository/`)

GORM-based PostgreSQL. `NewPostgresDB` opens a connection and runs `AutoMigrate` on startup. `PostgresDB` exposes typed sub-repositories (`Users()`, `Rooms()`, `Messages()`).

### Middleware (`internal/middleware/`)

- `auth.go`: API key auth via `Authorization: Bearer <key>` header; injects `*repository.User` into context.
- `ratelimit.go`: Sliding-window rate limiter backed by Redis (uses `github.com/EwanGreer/cache`).

## Configuration

Config file: `$HOME/.chatatui.toml` or passed via `--config`. See `dev-config.toml` for a local example.

Key fields under `[server]`: `addr`, `database_dsn`, `redis_url`, `message_history_limit`, `room_list_limit`, `rate_limit_requests`, `rate_limit_window_secs`.

## Mocks

Mocks are generated with [mockery](https://vektra.github.io/mockery/) and live in `_mocks/` subdirectories next to the package that defines the interface. Regenerate with:

```bash
mockery --all
```

## Key Dependencies

- **github.com/charmbracelet/bubbletea** + **bubbles**: TUI framework
- **github.com/go-chi/chi/v5**: HTTP router
- **github.com/coder/websocket**: WebSocket library
- **github.com/redis/go-redis/v9**: Redis client
- **gorm.io/gorm** + **gorm.io/driver/postgres**: ORM
- **github.com/spf13/cobra** + **github.com/spf13/viper**: CLI and config
- **github.com/stretchr/testify**: Test assertions and mocks
