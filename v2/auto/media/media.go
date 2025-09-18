// Package media provides automatic initialization for macgo v2 with media permissions.
//
// Import this package to automatically enable media capture permissions:
//
//	import (
//	    _ "github.com/tmc/misc/macgo/v2/auto/media"
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
	macgo "github.com/tmc/misc/macgo/v2"
)

func init() {
	// Enable media capture permissions
	if err := macgo.Request(
		macgo.Camera,
		macgo.Microphone,
	); err != nil {
		return
	}
}