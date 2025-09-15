// Package calendar provides calendar access entitlement for macOS apps.
// Import this package with the blank identifier to enable calendar access:
//
//	import _ "github.com/tmc/misc/macgo/entitlements/calendar"
package calendar

import (
	"github.com/tmc/misc/macgo"
	"github.com/tmc/misc/macgo/entitlement"
)

func init() {
	entitlement.EnableCalendars()
	// Also sync with main package for compatibility
	macgo.RequestEntitlement(macgo.EntCalendars)
}
