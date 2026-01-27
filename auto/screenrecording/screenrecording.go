// Package screenrecording provides automatic initialization for macgo with screen recording permission.
//
// Import this package to automatically enable screen recording permission:
//
//	import (
//	    _ "github.com/tmc/macgo/auto/screenrecording"
//	)
//
// This enables:
// - Screen recording/capture access via ScreenCaptureKit or CGWindowListCreateImage
//
// Perfect for screen sharing, recording, and browser automation applications
// that need to capture screen content.
package screenrecording

import (
	"fmt"
	"os"

	macgo "github.com/tmc/macgo"
)

func init() {
	// Enable screen recording permission
	if err := macgo.Request(
		macgo.ScreenRecording,
	); err != nil {
		// Log the error for debugging, but allow the app to continue
		fmt.Fprintf(os.Stderr, "macgo/auto/screenrecording: failed to request screen recording permission: %v\n", err)
		return
	}
}
