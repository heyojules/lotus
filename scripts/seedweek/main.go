// seedweek inserts synthetic log records into the tiny-telemetry DuckDB,
// spread across the past 7 days with a realistic distribution.
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

var services = []string{"api-gateway", "user-service", "payment-service", "auth-service", "notification-service"}
var severities = []string{"INFO", "INFO", "INFO", "INFO", "WARN", "WARN", "ERROR", "DEBUG", "DEBUG", "TRACE", "FATAL"}
var apps = []string{"web-frontend", "mobile-backend", "batch-processor", "default"}

var messagesInfo = []string{
	"Request completed successfully",
	"Health check passed",
	"Cache hit for user profile",
	"Database connection pool: 12/20 active",
	"Processed batch of 150 events",
	"JWT token validated",
	"Rate limiter: 423/1000 requests used",
	"Scheduled job completed in 230ms",
	"Loaded configuration from environment",
	"Websocket connection established",
	"TLS handshake completed",
	"Downstream service responded in 45ms",
	"Session created for user u_8a3f2b",
	"Metric flush: 48 datapoints sent",
}

var messagesWarn = []string{
	"Slow query detected: SELECT * FROM orders took 2340ms",
	"Connection pool nearing capacity: 18/20",
	"Retry attempt 2/3 for upstream call",
	"Deprecated API endpoint /v1/users called",
	"Memory usage at 78% of limit",
	"Request timeout approaching: 4800ms of 5000ms",
	"Certificate expires in 14 days",
	"Disk usage on /data at 85%",
}

var messagesError = []string{
	"Failed to connect to database: connection refused",
	"Payment processing failed: insufficient funds",
	"Unhandled exception in request handler",
	"Redis connection lost, reconnecting",
	"External API returned 503: service unavailable",
	"Failed to parse request body: invalid JSON",
	"Authentication failed: invalid token signature",
	"Queue consumer lag exceeded threshold: 15000 messages",
}

var messagesFatal = []string{
	"Out of memory: killing process",
	"Cannot bind to port 8080: address already in use",
	"Database migration failed: schema version mismatch",
}

var messagesDebug = []string{
	"Resolving DNS for api.stripe.com",
	"Request headers: content-type=application/json",
	"SQL plan sampled",
	"Cache TTL for key session:abc123 = 1800s",
	"gRPC channel state: READY",
}

var messagesTrace = []string{
	"Span entered", "Span exited", "Internal checkpoint reached",
}

var endpoints = []string{"/api/v2/users", "/api/v2/orders", "/api/v2/payments", "/api/v2/health", "/api/v2/products"}
var methods = []string{"GET", "POST", "PUT", "DELETE"}
var envs = []string{"production", "staging", "development"}
var regions = []string{"us-east-1", "us-west-2", "eu-west-1"}

func pick(s []string) string { return s[rand.Intn(len(s))] }

func sevNum(sev string) int {
	switch sev {
	case "TRACE":
		return 1
	case "DEBUG":
		return 5
	case "INFO":
		return 9
	case "WARN":
		return 13
	case "ERROR":
		return 17
	case "FATAL":
		return 21
	}
	return 9
}

func msgForSev(sev string) string {
	switch sev {
	case "TRACE":
		return pick(messagesTrace)
	case "DEBUG":
		return pick(messagesDebug)
	case "WARN":
		return pick(messagesWarn)
	case "ERROR":
		return pick(messagesError)
	case "FATAL":
		return pick(messagesFatal)
	default:
		return pick(messagesInfo)
	}
}

func main() {
	home, _ := os.UserHomeDir()
	dbPath := filepath.Join(home, ".local/share/tiny-telemetry/tiny-telemetry.duckdb")

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open db: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Distribution: today = ~3000, yesterday = ~2000, day-2 = ~1500, etc.
	// Total ~11,500 logs across 7 days.
	type dayPlan struct {
		daysAgo int
		count   int
	}
	days := []dayPlan{
		{0, 3000},
		{1, 2000},
		{2, 1500},
		{3, 1200},
		{4, 1000},
		{5, 800},
		{6, 500},
	}

	now := time.Now()
	totalInserted := 0

	for _, dp := range days {
		// Spread logs across 8am-11pm for that day
		dayStart := time.Date(now.Year(), now.Month(), now.Day()-dp.daysAgo, 8, 0, 0, 0, now.Location())
		dayEnd := time.Date(now.Year(), now.Month(), now.Day()-dp.daysAgo, 23, 0, 0, 0, now.Location())

		// For today, cap at current time
		if dp.daysAgo == 0 && dayEnd.After(now) {
			dayEnd = now
		}

		span := dayEnd.Sub(dayStart)
		if span <= 0 {
			continue
		}

		fmt.Printf("Day -%d: inserting %d logs (%s to %s)...\n", dp.daysAgo, dp.count, dayStart.Format("Jan 02 15:04"), dayEnd.Format("15:04"))

		// Batch insert using a transaction
		tx, err := db.Begin()
		if err != nil {
			fmt.Fprintf(os.Stderr, "begin tx: %v\n", err)
			os.Exit(1)
		}

		stmt, err := tx.Prepare(`INSERT INTO logs (timestamp, orig_timestamp, level, level_num, message, raw_line, service, hostname, pid, attributes, source, app, event_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
		if err != nil {
			fmt.Fprintf(os.Stderr, "prepare: %v\n", err)
			os.Exit(1)
		}

		for i := 0; i < dp.count; i++ {
			// Random timestamp within the day window
			offset := time.Duration(rand.Int63n(int64(span)))
			ts := dayStart.Add(offset)

			sev := pick(severities)
			svc := pick(services)
			app := pick(apps)
			msg := msgForSev(sev)
			hostname := "node-" + fmt.Sprintf("%02d", rand.Intn(5)+1)

			attrs := map[string]string{
				"service.name":     svc,
				"host.name":        hostname,
				"env":              pick(envs),
				"region":           pick(regions),
				"http.method":      pick(methods),
				"http.target":      pick(endpoints),
				"http.status_code": fmt.Sprintf("%d", 200+rand.Intn(400)),
				"duration.ms":      fmt.Sprintf("%d", rand.Intn(800)+1),
			}
			attrsJSON, _ := json.Marshal(attrs)

			eventID := fmt.Sprintf("seed-%d-%d-%d", dp.daysAgo, i, rand.Int63())

			_, err := stmt.Exec(
				ts, ts, sev, sevNum(sev), msg, "",
				svc, hostname, 0, string(attrsJSON), "seed", app, eventID,
			)
			if err != nil {
				fmt.Fprintf(os.Stderr, "insert row %d: %v\n", i, err)
				tx.Rollback()
				os.Exit(1)
			}
		}

		stmt.Close()
		if err := tx.Commit(); err != nil {
			fmt.Fprintf(os.Stderr, "commit: %v\n", err)
			os.Exit(1)
		}

		totalInserted += dp.count
	}

	fmt.Printf("\nDone! Inserted %d logs across 7 days.\n", totalInserted)

	// Quick verification
	var count int64
	db.QueryRow("SELECT COUNT(*) FROM logs").Scan(&count)
	fmt.Printf("Total logs in DB: %d\n", count)
}
