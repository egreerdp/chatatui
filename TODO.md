# TODO

## Client

### UI/UX
- [ ] Add room metadata display (active users, total messages, creation date)
- [ ] Allow users to view and update their personal details (username, friends, etc.)
- [ ] Create init command for basic app setup (generate config, register user)
- [ ] Add room creation UI (currently rooms are auto-created server-side)
- [ ] Add message search/filtering
- [ ] Show typing indicators when others are typing
- [ ] Display user presence/online status in room sidebar
- [ ] Add settings/preferences panel
- [ ] Configurable sidebar width (currently hardcoded to 20 chars)
- [ ] Message timestamps display

### Networking
- [ ] Auto-reconnect on WebSocket disconnect
- [ ] Connection status indicator in UI
- [ ] Retry logic for failed message sends
- [ ] Handle server unavailable gracefully (currently shows generic error)

## Server

### API Endpoints
- [ ] GET/PUT `/users/{id}` - user profile endpoints
- [ ] POST `/rooms` - explicit room creation (vs auto-create on WS connect)
- [ ] DELETE `/rooms/{id}` - room deletion/archiving
- [ ] PUT/DELETE `/messages/{id}` - message editing/deletion
- [ ] GET `/rooms/{id}/members` - list room members with presence

### Features
- [ ] Friends table with repo functions and endpoints
- [ ] User blocking/muting functionality
- [ ] Typing indicators (broadcast typing events)
- [ ] Presence tracking (online/offline/away status)
- [ ] Read receipts
- [ ] Rate limiting on endpoints
- [ ] Pagination on `/rooms` endpoint (currently hardcoded 100 limit)
- [ ] Return messages as structured model (author, timestamp, edited, etc.) not just text

### Configuration
- [x] Server config file/section (port, db path, limits)
- [x] Configurable message history limit (currently hardcoded 50)
- [x] Configurable room list limit (currently hardcoded 100)

## Infrastructure

### Error Handling
- [ ] Replace panics with graceful error handling in sqlite.go
- [ ] Surface message persistence errors to user (currently silent)
- [ ] Structured error responses from API endpoints
- [ ] Error wrapping with context throughout codebase

### Observability
- [ ] Add structured logging (replace log.Println)
- [ ] Add metrics (message count, active connections, latency)
- [ ] Health check endpoint beyond `/heartbeat`
- [ ] Request tracing

### Testing
- [ ] Unit tests for hub/room/client
- [ ] Integration tests for API endpoints
- [ ] E2E tests for WebSocket flow
- [ ] Repository layer tests

### DevOps
- [ ] Dockerfile for server
- [ ] Docker Compose for local dev (server + sqlite)
- [ ] CI/CD pipeline (build, test, lint)
