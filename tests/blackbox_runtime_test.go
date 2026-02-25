package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type blackboxConfig struct {
	DBPath              string
	JournalPath         string
	JournalEnabled      bool
	InsertBatchSize     int
	InsertFlushInterval time.Duration
	InsertFlushQueue    int
}

type blackboxServer struct {
	cmd     *exec.Cmd
	apiAddr string
	tcpAddr string
	output  *bytes.Buffer
	exitCh  chan error
	exited  bool
	exitErr error
}

var (
	lotusBuildOnce sync.Once
	lotusBinPath   string
	lotusBuildErr  error
)

func TestBlackBox_ReplaysPreseededJournal(t *testing.T) {
	baseDir := t.TempDir()
	cfg := blackboxConfig{
		DBPath:              filepath.Join(baseDir, "lotus.duckdb"),
		JournalPath:         filepath.Join(baseDir, "ingest.journal"),
		JournalEnabled:      true,
		InsertBatchSize:     64,
		InsertFlushInterval: 20 * time.Millisecond,
		InsertFlushQueue:    32,
	}
	const total = 24
	seedJournalFixture(t, cfg.JournalPath, "preseed-app", total, 0)
	srv1 := startBlackboxServer(t, cfg)
	waitForAppCountHTTP(t, srv1.apiAddr, "preseed-app", total, 10*time.Second)
	srv1.Kill(t)
}

func TestBlackBox_ReplaySkipsCommittedPrefix(t *testing.T) {
	baseDir := t.TempDir()
	cfg := blackboxConfig{
		DBPath:              filepath.Join(baseDir, "lotus.duckdb"),
		JournalPath:         filepath.Join(baseDir, "ingest.journal"),
		JournalEnabled:      true,
		InsertBatchSize:     64,
		InsertFlushInterval: 20 * time.Millisecond,
		InsertFlushQueue:    32,
	}
	const total = 30
	const committed = 22
	seedJournalFixture(t, cfg.JournalPath, "partial-app", total, committed)
	srv1 := startBlackboxServer(t, cfg)
	waitForAppCountHTTP(t, srv1.apiAddr, "partial-app", total-committed, 10*time.Second)
	srv1.Kill(t)
}

func TestBlackBox_JournalToggleBehavior(t *testing.T) {
	baseDir := t.TempDir()
	enabledCfg := blackboxConfig{
		DBPath:              filepath.Join(baseDir, "lotus.duckdb"),
		JournalPath:         filepath.Join(baseDir, "ingest.journal"),
		JournalEnabled:      true,
		InsertBatchSize:     64,
		InsertFlushInterval: 20 * time.Millisecond,
		InsertFlushQueue:    32,
	}

	srv1 := startBlackboxServer(t, enabledCfg)
	lines := generateJSONBurst(80, "journal-on-app", "journal-on-svc")
	sendTCPLines(t, srv1.tcpAddr, lines)
	waitForAppCountHTTP(t, srv1.apiAddr, "journal-on-app", int64(len(lines)), 10*time.Second)
	waitForJournalLineCount(t, enabledCfg.JournalPath, len(lines), 10*time.Second)
	if _, err := os.Stat(enabledCfg.JournalPath + ".commit"); err != nil {
		t.Fatalf("expected commit file when journal is enabled: %v", err)
	}
	srv1.Kill(t)

	disabledCfg := blackboxConfig{
		DBPath:              filepath.Join(baseDir, "lotus-nojournal.duckdb"),
		JournalPath:         filepath.Join(baseDir, "ingest-disabled.journal"),
		JournalEnabled:      false,
		InsertBatchSize:     64,
		InsertFlushInterval: 20 * time.Millisecond,
		InsertFlushQueue:    32,
	}
	srv2 := startBlackboxServer(t, disabledCfg)
	lines = generateJSONBurst(40, "journal-off-app", "journal-off-svc")
	sendTCPLines(t, srv2.tcpAddr, lines)
	waitForAppCountHTTP(t, srv2.apiAddr, "journal-off-app", int64(len(lines)), 10*time.Second)
	srv2.Kill(t)
	if _, err := os.Stat(disabledCfg.JournalPath); !os.IsNotExist(err) {
		t.Fatalf("expected no journal file when journal is disabled; err=%v", err)
	}
}

