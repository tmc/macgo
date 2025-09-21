// Package files provides automatic initialization for macgo with file access.
//
// Import this package to automatically enable file system access:
//
//	import (
//	    _ "github.com/tmc/misc/macgo/auto/files"
//	)
//
// This enables sandboxed file access where users can select files and folders
// for your app to access. Great for file processing utilities, editors, and
// document-based applications.
package files

import (
	"fmt"
	"os"

	macgo "github.com/tmc/misc/macgo"
)

func init() {
	// Enable file access permissions - simplified in v2
	if err := macgo.Request(macgo.Files); err != nil {
		// Log the error for debugging, but allow the app to continue
		fmt.Fprintf(os.Stderr, "macgo/auto/files: failed to request file permissions: %v\n", err)
		return
	}
}
