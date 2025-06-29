package calendar

import (
	"testing"

	"github.com/tmc/misc/macgo"
	"github.com/tmc/misc/macgo/entitlements"
)

func TestCalendarPackageInit(t *testing.T) {
	// Test that the calendar entitlement is registered during package init
	// Since init() already ran when the package was imported, we simulate this
	// by creating a fresh config and calling the registration manually

	// Create a backup of the original config
	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	// Reset config to clean state
	macgo.DefaultConfig = macgo.NewConfig()

	// Simulate what init() does
	entitlements.Register(entitlements.EntCalendars, true)

	// Check if it's registered
	expectedEntitlement := macgo.Entitlement(entitlements.EntCalendars)
	if val, ok := macgo.DefaultConfig.Entitlements[expectedEntitlement]; !ok || !val {
		t.Errorf("Expected calendar entitlement %s to be registered as true during package init", expectedEntitlement)
	}
}

func TestCalendarEntitlementValue(t *testing.T) {
	// Verify that the calendar entitlement constant has the correct value
	expected := "com.apple.security.personal-information.calendars"
	if string(entitlements.EntCalendars) != expected {
		t.Errorf("Expected calendar entitlement to be %q, got %q", expected, string(entitlements.EntCalendars))
	}
}

func TestCalendarEntitlementMatchesMainPackage(t *testing.T) {
	// Verify that our calendar entitlement matches the main package
	if string(entitlements.EntCalendars) != string(macgo.EntCalendars) {
		t.Errorf("Calendar entitlement mismatch: entitlements package %q != macgo package %q",
			entitlements.EntCalendars, macgo.EntCalendars)
	}
}

func TestPackageImportSideEffect(t *testing.T) {
	// Check that the calendar entitlement exists in the default config
	calendarEnt := macgo.Entitlement(entitlements.EntCalendars)

	// The entitlement should be present and set to true
	if val, exists := macgo.DefaultConfig.Entitlements[calendarEnt]; !exists {
		t.Error("Calendar entitlement should be registered after package import")
	} else if !val {
		t.Error("Calendar entitlement should be set to true after package import")
	}
}

func TestMultipleImports(t *testing.T) {
	// Create a backup of the original config
	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	// Reset config
	macgo.DefaultConfig = macgo.NewConfig()

	// Simulate multiple registrations (like multiple imports)
	entitlements.Register(entitlements.EntCalendars, true)
	entitlements.Register(entitlements.EntCalendars, true)
	entitlements.Register(entitlements.EntCalendars, true)

	// Should still work correctly
	calendarEnt := macgo.Entitlement(entitlements.EntCalendars)
	if val, exists := macgo.DefaultConfig.Entitlements[calendarEnt]; !exists || !val {
		t.Error("Calendar entitlement should remain registered and true after multiple registrations")
	}
}

func TestDocumentationExample(t *testing.T) {
	// The documentation shows: import _ "github.com/tmc/misc/macgo/entitlements/calendar"
	// This should enable calendar access by registering the entitlement during init()

	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	// Reset config to simulate a fresh import
	macgo.DefaultConfig = macgo.NewConfig()

	// Simulate what init() does
	entitlements.Register(entitlements.EntCalendars, true)

	// Verify the entitlement is registered
	calendarEnt := macgo.Entitlement(entitlements.EntCalendars)
	if val, exists := macgo.DefaultConfig.Entitlements[calendarEnt]; !exists || !val {
		t.Error("Calendar entitlement should be registered and enabled after init simulation")
	}
}
