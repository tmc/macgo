// Package photos provides photos library access entitlement for macOS apps.
// Import this package with the blank identifier to enable photos access:
//
//	import _ "github.com/tmc/misc/macgo/entitlements/photos"
package photos

import (
	"github.com/tmc/misc/macgo"
	"github.com/tmc/misc/macgo/entitlement"
)

func init() {
	entitlement.EnablePhotos()
	// Also sync with main package for compatibility
	macgo.RequestEntitlement(macgo.EntPhotos)
}
