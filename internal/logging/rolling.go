package logging

import (
	"bufio"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Entry struct {
	Timestamp time.Time `json:"timestamp"`
	Stream    string    `json:"stream"`
	Line      string    `json:"line"`
}

type RollingStore struct {
	mu        sync.Mutex
	dir       string
	maxLines  int
	maxFiles  int
	filename  string
	lineCount int
}

func NewRollingStore(dir string, maxLines, maxFiles int) *RollingStore {
	return &RollingStore{
		dir:      dir,
		maxLines: maxLines,
		maxFiles: maxFiles,
	}
}

func (r *RollingStore) Append(entry Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.filename == "" {
		if err := os.MkdirAll(r.dir, 0o755); err != nil {
			return err
		}
		r.filename = filepath.Join(r.dir, time.Now().Format("20060102_150405")+".log")
	}

	file, err := os.OpenFile(r.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	_, err = writer.WriteString(entry.Timestamp.Format(time.RFC3339) + " " + entry.Stream + " " + entry.Line + "\n")
	if err != nil {
		return err
	}
	if err := writer.Flush(); err != nil {
		return err
	}

	r.lineCount++
	if r.lineCount >= r.maxLines {
		r.rotate()
	}
	return nil
}

func (r *RollingStore) rotate() {
	if r.filename == "" {
		return
	}
	r.filename = ""
	r.lineCount = 0
	_ = r.cleanup()
}

func (r *RollingStore) cleanup() error {
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		return err
	}
	if len(entries) <= r.maxFiles {
		return nil
	}

	type fileInfo struct {
		name string
		mod  time.Time
	}
	files := make([]fileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, fileInfo{name: entry.Name(), mod: info.ModTime()})
	}

	for len(files) > r.maxFiles {
		oldest := 0
		for i := 1; i < len(files); i++ {
			if files[i].mod.Before(files[oldest].mod) {
				oldest = i
			}
		}
		_ = os.Remove(filepath.Join(r.dir, files[oldest].name))
		files = append(files[:oldest], files[oldest+1:]...)
	}
	return nil
}
