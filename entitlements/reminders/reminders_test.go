package reminders

import (
	"testing"

	"github.com/tmc/misc/macgo"
	"github.com/tmc/misc/macgo/entitlements"
)

func TestRemindersPackageInit(t *testing.T) {
	// Test that the reminders entitlement is registered during package init
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
	entitlements.Register(entitlements.EntReminders, true)

	// Check if it's registered
	expectedEntitlement := macgo.Entitlement(entitlements.EntReminders)
	if val, ok := macgo.DefaultConfig.Entitlements[expectedEntitlement]; !ok || !val {
		t.Errorf("Expected reminders entitlement %s to be registered as true during package init", expectedEntitlement)
	}
}

func TestRemindersEntitlementValue(t *testing.T) {
	// Verify that the reminders entitlement constant has the correct value
	expected := "com.apple.security.personal-information.reminders"
	if string(entitlements.EntReminders) != expected {
		t.Errorf("Expected reminders entitlement to be %q, got %q", expected, string(entitlements.EntReminders))
	}
}

func TestRemindersEntitlementMatchesMainPackage(t *testing.T) {
	// Verify that our reminders entitlement matches the main package
	if string(entitlements.EntReminders) != string(macgo.EntReminders) {
		t.Errorf("Reminders entitlement mismatch: entitlements package %q != macgo package %q",
			entitlements.EntReminders, macgo.EntReminders)
	}
}

func TestPackageImportSideEffect(t *testing.T) {
	// Check that the reminders entitlement exists in the default config
	remindersEnt := macgo.Entitlement(entitlements.EntReminders)

	// The entitlement should be present and set to true
	if val, exists := macgo.DefaultConfig.Entitlements[remindersEnt]; !exists {
		t.Error("Reminders entitlement should be registered after package import")
	} else if !val {
		t.Error("Reminders entitlement should be set to true after package import")
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
	entitlements.Register(entitlements.EntReminders, true)
	entitlements.Register(entitlements.EntReminders, true)
	entitlements.Register(entitlements.EntReminders, true)

	// Should still work correctly
	remindersEnt := macgo.Entitlement(entitlements.EntReminders)
	if val, exists := macgo.DefaultConfig.Entitlements[remindersEnt]; !exists || !val {
		t.Error("Reminders entitlement should remain registered and true after multiple registrations")
	}
}

func TestDocumentationExample(t *testing.T) {
	// The documentation shows: import _ "github.com/tmc/misc/macgo/entitlements/reminders"
	// This should enable reminders access by registering the entitlement during init()

	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	// Reset config to simulate a fresh import
	macgo.DefaultConfig = macgo.NewConfig()

	// Simulate what init() does
	entitlements.Register(entitlements.EntReminders, true)

	// Verify the entitlement is registered
	remindersEnt := macgo.Entitlement(entitlements.EntReminders)
	if val, exists := macgo.DefaultConfig.Entitlements[remindersEnt]; !exists || !val {
		t.Error("Reminders entitlement should be registered and enabled after init simulation")
	}
}
