// Package all imports all macgo entitlements.
// Import this package with the blank identifier to enable all supported permissions:
//
//	import _ "github.com/tmc/misc/macgo/entitlements/all"
package all

import (
	"github.com/tmc/misc/macgo"
	"github.com/tmc/misc/macgo/entitlement"

	// Import all entitlement subpackages to trigger their init() functions
	_ "github.com/tmc/misc/macgo/entitlements/calendar"
	_ "github.com/tmc/misc/macgo/entitlements/camera"
	_ "github.com/tmc/misc/macgo/entitlements/contacts"
	_ "github.com/tmc/misc/macgo/entitlements/location"
	_ "github.com/tmc/misc/macgo/entitlements/mic"
	_ "github.com/tmc/misc/macgo/entitlements/photos"
	_ "github.com/tmc/misc/macgo/entitlements/reminders"
)

func init() {
	entitlement.EnableAllTCCPermissions()
	// Also sync with main package for compatibility
	macgo.RequestEntitlements(
		macgo.EntCamera,
		macgo.EntMicrophone,
		macgo.EntLocation,
		macgo.EntAddressBook,
		macgo.EntPhotos,
		macgo.EntCalendars,
		macgo.EntReminders,
	)
}
