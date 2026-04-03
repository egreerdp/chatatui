# Chatatui — TUI Product Requirements Document

> This document covers UX and interaction design for the terminal client only.
> For server-side and API requirements, see [PRD.md](./PRD.md).

---

## 1. Overview

The chatatui TUI is a keyboard-driven terminal chat client built with [Bubble Tea](https://github.com/charmbracelet/bubbletea). It targets developers and power users who prefer terminal workflows and expect a first-class keyboard experience with no mouse dependency.

### Design principles

| Principle | Meaning |
|-----------|---------|
| **Keyboard-first** | Every action is reachable by keyboard. Mouse is optional (scroll only). |
| **Terminal-native** | Works in any ANSI-256-color terminal. No external renderers. |
| **Responsive** | Layout adapts to any terminal size ≥ 80×24. |
| **Minimal chrome** | UI surface is reserved for content, not decoration. |
| **Actionable errors** | Errors tell the user what to do, not just what went wrong. |

---

## 2. Current State

### What works

| Screen / Feature | Status | Notes |
|-----------------|--------|-------|
| Main layout (sidebar + viewport + input) | Working | Sidebar width hardcoded at 20 chars |
| Room list in sidebar | Working | Fetched on boot and every 5s |
| Room creation modal (`n`) | Working | `POST /rooms`, auto-connects to new room |
| Message viewport | Working | History loaded on join, scrollable |
| Typing indicators | Working | Debounced 2s send, 4s expiry |
| Connection status indicator | Working | Colored dot: green/yellow/red |
| Auto-reconnect | Working | Exponential backoff 1s–30s |
| Character counter | Working | Color-coded: muted/yellow/red |

### Known gaps

| Gap | Impact |
|-----|--------|
| No registration screen | New users must run `curl` to get an API key |
| No `chatatui init` wizard | Config must be hand-edited |
| Sidebar width hardcoded | Breaks at narrow terminal widths |
| No minimum terminal size check | Layout corrupts below 80×24 |
| No help overlay | Key bindings not discoverable |
| Input allows send over 300-char limit | Only warns, does not prevent |
| No smart auto-scroll | Always jumps to bottom, even when user is scrolled up |

---

## 3. Screen Inventory

| Screen | Phase | Status | Trigger |
|--------|-------|--------|---------|
| Registration screen | 0 | Missing | First boot with no API key |
| Main layout | 0 | Exists | After successful auth |
| Room creation modal | 0 | Exists | `n` key |
| Help overlay | 0 | Missing | `?` key |
| `chatatui init` CLI wizard | 0 | Stub | `chatatui init` subcommand |
| Profile screen | 1 | Not started | `p` key |
| Room search | 1 | Not started | `/` key |
| Member list panel | 1 | Not started | `m` key |
| Pending invites panel | 1 | Not started | `i` key |

---

## 4. Navigation Model & Keybindings

### 4.1 Focus model

There are four focus states, cycling in order:

```
focusRooms ←→ focusInput
                ↑
           focusMessages (viewport, scroll mode)
```

`Tab` / `Shift+Tab` cycle between `focusRooms` and `focusInput`. `[` / `]` and `←` / `→` move between panels. `Esc` from `focusInput` returns to `focusRooms`.

### 4.2 Keybinding table

#### Global (all focus states)

| Key | Action |
|-----|--------|
| `Ctrl+C` | Quit unconditionally |
| `q` | Quit (blocked when `focusInput` or `focusCreateRoom`) |
| `?` | Toggle help overlay *(Phase 0 addition)* |

#### Room sidebar (`focusRooms`)

| Key | Action |
|-----|--------|
| `↑` / `k` | Move selection up |
| `↓` / `j` | Move selection down |
| `Enter` | Join selected room |
| `n` | Open room creation modal |
| `r` | Refresh room list |
| `]` / `→` | Move focus to input |
| `/` | Focus room search input *(Phase 1)* |

#### Message input (`focusInput`)

| Key | Action |
|-----|--------|
| `Enter` | Send message (blocked if empty or over limit) |
| `[` / `←` / `Esc` | Move focus to rooms |
| `Tab` | Move focus to rooms |
| `Ctrl+L` | Clear viewport *(Phase 0 addition)* |
| `Shift+Enter` | Insert newline *(Phase 2)* |

#### Message viewport (`focusMessages`)

| Key | Action |
|-----|--------|
| `PgUp` / `PgDn` | Scroll viewport |
| `↑` / `↓` | Scroll one line |
| `e` | Edit selected message *(Phase 2)* |
| `d` | Delete selected message *(Phase 2)* |
| `r` | Add reaction to selected message *(Phase 2)* |
| `t` | Reply in thread to selected message *(Phase 2)* |

#### Room creation modal (`focusCreateRoom`)

| Key | Action |
|-----|--------|
| `Enter` | Submit (blocked if input empty) |
| `Esc` | Cancel, return to `focusRooms` |

---

## 5. Component Specifications

### 5.1 Registration Screen

Shown on first boot when no `api_key` is present in `~/.chatatui.toml`.

**Layout:**

```
╭──────────────────────────────────────╮
│                                      │
│          Welcome to chatatui         │
│                                      │
│  Choose a username                   │
│  ┌────────────────────────────────┐  │
│  │ ▌                              │  │
│  └────────────────────────────────┘  │
│  Username must be 3-24 chars         │
│  [Enter] register  [Ctrl+C] quit     │
│                                      │
╰──────────────────────────────────────╯
```

**States:**

| State | Display |
|-------|---------|
| Idle | Empty input, hint text below |
| Invalid | Red hint: `"3–24 chars, letters/numbers/-/_"` |
| Valid | Green hint: `"Looks good!"` |
| Submitting | Input disabled, spinner, `"Registering..."` |
| Error | Red message below input, input re-enabled (e.g. `"Username taken — choose another"`) |
| Success | Transition to loading rooms |

**Behaviour:**
- Validate on every keystroke (no submit required to see validation state)
- On success: write `api_key` to `~/.chatatui.toml`, then proceed to room list
- `Ctrl+C` exits the application from this screen

### 5.2 Main Layout

```
╭──────────────────────────────────────────────────────────────────────────╮
│ Rooms                │ #general ● connected                               │
│──────────────────────│────────────────────────────────────────────────────│
│ > general            │ ╭──────────────────────────────────────────────╮  │
│   random             │ │ 09:14 alice  hello everyone                  │  │
│   dev                │ │ 09:15 bob    hey! what's up                  │  │
│                      │ │ 09:16 alice  not much, just coding           │  │
│                      │ ╰──────────────────────────────────────────────╯  │
│                      │ alice is typing...                                 │
│                      │ ┌────────────────────────────────────────────────┐│
│                      │ │ ▌                                              ││
│                      │ └────────────────────────────────────────────────┘│
│ tab/←→ panels  j/k nav  n new  r refresh  enter join/send  ? help  q quit│
╰──────────────────────────────────────────────────────────────────────────╯
```

**Responsive sizing:**
- Minimum supported terminal: 80×24
- Below minimum: full-screen message `"Terminal too small (need 80×24)"` with current size shown
- Sidebar width: `max(20, min(30, termWidth/5))`
- Viewport fills remaining width after sidebar and divider

### 5.3 Sidebar

- Active room: `> Room Name`
- Inactive rooms: `  Room Name`
- Scrollable when room count exceeds visible height
- Empty state: `  (no rooms)` with dim hint `  Press n to create one`
- Phase 1 additions:
  - Presence dot: `● Room Name` (green = online activity, grey = quiet)
  - Member count: `● Room Name (3)`
  - Unread badge: `  Room Name [2]`
  - Search input at top when `/` pressed

### 5.4 Message Viewport

**Message format:**

```
HH:MM  Username    message text wraps here and the
                   continuation lines are indented
                   to align with the message start
```

- Username column width = longest username currently visible (capped at 16 chars)
- System messages: centered, muted color, no author column
- Error messages: red, left-aligned, no author column
- Phase 2 additions:
  - `(edited)` appended in muted color on edited messages
  - `[message deleted]` tombstone in muted color

**Scroll behaviour:**
- Auto-scroll to bottom on new message **unless** user has scrolled up
- When scrolled up and new messages arrive: show `↓ N new message(s)` indicator at bottom right
- Clicking / pressing `End` on the indicator jumps to bottom and clears it

### 5.5 Input Component

- Single-line text input
- Width: viewport width minus 2 chars padding
- Char counter in status bar (right side): `284 / 300`
  - Muted when >50 remaining
  - Yellow when ≤50 remaining
  - Red when ≤10 remaining
- Submission blocked (not just warned) when over the 300-char limit
- Input cleared on successful send
- Phase 2: `Shift+Enter` inserts a newline; input grows up to 5 lines before scrolling

### 5.6 Status Bar

Single line at the bottom of the main layout (inside the outer border):

```
● Connected to #general                                          284 / 300
```

- Left: connection state and room name
- Right: char counter (only shown when `focusInput`)
- Connection states: `● Connected to #room` (green) / `⟳ Reconnecting...` (yellow) / `✕ Disconnected` (red)

### 5.7 Typing Indicator

One line between the viewport and the input box:

| Active typers | Display |
|---------------|---------|
| 0 | ` ` (blank line; space preserved) |
| 1 | `alice is typing...` |
| 2 | `alice and bob are typing...` |
| 3+ | `Several people are typing...` |

- Phase 1: animate the ellipsis (`·` → `··` → `···` → `·`) on a 400ms tick

### 5.8 Help Overlay

Triggered by `?`. Rendered as a centered modal over the main layout.

```
╭── Key Bindings ────────────────────────────────╮
│                                                │
│  Navigation                                    │
│  tab / shift+tab    switch panel focus         │
│  [ / ]  ←  →        move between panels       │
│  j / k  ↑  ↓        navigate room list        │
│                                                │
│  Rooms                                         │
│  n                  create new room            │
│  r                  refresh room list          │
│  enter              join selected room         │
│                                                │
│  Messaging                                     │
│  enter              send message               │
│  ctrl+l             clear viewport             │
│                                                │
│  General                                       │
│  q / ctrl+c         quit                       │
│  ?                  close this help            │
│                                                │
╰────────────────────────────────────────────────╯
```

---

## 6. State Machine

```
Startup
├── No api_key in config  ──►  RegistrationScreen
│     ├── Submit success  ──►  LoadingRooms
│     └── Ctrl+C          ──►  Exit
│
└── api_key present       ──►  LoadingRooms
      ├── Fetch OK         ──►  RoomList (auto-connect to first room)
      └── Fetch fail       ──►  RoomList (error shown in sidebar)

RoomList
└── Select + Enter        ──►  ConnectingToRoom
      ├── Connected        ──►  ChatView
      └── Failed           ──►  ChatView (disconnected state)

ChatView
├── Connection lost       ──►  Reconnecting (banner, backoff loop)
│     └── Reconnected     ──►  ChatView
└── q / Ctrl+C            ──►  Exit
```

---

## 7. Terminal Compatibility

| Requirement | Target |
|-------------|--------|
| Color depth | ANSI-256 (`xterm-256color`); graceful fallback to 8-color |
| Minimum size | 80 columns × 24 rows |
| Mouse | Optional — scroll viewport only; layout must work without |
| Graphics | None (Sixel/Kitty planned for Phase 3 image previews) |
| Known-tested terminals | iTerm2, Alacritty, Windows Terminal, tmux, GNU Screen |
| Color detection | `TERM` / `COLORTERM` env vars; Lipgloss `HasDarkBackground()` |

---

## 8. Phase 0 UX Requirements

| Req | Description | PRD ref |
|-----|-------------|---------|
| U-01 | First-boot registration screen with live username validation | R-01, R-02 |
| U-02 | API key written to `~/.chatatui.toml` automatically on registration success | R-03 |
| U-03 | Subsequent boots skip registration and go straight to room list | R-04 |
| U-04 | Registration errors shown inline with actionable text (e.g. `"Username taken — choose another"`) | R-05 |
| U-05 | `chatatui init` interactive CLI wizard: prompts server address and username, calls `POST /register`, writes config | R-06, R-17 |
| U-06 | Config validation on boot shows specific field-level errors, not panics | R-19 |
| U-07 | Sidebar width adapts to terminal width using `max(20, min(30, termWidth/5))` | — |
| U-08 | Terminal size below 80×24 shows `"Terminal too small (need 80×24)"` instead of a corrupted layout | — |
| U-09 | `?` opens help overlay with full keybinding reference | — |
| U-10 | Input submission is blocked (not just warned) when message exceeds 300 chars | — |
| U-11 | `↓ N new message(s)` indicator appears when user is scrolled up and new messages arrive | — |
| U-12 | Auto-scroll to bottom on new message only when user is already at the bottom | — |

---

## 9. Phase 1 UX Requirements

| Req | Description | PRD ref |
|-----|-------------|---------|
| U-13 | `/` focuses a search input at the top of the sidebar; filters room list in real-time | R-35 |
| U-14 | Profile screen (`p`) shows username, join date, masked API key | R-31, R-34 |
| U-15 | Profile screen allows in-place username edit with same validation rules as registration | R-31 |
| U-16 | Member list panel (`m`) opens to the right, shows each member with online/away/offline indicator | R-39, R-40, R-41 |
| U-17 | Presence dot in sidebar reflects live activity in the room | R-40, R-41 |
| U-18 | Pending invites panel (`i`) shows sender name, room name, and `[Accept]` / `[Decline]` controls | R-36 |
| U-19 | Room creation modal gains a public/private toggle | R-37 |
| U-20 | Unread message count badge shown on room list items when not the active room | — |
| U-21 | Typing ellipsis animates (400ms cycle) | — |

---

## 10. Phase 2 UX Requirements

| Req | Description | PRD ref |
|-----|-------------|---------|
| U-22 | Message selection mode: `↑/↓` selects a message when viewport is focused | R-45, R-46, R-47, R-48 |
| U-23 | `e` on selected message opens inline edit; `Enter` saves, `Esc` cancels; `(edited)` appended after save | R-45 |
| U-24 | `d` on selected message shows `"Delete this message? [y/N]"` confirmation; on confirm, message replaced with tombstone | R-46 |
| U-25 | `r` on selected message opens emoji reaction picker; reactions rendered inline below message | R-47 |
| U-26 | `t` on selected message opens a thread sidebar on the right; thread follows same input/send model | R-48 |
| U-27 | `/find <query>` in the message input opens a search results overlay; results are selectable and jump to message | R-49 |
| U-28 | Markdown rendered in viewport: `**bold**`, `_italic_`, `` `code` ``, fenced code blocks, `[links](url)` underlined | R-50 |
| U-29 | `Shift+Enter` inserts a newline in the input; input box grows up to 5 lines before scrolling internally | — |

---

## 11. Open Questions

| # | Question |
|---|----------|
| Q1 | Should `chatatui init` be interactive (Bubble Tea wizard) or a simple `cobra` prompt loop? |
| Q2 | Should the member list replace the sidebar or open as an additional panel? |
| Q3 | Terminal bell / OS notification on mention — in scope for Phase 1? |
| Q4 | Should room search be server-side (API call) or client-side (filter fetched list)? |
