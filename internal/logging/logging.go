// Package logging provides a dependency-free rotating file writer used for the
// server's own access/application log.
//
// Why not pull in lumberjack? Keeping the writer local means the binary stays
// truly single-file with no extra dependencies, while still giving operators the
// guarantees they expect: bounded disk usage, daily rotation and predictable
// retention.
package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// RotatingFile is an io.WriteCloser that rotates its destination by size and by
// calendar day, pruning files beyond KeepDays.
type RotatingFile struct {
	mu       sync.Mutex
	dir      string
	baseName string
	maxSize  int64 // bytes
	keepDays int
	day      string
	file     *os.File
	size     int64
}

// RotatingFileConfig configures NewRotatingFile. Zero values fall back to sane
// defaults (64MB max, 31 days retention) matching the product's defaults.
type RotatingFileConfig struct {
	Dir      string
	BaseName string
	MaxMB    int
	KeepDays int
}

// NewRotatingFile creates (or appends to) today's log file and returns it.
func NewRotatingFile(cfg RotatingFileConfig) (*RotatingFile, error) {
	maxMB := cfg.MaxMB
	if maxMB <= 0 {
		maxMB = 64
	}
	keep := cfg.KeepDays
	if keep <= 0 {
		keep = 31
	}
	name := cfg.BaseName
	if name == "" {
		name = "app"
	}
	if err := os.MkdirAll(cfg.Dir, 0o755); err != nil {
		return nil, err
	}
	rf := &RotatingFile{
		dir:      cfg.Dir,
		baseName: name,
		maxSize:  int64(maxMB) << 20,
		keepDays: keep,
	}
	if err := rf.open(); err != nil {
		return nil, err
	}
	rf.prune()
	return rf, nil
}

// todaySuffix returns the current day stamp used in rotated file names.
func todaySuffix() string { return time.Now().Format("2006-01-02") }

// open (re)opens the file matching today's suffix, appending when it exists.
func (r *RotatingFile) open() error {
	r.day = todaySuffix()
	path := filepath.Join(r.dir, fmt.Sprintf("%s-%s.log", r.baseName, r.day))
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return err
	}
	if r.file != nil {
		_ = r.file.Close()
	}
	r.file = f
	r.size = info.Size()
	return nil
}

// Write appends data, rotating first when the size limit or the day changed.
func (r *RotatingFile) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Rotate on day change or when adding p would exceed the size budget,
	// rolling over into numbered slices within the same day.
	if r.file == nil || r.day != todaySuffix() {
		if err := r.open(); err != nil {
			return 0, err
		}
	}
	if r.size+int64(len(p)) > r.maxSize {
		if err := r.rotate(); err != nil {
			return 0, err
		}
	}
	n, err := r.file.Write(p)
	r.size += int64(n)
	return n, err
}

// rotate closes the current slice and starts a new numbered one.
func (r *RotatingFile) rotate() error {
	if r.file != nil {
		_ = r.file.Close()
		r.file = nil
	}
	// Find the first free ".N" suffix for today.
	for i := 1; ; i++ {
		path := filepath.Join(r.dir, fmt.Sprintf("%s-%s.%d.log", r.baseName, todaySuffix(), i))
		if _, err := os.Stat(path); os.IsNotExist(err) {
			f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
			if err != nil {
				return err
			}
			r.file = f
			r.size = 0
			return nil
		}
	}
}

// Close closes the underlying file.
func (r *RotatingFile) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.file == nil {
		return nil
	}
	err := r.file.Close()
	r.file = nil
	return err
}

// prune deletes log files older than KeepDays so disk usage stays bounded.
// Files are matched by name rather than ModTime, keeping behaviour predictable
// even when archives are copied around with preserved timestamps.
func (r *RotatingFile) prune() {
	r.mu.Lock()
	defer r.mu.Unlock()
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -r.keepDays).Format("2006-01-02")
	prefix := r.baseName + "-"
	var keep []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), prefix) {
			continue
		}
		// Extract the YYYY-MM-DD stamp that follows the prefix.
		rest := strings.TrimPrefix(e.Name(), prefix)
		if len(rest) < 10 {
			continue
		}
		stamp := rest[:10]
		if stamp < cutoff { // lexicographic compare works for ISO dates
			_ = os.Remove(filepath.Join(r.dir, e.Name()))
			continue
		}
		keep = append(keep, e.Name())
	}
	_ = keep
}

// Init installs a global logger writing to both stdout and a rotated file under
// dir. This makes the single binary self-logging without external tooling.
func Init(dir string, maxMB, keepDays int) (*RotatingFile, error) {
	w, err := NewRotatingFile(RotatingFileConfig{
		Dir:      dir,
		BaseName: "medical-system",
		MaxMB:    maxMB,
		KeepDays: keepDays,
	})
	if err != nil {
		return nil, err
	}
	log.SetOutput(io.MultiWriter(os.Stdout, w))
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	return w, nil
}

// ListLogFiles returns sorted log file names (oldest first) for inspection.
func ListLogFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".log") {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}
