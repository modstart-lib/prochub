//go:build linux

package updater

import (
	"errors"
	"os"
	"testing"
)

// TestCheckAppImageDependencies inspects a real AppImage when provided via
// PROCHUB_TEST_APPIMAGE. It asserts that the dependency probe either passes or
// reports a typed MissingDependencyError (never a crash or a silent failure).
func TestCheckAppImageDependencies(t *testing.T) {
	path := os.Getenv("PROCHUB_TEST_APPIMAGE")
	if path == "" {
		t.Skip("set PROCHUB_TEST_APPIMAGE to a real AppImage")
	}
	err := checkAppImageDependencies(path)
	if err == nil {
		return
	}
	var dep *MissingDependencyError
	if !errors.As(err, &dep) {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Logf("missing dependency: %s (%s)", dep.Library, dep.Package)
}
