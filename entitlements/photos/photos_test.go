package photos

import (
	"testing"

	"github.com/tmc/misc/macgo"
	"github.com/tmc/misc/macgo/entitlements"
)

func TestPhotosPackageInit(t *testing.T) {
	// Test that the photos entitlement is registered during package init
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
	entitlements.Register(entitlements.EntPhotos, true)

	// Check if it's registered
	expectedEntitlement := macgo.Entitlement(entitlements.EntPhotos)
	if val, ok := macgo.DefaultConfig.Entitlements[expectedEntitlement]; !ok || !val {
		t.Errorf("Expected photos entitlement %s to be registered as true during package init", expectedEntitlement)
	}
}

func TestPhotosEntitlementValue(t *testing.T) {
	// Verify that the photos entitlement constant has the correct value
	expected := "com.apple.security.personal-information.photos-library"
	if string(entitlements.EntPhotos) != expected {
		t.Errorf("Expected photos entitlement to be %q, got %q", expected, string(entitlements.EntPhotos))
	}
}

func TestPhotosEntitlementMatchesMainPackage(t *testing.T) {
	// Verify that our photos entitlement matches the main package
	if string(entitlements.EntPhotos) != string(macgo.EntPhotos) {
		t.Errorf("Photos entitlement mismatch: entitlements package %q != macgo package %q",
			entitlements.EntPhotos, macgo.EntPhotos)
	}
}

func TestPackageImportSideEffect(t *testing.T) {
	// Check that the photos entitlement exists in the default config
	photosEnt := macgo.Entitlement(entitlements.EntPhotos)

	// The entitlement should be present and set to true
	if val, exists := macgo.DefaultConfig.Entitlements[photosEnt]; !exists {
		t.Error("Photos entitlement should be registered after package import")
	} else if !val {
		t.Error("Photos entitlement should be set to true after package import")
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
	entitlements.Register(entitlements.EntPhotos, true)
	entitlements.Register(entitlements.EntPhotos, true)
	entitlements.Register(entitlements.EntPhotos, true)

	// Should still work correctly
	photosEnt := macgo.Entitlement(entitlements.EntPhotos)
	if val, exists := macgo.DefaultConfig.Entitlements[photosEnt]; !exists || !val {
		t.Error("Photos entitlement should remain registered and true after multiple registrations")
	}
}

func TestDocumentationExample(t *testing.T) {
	// The documentation shows: import _ "github.com/tmc/misc/macgo/entitlements/photos"
	// This should enable photos access by registering the entitlement during init()

	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	// Reset config to simulate a fresh import
	macgo.DefaultConfig = macgo.NewConfig()

	// Simulate what init() does
	entitlements.Register(entitlements.EntPhotos, true)

	// Verify the entitlement is registered
	photosEnt := macgo.Entitlement(entitlements.EntPhotos)
	if val, exists := macgo.DefaultConfig.Entitlements[photosEnt]; !exists || !val {
		t.Error("Photos entitlement should be registered and enabled after init simulation")
	}
}
