#!/usr/bin/env bash
# Generates OTEL JSON log lines to stdout for testing the TUI.
# Usage: ./scripts/mockdata.sh | ./build/lotus
#   or:  ./scripts/mockdata.sh --stream   (continuous, ~5 lines/sec)
#   or:  ./scripts/mockdata.sh --stream --lps 50

set -euo pipefail

HOSTNAME_VAL="$(hostname)"

SERVICES=("api-gateway" "user-service" "payment-service" "auth-service" "notification-service")
SEVERITIES=("INFO" "INFO" "INFO" "INFO" "INFO" "WARN" "WARN" "ERROR" "ERROR" "FATAL" "DEBUG" "DEBUG" "TRACE")
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
  "Redis connection lost, reconnecting"
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
  "SQL plan sampled"
  "Cache TTL for key session:abc123 = 1800s"
  "gRPC channel state: READY"
)
MESSAGES_TRACE=(
  "Span entered"
  "Span exited"
  "Internal checkpoint reached"
)

ENDPOINTS=("/api/v2/users" "/api/v2/orders" "/api/v2/payments" "/api/v2/auth/login" "/api/v2/health" "/api/v2/products" "/api/v2/notifications" "/api/v2/search")
METHODS=("GET" "POST" "PUT" "DELETE" "PATCH")
ENVS=("production" "staging" "development")
REGIONS=("us-east-1" "us-west-2" "eu-west-1" "ap-southeast-1")
VERSIONS=("1.4.2" "1.5.0" "1.5.1" "2.0.0-beta.3")

pick() { local arr=("$@"); echo "${arr[$((RANDOM % ${#arr[@]}))]}"; }
rand_between() { echo $(( $1 + RANDOM % ($2 - $1 + 1) )); }

rand_hex() {
  local length="$1"
  local out=""
  while [ "${#out}" -lt "$length" ]; do
    out+="$(printf '%04x' "$RANDOM")"
  done
  echo "${out:0:length}"
}

severity_number() {
  case "$1" in
    TRACE) echo 1 ;;
    DEBUG) echo 5 ;;
    INFO) echo 9 ;;
    WARN) echo 13 ;;
    ERROR) echo 17 ;;
    FATAL) echo 21 ;;
    *) echo 9 ;;
  esac
}

emit_line() {
  local svc sev msg
  svc=$(pick "${SERVICES[@]}")
  sev=$(pick "${SEVERITIES[@]}")

  case "$sev" in
    TRACE) msg=$(pick "${MESSAGES_TRACE[@]}") ;;
    DEBUG) msg=$(pick "${MESSAGES_DEBUG[@]}") ;;
    WARN) msg=$(pick "${MESSAGES_WARN[@]}") ;;
    ERROR) msg=$(pick "${MESSAGES_ERROR[@]}") ;;
    FATAL) msg=$(pick "${MESSAGES_FATAL[@]}") ;;
    *) msg=$(pick "${MESSAGES_INFO[@]}") ;;
  esac

  local sec nano time_nano endpoint method status_code duration trace_id span_id env region version app sev_num
  sec="$(date +%s)"
  nano="$(printf '%09d' $((RANDOM * 1000000 % 1000000000)))"
  time_nano="${sec}${nano}"
  endpoint=$(pick "${ENDPOINTS[@]}")
  method=$(pick "${METHODS[@]}")
  duration=$(rand_between 1 800)
  trace_id=$(rand_hex 32)
  span_id=$(rand_hex 16)
  env=$(pick "${ENVS[@]}")
  region=$(pick "${REGIONS[@]}")
  version=$(pick "${VERSIONS[@]}")
  app="$svc"
  sev_num="$(severity_number "$sev")"

  if [ "$sev" = "ERROR" ] || [ "$sev" = "FATAL" ]; then
    status_code=$(pick 500 502 503)
  elif [ "$sev" = "WARN" ]; then
    status_code=$(pick 400 401 408 429)
  else
    status_code=$(pick 200 200 200 201 204 304)
  fi

  printf '{"timeUnixNano":"%s","severityNumber":%s,"severityText":"%s","body":{"stringValue":"%s"},"traceId":"%s","spanId":"%s","attributes":[{"key":"app","value":{"stringValue":"%s"}},{"key":"service.name","value":{"stringValue":"%s"}},{"key":"host.name","value":{"stringValue":"%s"}},{"key":"env","value":{"stringValue":"%s"}},{"key":"region","value":{"stringValue":"%s"}},{"key":"version","value":{"stringValue":"%s"}},{"key":"http.method","value":{"stringValue":"%s"}},{"key":"http.target","value":{"stringValue":"%s"}},{"key":"http.status_code","value":{"intValue":"%d"}},{"key":"duration.ms","value":{"doubleValue":%d}}]}' \
    "$time_nano" "$sev_num" "$sev" "$msg" "$trace_id" "$span_id" "$app" "$svc" "$HOSTNAME_VAL" "$env" "$region" "$version" "$method" "$endpoint" "$status_code" "$duration"
  printf '\n'
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
  echo "Streaming mock OTEL logs at ~${LPS}/sec (Ctrl+C to stop)..." >&2
  while true; do
    emit_line
    sleep "$SLEEP_INTERVAL"
  done
else
  for _ in $(seq 1 "$COUNT"); do
    emit_line
  done
fi
