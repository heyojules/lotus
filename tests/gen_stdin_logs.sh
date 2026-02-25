#!/usr/bin/env bash
# gen_stdin_logs.sh — Generates OTEL log-record JSON lines to stdout for piping into Lotus via stdin.
# Usage: ./tests/gen_stdin_logs.sh | ./lotus --config cmd/lotus/config.yml
# Env vars: RATE (logs/sec, default 5), DURATION (seconds, default infinite)

set -euo pipefail

RATE="${RATE:-5}"
DURATION="${DURATION:-0}"

ADJECTIVES=(blazing silent cosmic rapid frozen amber golden crimson velvet nimble hollow drifting molten steady rustic polished)
NOUNS=(falcon reef nebula piston glacier orchid beacon vortex condor sprocket tundra ember lotus cedar anvil)

HTTP_METHODS=(GET POST PUT DELETE PATCH)
HTTP_PATHS=(/api/users /api/orders /api/products /api/auth/login /api/health /api/search /api/payments /api/settings)
HTTP_CODES=(200 200 200 200 201 204 400 401 404 500 502 503)
SERVICES=(gateway auth-service user-service order-service payment-service search-service)

INFO_MESSAGES=(
  "Request completed successfully"
  "Cache hit for key"
  "Database query executed"
  "Connection pool status: healthy"
  "Background job enqueued"
  "Health check passed"
  "Session validated"
  "Metrics flushed"
  "Message published to queue"
)

WARN_MESSAGES=(
  "Slow query detected"
  "Connection pool near capacity"
  "Rate limit approaching threshold"
  "Deprecated API endpoint called"
  "Response time exceeded SLA"
)

ERROR_MESSAGES=(
  "Failed to connect to database"
  "Request timeout exceeded"
  "Authentication token expired"
  "Upstream service returned 503"
  "Circuit breaker tripped"
)

FATAL_MESSAGES=(
  "Unrecoverable database corruption detected"
  "Panic: nil pointer dereference"
)

# Pick 3-5 random app names
NUM_APPS=$((RANDOM % 3 + 3))
APPS=()
for ((i = 0; i < NUM_APPS; i++)); do
  adj="${ADJECTIVES[$((RANDOM % ${#ADJECTIVES[@]}))]}"
  noun="${NOUNS[$((RANDOM % ${#NOUNS[@]}))]}"
  APPS+=("${adj}-${noun}")
done

# Log setup info to stderr so it doesn't interfere with stdout piping
echo "Generating OTEL stdin logs (pipe into Lotus)" >&2
echo "Apps: ${APPS[*]}" >&2
echo "Rate: ${RATE}/sec | Duration: ${DURATION}s (0=infinite)" >&2
echo "Press Ctrl+C to stop" >&2

cleanup() {
  echo "" >&2
  echo "Stopped." >&2
  exit 0
}
trap cleanup SIGINT SIGTERM

rand_uuid() {
  printf '%04x%04x-%04x-%04x-%04x-%04x%04x%04x' \
    $((RANDOM)) $((RANDOM)) $((RANDOM)) \
    $(((RANDOM & 0x0fff) | 0x4000)) \
    $(((RANDOM & 0x3fff) | 0x8000)) \
    $((RANDOM)) $((RANDOM)) $((RANDOM))
}

SLEEP_INTERVAL=$(awk "BEGIN {printf \"%.4f\", 1.0 / ${RATE}}")
START_TIME=$(date +%s)

while true; do
  if [[ "$DURATION" -gt 0 ]]; then
    NOW=$(date +%s)
    if (( NOW - START_TIME >= DURATION )); then
      echo "Duration reached (${DURATION}s)." >&2
      break
    fi
  fi

  APP="${APPS[$((RANDOM % ${#APPS[@]}))]}"
  SERVICE="${SERVICES[$((RANDOM % ${#SERVICES[@]}))]}"
  REQUEST_ID=$(rand_uuid)
  USER_ID=$((RANDOM % 9000 + 1000))
  DURATION_MS=$((RANDOM % 2000 + 5))
  METHOD="${HTTP_METHODS[$((RANDOM % ${#HTTP_METHODS[@]}))]}"
  PATH_VAL="${HTTP_PATHS[$((RANDOM % ${#HTTP_PATHS[@]}))]}"
  STATUS="${HTTP_CODES[$((RANDOM % ${#HTTP_CODES[@]}))]}"
  TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  UNIX_TS=$(date +%s)

  # Weighted severity
  ROLL=$((RANDOM % 100))
  if (( ROLL < 70 )); then
    SEV="INFO"
    MSG="${INFO_MESSAGES[$((RANDOM % ${#INFO_MESSAGES[@]}))]}"
  elif (( ROLL < 85 )); then
    SEV="WARN"
    MSG="${WARN_MESSAGES[$((RANDOM % ${#WARN_MESSAGES[@]}))]}"
  elif (( ROLL < 95 )); then
    SEV="ERROR"
    MSG="${ERROR_MESSAGES[$((RANDOM % ${#ERROR_MESSAGES[@]}))]}"
  elif (( ROLL < 98 )); then
    SEV="DEBUG"
    MSG="Debug checkpoint: processing step $((RANDOM % 20 + 1))"
  else
    SEV="FATAL"
    MSG="${FATAL_MESSAGES[$((RANDOM % ${#FATAL_MESSAGES[@]}))]}"
  fi

  TIME_NANO=$(awk -v s="$UNIX_TS" 'BEGIN { printf "%.0f", s * 1000000000 }')
  echo "{\"timeUnixNano\":\"${TIME_NANO}\",\"severityText\":\"${SEV}\",\"body\":{\"stringValue\":\"${MSG}\"},\"attributes\":[{\"key\":\"app\",\"value\":{\"stringValue\":\"${APP}\"}},{\"key\":\"service.name\",\"value\":{\"stringValue\":\"${SERVICE}\"}},{\"key\":\"request.id\",\"value\":{\"stringValue\":\"${REQUEST_ID}\"}},{\"key\":\"user.id\",\"value\":{\"intValue\":\"${USER_ID}\"}},{\"key\":\"duration.ms\",\"value\":{\"doubleValue\":${DURATION_MS}}},{\"key\":\"http.method\",\"value\":{\"stringValue\":\"${METHOD}\"}},{\"key\":\"http.target\",\"value\":{\"stringValue\":\"${PATH_VAL}\"}},{\"key\":\"http.status_code\",\"value\":{\"intValue\":\"${STATUS}\"}},{\"key\":\"event.time.rfc3339\",\"value\":{\"stringValue\":\"${TS}\"}}]}"

  sleep "$SLEEP_INTERVAL"
done
