// Package camera provides camera access entitlement for macOS apps.
// Import this package with the blank identifier to enable camera access:
//
//	import _ "github.com/tmc/misc/macgo/entitlements/camera"
package camera

import (
	"github.com/tmc/misc/macgo"
	"github.com/tmc/misc/macgo/entitlement"
)

func init() {
	entitlement.EnableCamera()
	// Also sync with main package for compatibility
	macgo.RequestEntitlement(macgo.EntCamera)
}
