// Package files provides automatic initialization for macgo v2 with file access.
//
// Import this package to automatically enable file system access:
//
//	import (
//	    _ "github.com/tmc/misc/macgo/v2/auto/files"
//	)
//
// This enables sandboxed file access where users can select files and folders
// for your app to access. Great for file processing utilities, editors, and
// document-based applications.
package files

import (
	macgo "github.com/tmc/misc/macgo/v2"
)

func init() {
	// Enable file access permissions - simplified in v2
	if err := macgo.Request(macgo.Files); err != nil {
		return
	}
}