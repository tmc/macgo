// Package sandbox provides automatic initialization for macgo v2 with app sandboxing.
//
// Import this package to automatically set up app sandboxing:
//
//	import (
//	    _ "github.com/tmc/misc/macgo/v2/auto/sandbox"
//	)
//
// This enables app sandboxing which provides security isolation but limits
// file system access to user-selected files only. Perfect for apps that
// process user documents safely.
package sandbox

import (
	macgo "github.com/tmc/misc/macgo/v2"
)

func init() {
	// Enable app sandbox - much simpler in v2!
	if err := macgo.Request(macgo.Sandbox); err != nil {
		return
	}
}