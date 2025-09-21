// Package media provides automatic initialization for macgo with media permissions.
//
// Import this package to automatically enable media capture permissions:
//
//	import (
//	    _ "github.com/tmc/misc/macgo/auto/media"
//	)
//
// This enables:
// - Camera access for video capture
// - Microphone access for audio recording
//
// Perfect for media applications, video conferencing, and content creation
// tools that need camera and microphone access.
package media

import (
	"fmt"
	"os"

	macgo "github.com/tmc/misc/macgo"
)

func init() {
	// Enable media capture permissions
	if err := macgo.Request(
		macgo.Camera,
		macgo.Microphone,
	); err != nil {
		// Log the error for debugging, but allow the app to continue
		fmt.Fprintf(os.Stderr, "macgo/auto/media: failed to request media permissions: %v\n", err)
		return
	}
}
