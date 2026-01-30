//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>

void hideDockIcon() {
    dispatch_async(dispatch_get_main_queue(), ^{
        [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
    });
}

void showDockIcon() {
    dispatch_async(dispatch_get_main_queue(), ^{
        [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
        [NSApp activateIgnoringOtherApps:YES];
    });
}
*/
import "C"

// HideDockIcon hides the application from the Dock on macOS
func HideDockIcon() {
	C.hideDockIcon()
}

// ShowDockIcon shows the application in the Dock on macOS
func ShowDockIcon() {
	C.showDockIcon()
}
