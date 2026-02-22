#!/usr/bin/env bash
# gen_mixed_load.sh — Launches multiple log generators in parallel to simulate a busy environment.
# Usage: ./tests/gen_mixed_load.sh [-lps N] [-d SECONDS] [-p PORT]
# Env vars: PORT (default 4000), DURATION (seconds, default infinite), LPS (logs/sec, default 8)

set -euo pipefail

PORT="${PORT:-4000}"
DURATION="${DURATION:-0}"
LPS="${LPS:-8}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Parse command line flags
while [[ $# -gt 0 ]]; do
  case $1 in
    -lps|--lps)
      LPS="$2"
      shift 2
      ;;
    -d|--duration)
      DURATION="$2"
      shift 2
      ;;
    -p|--port)
      PORT="$2"
      shift 2
      ;;
    -h|--help)
      cat <<EOF
Usage: $0 [-lps N] [-d SECONDS] [-p PORT]

Options:
  -lps, --lps N         Target logs per second (default: 8)
  -d,   --duration N    Run for N seconds, 0=infinite (default: 0)
  -p,   --port N        TCP port to send to (default: 4000)
  -h,   --help          Show this help

Examples:
  $0                    # ~8 logs/sec, default
  $0 -lps 50            # ~50 logs/sec
  $0 -lps 100 -d 30     # ~100 logs/sec for 30 seconds
  $0 -lps 200 -p 5000   # ~200 logs/sec to port 5000
EOF
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      echo "Use -h for help" >&2
      exit 1
      ;;
  esac
done

# Distribute LPS across 4 generators with a 3:2:2:1 ratio (JSON:JSON:Text:Text)
# Using awk for integer rounding
read -r R_JSON1 R_JSON2 R_TEXT1 R_TEXT2 <<< "$(awk -v lps="$LPS" 'BEGIN {
  r1 = int(lps * 3 / 8 + 0.5)
  r2 = int(lps * 2 / 8 + 0.5)
  r3 = int(lps * 2 / 8 + 0.5)
  r4 = lps - r1 - r2 - r3
  if (r4 < 1) r4 = 1
  print r1, r2, r3, r4
}')"

TOTAL=$((R_JSON1 + R_JSON2 + R_TEXT1 + R_TEXT2))

PIDS=()

cleanup() {
  echo ""
  echo "Stopping all generators..."
  for pid in "${PIDS[@]}"; do
    kill "$pid" 2>/dev/null || true
  done
  wait 2>/dev/null
  echo "All generators stopped."
  exit 0
}
trap cleanup SIGINT SIGTERM

echo "=== Mixed Load Test ==="
echo "Port: ${PORT} | Duration: ${DURATION}s (0=infinite) | Target: ~${LPS} logs/sec"
echo "Launching generators..."
echo ""

# JSON logs
PORT="$PORT" RATE="$R_JSON1" DURATION="$DURATION" "$SCRIPT_DIR/gen_json_logs.sh" &
PIDS+=($!)
echo "[PID $!] JSON log generator (${R_JSON1}/sec)"

# Another JSON stream
PORT="$PORT" RATE="$R_JSON2" DURATION="$DURATION" "$SCRIPT_DIR/gen_json_logs.sh" &
PIDS+=($!)
echo "[PID $!] JSON log generator (${R_JSON2}/sec)"

# Text logs
PORT="$PORT" RATE="$R_TEXT1" DURATION="$DURATION" "$SCRIPT_DIR/gen_text_logs.sh" &
PIDS+=($!)
echo "[PID $!] Text log generator (${R_TEXT1}/sec)"

# Another text stream
PORT="$PORT" RATE="$R_TEXT2" DURATION="$DURATION" "$SCRIPT_DIR/gen_text_logs.sh" &
PIDS+=($!)
echo "[PID $!] Text log generator (${R_TEXT2}/sec)"

echo ""
echo "All generators running (~${TOTAL} logs/sec total). Press Ctrl+C to stop."
echo ""

# Wait for all children
wait