func startBlackboxServer(t *testing.T, cfg blackboxConfig) *blackboxServer {
	t.Helper()

	repoRoot := findRepoRoot(t)
	apiPort := freeTCPPort(t)
	tcpPort := freeTCPPort(t)
	socketPath := filepath.Join(filepath.Dir(cfg.DBPath), fmt.Sprintf("lotus-%d.sock", time.Now().UnixNano()))

	configPath := filepath.Join(filepath.Dir(cfg.DBPath), fmt.Sprintf("config-%d.yml", time.Now().UnixNano()))
	configBody := fmt.Sprintf(`host: 127.0.0.1
tcp-enabled: true
tcp-port: %d
api-enabled: true
api-port: %d
db-path: %q
socket-path: %q
query-timeout: 5s
insert-batch-size: %d
insert-flush-interval: %s
insert-flush-queue-size: %d
journal-enabled: %t
journal-path: %q
backup-enabled: false
`, tcpPort, apiPort, cfg.DBPath, socketPath, cfg.InsertBatchSize, cfg.InsertFlushInterval.String(), cfg.InsertFlushQueue, cfg.JournalEnabled, cfg.JournalPath)
	if err := os.WriteFile(configPath, []byte(configBody), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var out bytes.Buffer
	cmd := exec.Command(lotusBinary(t), "--config", configPath)
	cmd.Dir = repoRoot
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Start(); err != nil {
		t.Fatalf("start lotus process: %v", err)
	}

	srv := &blackboxServer{
		cmd:     cmd,
		apiAddr: fmt.Sprintf("127.0.0.1:%d", apiPort),
		tcpAddr: fmt.Sprintf("127.0.0.1:%d", tcpPort),
		output:  &out,
		exitCh:  make(chan error, 1),
	}
	go func() {
		srv.exitCh <- cmd.Wait()
	}()

	waitEventually(t, 20*time.Second, 50*time.Millisecond, func() bool {
		if exited, err := srv.pollExited(); exited {
			t.Fatalf("lotus exited before ready: %v\n%s", err, srv.output.String())
		}
		resp, err := http.Get("http://" + srv.apiAddr + "/api/health")
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, "lotus api failed to become ready")

	t.Cleanup(func() {
		if exited, _ := srv.pollExited(); exited {
			return
		}
		_ = srv.cmd.Process.Kill()
		_, _ = srv.waitExited(3 * time.Second)
	})

	return srv
}

func lotusBinary(t *testing.T) string {
	t.Helper()
	lotusBuildOnce.Do(func() {
		repoRoot := findRepoRoot(t)
		tmpDir, err := os.MkdirTemp("", "lotus-blackbox-bin-*")
		if err != nil {
			lotusBuildErr = fmt.Errorf("mktemp bin dir: %w", err)
			return
		}
		lotusBinPath = filepath.Join(tmpDir, "lotus")

		cmd := exec.Command("go", "build", "-o", lotusBinPath, "./cmd/lotus")
		cmd.Dir = repoRoot
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Run(); err != nil {
			lotusBuildErr = fmt.Errorf("build lotus binary: %w\n%s", err, out.String())
			return
		}
	})
	if lotusBuildErr != nil {
		t.Fatalf("%v", lotusBuildErr)
	}
	return lotusBinPath
}

func (s *blackboxServer) Kill(t *testing.T) {
	t.Helper()
	if s.cmd.Process == nil {
		t.Fatalf("process not started")
	}
	if exited, _ := s.pollExited(); exited {
		return
	}
	if err := s.cmd.Process.Kill(); err != nil {
		t.Fatalf("kill process: %v", err)
	}
	if _, ok := s.waitExited(5 * time.Second); !ok {
		t.Fatalf("process did not exit after kill; output:\n%s", s.output.String())
	}
}

func (s *blackboxServer) pollExited() (bool, error) {
	if s.exited {
		return true, s.exitErr
	}
	select {
	case err := <-s.exitCh:
		s.exited = true
		s.exitErr = err
		return true, err
	default:
		return false, nil
	}
}

func (s *blackboxServer) waitExited(timeout time.Duration) (error, bool) {
	if s.exited {
		return s.exitErr, true
	}
	select {
	case err := <-s.exitCh:
		s.exited = true
		s.exitErr = err
		return err, true
	case <-time.After(timeout):
		return nil, false
	}
}

func appCountHTTP(t *testing.T, addr, app string) int64 {
	t.Helper()
	escaped := strings.ReplaceAll(app, "'", "''")
	code, resp, err := postSQL(addr, fmt.Sprintf("SELECT COUNT(*) AS c FROM logs WHERE app = '%s'", escaped))
	if err != nil || code != http.StatusOK || len(resp.Rows) != 1 {
		return -1
	}
	switch v := resp.Rows[0]["c"].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	default:
		return -1
	}
}

func waitForAppCountHTTP(t *testing.T, addr, app string, expected int64, timeout time.Duration) {
	t.Helper()
	waitEventually(t, timeout, 50*time.Millisecond, func() bool {
		return appCountHTTP(t, addr, app) == expected
	}, fmt.Sprintf("app=%s expected count=%d", app, expected))
}

func waitForJournalLineCount(t *testing.T, path string, expected int, timeout time.Duration) {
	t.Helper()
	waitEventually(t, timeout, 50*time.Millisecond, func() bool {
		data, err := os.ReadFile(path)
		if err != nil {
			return false
		}
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) == 1 && lines[0] == "" {
			return false
		}
		return len(lines) >= expected
	}, fmt.Sprintf("journal lines >= %d", expected))
}

