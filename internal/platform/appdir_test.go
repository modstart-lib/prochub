package platform

import (
	"os"
	"path/filepath"
	"testing"
)

// setupEnv points HOME/USERPROFILE at base and clears the PROCHUB_DATA_ROOT
// override so each test starts from a clean environment.
func setupEnv(t *testing.T, base string) {
	t.Helper()
	t.Setenv("HOME", base)
	t.Setenv("USERPROFILE", base)
	t.Setenv("PROCHUB_DATA_ROOT", "")
}

// writeTestClientConfig writes a client.json at path (creating its directory).
func writeTestClientConfig(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for client.json failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write client.json failed: %v", err)
	}
}

func TestDataDir(t *testing.T) {
	base := t.TempDir()
	setupEnv(t, base)

	dir, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir() failed: %v", err)
	}
	if want := filepath.Join(base, appDirName, dataDirName); dir != want {
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
	setupEnv(t, base)

	dir := MustDataDir()
	if want := filepath.Join(base, appDirName, dataDirName); dir != want {
		t.Errorf("MustDataDir() = %q, want %q", dir, want)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("data dir was not created: %v", err)
	}
}

func TestDataDirEnvOverride(t *testing.T) {
	base := t.TempDir()
	setupEnv(t, base)
	envRoot := filepath.Join(base, "custom-data")
	t.Setenv("PROCHUB_DATA_ROOT", envRoot)

	dir, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir() failed: %v", err)
	}
	if dir != envRoot {
		t.Errorf("DataDir() = %q, want %q", dir, envRoot)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("override data dir was not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("override data dir is not a directory")
	}
}

func TestDataDirEnvTilde(t *testing.T) {
	base := t.TempDir()
	setupEnv(t, base)
	t.Setenv("PROCHUB_DATA_ROOT", "~/custom-data")

	dir, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir() failed: %v", err)
	}
	if want := filepath.Join(base, "custom-data"); dir != want {
		t.Errorf("DataDir() = %q, want %q", dir, want)
	}
}

func TestDataDirEnvBlankFallsBack(t *testing.T) {
	base := t.TempDir()
	setupEnv(t, base)
	t.Setenv("PROCHUB_DATA_ROOT", "   ")

	dir, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir() failed: %v", err)
	}
	if want := filepath.Join(base, appDirName, dataDirName); dir != want {
		t.Errorf("DataDir() = %q, want %q", dir, want)
	}
}

func TestDataDirClientConfig(t *testing.T) {
	base := t.TempDir()
	setupEnv(t, base)
	writeTestClientConfig(t, filepath.Join(base, appDirName, clientConfigName), `{"dataRoot": "~/custom-client-data"}`)

	dir, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir() failed: %v", err)
	}
	if want := filepath.Join(base, "custom-client-data"); dir != want {
		t.Errorf("DataDir() = %q, want %q", dir, want)
	}
}

func TestDataDirClientConfigMissing(t *testing.T) {
	base := t.TempDir()
	setupEnv(t, base)

	dir, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir() failed: %v", err)
	}
	if want := filepath.Join(base, appDirName, dataDirName); dir != want {
		t.Errorf("DataDir() = %q, want %q", dir, want)
	}
	path := filepath.Join(base, appDirName, clientConfigName)
	if _, err := os.Stat(path); err != nil {
		t.Errorf("client.json was not created: %v", err)
	}
}

func TestDataDirClientConfigBlankFallsBack(t *testing.T) {
	base := t.TempDir()
	setupEnv(t, base)
	writeTestClientConfig(t, filepath.Join(base, appDirName, clientConfigName), `{"dataRoot": "   "}`)

	dir, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir() failed: %v", err)
	}
	if want := filepath.Join(base, appDirName, dataDirName); dir != want {
		t.Errorf("DataDir() = %q, want %q", dir, want)
	}
}

func TestDataDirEnvOverridesClientConfig(t *testing.T) {
	base := t.TempDir()
	setupEnv(t, base)
	writeTestClientConfig(t, filepath.Join(base, appDirName, clientConfigName), `{"dataRoot": "~/client-data"}`)
	t.Setenv("PROCHUB_DATA_ROOT", "~/env-data")

	dir, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir() failed: %v", err)
	}
	if want := filepath.Join(base, "env-data"); dir != want {
		t.Errorf("DataDir() = %q, want %q", dir, want)
	}
}
