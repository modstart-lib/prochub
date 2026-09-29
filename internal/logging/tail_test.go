package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeLogFile(t *testing.T, dir, name string, lines []string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	content := ""
	for _, line := range lines {
		content += line + "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write log file failed: %v", err)
	}
	// Spacing out modification times makes file ordering deterministic.
	mod := time.Now().Add(time.Duration(len(name)) * time.Second)
	_ = os.Chtimes(path, mod, mod)
	return path
}

func TestTailFiles(t *testing.T) {
	dir := t.TempDir()
	writeLogFile(t, dir, "a.log", []string{"old-1", "old-2"})
	writeLogFile(t, dir, "b.log", []string{"new-1", "new-2", "new-3"})

	lines, err := TailFiles(dir, 2)
	if err != nil {
		t.Fatalf("TailFiles failed: %v", err)
	}
	if len(lines) != 2 || lines[0] != "new-2" || lines[1] != "new-3" {
		t.Fatalf("unexpected tail: %v", lines)
	}

	// A limit larger than the total returns everything, oldest file first.
	all, err := TailFiles(dir, 100)
	if err != nil {
		t.Fatalf("TailFiles failed: %v", err)
	}
	if len(all) != 5 || all[0] != "old-1" || all[4] != "new-3" {
		t.Fatalf("unexpected full result: %v", all)
	}

	// A missing directory is not an error.
	empty, err := TailFiles(filepath.Join(dir, "missing"), 10)
	if err != nil || len(empty) != 0 {
		t.Fatalf("expected empty result for missing dir, got %v / %v", empty, err)
	}
}

func TestTailerFollow(t *testing.T) {
	dir := t.TempDir()
	path := writeLogFile(t, dir, "cur.log", []string{"first"})

	tailer := NewTailer(dir)
	if err := tailer.SeekEnd(); err != nil {
		t.Fatalf("SeekEnd failed: %v", err)
	}

	// Nothing new yet.
	lines, err := tailer.Next()
	if err != nil || len(lines) != 0 {
		t.Fatalf("expected no new lines, got %v / %v", lines, err)
	}

	// A partial line without a trailing newline is not emitted yet.
	appendToFile(t, path, "second")
	lines, err = tailer.Next()
	if err != nil || len(lines) != 0 {
		t.Fatalf("expected partial line to be withheld, got %v / %v", lines, err)
	}

	// Completing the line emits it.
	appendToFile(t, path, "\nthird\n")
	lines, err = tailer.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if len(lines) != 2 || lines[0] != "second" || lines[1] != "third" {
		t.Fatalf("unexpected lines: %v", lines)
	}
}

func appendToFile(t *testing.T, path, text string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatalf("open append failed: %v", err)
	}
	defer f.Close()
	if _, err := fmt.Fprint(f, text); err != nil {
		t.Fatalf("append failed: %v", err)
	}
}
