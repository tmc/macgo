package entitlements

import (
	"os"
	"testing"

	"github.com/tmc/misc/macgo"
)

// TestIntegrationWithMainPackage tests the integration between the entitlements package
// and the main macgo package
func TestIntegrationWithMainPackage(t *testing.T) {
	// Create a backup of the original config
	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	// Reset config
	macgo.DefaultConfig = macgo.NewConfig()

	// Test that entitlements package functions work with the main package
	SetCamera()
	SetMic()
	SetLocation()

	// Verify that the entitlements were registered in the main package
	expectedEntitlements := []macgo.Entitlement{
		macgo.EntCamera,
		macgo.EntMicrophone,
		macgo.EntLocation,
	}

	for _, ent := range expectedEntitlements {
		if val, exists := macgo.DefaultConfig.Entitlements[ent]; !exists || !val {
			t.Errorf("Expected entitlement %s to be registered in main package", ent)
		}
	}
}

func TestRegisterIntegrationWithMainPackage(t *testing.T) {
	// Create a backup of the original config
	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	// Reset config
	macgo.DefaultConfig = macgo.NewConfig()

	// Test the Register function from entitlements package
	Register(EntPhotos, true)
	Register(EntCalendars, false) // false should not register

	// Check that true registration worked
	if val, exists := macgo.DefaultConfig.Entitlements[macgo.EntPhotos]; !exists || !val {
		t.Error("Expected photos entitlement to be registered as true")
	}

	// Check that false registration didn't add the entitlement
	if _, exists := macgo.DefaultConfig.Entitlements[macgo.EntCalendars]; exists {
		t.Error("Expected calendar entitlement not to be registered when value is false")
	}
}

func TestJSONIntegrationWithMainPackage(t *testing.T) {
	// Create a backup of the original config
	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	// Reset config
	macgo.DefaultConfig = macgo.NewConfig()

	// Test JSON loading through entitlements package
	testJSON := `{
		"com.apple.security.device.camera": true,
		"com.apple.security.device.microphone": true,
		"com.apple.security.personal-information.location": false,
		"com.apple.security.app-sandbox": true
	}`

	err := RegisterEntitlements([]byte(testJSON))
	if err != nil {
		t.Fatalf("Failed to register entitlements: %v", err)
	}

	// Verify the entitlements were registered in the main package
	expectedEntitlements := map[macgo.Entitlement]bool{
		macgo.EntCamera:     true,
		macgo.EntMicrophone: true,
		macgo.EntLocation:   false,
		macgo.EntAppSandbox: true,
	}

	for ent, expectedVal := range expectedEntitlements {
		if val, exists := macgo.DefaultConfig.Entitlements[ent]; !exists {
			t.Errorf("Expected entitlement %s to be registered", ent)
		} else if val != expectedVal {
			t.Errorf("Expected entitlement %s to be %v, got %v", ent, expectedVal, val)
		}
	}
}

