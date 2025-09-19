// Package auto provides automatic initialization for macgo v2 API.
//
// Import this package to automatically initialize macgo with no permissions:
//
//	import (
//	    _ "github.com/tmc/misc/macgo/auto"
//	)
//
// This is the simplest way to use macgo v2 - just import and your app will
// be properly bundled for macOS without any additional permissions.
//
// For apps that need specific permissions, use one of the specialized auto
// packages like auto/sandbox, auto/files, auto/camera, etc.
package auto

import (
	macgo "github.com/tmc/misc/macgo"
)

func init() {
	// Use the simplest possible v2 configuration - no permissions needed
	// This creates a proper macOS app bundle but requests no special permissions
	if err := macgo.Request(); err != nil {
		// In v2, we can handle errors gracefully instead of panicking
		// The app will still run, just without proper bundling
		return
	}
}