func seedJournalFixture(t *testing.T, journalPath, app string, total int64, committed int64) {
	t.Helper()
	if total <= 0 {
		t.Fatalf("total must be > 0")
	}
	if committed < 0 || committed > total {
		t.Fatalf("invalid committed=%d for total=%d", committed, total)
	}

	if err := os.MkdirAll(filepath.Dir(journalPath), 0755); err != nil {
		t.Fatalf("mkdir journal dir: %v", err)
	}
	f, err := os.OpenFile(journalPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("open journal fixture: %v", err)
	}
	defer f.Close()

	baseTS := time.Date(2026, 2, 25, 0, 0, 0, 0, time.UTC)
	for i := int64(1); i <= total; i++ {
		entry := map[string]any{
			"seq": i,
			"record": map[string]any{
				"Timestamp":  baseTS.Add(time.Duration(i) * time.Second).Format(time.RFC3339Nano),
				"Level":      "INFO",
				"LevelNum":   30,
				"Message":    fmt.Sprintf("seed-%d", i),
				"RawLine":    fmt.Sprintf(`{"msg":"seed-%d"}`, i),
				"Service":    "seed-svc",
				"Hostname":   "seed-host",
				"PID":        1234,
				"Attributes": map[string]string{"seed": "true"},
				"Source":     "tcp",
				"App":        app,
				"EventID":    fmt.Sprintf("seed-%d", i),
			},
		}
		line, merr := json.Marshal(entry)
		if merr != nil {
			t.Fatalf("marshal journal entry: %v", merr)
		}
		if _, werr := f.Write(append(line, '\n')); werr != nil {
			t.Fatalf("write journal entry: %v", werr)
		}
	}
	if err := f.Sync(); err != nil {
		t.Fatalf("sync journal fixture: %v", err)
	}

	commitPath := journalPath + ".commit"
	if err := os.WriteFile(commitPath, []byte(strconv.FormatInt(committed, 10)+"\n"), 0644); err != nil {
		t.Fatalf("write commit fixture: %v", err)
	}
}

func freeTCPPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve tcp port: %v", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find repo root from %s", wd)
		}
		dir = parent
	}
}
