package all

import (
	"testing"

	"github.com/tmc/misc/macgo"
	"github.com/tmc/misc/macgo/entitlements"
)

func TestAllPackageImportsAllEntitlements(t *testing.T) {
	// The "all" package should import all the individual entitlement packages
	// This means all their init() functions should have run, registering all entitlements

	// Create a backup of the original config
	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	// Expected entitlements that should be registered by the "all" package
	expectedEntitlements := []macgo.Entitlement{
		macgo.Entitlement(entitlements.EntCamera),
		macgo.Entitlement(entitlements.EntMicrophone),
		macgo.Entitlement(entitlements.EntLocation),
		macgo.Entitlement(entitlements.EntAddressBook),
		macgo.Entitlement(entitlements.EntPhotos),
		macgo.Entitlement(entitlements.EntCalendars),
		macgo.Entitlement(entitlements.EntReminders),
	}

	// Check that each expected entitlement is registered
	for _, expectedEnt := range expectedEntitlements {
		if val, exists := macgo.DefaultConfig.Entitlements[expectedEnt]; !exists {
			t.Errorf("Expected entitlement %s to be registered by the 'all' package", expectedEnt)
		} else if !val {
			t.Errorf("Expected entitlement %s to be set to true by the 'all' package", expectedEnt)
		}
	}
}

func TestAllPackageDocumentation(t *testing.T) {
	// The documentation says: import _ "github.com/tmc/misc/macgo/entitlements/all"
	// This should enable all supported permissions

	// We can't re-test the init() function, but we can verify that all expected
	// entitlements are present in the configuration

	allTCCEntitlements := []macgo.Entitlement{
		macgo.Entitlement(entitlements.EntCamera),
		macgo.Entitlement(entitlements.EntMicrophone),
		macgo.Entitlement(entitlements.EntLocation),
		macgo.Entitlement(entitlements.EntAddressBook),
		macgo.Entitlement(entitlements.EntPhotos),
		macgo.Entitlement(entitlements.EntCalendars),
		macgo.Entitlement(entitlements.EntReminders),
	}

	for _, ent := range allTCCEntitlements {
		if val, exists := macgo.DefaultConfig.Entitlements[ent]; !exists || !val {
			t.Errorf("Expected TCC entitlement %s to be enabled by the 'all' package", ent)
		}
	}
}

func TestAllPackageImportsCorrectSubpackages(t *testing.T) {
	// This test verifies that the "all" package correctly imports the expected subpackages
	// by checking that their entitlements are registered

	// Mapping of expected subpackages to their entitlements
	subpackageEntitlements := map[string]macgo.Entitlement{
		"camera":    macgo.Entitlement(entitlements.EntCamera),
		"mic":       macgo.Entitlement(entitlements.EntMicrophone),
		"location":  macgo.Entitlement(entitlements.EntLocation),
		"contacts":  macgo.Entitlement(entitlements.EntAddressBook),
		"photos":    macgo.Entitlement(entitlements.EntPhotos),
		"calendar":  macgo.Entitlement(entitlements.EntCalendars),
		"reminders": macgo.Entitlement(entitlements.EntReminders),
	}

	for packageName, expectedEnt := range subpackageEntitlements {
		if val, exists := macgo.DefaultConfig.Entitlements[expectedEnt]; !exists {
			t.Errorf("Expected entitlement %s from subpackage %s to be registered", expectedEnt, packageName)
		} else if !val {
			t.Errorf("Expected entitlement %s from subpackage %s to be set to true", expectedEnt, packageName)
		}
	}
}

func TestAllPackageEntitlementConsistency(t *testing.T) {
	// Test that the entitlements registered by the "all" package are consistent
	// with what we expect from the individual subpackages

	// Get all registered entitlements
	registeredEntitlements := make(map[macgo.Entitlement]bool)
	for ent, val := range macgo.DefaultConfig.Entitlements {
		registeredEntitlements[ent] = val
	}

	// Define the expected TCC entitlements
	expectedTCCEntitlements := []macgo.Entitlement{
		macgo.Entitlement(entitlements.EntCamera),
		macgo.Entitlement(entitlements.EntMicrophone),
		macgo.Entitlement(entitlements.EntLocation),
		macgo.Entitlement(entitlements.EntAddressBook),
		macgo.Entitlement(entitlements.EntPhotos),
		macgo.Entitlement(entitlements.EntCalendars),
		macgo.Entitlement(entitlements.EntReminders),
	}

	// Check that all expected TCC entitlements are present and enabled
	for _, expectedEnt := range expectedTCCEntitlements {
		if val, exists := registeredEntitlements[expectedEnt]; !exists {
			t.Errorf("Expected TCC entitlement %s to be registered", expectedEnt)
		} else if !val {
			t.Errorf("Expected TCC entitlement %s to be enabled (true)", expectedEnt)
		}
	}
}

