//go:build linux

package updater

import (
	"debug/elf"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// MissingDependencyError blocks an update when the downloaded AppImage needs a
// system library that is not installed. The app must not replace itself with a
// binary that cannot start.
type MissingDependencyError struct {
	Library string
	Package string
	Hint    string
}

func (e *MissingDependencyError) Error() string {
	return fmt.Sprintf("missing system dependency %s; %s", e.Library, e.Hint)
}

// checkAppImageDependencies extracts the AppImage in a temp directory and
// verifies that every WebKitGTK library referenced by its main binary exists on
// this system. It is intentionally a pre-flight check: replacing the running
// AppImage with an unrunnable one would leave the user without a working app.
func checkAppImageDependencies(path string) error {
	libs, err := appImageNeededLibraries(path)
	if err != nil {
		return fmt.Errorf("cannot inspect update package dependencies: %w", err)
	}
	for _, lib := range libs {
		if !strings.Contains(lib, "libwebkit2gtk-") {
			continue
		}
		if systemHasLibrary(lib) {
			continue
		}
		pkg, hint := webkitPackageHint(lib)
		return &MissingDependencyError{Library: lib, Package: pkg, Hint: hint}
	}
	return nil
}

func appImageNeededLibraries(path string) ([]string, error) {
	tmp, err := os.MkdirTemp("", "prochub-appimage-check-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	if err := os.Chmod(path, 0o755); err != nil {
		return nil, err
	}
	cmd := exec.Command(path, "--appimage-extract")
	cmd.Dir = tmp
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("extract AppImage: %v: %s", err, strings.TrimSpace(string(out)))
	}

	root := filepath.Join(tmp, "squashfs-root")
	binary, err := findAppImageBinary(root)
	if err != nil {
		return nil, err
	}
	file, err := elf.Open(binary)
	if err != nil {
		return nil, fmt.Errorf("read ELF %s: %w", binary, err)
	}
	defer file.Close()
	return file.ImportedLibraries()
}

func findAppImageBinary(root string) (string, error) {
	// Normal Wails AppImage layout: squashfs-root/usr/bin/<name>.
	candidates, _ := filepath.Glob(filepath.Join(root, "usr", "bin", "*"))
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && info.Mode().IsRegular() {
			if strings.EqualFold(filepath.Base(candidate), "prochub") {
				return candidate, nil
			}
		}
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
			if f, err := elf.Open(candidate); err == nil {
				f.Close()
				return candidate, nil
			}
		}
	}
	return "", fmt.Errorf("no executable found in AppImage")
}

func systemHasLibrary(library string) bool {
	// ldconfig is the authoritative dynamic linker cache on most distros.
	for _, ldconfig := range []string{"ldconfig", "/sbin/ldconfig", "/usr/sbin/ldconfig"} {
		if out, err := exec.Command(ldconfig, "-p").Output(); err == nil {
			if strings.Contains(string(out), library) {
				return true
			}
			break
		}
	}
	// Fallback for minimal systems without ldconfig.
	patterns := []string{
		"/usr/lib/*/" + library,
		"/lib/*/" + library,
		"/usr/lib/" + library,
		"/lib/" + library,
		"/usr/local/lib/" + library,
	}
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		if len(matches) > 0 {
			return true
		}
	}
	return false
}

func webkitPackageHint(library string) (string, string) {
	if strings.Contains(library, "webkit2gtk-4.1") {
		return "libwebkit2gtk-4.1-0",
			"install the WebKitGTK 4.1 runtime (Debian/Ubuntu: sudo apt install libwebkit2gtk-4.1-0; Fedora: sudo dnf install webkit2gtk4.1)"
	}
	return "libwebkit2gtk-4.0-37",
		"install the WebKitGTK 4.0 runtime (Debian/Ubuntu: sudo apt install libwebkit2gtk-4.0-37; Fedora: sudo dnf install webkit2gtk4.0)"
}
