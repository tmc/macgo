package entitlements

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/tmc/misc/macgo"
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

func TestRegisterFunction(t *testing.T) {
	// Create a backup of the original config
	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	// Reset config for testing
	macgo.DefaultConfig = macgo.NewConfig()

	tests := []struct {
		name        string
		entitlement Entitlement
		value       bool
		expected    bool
	}{
		{"Register Camera True", EntCamera, true, true},
		{"Register Microphone False", EntMicrophone, false, false},
		{"Register Location True", EntLocation, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear existing entitlements
			macgo.DefaultConfig.Entitlements = make(map[macgo.Entitlement]bool)

			Register(tt.entitlement, tt.value)

			if tt.value {
				// When value is true, the entitlement should be present and true
				if val, ok := macgo.DefaultConfig.Entitlements[macgo.Entitlement(tt.entitlement)]; !ok || !val {
					t.Errorf("Expected entitlement %s to be registered as true, got %v", tt.entitlement, val)
				}
			} else {
				// When value is false, the entitlement should not be registered
				if _, ok := macgo.DefaultConfig.Entitlements[macgo.Entitlement(tt.entitlement)]; ok {
					t.Errorf("Expected entitlement %s not to be registered when value is false", tt.entitlement)
				}
			}
		})
	}
}

func TestTCCPermissionFunctions(t *testing.T) {
	// Create a backup of the original config
	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	tests := []struct {
		name        string
		fn          func()
		entitlement macgo.Entitlement
	}{
		{"SetCamera", SetCamera, macgo.Entitlement(EntCamera)},
		{"SetMic", SetMic, macgo.Entitlement(EntMicrophone)},
		{"SetLocation", SetLocation, macgo.Entitlement(EntLocation)},
		{"SetContacts", SetContacts, macgo.Entitlement(EntAddressBook)},
		{"SetPhotos", SetPhotos, macgo.Entitlement(EntPhotos)},
		{"SetCalendar", SetCalendar, macgo.Entitlement(EntCalendars)},
		{"SetReminders", SetReminders, macgo.Entitlement(EntReminders)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset config
			macgo.DefaultConfig = macgo.NewConfig()

			tt.fn()

			if val, ok := macgo.DefaultConfig.Entitlements[tt.entitlement]; !ok || !val {
				t.Errorf("Expected entitlement %s to be set to true", tt.entitlement)
			}
		})
	}
}

func TestAppSandboxFunctions(t *testing.T) {
	// Create a backup of the original config
	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	tests := []struct {
		name        string
		fn          func()
		entitlement macgo.Entitlement
	}{
		{"SetAppSandbox", SetAppSandbox, macgo.Entitlement(EntAppSandbox)},
		{"SetNetworkClient", SetNetworkClient, macgo.Entitlement(EntNetworkClient)},
		{"SetNetworkServer", SetNetworkServer, macgo.Entitlement(EntNetworkServer)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset config
			macgo.DefaultConfig = macgo.NewConfig()

			tt.fn()

			if val, ok := macgo.DefaultConfig.Entitlements[tt.entitlement]; !ok || !val {
				t.Errorf("Expected entitlement %s to be set to true", tt.entitlement)
			}
		})
	}
}

func TestDeviceAccessFunctions(t *testing.T) {
	// Create a backup of the original config
	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	tests := []struct {
		name        string
		fn          func()
		entitlement macgo.Entitlement
	}{
		{"SetBluetooth", SetBluetooth, macgo.Entitlement(EntBluetooth)},
		{"SetUSB", SetUSB, macgo.Entitlement(EntUSB)},
		{"SetAudioInput", SetAudioInput, macgo.Entitlement(EntAudioInput)},
		{"SetPrinting", SetPrinting, macgo.Entitlement(EntPrint)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset config
			macgo.DefaultConfig = macgo.NewConfig()

			tt.fn()

			if val, ok := macgo.DefaultConfig.Entitlements[tt.entitlement]; !ok || !val {
				t.Errorf("Expected entitlement %s to be set to true", tt.entitlement)
			}
		})
	}
}

func TestHardenedRuntimeFunctions(t *testing.T) {
	// Create a backup of the original config
	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	tests := []struct {
		name        string
		fn          func()
		entitlement macgo.Entitlement
	}{
		{"SetAllowJIT", SetAllowJIT, macgo.Entitlement(EntAllowJIT)},
		{"SetAllowUnsignedMemory", SetAllowUnsignedMemory, macgo.Entitlement(EntAllowUnsignedExecutableMemory)},
		{"SetAllowDyldEnvVars", SetAllowDyldEnvVars, macgo.Entitlement(EntAllowDyldEnvVars)},
		{"SetDisableLibraryValidation", SetDisableLibraryValidation, macgo.Entitlement(EntDisableLibraryValidation)},
		{"SetDisableExecutablePageProtection", SetDisableExecutablePageProtection, macgo.Entitlement(EntDisableExecutablePageProtection)},
		{"SetDebugger", SetDebugger, macgo.Entitlement(EntDebugger)},
		{"SetVirtualization", SetVirtualization, macgo.Entitlement(EntVirtualization)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset config
			macgo.DefaultConfig = macgo.NewConfig()

			tt.fn()

			if val, ok := macgo.DefaultConfig.Entitlements[tt.entitlement]; !ok || !val {
				t.Errorf("Expected entitlement %s to be set to true", tt.entitlement)
			}
		})
	}
}

