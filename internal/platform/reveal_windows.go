//go:build windows

package platform

import "os/exec"

// RevealPath opens the given path in the system file manager (Explorer).
func RevealPath(path string) error {
	return exec.Command("explorer", path).Start()
}
