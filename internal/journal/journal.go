package journal

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/tinytelemetry/lotus/internal/model"
)

const (
	defaultFileMode = 0644
	defaultDirMode  = 0755
)

type entry struct {
	Seq    uint64          `json:"seq"`
	Record model.LogRecord `json:"record"`
}

// Journal provides a durable append-only log for ingested records.
// It stores one JSON entry per line and tracks commit progress in a sidecar file.
type Journal struct {
	mu         sync.Mutex
	path       string
	commitPath string
	file       *os.File
	nextSeq    uint64
	committed  uint64
}

// Open creates or opens a journal at path. On startup it compacts committed
// entries and ignores a partially written trailing line.
func Open(path string) (*Journal, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("journal: path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), defaultDirMode); err != nil {
		return nil, fmt.Errorf("journal: mkdir: %w", err)
	}

	commitPath := path + ".commit"
	committed, err := readCommitted(commitPath)
	if err != nil {
		return nil, err
	}

	maxSeq, err := compactCommitted(path, committed)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_RDWR, defaultFileMode)
	if err != nil {
		return nil, fmt.Errorf("journal: open: %w", err)
	}

	next := maxSeq + 1
	if committed+1 > next {
		next = committed + 1
	}

	return &Journal{
		path:       path,
		commitPath: commitPath,
		file:       f,
		nextSeq:    next,
		committed:  committed,
	}, nil
}

// Append persists one record and returns its sequence number.
func (j *Journal) Append(record *model.LogRecord) (uint64, error) {
	if record == nil {
		return 0, errors.New("journal: nil record")
	}

	j.mu.Lock()
	defer j.mu.Unlock()

	seq := j.nextSeq
	j.nextSeq++

	e := entry{
		Seq:    seq,
		Record: cloneRecord(record),
	}
	line, err := json.Marshal(e)
	if err != nil {
		return 0, fmt.Errorf("journal: marshal entry: %w", err)
	}
	line = append(line, '\n')

	if _, err := j.file.Write(line); err != nil {
		return 0, fmt.Errorf("journal: write entry: %w", err)
	}
	if err := j.file.Sync(); err != nil {
		return 0, fmt.Errorf("journal: sync entry: %w", err)
	}
	return seq, nil
}

// Commit marks all entries up to seq as committed.
func (j *Journal) Commit(seq uint64) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if seq <= j.committed {
		return nil
	}
	j.committed = seq
	if err := writeCommitted(j.commitPath, seq); err != nil {
		return err
	}
	return nil
}

// Committed returns the highest committed sequence number.
func (j *Journal) Committed() uint64 {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.committed
}

// Replay calls fn for each uncommitted entry in sequence order.
func (j *Journal) Replay(fn func(seq uint64, record *model.LogRecord) error) error {
	if fn == nil {
		return errors.New("journal: replay callback is nil")
	}

	j.mu.Lock()
	path := j.path
	committed := j.committed
	j.mu.Unlock()

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("journal: open for replay: %w", err)
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("journal: replay read: %w", err)
		}
		if len(line) == 0 {
			if errors.Is(err, io.EOF) {
				return nil
			}
			continue
		}
		if !strings.HasSuffix(string(line), "\n") {
			// Ignore a potentially partial trailing line.
			return nil
		}

		var e entry
		if uerr := json.Unmarshal(line, &e); uerr != nil {
			// Stop at first malformed line and keep replay deterministic.
			return nil
		}
		if e.Seq <= committed {
			if errors.Is(err, io.EOF) {
				return nil
			}
			continue
		}
		rec := e.Record
		if rerr := fn(e.Seq, &rec); rerr != nil {
			return rerr
		}

		if errors.Is(err, io.EOF) {
			return nil
		}
	}
}

// Close closes the underlying journal file.
func (j *Journal) Close() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.file == nil {
		return nil
	}
	err := j.file.Close()
	j.file = nil
	return err
}

func cloneRecord(r *model.LogRecord) model.LogRecord {
	out := *r
	if len(r.Attributes) == 0 {
		out.Attributes = map[string]string{}
		return out
	}
	attrs := make(map[string]string, len(r.Attributes))
	for k, v := range r.Attributes {
		attrs[k] = v
	}
	out.Attributes = attrs
	return out
}

func readCommitted(path string) (uint64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, fmt.Errorf("journal: read commit file: %w", err)
	}
	s := strings.TrimSpace(string(data))
	if s == "" {
		return 0, nil
	}
	seq, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("journal: parse commit seq: %w", err)
	}
	return seq, nil
}

func writeCommitted(path string, seq uint64) error {
	tmp := path + ".tmp"
	payload := []byte(strconv.FormatUint(seq, 10) + "\n")
	if err := os.WriteFile(tmp, payload, defaultFileMode); err != nil {
		return fmt.Errorf("journal: write commit tmp: %w", err)
	}

	f, err := os.OpenFile(tmp, os.O_RDWR, defaultFileMode)
	if err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("journal: open commit tmp: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("journal: sync commit tmp: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("journal: close commit tmp: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("journal: rename commit file: %w", err)
	}
	return nil
}

func compactCommitted(path string, committed uint64) (uint64, error) {
	src, err := os.OpenFile(path, os.O_CREATE|os.O_RDONLY, defaultFileMode)
	if err != nil {
		return 0, fmt.Errorf("journal: open source for compact: %w", err)
	}
	defer src.Close()

	tmpPath := path + ".compact"
	dst, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, defaultFileMode)
	if err != nil {
		return 0, fmt.Errorf("journal: open compact tmp: %w", err)
	}

	reader := bufio.NewReader(src)
	var maxSeq uint64

	for {
		line, rerr := reader.ReadBytes('\n')
		if rerr != nil && !errors.Is(rerr, io.EOF) {
			_ = dst.Close()
			_ = os.Remove(tmpPath)
			return 0, fmt.Errorf("journal: compact read: %w", rerr)
		}
		if len(line) == 0 {
			if errors.Is(rerr, io.EOF) {
				break
			}
			continue
		}
		if !strings.HasSuffix(string(line), "\n") {
			// Ignore potentially partial trailing line.
			break
		}

		var e entry
		if uerr := json.Unmarshal(line, &e); uerr != nil {
			break
		}
		if e.Seq > maxSeq {
			maxSeq = e.Seq
		}
		if e.Seq > committed {
			if _, werr := dst.Write(line); werr != nil {
				_ = dst.Close()
				_ = os.Remove(tmpPath)
				return 0, fmt.Errorf("journal: compact write: %w", werr)
			}
		}
		if errors.Is(rerr, io.EOF) {
			break
		}
	}

	if err := dst.Sync(); err != nil {
		_ = dst.Close()
		_ = os.Remove(tmpPath)
		return 0, fmt.Errorf("journal: compact sync: %w", err)
	}
	if err := dst.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return 0, fmt.Errorf("journal: compact close: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return 0, fmt.Errorf("journal: compact rename: %w", err)
	}
	return maxSeq, nil
}