func TestSetCustomEntitlement(t *testing.T) {
	// Create a backup of the original config
	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	tests := []struct {
		name  string
		key   string
		value bool
	}{
		{"Custom Entitlement True", "com.example.custom.permission", true},
		{"Custom Entitlement False", "com.example.custom.other", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset config
			macgo.DefaultConfig = macgo.NewConfig()

			SetCustomEntitlement(tt.key, tt.value)

			if tt.value {
				if val, ok := macgo.DefaultConfig.Entitlements[macgo.Entitlement(tt.key)]; !ok || !val {
					t.Errorf("Expected custom entitlement %s to be set to true", tt.key)
				}
			} else {
				if _, ok := macgo.DefaultConfig.Entitlements[macgo.Entitlement(tt.key)]; ok {
					t.Errorf("Expected custom entitlement %s not to be registered when value is false", tt.key)
				}
			}
		})
	}
}

func TestRegisterEntitlements(t *testing.T) {
	// Create a backup of the original config
	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	testJSON := `{
		"com.apple.security.device.camera": true,
		"com.apple.security.device.microphone": true,
		"com.apple.security.app-sandbox": false
	}`

	// Reset config
	macgo.DefaultConfig = macgo.NewConfig()

	err := RegisterEntitlements([]byte(testJSON))
	if err != nil {
		t.Fatalf("Failed to register entitlements: %v", err)
	}

	// Check that entitlements were registered correctly
	expectedEntitlements := map[macgo.Entitlement]bool{
		macgo.Entitlement(EntCamera):     true,
		macgo.Entitlement(EntMicrophone): true,
		macgo.Entitlement(EntAppSandbox): false,
	}

	for ent, expectedVal := range expectedEntitlements {
		if val, ok := macgo.DefaultConfig.Entitlements[ent]; !ok {
			t.Errorf("Expected entitlement %s to be registered", ent)
		} else if val != expectedVal {
			t.Errorf("Expected entitlement %s to be %v, got %v", ent, expectedVal, val)
		}
	}
}

func TestRegisterEntitlementsFromReader(t *testing.T) {
	// Create a backup of the original config
	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	testJSON := `{
		"com.apple.security.personal-information.location": true,
		"com.apple.security.network.client": true
	}`

	// Reset config
	macgo.DefaultConfig = macgo.NewConfig()

	reader := strings.NewReader(testJSON)
	err := RegisterEntitlementsFromReader(reader)
	if err != nil {
		t.Fatalf("Failed to register entitlements from reader: %v", err)
	}

	// Check that entitlements were registered
	expectedEntitlements := []macgo.Entitlement{
		macgo.Entitlement(EntLocation),
		macgo.Entitlement(EntNetworkClient),
	}

	for _, ent := range expectedEntitlements {
		if val, ok := macgo.DefaultConfig.Entitlements[ent]; !ok || !val {
			t.Errorf("Expected entitlement %s to be registered as true", ent)
		}
	}
}

