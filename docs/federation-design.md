# Federation Design

Visual reference for the federated chat network described in the federation epic (#81). Each diagram targets one mental model — read whichever helps the question you're holding.

> Companion to: the federation PRD (Goals / Non-goals / API shape live there). This file shows *how the pieces move*, not *what they are*.

---

## 1. Component / package map

Where the new code lives and which packages depend on which. Solid arrows are direct calls; dashed arrows are HTTP across a network boundary.

```mermaid
graph LR
    subgraph "TUI client (internal/client/ui)"
        UI[update.go / view.go]
        CMD[commands.go]
    end

    subgraph "server-b (this server)"
        subgraph "API (internal/server/api)"
            WS[ws_handler.go]
            ME[me_handler.go]
            FED_R["federation/*_handler.go"]
            ADMIN_H[federation/admin_handler.go]
        end

        subgraph "Service (internal/service)"
            CHAT[chat.go]
            FED_S[federation.go]
            ADMIN_S[admin.go]
        end

        subgraph "Federation core (internal/federation)"
            CLIENT[client.go]
            VERIFIER[verifier.go]
            DISCOVERY[discovery.go]
            RL[rate_limiter.go]
            WK[wellknown.go]
        end

        subgraph "Repository (internal/repository)"
            REPOS[(federation.go<br/>peers, room_cache,<br/>blocked, bans)]
            CORE_R[(message.go<br/>room.go / user.go)]
        end

        DB[(Postgres)]
        REDIS[(Redis)]
    end

    subgraph "server-a (peer)"
        REMOTE_WK[/.well-known/chatatui/]
        REMOTE_VERIFY[/federation/verify/]
        REMOTE_ROOMS[/federation/rooms/]
    end

    CMD -.->|HTTPS| FED_R
    CMD -.->|WSS + headers| WS
    UI --> CMD

    WS --> CHAT
    WS --> FED_S
    ME --> CORE_R
    FED_R --> CHAT
    FED_R --> VERIFIER
    ADMIN_H --> ADMIN_S

    FED_S --> CLIENT
    FED_S --> REPOS
    VERIFIER --> CORE_R
    VERIFIER --> RL
    DISCOVERY --> CLIENT
    DISCOVERY --> REPOS
    ADMIN_S --> REPOS
    CHAT --> CORE_R

    REPOS --> DB
    CORE_R --> DB
    RL --> REDIS

    CLIENT -.->|HTTPS| REMOTE_WK
    CLIENT -.->|HTTPS| REMOTE_VERIFY
    CLIENT -.->|HTTPS| REMOTE_ROOMS
```

**Reading guide.** `FederationService` is the orchestrator on the inbound side; `FederationClient` + `DiscoveryJob` are the outbound side. The two never talk directly — they share state through repositories.

---

## 2. Cross-server join sequence

The 10-step join from the PRD, drawn out. Three actors — Alice's TUI, server-b (room owner), server-a (Alice's home server).

```mermaid
sequenceDiagram
    autonumber
    participant TUI as Alice TUI
    participant B as server-b<br/>(room owner)
    participant A as server-a<br/>(home server)

    Note over TUI,A: User picks general@server-b.com from federated search

    TUI->>B: GET /.well-known/chatatui
    B-->>TUI: federation_url, verify_url, ...

    TUI->>B: WSS /ws/<room-uuid><br/>Authorization: Bearer <key><br/>X-Federated-Identity: alice@server-a.com<br/>X-Federated-User-ID: <alice's uuid>

    Note over B: Detect federation headers<br/>Parse domain = server-a.com<br/>Check blocked_domains (reject if blocked)

    B->>A: GET /.well-known/chatatui
    A-->>B: verify_url, ...

    B->>A: POST /federation/verify<br/>{ user_id, api_key }

    Note over A: Rate-limit by caller domain<br/>Check own blocked_domains<br/>Look up user by (id + key hash)

    A-->>B: 200 { verified: true, username, federated_id }

    Note over B: Upsert peer in federation_peers<br/>Trigger DiscoveryJob.immediate(server-a.com)<br/>Check room_bans for this room<br/>(reject if banned)

    B->>B: JoinRoomFederated(roomID, "alice@server-a.com")<br/>Insert room_member with federated_identity<br/>Load message history

    B-->>TUI: history (existing connectedMsg flow)

    Note over TUI,A: server-a is uninvolved from here on

    TUI->>B: { content: "hello" }
    B->>B: Save message with federated_sender = "alice@server-a.com"
    B-->>TUI: WireMessage { author, federated_author, ... }<br/>(broadcast to all room clients)
```

**Watch points.** Steps 4 and 7 are the only network hops between server-b and server-a; once verification returns, server-a is out of the picture for the lifetime of Alice's connection.

---

## 3. Trust & verification flow

What each side knows, trusts, and proves. Useful when reasoning about attacks ("can server-x impersonate alice?") or about why a step exists.