func TestAllPackageComparison(t *testing.T) {
	// Test that importing the "all" package gives the same result as calling
	// the convenience function SetAllTCCPermissions()

	// Create a backup of the original config
	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	// Get the current entitlements from the "all" package import
	allPackageEntitlements := make(map[macgo.Entitlement]bool)
	for ent, val := range macgo.DefaultConfig.Entitlements {
		allPackageEntitlements[ent] = val
	}

	// Reset config and use the convenience function
	macgo.DefaultConfig = macgo.NewConfig()
	entitlements.SetAllTCCPermissions()

	// Get the entitlements from the convenience function
	convenienceEntitlements := make(map[macgo.Entitlement]bool)
	for ent, val := range macgo.DefaultConfig.Entitlements {
		convenienceEntitlements[ent] = val
	}

	// Compare the two sets of entitlements
	for ent, val := range allPackageEntitlements {
		if convenienceVal, exists := convenienceEntitlements[ent]; !exists {
			t.Errorf("Entitlement %s from 'all' package not found in convenience function", ent)
		} else if val != convenienceVal {
			t.Errorf("Entitlement %s value mismatch: 'all' package=%v, convenience function=%v", ent, val, convenienceVal)
		}
	}

	// Check the reverse - all convenience function entitlements should be in the "all" package
	for ent, val := range convenienceEntitlements {
		if allVal, exists := allPackageEntitlements[ent]; !exists {
			t.Errorf("Entitlement %s from convenience function not found in 'all' package", ent)
		} else if val != allVal {
			t.Errorf("Entitlement %s value mismatch: convenience function=%v, 'all' package=%v", ent, val, allVal)
		}
	}
}

func TestAllPackageMultipleImports(t *testing.T) {
	// Test that importing the "all" package multiple times doesn't cause issues
	// This is mainly a sanity check since Go's import system handles this

	// Create a backup of the original config
	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	// Simulate multiple imports by calling the registration functions multiple times
	// (since we can't actually re-import the package in tests)
	entitlements.Register(entitlements.EntCamera, true)
	entitlements.Register(entitlements.EntMicrophone, true)
	entitlements.Register(entitlements.EntLocation, true)
	entitlements.Register(entitlements.EntAddressBook, true)
	entitlements.Register(entitlements.EntPhotos, true)
	entitlements.Register(entitlements.EntCalendars, true)
	entitlements.Register(entitlements.EntReminders, true)

	// Repeat the registrations
	entitlements.Register(entitlements.EntCamera, true)
	entitlements.Register(entitlements.EntMicrophone, true)
	entitlements.Register(entitlements.EntLocation, true)
	entitlements.Register(entitlements.EntAddressBook, true)
	entitlements.Register(entitlements.EntPhotos, true)
	entitlements.Register(entitlements.EntCalendars, true)
	entitlements.Register(entitlements.EntReminders, true)

	// All entitlements should still be correctly registered
	expectedEntitlements := []macgo.Entitlement{
		macgo.Entitlement(entitlements.EntCamera),
		macgo.Entitlement(entitlements.EntMicrophone),
		macgo.Entitlement(entitlements.EntLocation),
		macgo.Entitlement(entitlements.EntAddressBook),
		macgo.Entitlement(entitlements.EntPhotos),
		macgo.Entitlement(entitlements.EntCalendars),
		macgo.Entitlement(entitlements.EntReminders),
	}

	for _, ent := range expectedEntitlements {
		if val, exists := macgo.DefaultConfig.Entitlements[ent]; !exists || !val {
			t.Errorf("Expected entitlement %s to remain registered and enabled after multiple imports", ent)
		}
	}
}
