package mic

import (
	"testing"

	"github.com/tmc/misc/macgo"
	"github.com/tmc/misc/macgo/entitlements"
)

func TestMicPackageInit(t *testing.T) {
	// Test that the microphone entitlement is registered during package init
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
	entitlements.Register(entitlements.EntMicrophone, true)

	// Check if it's registered
	expectedEntitlement := macgo.Entitlement(entitlements.EntMicrophone)
	if val, ok := macgo.DefaultConfig.Entitlements[expectedEntitlement]; !ok || !val {
		t.Errorf("Expected microphone entitlement %s to be registered as true during package init", expectedEntitlement)
	}
}

func TestMicEntitlementValue(t *testing.T) {
	// Verify that the microphone entitlement constant has the correct value
	expected := "com.apple.security.device.microphone"
	if string(entitlements.EntMicrophone) != expected {
		t.Errorf("Expected microphone entitlement to be %q, got %q", expected, string(entitlements.EntMicrophone))
	}
}

func TestMicEntitlementMatchesMainPackage(t *testing.T) {
	// Verify that our microphone entitlement matches the main package
	if string(entitlements.EntMicrophone) != string(macgo.EntMicrophone) {
		t.Errorf("Microphone entitlement mismatch: entitlements package %q != macgo package %q", 
			entitlements.EntMicrophone, macgo.EntMicrophone)
	}
}

func TestPackageImportSideEffect(t *testing.T) {
	// Check that the microphone entitlement exists in the default config
	micEnt := macgo.Entitlement(entitlements.EntMicrophone)
	
	// The entitlement should be present and set to true
	if val, exists := macgo.DefaultConfig.Entitlements[micEnt]; !exists {
		t.Error("Microphone entitlement should be registered after package import")
	} else if !val {
		t.Error("Microphone entitlement should be set to true after package import")
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
	entitlements.Register(entitlements.EntMicrophone, true)
	entitlements.Register(entitlements.EntMicrophone, true)
	entitlements.Register(entitlements.EntMicrophone, true)

	// Should still work correctly
	micEnt := macgo.Entitlement(entitlements.EntMicrophone)
	if val, exists := macgo.DefaultConfig.Entitlements[micEnt]; !exists || !val {
		t.Error("Microphone entitlement should remain registered and true after multiple registrations")
	}
}

func TestDocumentationExample(t *testing.T) {
	// The documentation shows: import _ "github.com/tmc/misc/macgo/entitlements/mic"
	// This should enable microphone access by registering the entitlement during init()
	
	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	// Reset config to simulate a fresh import
	macgo.DefaultConfig = macgo.NewConfig()

	// Simulate what init() does
	entitlements.Register(entitlements.EntMicrophone, true)

	// Verify the entitlement is registered
	micEnt := macgo.Entitlement(entitlements.EntMicrophone)
	if val, exists := macgo.DefaultConfig.Entitlements[micEnt]; !exists || !val {
		t.Error("Microphone entitlement should be registered and enabled after init simulation")
	}
}