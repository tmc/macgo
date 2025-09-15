package entitlements

import (
	"testing"
)

func TestEntitlementConstants(t *testing.T) {
	tests := []struct {
		name     string
		ent      Entitlement
		expected string
	}{
		{"App Sandbox", EntAppSandbox, "com.apple.security.app-sandbox"},
		{"Network Client", EntNetworkClient, "com.apple.security.network.client"},
		{"Network Server", EntNetworkServer, "com.apple.security.network.server"},
		{"Camera", EntCamera, "com.apple.security.device.camera"},
		{"Microphone", EntMicrophone, "com.apple.security.device.microphone"},
		{"Bluetooth", EntBluetooth, "com.apple.security.device.bluetooth"},
		{"USB", EntUSB, "com.apple.security.device.usb"},
		{"Audio Input", EntAudioInput, "com.apple.security.device.audio-input"},
		{"Print", EntPrint, "com.apple.security.print"},
		{"Address Book", EntAddressBook, "com.apple.security.personal-information.addressbook"},
		{"Location", EntLocation, "com.apple.security.personal-information.location"},
		{"Calendars", EntCalendars, "com.apple.security.personal-information.calendars"},
		{"Photos", EntPhotos, "com.apple.security.personal-information.photos-library"},
		{"Reminders", EntReminders, "com.apple.security.personal-information.reminders"},
		{"User Selected Read Only", EntUserSelectedReadOnly, "com.apple.security.files.user-selected.read-only"},
		{"User Selected Read Write", EntUserSelectedReadWrite, "com.apple.security.files.user-selected.read-write"},
		{"Allow JIT", EntAllowJIT, "com.apple.security.cs.allow-jit"},
		{"Allow Unsigned Executable Memory", EntAllowUnsignedExecutableMemory, "com.apple.security.cs.allow-unsigned-executable-memory"},
		{"Allow DYLD Env Vars", EntAllowDyldEnvVars, "com.apple.security.cs.allow-dyld-environment-variables"},
		{"Disable Library Validation", EntDisableLibraryValidation, "com.apple.security.cs.disable-library-validation"},
		{"Disable Executable Page Protection", EntDisableExecutablePageProtection, "com.apple.security.cs.disable-executable-page-protection"},
		{"Debugger", EntDebugger, "com.apple.security.cs.debugger"},
		{"Virtualization", EntVirtualization, "com.apple.security.virtualization"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.ent) != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, string(tt.ent))
			}
		})
	}
}

func TestPlaceholderFunctions(t *testing.T) {
	// Test that placeholder functions exist and don't panic
	tests := []struct {
		name string
		fn   func()
	}{
		{"SetAllTCCPermissions", SetAllTCCPermissions},
		{"SetCamera", SetCamera},
		{"SetMic", SetMic},
		{"SetLocation", SetLocation},
		{"SetContacts", SetContacts},
		{"SetPhotos", SetPhotos},
		{"SetCalendar", SetCalendar},
		{"SetReminders", SetReminders},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just ensure the function doesn't panic when called
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Function %s panicked: %v", tt.name, r)
				}
			}()
			tt.fn()
		})
	}
}

func TestEntitlementTypes(t *testing.T) {
	// Test that Entitlements map works correctly
	ents := make(Entitlements)
	ents[EntCamera] = true
	ents[EntMicrophone] = false

	if ents[EntCamera] != true {
		t.Error("Expected EntCamera to be true")
	}
	if ents[EntMicrophone] != false {
		t.Error("Expected EntMicrophone to be false")
	}
}

func TestEntitlementStringConversion(t *testing.T) {
	// Test that entitlements can be converted to/from strings
	testEnt := Entitlement("com.example.test")
	if string(testEnt) != "com.example.test" {
		t.Error("String conversion failed")
	}
}