```mermaid
flowchart TB
    classDef known fill:#0e8a16,color:#fff,stroke:#0e8a16
    classDef trusted fill:#1d76db,color:#fff,stroke:#1d76db
    classDef proven fill:#d93f0b,color:#fff,stroke:#d93f0b
    classDef threat fill:#d73a4a,color:#fff,stroke:#d73a4a

    START([TUI dials server-b<br/>with federation headers])

    Q1{"server-b: is the<br/>X-Federated-Identity domain<br/>equal to my own domain?"}
    REJ1["Reject 403<br/>self-claim guard"]:::threat

    Q2{"server-b: is the<br/>caller domain in<br/>blocked_domains?"}
    REJ2["Reject 403"]:::threat

    FETCH["Fetch caller<br/>.well-known via TLS"]:::trusted
    NOTE_TLS["TLS cert pins identity:<br/>only server-a.com can serve<br/>https://server-a.com/.well-known"]:::known

    POST["POST /federation/verify<br/>over TLS"]:::trusted

    Q3{"server-a: is the<br/>server-b domain<br/>in my blocked_domains?"}
    REJ3["Reject 403"]:::threat

    Q4{"server-a: rate limit<br/>exceeded for server-b?"}
    REJ4["Reject 429"]:::threat

    Q5{"server-a: does the<br/>user_id + api_key pair<br/>match a real user?"}
    REJ5["Reject 403<br/>verified: false"]:::threat

    PROVEN["server-b now has proof:<br/>1. server-a vouches for alice<br/>2. alice home is server-a<br/>3. caller domain = server-a"]:::proven

    Q6{"server-b: is<br/>alice@server-a.com<br/>banned from this room?"}
    REJ6["Reject 403"]:::threat

    JOIN([Join room as<br/>federated member]):::known

    START --> Q1
    Q1 -->|yes| REJ1
    Q1 -->|no| Q2
    Q2 -->|yes| REJ2
    Q2 -->|no| FETCH
    FETCH --> NOTE_TLS
    NOTE_TLS --> POST
    POST --> Q3
    Q3 -->|yes| REJ3
    Q3 -->|no| Q4
    Q4 -->|yes| REJ4
    Q4 -->|no| Q5
    Q5 -->|no| REJ5
    Q5 -->|yes| PROVEN
    PROVEN --> Q6
    Q6 -->|yes| REJ6
    Q6 -->|no| JOIN
```

**The trust pillar.** v1 has no request signing — TLS certificate validation is the entire trust mechanism. A claim "I am server-a.com" is only believed because we contacted `https://server-a.com/...` and the cert chain checked out.

---

## 4. Discovery feedback loop

How the federated room cache stays warm. The interesting part is the **immediate** path that fires whenever a foreign user joins for the first time.

```mermaid
flowchart LR
    subgraph "Trigger 1: foreign join"
        JOIN[FederationService<br/>.VerifyAndRecordForeignJoin] --> UPSERT[Upsert peer<br/>in federation_peers]
        UPSERT --> SIGNAL[send to DiscoveryJob.immediate<br/>non-blocking]
    end

    subgraph "Trigger 2: periodic"
        TICK([10m ticker])
        TICK --> WALK[Walk all peers]
        WALK --> STALE{rooms_fetched_at<br/>older than<br/>staleAfter?}
        STALE -->|no| SKIP[skip]
        STALE -->|yes| ENQUEUE[enqueue refresh]
    end

    SIGNAL --> WORKER
    ENQUEUE --> WORKER

    subgraph "DiscoveryJob worker"
        WORKER[pick next domain]
        WORKER --> CALL[FederationClient<br/>.FetchRooms domain]
        CALL --> OK{2xx?}
        OK -->|no| RETRY[leave rooms_fetched_at<br/>untouched]
        RETRY --> DEAD{"7+ days<br/>since last success?"}
        DEAD -->|yes| MARKDEAD[mark peer dead<br/>stop polling]
        DEAD -->|no| WORKER
        OK -->|yes| TX[(BEGIN TX)]
        TX --> DEL[DELETE FROM<br/>federated_room_cache<br/>WHERE peer_id = ...]
        DEL --> INS[INSERT new rows]
        INS --> BUMP[UPDATE peers<br/>SET rooms_fetched_at = now]
        BUMP --> COMMIT[(COMMIT)]
        COMMIT --> WORKER
    end

    subgraph "Read side"
        SEARCH[GET /rooms/federated?q=...] --> READ[(federated_room_cache)]
        READ --> STALEFLAG[mark each row<br/>stale = fetched_at older<br/>than staleAfter]
        STALEFLAG --> RESP[JSON response]
    end
```

**Why full-replace.** A peer's room list isn't authoritative anywhere in our DB — losing one entry between fetches is fine, double-counting is annoying. Full-replace inside one transaction trades a tiny window of "no rooms cached for peer X" for a much simpler invariant.

