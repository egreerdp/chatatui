# Chatatui — Product Requirements Document

## 1. Overview

Chatatui is a terminal-based real-time chat application. Users connect via a TUI client over WebSocket to a central server that manages rooms, message persistence, and presence. The project targets developers and power users who prefer terminal workflows.

**Deployment models:** Self-hosted (single binary + Postgres + Redis) and managed/hosted.

**Current state:** Functional prototype. Users can register (out-of-band), join rooms, send/receive messages with typing indicators, and auto-reconnect on disconnect. Core infrastructure exists (auth, rate limiting, broadcast pool, Docker Compose) but several foundational pieces are missing or incomplete.

---

## 2. Current State Assessment

### What works

| Area | Status | Notes |
|------|--------|-------|
| WebSocket messaging | Working | JSON wire protocol, read/write pumps, broadcast pool for 10+ clients |
| Room list & join | Working | `GET /rooms`, auto-join first room, manual join via sidebar |
| Room creation | Working | `POST /rooms` from TUI modal, `n` key |
| Typing indicators | Working | Debounced send (2s), 4s expiry, broadcast to room |
| Auto-reconnect | Working | Exponential backoff 1s-30s |
| Message history | Working | Loads N newest messages on room join |
| Registration API | Working | `POST /register` returns API key |
| Rate limiting | Working | Per-user, Redis-backed, configurable window |
| Docker dev env | Working | Compose with hot reload + Delve debugging |
| Config | Partial | TOML via Viper, but no CLI setup flow |

### What's missing or broken

| Area | Gap | Impact |
|------|-----|--------|
| Client registration | No TUI flow; requires `curl` | New users cannot onboard without external tooling |
| Username uniqueness | `name` has no unique constraint | Impersonation possible; no identity guarantee |
| User discovery | No way to find or add other users | Rooms exist but there's no social graph |
| Room discovery | Flat list, no search, no invites | Unusable beyond a handful of rooms |
| Service layer | Handler calls DB directly | Tight coupling, hard to test, hard to swap storage |
| Schema design | Integer PKs with separate UUIDs | Naming confusion (`roomID` vs `roomUUID`), unnecessary indirection |
| User profiles | No GET/PUT user endpoint | Users cannot view or update their identity |
| Presence | No online/offline tracking | No way to know who is in a room or active |
| Message editing/deletion | Not supported | No way to correct mistakes |
| Pagination | Rooms endpoint has a hard limit, no cursor | Won't scale beyond ~100 rooms |
| Error handling | Panics on DB failure, silent message persistence errors | Crashes in production, data loss without notice |
| Testing | Zero tests | No safety net for changes |
| CI/CD | None | No automated quality gates |
| Structured logging | Uses `log.Println` | No levels, no correlation, hard to debug |

---

## 3. Product Vision

### Phase 0 — Solid Foundations (current focus)

Get the basics right. A new user should be able to install the client, register, find a room, and chat — all from the terminal, with no external tooling. The server should be cleanly layered and tested.

### Phase 1 — Social & Discovery

Users can find each other, manage contacts, and discover rooms. Rooms have owners and access control. Presence shows who is online.

### Phase 2 — Rich Messaging

Message editing, deletion, reactions, threads, and search. Markdown rendering in the TUI.

### Phase 3 — Files & Integrations

File sharing (upload/download via pre-signed URLs), link previews, webhook integrations, bot framework.

### Explicitly out of scope

- VoIP / voice / video
- E2E encryption (may revisit later, but not a near-term goal)

---

## 4. Phase 0 — Requirements

Phase 0 is the immediate priority. Everything here should be completed before moving to Phase 1.

### 4.1 Client Registration & Onboarding

**Goal:** A new user can go from `chatatui` to chatting with zero external steps.

