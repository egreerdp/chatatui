# Architecture

## Client-Server Data Flow

```mermaid
sequenceDiagram
    participant Client as TUI Client
    participant API as HTTP API
    participant Hub as Hub
    participant Room as Room
    participant DB as SQLite

    Note over Client,DB: Registration & Setup
    Client->>API: POST /register {username}
    API->>DB: Create user
    DB-->>API: User + API key
    API-->>Client: {api_key}

    Note over Client,DB: Fetching Rooms
    Client->>API: GET /rooms (API key header)
    API->>DB: List rooms
    DB-->>API: Rooms[]
    API-->>Client: [{id, name}]

    Note over Client,DB: Joining a Room
    Client->>API: WS /ws/{roomID} (API key header)
    API->>Hub: GetOrCreateRoom(roomID)
    Hub-->>API: Room
    API->>DB: AddMember(room, user)
    API->>Room: Add(client)
    API->>DB: GetByRoom(roomID, limit=50)
    DB-->>API: Message history
    API-->>Client: Historical messages

    Note over Client,DB: Sending Messages
    Client->>Room: Send message (WebSocket)
    Room->>DB: Create message
    Room->>Room: Broadcast to other clients
    Room-->>Client: Message (to other clients)

    Note over Client,DB: Disconnect
    Client->>Room: Close WebSocket
    Room->>Room: Remove(client)
```

## Components

| Component | Location | Description |
|-----------|----------|-------------|
| TUI Client | `internal/client/ui/` | Bubble Tea terminal UI |
| HTTP API | `internal/server/api/` | Chi router with REST + WebSocket endpoints |
| Hub | `internal/server/hub/hub.go` | Manages all rooms, thread-safe |
| Room | `internal/server/hub/room.go` | Manages clients in a room, broadcasts messages |
| Client | `internal/server/hub/client.go` | WebSocket read/write pumps per connection |
| SQLite | `internal/repository/` | GORM-based persistence layer |
