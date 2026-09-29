//go:build !darwin && !windows

package platform

import "os/exec"

// RevealPath opens the given path in the system file manager via xdg-open.
func RevealPath(path string) error {
	return exec.Command("xdg-open", path).Start()
}
