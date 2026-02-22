//go:build !darwin

package main

// runTrayStart calls the systray start function directly on non-Darwin platforms.
func runTrayStart(start func()) {
	start()
}