| Req | Description |
|-----|-------------|
| R-01 | On first boot with no API key in config, show a registration screen |
| R-02 | Registration screen collects a username and calls `POST /register` |
| R-03 | Returned API key is persisted to `~/.chatatui.toml` automatically |
| R-04 | On subsequent boots, skip registration and proceed to room list |
| R-05 | If registration fails (duplicate name, server down), show actionable error in TUI |
| R-06 | Provide a `chatatui init` CLI command as an alternative setup path |

**Related issues:** #13, #14

### 4.2 Username Integrity

| Req | Description |
|-----|-------------|
| R-07 | `name` column gets a unique index at the database level |
| R-08 | `POST /register` returns `409 Conflict` on duplicate name with a descriptive message |
| R-09 | Username validation: alphanumeric + hyphens/underscores, 3-24 chars, case-insensitive uniqueness |

### 4.3 Service Layer

**Goal:** Handlers deal with HTTP/WebSocket concerns only. Business logic and persistence live in a service layer.

| Req | Description |
|-----|-------------|
| R-10 | Introduce `internal/service/` with interfaces for room, user, and message operations |
| R-11 | `WSHandler` and `client.go` depend on service interfaces, not repositories |
| R-12 | Repositories remain as the persistence implementation behind the service |

**Related issues:** #11

### 4.4 Schema Cleanup

| Req | Description |
|-----|-------------|
| R-13 | UUID becomes the primary key for `rooms`, `users`, and `messages` |
| R-14 | Integer auto-increment IDs are removed from all models |
| R-15 | Foreign keys reference UUIDs consistently |
| R-16 | Migration path documented for existing databases |

**Related issues:** #12

### 4.5 Configuration & Setup

| Req | Description |
|-----|-------------|
| R-17 | `chatatui init` generates a config file with sensible defaults and prompts for server address |
| R-18 | Server address defaults to a well-known value (e.g., `localhost:8080`) for local dev |
| R-19 | Config validation on boot: missing required fields produce clear errors, not panics |

### 4.6 Error Handling & Resilience

| Req | Description |
|-----|-------------|
| R-20 | Replace `panic()` in database init with returned errors and graceful shutdown |
| R-21 | Message persistence failures are surfaced to the sender (not silently dropped) |
| R-22 | API endpoints return structured JSON errors: `{"error": "message", "code": "DUPLICATE_NAME"}` |

### 4.7 Testing

| Req | Description |
|-----|-------------|
| R-23 | Unit tests for hub, room, client, and broadcast pool |
| R-24 | Integration tests for all HTTP endpoints (register, rooms CRUD) |
| R-25 | Integration tests for WebSocket flow (connect, send, receive, history, typing) |
| R-26 | Repository layer tests against a real database (not mocks) |

### 4.8 CI/CD

| Req | Description |
|-----|-------------|
| R-27 | GitHub Actions workflow: lint (`golangci-lint`), test, build on every push/PR |
| R-28 | Tests run against a Postgres service container (not SQLite) |

### 4.9 Observability

| Req | Description |
|-----|-------------|
| R-29 | Replace `log.Println` with `slog` structured logging throughout |
| R-30 | Log levels: debug (verbose hub activity), info (connections, room events), warn (rate limits, retries), error (persistence failures, panics recovered) |

---

## 5. Phase 1 — Requirements (next)

These come after Phase 0 is complete.

### 5.1 User Profiles & Social

| Req | Description |
|-----|-------------|
| R-31 | `GET /users/{id}` and `PUT /users/{id}` for viewing and updating profiles |
| R-32 | Friends / contacts system with mutual add (request + accept) |
| R-33 | User blocking: blocked user's messages hidden client-side, server prevents DM |
| R-34 | TUI screen for managing profile, friends list, and block list |

### 5.2 Room Discovery & Management

| Req | Description |
|-----|-------------|
| R-35 | Room search by name (server-side, with fuzzy matching or prefix search) |
| R-36 | Room invites: owner can invite users, invitee sees pending invites in TUI |
| R-37 | Public vs. private rooms (public rooms appear in search, private require invite) |
| R-38 | Room owner/admin role: can rename, delete, kick members |
| R-39 | `GET /rooms/{id}/members` endpoint with online/offline status |

