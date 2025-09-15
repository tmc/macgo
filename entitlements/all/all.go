// Package all imports all macgo entitlements.
// Import this package with the blank identifier to enable all supported permissions:
//
//	import _ "github.com/tmc/misc/macgo/entitlements/all"
package all

import "github.com/tmc/misc/macgo/entitlement"

func init() {
	entitlement.EnableAllTCCPermissions()
}
