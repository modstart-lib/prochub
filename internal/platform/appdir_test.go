package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDataDir(t *testing.T) {
	base := t.TempDir()
	// Set both variables so the test works on every OS.
	t.Setenv("HOME", base)
	t.Setenv("USERPROFILE", base)

	dir, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir() failed: %v", err)
	}
	if want := filepath.Join(base, appDirName); dir != want {
		t.Errorf("DataDir() = %q, want %q", dir, want)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("data dir was not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("data dir is not a directory")
	}
}

func TestMustDataDir(t *testing.T) {
	base := t.TempDir()
	t.Setenv("HOME", base)
	t.Setenv("USERPROFILE", base)

	dir := MustDataDir()
	if want := filepath.Join(base, appDirName); dir != want {
		t.Errorf("MustDataDir() = %q, want %q", dir, want)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("data dir was not created: %v", err)
	}
}
