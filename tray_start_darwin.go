//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation

#include <dispatch/dispatch.h>

extern void goTrayStartCallback(void);

static void dispatchTrayStartToMainThread(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        goTrayStartCallback();
    });
}
*/
import "C"

// pendingTrayStart holds the systray start function until the main thread picks it up.
var pendingTrayStart func()

//export goTrayStartCallback
func goTrayStartCallback() {
	if pendingTrayStart != nil {
		f := pendingTrayStart
		pendingTrayStart = nil
		f()
	}
}

// runTrayStart dispatches the systray start function to the macOS main thread.
// NSStatusBarWindow (created inside fyne.io/systray's nativeStart) must be
// instantiated on the main thread, so we use dispatch_async on the main queue.
func runTrayStart(start func()) {
	pendingTrayStart = start
	C.dispatchTrayStartToMainThread()
}
