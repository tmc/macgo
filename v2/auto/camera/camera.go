// Package camera provides automatic initialization for macgo v2 with camera access.
//
// Import this package to automatically enable camera permissions:
//
//	import (
//	    _ "github.com/tmc/misc/macgo/v2/auto/camera"
//	)
//
// This enables camera access which will prompt the user for permission when
// your app first tries to access the camera. Perfect for video recording,
// conferencing, or photo capture applications.
package camera

import (
	macgo "github.com/tmc/misc/macgo/v2"
)

func init() {
	// Enable camera access - much simpler than v1
	if err := macgo.Request(macgo.Camera); err != nil {
		return
	}
}