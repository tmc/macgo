// Package sandbox provides automatic initialization for macgo with app sandboxing.
//
// Import this package to automatically set up app sandboxing:
//
//	import (
//	    _ "github.com/tmc/misc/macgo/auto/sandbox"
//	)
//
// This enables app sandboxing which provides security isolation but limits
// file system access to user-selected files only. Perfect for apps that
// process user documents safely.
package sandbox

import (
	"fmt"
	"os"

	macgo "github.com/tmc/misc/macgo"
)

func init() {
	// Enable app sandbox - much simpler !
	if err := macgo.Request(macgo.Sandbox); err != nil {
		// Log the error for debugging, but allow the app to continue
		fmt.Fprintf(os.Stderr, "macgo/auto/sandbox: failed to request sandbox permissions: %v\n", err)
		return
	}
}
