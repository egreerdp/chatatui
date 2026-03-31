# Testing Across Devices with ngrok

## Prerequisites

- [ngrok](https://ngrok.com/download) installed on the host machine
- The `chatatui` binary built on each client machine (`mise run build`)
- A running PostgreSQL instance accessible to the server

## Setup

### 1. Start the server

On the host machine:

```bash
mise run dev serve
```

The server listens on `:8080` by default.

### 2. Expose the server with ngrok

```bash
ngrok http 8080
```

Note the HTTPS URL from the output, e.g. `https://abc123.ngrok-free.app`.

### 3. Register a user for each participant

Each person needs their own API key. Run this once per user (substituting their name):

```bash
curl -s -X POST https://abc123.ngrok-free.app/register \
  -H "Content-Type: application/json" \
  -d '{"name": "alice"}' | jq .
```

Save the returned `api_key`.

### 4. Configure each client

On each machine, create or edit `~/.chatatui.toml`:

```toml
host    = "https://abc123.ngrok-free.app"
api_key = "<key from step 3>"
```

### 5. Run the client

```bash
mise run dev
```

Or run the binary directly if `mise` is not available on the remote machine:

```bash
./chatatui
```

## Notes

- The ngrok URL changes each time you restart ngrok (on the free tier). You will need to update `~/.chatatui.toml` on all client machines when this happens.
- ngrok's free tier displays a browser interstitial for the first HTTP request. This does not affect the TUI client.
- Registered users and rooms persist across server restarts. You only need to register once per user.