---

## 5. Data model

New tables (green) and modified existing tables (yellow). Only fields relevant to federation are shown.

```mermaid
erDiagram
    users {
        uuid id PK
        text name
        text api_key_hash
        bool is_admin "NEW"
    }

    rooms {
        uuid id PK
        text name
    }

    room_members {
        uuid id PK
        uuid room_id FK
        uuid user_id "NULLABLE NOW"
        text federated_identity "NEW NULLABLE"
    }

    messages {
        uuid id PK
        uuid room_id FK
        uuid sender_id "NULLABLE NOW"
        text federated_sender "NEW NULLABLE"
        text content
        timestamp created_at
    }

    federation_peers {
        uuid id PK
        text domain UK
        text federation_url
        timestamp last_seen_at
        timestamp rooms_fetched_at
        timestamp created_at
    }

    federated_room_cache {
        uuid id PK
        uuid peer_id FK
        text remote_id
        text name
        int member_count
        timestamp fetched_at
    }

    blocked_domains {
        uuid id PK
        text domain UK
        text reason
        timestamp created_at
        uuid blocked_by FK
    }

    room_bans {
        uuid id PK
        uuid room_id FK
        text federated_id
        text reason
        timestamp created_at
        text banned_by
    }

    users ||--o{ room_members : "joins"
    rooms ||--o{ room_members : "has"
    users ||--o{ messages : "sends"
    rooms ||--o{ messages : "in"
    federation_peers ||--o{ federated_room_cache : "advertises"
    rooms ||--o{ room_bans : "has bans"
    users ||--o{ blocked_domains : "blocked_by"
```

**Invariants worth holding in your head.**

- `room_members`: exactly one of `(user_id, federated_identity)` is non-null. Enforced by raw SQL since GORM can't express it.
- `messages`: same — exactly one of `(sender_id, federated_sender)` is non-null.
- `federated_room_cache.remote_id` is a string (the room UUID *on the remote server*), not a local FK. Local rooms and remote rooms never collide.
- Unique on `(peer_id, remote_id)` so re-fetching is idempotent.
- Unique on `(room_id, federated_id)` so bans can't double up.

---

## 6. Moderation enforcement points

Every place a block or ban is checked along the federated join path. Useful as a checklist when implementing or auditing #77.

```mermaid
sequenceDiagram
    autonumber
    participant TUI as Alice TUI
    participant B as server-b
    participant DB_B as server-b DB
    participant A as server-a
    participant DB_A as server-a DB

    TUI->>B: WSS /ws/<roomID> + federation headers

    rect rgb(255, 235, 235)
        Note over B,DB_B: Check 1: caller domain blocked?
        B->>DB_B: SELECT FROM blocked_domains WHERE domain = "server-a.com"
        DB_B-->>B: row?
        alt blocked
            B-->>TUI: 403 (close WS)
        end
    end

    rect rgb(255, 235, 235)
        Note over B: Check 2: self-claim guard
        B->>B: identity.Domain == cfg.Server.Domain?
        alt yes
            B-->>TUI: 403 self-claim
        end
    end

    B->>A: POST /federation/verify

    rect rgb(255, 235, 235)
        Note over A,DB_A: Check 3: server-a's own block list
        A->>DB_A: SELECT FROM blocked_domains WHERE domain = "server-b.com"
        DB_A-->>A: row?
        alt blocked
            A-->>B: 403
            B-->>TUI: 403 (propagated)
        end
    end

    rect rgb(255, 235, 235)
        Note over A: Check 4: rate limit
        A->>A: limiter.Allow("server-b.com")?
        alt over limit
            A-->>B: 429
            B-->>TUI: 429 (propagated)
        end
    end

    A-->>B: 200 verified

    rect rgb(255, 235, 235)
        Note over B,DB_B: Check 5: per-room ban
        B->>DB_B: SELECT FROM room_bans WHERE room_id=? AND federated_id=?
        DB_B-->>B: row?
        alt banned
            B-->>TUI: 403 banned from room
        end
    end

    B-->>TUI: connected (history follows)
```

**Five chokepoints, two databases, both directions.** Checks 1, 2, and 5 live in server-b's WS join path; checks 3 and 4 live inside server-a's `/federation/verify`. None of these checks should ever be skipped — even though check 3 looks redundant with check 1, they're mirror images and either side may have unique reasons to block.

---

## How to update these diagrams

GitHub renders Mermaid natively, so previews work in PRs. For local rendering during edits:

```bash
# Option 1: use the GitHub preview when pushing.
# Option 2: install mermaid-cli and render to PNG.
npx --yes @mermaid-js/mermaid-cli -i docs/federation-design.md -o /tmp/diagram.png
```

Keep changes diagram-by-diagram — small commits per section make review and rollback easier.
