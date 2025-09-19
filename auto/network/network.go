// Package network provides automatic initialization for macgo v2 with network access.
//
// Import this package to automatically enable network permissions:
//
//	import (
//	    _ "github.com/tmc/misc/macgo/auto/network"
//	)
//
// This enables network access for both client and server operations when
// running in a sandbox. Essential for web servers, API clients, or any
// networked application.
package network

import (
	"fmt"
	"os"

	macgo "github.com/tmc/misc/macgo"
)

func init() {
	// Enable network access - unified permission in v2
	if err := macgo.Request(macgo.Network); err != nil {
		// Log the error for debugging, but allow the app to continue
		fmt.Fprintf(os.Stderr, "macgo/auto/network: failed to request network permissions: %v\n", err)
		return
	}
}
