//go:build !darwin

package platform

// runTrayStart calls the systray start function directly on non-Darwin platforms.
func runTrayStart(start func()) {
	start()
}
