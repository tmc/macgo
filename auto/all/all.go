// Package all provides automatic initialization for macgo v2 with common permissions.
//
// Import this package to automatically enable a comprehensive set of permissions:
//
//	import (
//	    _ "github.com/tmc/misc/macgo/auto/all"
//	)
//
// This enables:
// - File system access (user-selected files and folders)
// - Network access (both client and server)
// - Camera access
// - Microphone access
//
// Perfect for multimedia applications that need broad system access.
// For more targeted permissions, use the specific auto packages instead.
package all

import (
	macgo "github.com/tmc/misc/macgo"
)

func init() {
	// Enable common permissions all at once - so clean in v2!
	if err := macgo.Request(
		macgo.Files,
		macgo.Network,
		macgo.Camera,
		macgo.Microphone,
	); err != nil {
		return
	}
}
