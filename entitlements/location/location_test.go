package location

import (
	"testing"

	"github.com/tmc/misc/macgo"
	"github.com/tmc/misc/macgo/entitlements"
)

func TestLocationPackageInit(t *testing.T) {
	// Test that the location entitlement is registered during package init
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
	macgo.RequestEntitlement(entitlements.EntLocation)

	// Check if it's registered
	expectedEntitlement := macgo.Entitlement(entitlements.EntLocation)
	if val, ok := macgo.DefaultConfig.Entitlements[expectedEntitlement]; !ok || !val {
		t.Errorf("Expected location entitlement %s to be registered as true during package init", expectedEntitlement)
	}
}

func TestLocationEntitlementValue(t *testing.T) {
	// Verify that the location entitlement constant has the correct value
	expected := "com.apple.security.personal-information.location"
	if string(entitlements.EntLocation) != expected {
		t.Errorf("Expected location entitlement to be %q, got %q", expected, string(entitlements.EntLocation))
	}
}

func TestLocationEntitlementMatchesMainPackage(t *testing.T) {
	// Verify that our location entitlement matches the main package
	if string(entitlements.EntLocation) != string(macgo.EntLocation) {
		t.Errorf("Location entitlement mismatch: entitlements package %q != macgo package %q",
			entitlements.EntLocation, macgo.EntLocation)
	}
}

func TestPackageImportSideEffect(t *testing.T) {
	// Check that the location entitlement exists in the default config
	locationEnt := macgo.Entitlement(entitlements.EntLocation)

	// The entitlement should be present and set to true
	if val, exists := macgo.DefaultConfig.Entitlements[locationEnt]; !exists {
		t.Error("Location entitlement should be registered after package import")
	} else if !val {
		t.Error("Location entitlement should be set to true after package import")
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
	macgo.RequestEntitlement(entitlements.EntLocation)
	macgo.RequestEntitlement(entitlements.EntLocation)
	macgo.RequestEntitlement(entitlements.EntLocation)

	// Should still work correctly
	locationEnt := macgo.Entitlement(entitlements.EntLocation)
	if val, exists := macgo.DefaultConfig.Entitlements[locationEnt]; !exists || !val {
		t.Error("Location entitlement should remain registered and true after multiple registrations")
	}
}

func TestDocumentationExample(t *testing.T) {
	// The documentation shows: import _ "github.com/tmc/misc/macgo/entitlements/location"
	// This should enable location access by registering the entitlement during init()

	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	// Reset config to simulate a fresh import
	macgo.DefaultConfig = macgo.NewConfig()

	// Simulate what init() does
	macgo.RequestEntitlement(entitlements.EntLocation)

	// Verify the entitlement is registered
	locationEnt := macgo.Entitlement(entitlements.EntLocation)
	if val, exists := macgo.DefaultConfig.Entitlements[locationEnt]; !exists || !val {
		t.Error("Location entitlement should be registered and enabled after init simulation")
	}
}
