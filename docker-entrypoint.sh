#!/usr/bin/env bash
set -euo pipefail

# ---- Configurable defaults (override via environment) ----
: "${APP_PKG:=.}"                                # main package path (root of project)
: "${APP_BIN:=/tmp/app}"                         # compiled binary path inside container
: "${APP_CMD:=serve}"                            # subcommand to run (serve starts the server)
: "${DLV_ADDR:=0.0.0.0:2345}"                    # delve listen address
: "${AIR_TMPDIR:=/tmp/air}"                      # air temp dir
: "${AIR_EXCLUDE_DIRS:=vendor,.git,$AIR_TMPDIR}" # comma-separated
: "${AIR_DELAY_MS:=500}"                         # debounce rebuild delay

mkdir -p "$(dirname "$APP_BIN")" "$AIR_TMPDIR"

# Build and run commands
GO_BUILD_CMD="go build -buildvcs=false -gcflags='all=-N -l' -o ${APP_BIN} ${APP_PKG}"
DLV_RUN_CMD="dlv exec ${APP_BIN} --headless --listen=${DLV_ADDR} --api-version=2 --accept-multiclient --continue -- ${APP_CMD}"

# Generate a throwaway .air.toml each run (so env overrides Just Work™)
AIR_TOML="$(mktemp)"
cat >"$AIR_TOML" <<EOF
root = "."
tmp_dir = "${AIR_TMPDIR}"

[build]
cmd = "${GO_BUILD_CMD}"
bin = "${APP_BIN}"
# Run the freshly built binary through Delve with the serve subcommand
full_bin = "${DLV_RUN_CMD}"
delay = ${AIR_DELAY_MS}
include_ext = ["go", "toml"]
exclude_dir = [$(printf '"%s",' ${AIR_EXCLUDE_DIRS//,/ } | sed 's/,$//')]

[log]
time = true
EOF

echo "==> Starting chatatui development server"
echo "==> Air config:"
cat "$AIR_TOML"
echo

exec air -c "$AIR_TOML"
