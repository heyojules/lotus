#!/usr/bin/env bash
# gen_text_logs.sh — Compatibility wrapper that now emits OTEL log-record JSON to TCP.
# Usage: ./tests/gen_text_logs.sh
# Env vars: PORT (default 4000), RATE (logs/sec, default 5), DURATION (seconds, default infinite)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
exec "$SCRIPT_DIR/gen_json_logs.sh" "$@"
