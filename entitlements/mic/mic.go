// Package mic provides microphone access entitlement for macOS apps.
// Import this package with the blank identifier to enable microphone access:
//
//	import _ "github.com/tmc/misc/macgo/entitlements/mic"
package mic

import (
	"github.com/tmc/misc/macgo"
	"github.com/tmc/misc/macgo/entitlement"
)

func init() {
	entitlement.EnableMicrophone()
	// Also sync with main package for compatibility
	macgo.RequestEntitlement(macgo.EntMicrophone)
}