func TestFileIntegrationWithMainPackage(t *testing.T) {
	// Create a backup of the original config
	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "entitlements-integration-test-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	testJSON := `{
		"com.apple.security.personal-information.photos-library": true,
		"com.apple.security.device.bluetooth": true,
		"com.apple.security.network.client": true
	}`

	if _, err := tmpFile.Write([]byte(testJSON)); err != nil {
		t.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatal(err)
	}

	// Reset config
	macgo.DefaultConfig = macgo.NewConfig()

	// Load from file through entitlements package
	err = RegisterEntitlementsFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to register entitlements from file: %v", err)
	}

	// Verify the entitlements were registered in the main package
	expectedEntitlements := []macgo.Entitlement{
		macgo.EntPhotos,
		macgo.EntBluetooth,
		macgo.EntNetworkClient,
	}

	for _, ent := range expectedEntitlements {
		if val, exists := macgo.DefaultConfig.Entitlements[ent]; !exists || !val {
			t.Errorf("Expected entitlement %s to be registered as true", ent)
		}
	}
}

func TestConvenienceFunctionsIntegrationWithMainPackage(t *testing.T) {
	// Create a backup of the original config
	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	tests := []struct {
		name        string
		fn          func()
		expectedEnt []macgo.Entitlement
	}{
		{
			"SetAllTCCPermissions",
			SetAllTCCPermissions,
			[]macgo.Entitlement{
				macgo.EntCamera, macgo.EntMicrophone, macgo.EntLocation,
				macgo.EntAddressBook, macgo.EntPhotos, macgo.EntCalendars, macgo.EntReminders,
			},
		},
		{
			"SetAllDeviceAccess",
			SetAllDeviceAccess,
			[]macgo.Entitlement{
				macgo.EntCamera, macgo.EntMicrophone, macgo.EntBluetooth,
				macgo.EntUSB, macgo.EntAudioInput, macgo.EntPrint,
			},
		},
		{
			"SetAllNetworking",
			SetAllNetworking,
			[]macgo.Entitlement{macgo.EntNetworkClient, macgo.EntNetworkServer},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset config
			macgo.DefaultConfig = macgo.NewConfig()

			// Call the convenience function
			tt.fn()

			// Verify all expected entitlements are registered
			for _, ent := range tt.expectedEnt {
				if val, exists := macgo.DefaultConfig.Entitlements[ent]; !exists || !val {
					t.Errorf("Expected entitlement %s to be registered as true", ent)
				}
			}
		})
	}
}

func TestRequestEntitlementsIntegrationWithMainPackage(t *testing.T) {
	// Create a backup of the original config
	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	// Reset config
	macgo.DefaultConfig = macgo.NewConfig()

	// Test the RequestEntitlements function from entitlements package
	RequestEntitlements(
		EntCamera,
		"com.apple.security.device.microphone",
		EntLocation,
	)

	// Verify entitlements were registered in the main package
	expectedEntitlements := []macgo.Entitlement{
		macgo.EntCamera,
		macgo.EntMicrophone,
		macgo.EntLocation,
	}

	for _, ent := range expectedEntitlements {
		if val, exists := macgo.DefaultConfig.Entitlements[ent]; !exists || !val {
			t.Errorf("Expected entitlement %s to be registered as true", ent)
		}
	}
}

func TestCustomEntitlementIntegrationWithMainPackage(t *testing.T) {
	// Create a backup of the original config
	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	// Reset config
	macgo.DefaultConfig = macgo.NewConfig()

	// Test custom entitlement setting
	customEntitlement := "com.example.custom.permission"
	SetCustomEntitlement(customEntitlement, true)

	// Verify it was registered in the main package
	if val, exists := macgo.DefaultConfig.Entitlements[macgo.Entitlement(customEntitlement)]; !exists || !val {
		t.Errorf("Expected custom entitlement %s to be registered as true", customEntitlement)
	}

	// Test with false value
	customEntitlementFalse := "com.example.custom.disabled"
	SetCustomEntitlement(customEntitlementFalse, false)

	// Should not be registered
	if _, exists := macgo.DefaultConfig.Entitlements[macgo.Entitlement(customEntitlementFalse)]; exists {
		t.Errorf("Expected custom entitlement %s not to be registered when value is false", customEntitlementFalse)
	}
}

func TestMainPackageCompatibility(t *testing.T) {
	// Test that entitlements package works alongside main package functions
	// Create a backup of the original config
	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	// Reset config
	macgo.DefaultConfig = macgo.NewConfig()

	// Use both entitlements package and main package functions
	SetCamera()                                              // From entitlements package
	macgo.RequestEntitlement(macgo.EntMicrophone)            // From main package
	SetLocation()                                            // From entitlements package
	macgo.RequestEntitlements(macgo.EntPhotos, macgo.EntUSB) // From main package

	// Verify all were registered
	expectedEntitlements := []macgo.Entitlement{
		macgo.EntCamera,
		macgo.EntMicrophone,
		macgo.EntLocation,
		macgo.EntPhotos,
		macgo.EntUSB,
	}

	for _, ent := range expectedEntitlements {
		if val, exists := macgo.DefaultConfig.Entitlements[ent]; !exists || !val {
			t.Errorf("Expected entitlement %s to be registered as true", ent)
		}
	}
}

func TestLoadJSONComparedToMainPackage(t *testing.T) {
	// Test that loading JSON through entitlements package gives same result as main package

	testJSON := `{
		"com.apple.security.device.camera": true,
		"com.apple.security.device.microphone": true,
		"com.apple.security.personal-information.location": true
	}`

	// Test entitlements package approach
	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	macgo.DefaultConfig = macgo.NewConfig()
	err := RegisterEntitlements([]byte(testJSON))
	if err != nil {
		t.Fatal(err)
	}
	entitlementsResult := make(map[macgo.Entitlement]bool)
	for k, v := range macgo.DefaultConfig.Entitlements {
		entitlementsResult[k] = v
	}

	// Test main package approach
	macgo.DefaultConfig = macgo.NewConfig()
	err = macgo.LoadEntitlementsFromJSON([]byte(testJSON))
	if err != nil {
		t.Fatal(err)
	}
	mainResult := make(map[macgo.Entitlement]bool)
	for k, v := range macgo.DefaultConfig.Entitlements {
		mainResult[k] = v
	}

	// Compare results
	if len(entitlementsResult) != len(mainResult) {
		t.Errorf("Different number of entitlements: entitlements package=%d, main package=%d",
			len(entitlementsResult), len(mainResult))
	}

	for ent, val := range entitlementsResult {
		if mainVal, exists := mainResult[ent]; !exists {
			t.Errorf("Entitlement %s missing from main package result", ent)
		} else if val != mainVal {
			t.Errorf("Entitlement %s value mismatch: entitlements=%v, main=%v", ent, val, mainVal)
		}
	}
}

// TestTypeCompatibility tests that the entitlements package types
// are fully compatible with the main package types
func TestTypeCompatibility(t *testing.T) {
	// Create a backup of the original config
	originalConfig := macgo.DefaultConfig
	defer func() {
		macgo.DefaultConfig = originalConfig
	}()

	// Reset config
	macgo.DefaultConfig = macgo.NewConfig()

	// Test that entitlements.Entitlement can be used where macgo.Entitlement is expected
	var entitlementFromEntitlements Entitlement = EntCamera
	var entitlementFromMacgo macgo.Entitlement = macgo.EntCamera

	// These should be assignable to each other (through string conversion)
	if string(entitlementFromEntitlements) != string(entitlementFromMacgo) {
		t.Error("Entitlement types should be compatible")
	}

	// Test that we can use entitlements package constants with main package functions
	macgo.RequestEntitlement(string(EntMicrophone))
	macgo.RequestEntitlements(string(EntLocation), string(EntPhotos))

	// Verify they were registered
	expectedEntitlements := []macgo.Entitlement{macgo.EntMicrophone, macgo.EntLocation, macgo.EntPhotos}
	for _, ent := range expectedEntitlements {
		if val, exists := macgo.DefaultConfig.Entitlements[ent]; !exists || !val {
			t.Errorf("Expected entitlement %s to be registered", ent)
		}
	}
}