func TestRegisterEntitlementsFromFile(t *testing.T) {
	// Create a backup of the original config
	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "entitlements-test-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	testJSON := `{
		"com.apple.security.personal-information.photos-library": true,
		"com.apple.security.device.bluetooth": true
	}`

	if _, err := tmpFile.Write([]byte(testJSON)); err != nil {
		t.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatal(err)
	}

	// Reset config
	macgo.DefaultConfig = macgo.NewConfig()

	err = RegisterEntitlementsFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to register entitlements from file: %v", err)
	}

	// Check that entitlements were registered
	expectedEntitlements := []macgo.Entitlement{
		macgo.Entitlement(EntPhotos),
		macgo.Entitlement(EntBluetooth),
	}

	for _, ent := range expectedEntitlements {
		if val, ok := macgo.DefaultConfig.Entitlements[ent]; !ok || !val {
			t.Errorf("Expected entitlement %s to be registered as true", ent)
		}
	}
}

func TestRegisterEntitlementsFromFileError(t *testing.T) {
	err := RegisterEntitlementsFromFile("/nonexistent/path/entitlements.json")
	if err == nil {
		t.Error("Expected error when reading nonexistent file")
	}
}

func TestRegisterEntitlementsInvalidJSON(t *testing.T) {
	invalidJSON := `{"invalid": json}`
	err := RegisterEntitlements([]byte(invalidJSON))
	if err == nil {
		t.Error("Expected error when parsing invalid JSON")
	}
}

func TestConvenienceFunctions(t *testing.T) {
	// Create a backup of the original config
	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	tests := []struct {
		name                 string
		fn                   func()
		expectedEntitlements []macgo.Entitlement
	}{
		{
			"SetAllTCCPermissions",
			SetAllTCCPermissions,
			[]macgo.Entitlement{
				macgo.Entitlement(EntCamera),
				macgo.Entitlement(EntMicrophone),
				macgo.Entitlement(EntLocation),
				macgo.Entitlement(EntAddressBook),
				macgo.Entitlement(EntPhotos),
				macgo.Entitlement(EntCalendars),
				macgo.Entitlement(EntReminders),
			},
		},
		{
			"SetAllDeviceAccess",
			SetAllDeviceAccess,
			[]macgo.Entitlement{
				macgo.Entitlement(EntCamera),
				macgo.Entitlement(EntMicrophone),
				macgo.Entitlement(EntBluetooth),
				macgo.Entitlement(EntUSB),
				macgo.Entitlement(EntAudioInput),
				macgo.Entitlement(EntPrint),
			},
		},
		{
			"SetAllNetworking",
			SetAllNetworking,
			[]macgo.Entitlement{
				macgo.Entitlement(EntNetworkClient),
				macgo.Entitlement(EntNetworkServer),
			},
		},
		{
			"SetAll",
			SetAll,
			[]macgo.Entitlement{
				macgo.Entitlement(EntCamera),
				macgo.Entitlement(EntMicrophone),
				macgo.Entitlement(EntLocation),
				macgo.Entitlement(EntAddressBook),
				macgo.Entitlement(EntPhotos),
				macgo.Entitlement(EntCalendars),
				macgo.Entitlement(EntReminders),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset config
			macgo.DefaultConfig = macgo.NewConfig()

			tt.fn()

			for _, expectedEnt := range tt.expectedEntitlements {
				if val, ok := macgo.DefaultConfig.Entitlements[expectedEnt]; !ok || !val {
					t.Errorf("Expected entitlement %s to be set to true", expectedEnt)
				}
			}
		})
	}
}

func TestRequestEntitlements(t *testing.T) {
	// Create a backup of the original config
	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	// Reset config
	macgo.DefaultConfig = macgo.NewConfig()

	// Test with mixed types
	RequestEntitlements(
		EntCamera,
		"com.apple.security.device.microphone",
		EntLocation,
	)

	expectedEntitlements := []macgo.Entitlement{
		macgo.Entitlement(EntCamera),
		macgo.Entitlement(EntMicrophone),
		macgo.Entitlement(EntLocation),
	}

	for _, ent := range expectedEntitlements {
		if val, ok := macgo.DefaultConfig.Entitlements[ent]; !ok || !val {
			t.Errorf("Expected entitlement %s to be registered as true", ent)
		}
	}
}

func TestEntitlementTypeComparison(t *testing.T) {
	// Test that our entitlement constants match the main package constants
	entitlementPairs := map[Entitlement]macgo.Entitlement{
		EntAppSandbox:    macgo.EntAppSandbox,
		EntNetworkClient: macgo.EntNetworkClient,
		EntNetworkServer: macgo.EntNetworkServer,
		EntCamera:        macgo.EntCamera,
		EntMicrophone:    macgo.EntMicrophone,
		EntBluetooth:     macgo.EntBluetooth,
		EntUSB:           macgo.EntUSB,
		EntAudioInput:    macgo.EntAudioInput,
		EntPrint:         macgo.EntPrint,
		EntAddressBook:   macgo.EntAddressBook,
		EntLocation:      macgo.EntLocation,
		EntCalendars:     macgo.EntCalendars,
		EntPhotos:        macgo.EntPhotos,
		EntReminders:     macgo.EntReminders,
	}

	for entLocal, entMain := range entitlementPairs {
		if string(entLocal) != string(entMain) {
			t.Errorf("Entitlement mismatch: local %q != main %q", entLocal, entMain)
		}
	}
}

// Test edge cases and error conditions

func TestRegisterEntitlementsEmptyJSON(t *testing.T) {
	err := RegisterEntitlements([]byte("{}"))
	if err != nil {
		t.Errorf("Expected no error for empty JSON object, got: %v", err)
	}
}

func TestRegisterEntitlementsNilData(t *testing.T) {
	err := RegisterEntitlements(nil)
	if err == nil {
		t.Error("Expected error for nil data")
	}
}

func TestRegisterWithInvalidEntitlement(t *testing.T) {
	// Create a backup of the original config
	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	// Reset config
	macgo.DefaultConfig = macgo.NewConfig()

	// This should not crash or cause issues
	RequestEntitlements(123, nil, []int{1, 2, 3})

	// Config should still be valid but empty
	if len(macgo.DefaultConfig.Entitlements) != 0 {
		t.Errorf("Expected no entitlements to be registered for invalid types, got %d", len(macgo.DefaultConfig.Entitlements))
	}
}

func TestRegisterEntitlementsFromReaderError(t *testing.T) {
	// Test with reader that returns an error
	errorReader := &bytes.Buffer{}
	errorReader.WriteString("invalid json")

	err := RegisterEntitlementsFromReader(errorReader)
	if err == nil {
		t.Error("Expected error when reading invalid JSON")
	}
}
