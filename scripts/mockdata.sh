#!/usr/bin/env bash
# Generates pino-style NDJSON log lines to stdout for testing the TUI.
# Usage: ./scripts/mockdata.sh | ./build/lotus
#   or:  ./scripts/mockdata.sh --stream   (continuous, ~5 lines/sec)
#   or:  ./scripts/mockdata.sh --stream --lps 50

set -euo pipefail

HOSTNAME_VAL="$(hostname)"

SERVICES=("api-gateway" "user-service" "payment-service" "auth-service" "notification-service")
LEVELS=(30 30 30 30 30 40 40 50 50 60 20 20 10) # weighted: mostly info, some warn/error
MESSAGES_INFO=(
  "Request completed successfully"
  "Health check passed"
  "Cache hit for user profile"
  "Database connection pool: 12/20 active"
  "Processed batch of 150 events"
  "JWT token validated"
  "Rate limiter: 423/1000 requests used"
  "Scheduled job completed in 230ms"
  "Loaded configuration from environment"
  "Websocket connection established"
  "Served static asset /app.js in 3ms"
  "Session created for user u_8a3f2b"
  "Metric flush: 48 datapoints sent"
  "TLS handshake completed"
  "Downstream service responded in 45ms"
)
MESSAGES_WARN=(
  "Slow query detected: SELECT * FROM orders took 2340ms"
  "Connection pool nearing capacity: 18/20"
  "Retry attempt 2/3 for upstream call"
  "Deprecated API endpoint /v1/users called"
  "Memory usage at 78% of limit"
  "Request timeout approaching: 4800ms of 5000ms"
  "Certificate expires in 14 days"
  "Disk usage on /data at 85%"
)
MESSAGES_ERROR=(
  "Failed to connect to database: connection refused"
  "Payment processing failed: insufficient funds"
  "Unhandled exception in request handler"
  "Redis connection lost, reconnecting..."
  "External API returned 503: service unavailable"
  "Failed to parse request body: invalid JSON"
  "Authentication failed: invalid token signature"
  "Queue consumer lag exceeded threshold: 15000 messages"
)
MESSAGES_FATAL=(
  "Out of memory: killing process"
  "Cannot bind to port 8080: address already in use"
  "Database migration failed: schema version mismatch"
)
MESSAGES_DEBUG=(
  "Resolving DNS for api.stripe.com"
  "Request headers: content-type=application/json"
  "SQL: SELECT id, name FROM users WHERE active = true LIMIT 50"
  "Cache TTL for key session:abc123 = 1800s"
  "gRPC channel state: READY"
)

ENDPOINTS=("/api/v2/users" "/api/v2/orders" "/api/v2/payments" "/api/v2/auth/login" "/api/v2/health" "/api/v2/products" "/api/v2/notifications" "/api/v2/search")
METHODS=("GET" "POST" "PUT" "DELETE" "PATCH")
ENVS=("production" "staging" "development")
REGIONS=("us-east-1" "us-west-2" "eu-west-1" "ap-southeast-1")
VERSIONS=("1.4.2" "1.5.0" "1.5.1" "2.0.0-beta.3")
TRACE_IDS=()
for i in $(seq 1 20); do
  TRACE_IDS+=("$(printf '%032x' $((RANDOM * RANDOM + RANDOM)))")
done

pick() { local arr=("$@"); echo "${arr[$((RANDOM % ${#arr[@]}))]}" ; }
rand_between() { echo $(( $1 + RANDOM % ($2 - $1 + 1) )); }

emit_line() {
  local svc lvl_num msg
  svc=$(pick "${SERVICES[@]}")
  lvl_num=$(pick "${LEVELS[@]}")

  case $lvl_num in
    10) msg=$(pick "${MESSAGES_DEBUG[@]}") ;;
    20) msg=$(pick "${MESSAGES_DEBUG[@]}") ;;
    40) msg=$(pick "${MESSAGES_WARN[@]}") ;;
    50) msg=$(pick "${MESSAGES_ERROR[@]}") ;;
    60) msg=$(pick "${MESSAGES_FATAL[@]}") ;;
    *)  msg=$(pick "${MESSAGES_INFO[@]}") ;;
  esac

  # Pino uses unix epoch milliseconds for "time"
  local time_ms pid endpoint method status_code duration trace_id env region version
  time_ms="$(date +%s)$(printf '%03d' $((RANDOM % 1000)))"
  pid=$(rand_between 1000 9999)
  endpoint=$(pick "${ENDPOINTS[@]}")
  method=$(pick "${METHODS[@]}")
  duration=$(rand_between 1 800)
  trace_id=$(pick "${TRACE_IDS[@]}")
  env=$(pick "${ENVS[@]}")
  region=$(pick "${REGIONS[@]}")
  version=$(pick "${VERSIONS[@]}")

  if [ "$lvl_num" -ge 50 ]; then
    status_code=$(pick 500 502 503)
  elif [ "$lvl_num" -ge 40 ]; then
    status_code=$(pick 400 401 408 429)
  else
    status_code=$(pick 200 200 200 201 204 304)
  fi

  # Random subset of extra fields per line to keep it varied
  local extra=""
  (( RANDOM % 2 == 0 )) && extra="${extra},\"reqId\":\"req-$(rand_between 10000 99999)\""
  (( RANDOM % 2 == 0 )) && extra="${extra},\"userId\":\"usr-$(rand_between 1000 9999)\""
  (( RANDOM % 3 == 0 )) && extra="${extra},\"responseSize\":$(rand_between 128 65536)"
  (( RANDOM % 3 == 0 )) && extra="${extra},\"cacheHit\":$(pick true false)"
  (( RANDOM % 4 == 0 )) && extra="${extra},\"retryCount\":$(rand_between 0 3)"
  (( RANDOM % 4 == 0 )) && extra="${extra},\"queueDepth\":$(rand_between 0 500)"
  (( RANDOM % 3 == 0 )) && extra="${extra},\"correlationId\":\"$(printf '%08x-%04x-%04x' $((RANDOM*RANDOM)) $((RANDOM)) $((RANDOM)))\""

  printf '{"level":%d,"time":%s,"msg":"%s","pid":%d,"hostname":"%s","_app":"%s","env":"%s","region":"%s","version":"%s","method":"%s","path":"%s","statusCode":%d,"duration":%d,"traceId":"%s"%s}\n' \
    "$lvl_num" "$time_ms" "$msg" "$pid" "$HOSTNAME_VAL" "$svc" "$env" "$region" "$version" "$method" "$endpoint" "$status_code" "$duration" "$trace_id" "$extra"
}

STREAM=false
COUNT=500
LPS=5
for arg in "$@"; do
  case $arg in
    --stream) STREAM=true ;;
    --count=*) COUNT="${arg#*=}" ;;
    --lps=*) LPS="${arg#*=}" ;;
  esac
done

SLEEP_INTERVAL=$(awk "BEGIN {printf \"%.4f\", 1.0 / ${LPS}}")

if [ "$STREAM" = true ]; then
  echo "Streaming mock pino logs at ~${LPS}/sec (Ctrl+C to stop)..." >&2
  while true; do
    emit_line
    sleep "$SLEEP_INTERVAL"
  done
else
  for _ in $(seq 1 "$COUNT"); do
    emit_line
  done
fi
