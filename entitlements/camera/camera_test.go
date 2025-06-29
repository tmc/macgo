package camera

import (
	"testing"

	"github.com/tmc/misc/macgo"
	"github.com/tmc/misc/macgo/entitlements"
)

func TestCameraPackageInit(t *testing.T) {
	// Test that the camera entitlement is registered during package init
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
	entitlements.Register(entitlements.EntCamera, true)

	// Check if it's registered
	expectedEntitlement := macgo.Entitlement(entitlements.EntCamera)
	if val, ok := macgo.DefaultConfig.Entitlements[expectedEntitlement]; !ok || !val {
		t.Errorf("Expected camera entitlement %s to be registered as true during package init", expectedEntitlement)
	}
}

func TestCameraEntitlementValue(t *testing.T) {
	// Verify that the camera entitlement constant has the correct value
	expected := "com.apple.security.device.camera"
	if string(entitlements.EntCamera) != expected {
		t.Errorf("Expected camera entitlement to be %q, got %q", expected, string(entitlements.EntCamera))
	}
}

func TestCameraEntitlementMatchesMainPackage(t *testing.T) {
	// Verify that our camera entitlement matches the main package
	if string(entitlements.EntCamera) != string(macgo.EntCamera) {
		t.Errorf("Camera entitlement mismatch: entitlements package %q != macgo package %q",
			entitlements.EntCamera, macgo.EntCamera)
	}
}

// TestPackageImportSideEffect tests that importing this package has the expected side effect
func TestPackageImportSideEffect(t *testing.T) {
	// Since the package init() function should have already run when this test package was imported,
	// we should be able to verify that the camera entitlement was registered.
	//
	// Note: This test demonstrates that the blank import pattern works as expected.

	// Check that the camera entitlement exists in the default config
	cameraEnt := macgo.Entitlement(entitlements.EntCamera)

	// The entitlement should be present and set to true
	if val, exists := macgo.DefaultConfig.Entitlements[cameraEnt]; !exists {
		t.Error("Camera entitlement should be registered after package import")
	} else if !val {
		t.Error("Camera entitlement should be set to true after package import")
	}
}

// TestMultipleImports tests that importing the package multiple times doesn't cause issues
func TestMultipleImports(t *testing.T) {
	// Create a backup of the original config
	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	// Reset config
	macgo.DefaultConfig = macgo.NewConfig()

	// Simulate multiple registrations (like multiple imports)
	entitlements.Register(entitlements.EntCamera, true)
	entitlements.Register(entitlements.EntCamera, true)
	entitlements.Register(entitlements.EntCamera, true)

	// Should still work correctly
	cameraEnt := macgo.Entitlement(entitlements.EntCamera)
	if val, exists := macgo.DefaultConfig.Entitlements[cameraEnt]; !exists || !val {
		t.Error("Camera entitlement should remain registered and true after multiple registrations")
	}
}

// TestDocumentationExample tests the example from the package documentation
func TestDocumentationExample(t *testing.T) {
	// The documentation shows: import _ "github.com/tmc/misc/macgo/entitlements/camera"
	// This should enable camera access by registering the entitlement during init()

	// Since we can't re-run init(), we test that the Register function works as expected
	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	// Reset config to simulate a fresh import
	macgo.DefaultConfig = macgo.NewConfig()

	// Simulate what init() does
	entitlements.Register(entitlements.EntCamera, true)

	// Verify the entitlement is registered
	cameraEnt := macgo.Entitlement(entitlements.EntCamera)
	if val, exists := macgo.DefaultConfig.Entitlements[cameraEnt]; !exists || !val {
		t.Error("Camera entitlement should be registered and enabled after init simulation")
	}
}
