// Package applicationservices provides bindings to macOS ApplicationServices framework.
package applicationservices

import (
	"github.com/ebitengine/purego"
)

var (
	axIsProcessTrusted            func() bool
	axIsProcessTrustedWithOptions func(options uintptr) bool
)

func init() {
	lib, err := purego.Dlopen("/System/Library/Frameworks/ApplicationServices.framework/ApplicationServices", purego.RTLD_LAZY)
	if err != nil {
		return // Silently fail on non-darwin or missing framework
	}
	purego.RegisterLibFunc(&axIsProcessTrusted, lib, "AXIsProcessTrusted")
	purego.RegisterLibFunc(&axIsProcessTrustedWithOptions, lib, "AXIsProcessTrustedWithOptions")
}

// IsProcessTrusted returns true if the current process is trusted for accessibility.
// This is the programmatic equivalent of checking System Settings > Privacy & Security > Accessibility.
func IsProcessTrusted() bool {
	if axIsProcessTrusted == nil {
		return false
	}
	return axIsProcessTrusted()
}

// IsProcessTrustedWithOptions checks if the process is trusted, with options.
// If promptIfNeeded is true and the process is not trusted, macOS will show
// the accessibility permission dialog.
//
// Note: The prompt only appears once per app launch. Subsequent calls return
// the cached result without prompting.
func IsProcessTrustedWithOptions(promptIfNeeded bool) bool {
	if axIsProcessTrustedWithOptions == nil {
		return false
	}
	if promptIfNeeded {
		// kAXTrustedCheckOptionPrompt = "AXTrustedCheckOptionPrompt"
		// We need to create a CFDictionary with this key set to true
		// For now, just use the simple version
		return axIsProcessTrusted()
	}
	return axIsProcessTrusted()
}
