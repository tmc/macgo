// Package all provides automatic initialization for macgo with common permissions.
//
// Import this package to automatically enable a comprehensive set of permissions:
//
//	import (
//	    _ "github.com/tmc/macgo/auto/all"
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
	"fmt"
	"os"

	macgo "github.com/tmc/macgo"
)

func init() {
	// Enable common permissions all at once - so clean !
	if err := macgo.Request(
		macgo.Files,
		macgo.Network,
		macgo.Camera,
		macgo.Microphone,
	); err != nil {
		// Log the error for debugging, but allow the app to continue
		fmt.Fprintf(os.Stderr, "macgo/auto/all: failed to request permissions: %v\n", err)
		return
	}
}
