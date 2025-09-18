// Package dev provides automatic initialization for macgo v2 with developer tool permissions.
//
// Import this package to automatically enable permissions for development tools:
//
//	import (
//	    _ "github.com/tmc/misc/macgo/v2/auto/dev"
//	)
//
// This enables:
// - File system access for reading/writing project files
// - Network access for downloading dependencies and API calls
//
// Perfect for development utilities, build tools, code generators,
// project analyzers, and IDE integrations.
package dev

import (
	macgo "github.com/tmc/misc/macgo/v2"
)

func init() {
	// Enable development tool permissions
	if err := macgo.Request(
		macgo.Files,
		macgo.Network,
	); err != nil {
		return
	}
}