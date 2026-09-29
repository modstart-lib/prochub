//go:build darwin

package platform

import "os/exec"

// RevealPath opens the given path in the system file manager (Finder).
func RevealPath(path string) error {
	return exec.Command("open", path).Start()
}