### 5.3 Presence

| Req | Description |
|-----|-------------|
| R-40 | Track online/offline/away per user, broadcast presence changes to rooms |
| R-41 | TUI sidebar shows online status indicators next to room members |
| R-42 | Idle detection: mark user "away" after N minutes of no input |

### 5.4 Pagination

| Req | Description |
|-----|-------------|
| R-43 | Cursor-based pagination on `GET /rooms` and message history |
| R-44 | TUI supports infinite scroll / "load more" for messages and room list |

---

## 6. Phase 2 — Requirements (future)

### 6.1 Rich Messaging

| Req | Description |
|-----|-------------|
| R-45 | Message editing (`PUT /messages/{id}`) with "edited" indicator in TUI |
| R-46 | Message deletion (`DELETE /messages/{id}`) with tombstone display |
| R-47 | Reactions (emoji shortcodes, displayed inline) |
| R-48 | Threaded replies (reply-to a specific message ID) |
| R-49 | Message search across rooms (server-side full-text search) |
| R-50 | Markdown rendering in the TUI viewport |

---

## 7. Phase 3 — Requirements (future)

### 7.1 Files & Integrations

| Req | Description |
|-----|-------------|
| R-51 | File upload via pre-signed URL (S3-compatible storage) |
| R-52 | File messages rendered as download links in TUI |
| R-53 | Image preview in terminals that support it (Kitty, iTerm2 inline images) |
| R-54 | Webhook endpoint for external services to post messages into rooms |
| R-55 | Bot framework: bot users with API keys, can join rooms and respond to messages |

---

## 8. Architecture Considerations

### Horizontal Scaling

The current architecture is single-instance. For multi-instance deployment:

- **WebSocket fan-out:** Rooms are in-memory per server. A pub/sub layer (Redis Pub/Sub or NATS) is needed to broadcast messages across instances.
- **Presence:** Currently implicit (client in room map). Needs a shared store (Redis) for cross-instance presence.
- **Rate limiting:** Already Redis-backed — works across instances.
- **Session affinity:** Not required if pub/sub handles fan-out, but would reduce cross-instance chatter.

**Recommendation:** Introduce Redis Pub/Sub for room-level message fan-out as the first scaling step. This aligns with the existing Redis dependency and unblocks multi-instance deployment without a full architecture rewrite.

### Self-Hosting Ergonomics

For self-hosters, minimise the dependency surface:

- Single binary (already achieved via Go static build)
- Optional Redis (degrade gracefully: disable rate limiting, use in-memory presence)
- SQLite as an alternative to Postgres for small deployments (GORM already abstracts the driver)
- Provide a `docker-compose.yml` tuned for production (not just dev)
- Document minimum requirements and a quick-start guide

### TODO.md Reconciliation

The existing `TODO.md` has items that overlap with this PRD. Once this PRD is accepted, `TODO.md` should be retired in favour of GitHub issues linked to PRD requirements. Several `TODO.md` items are already done but not checked off (e.g., typing indicators, room creation UI, auto-reconnect, connection status indicator, rate limiting, structured messages).

---

## 9. Open Questions

| # | Question | Context |
|---|----------|---------|
| Q1 | Should rooms have a max member limit? | Broadcast pool scales, but very large rooms may need different UX (no typing indicators, paginated member list) |
| Q2 | Direct messages — separate concept or just a 2-person room? | Affects schema and UX. DMs-as-rooms is simpler but may feel wrong in the room list |
| Q3 | Should the managed offering require account creation beyond the API key? | Email/password auth, OAuth, or keep it API-key-only? |
| Q4 | Message retention policy? | Relevant for self-hosters with limited storage and for any managed offering |
| Q5 | Notifications? | Terminal bell, OS notifications, or out of scope for a TUI? |
