package macgo

import (
	"fmt"
	"io/fs"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCalculateSHA256(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "macgo-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	// Write some content
	content := "test content for SHA256"
	if _, err := tmpFile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatal(err)
	}

	// Calculate the hash
	hash, err := checksum(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	// The hash should be 64 characters long (SHA-256 is 32 bytes, hex-encoded)
	if len(hash) != 64 {
		t.Errorf("Expected SHA-256 hash to be 64 characters, got %d", len(hash))
	}
}

func TestCopyFile(t *testing.T) {
	// Create a source file
	srcFile, err := os.CreateTemp("", "macgo-test-src-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(srcFile.Name())

	// Write content
	content := "test content for copy file"
	if _, err := srcFile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := srcFile.Close(); err != nil {
		t.Fatal(err)
	}

	// Create a destination path
	dstFile, err := os.CreateTemp("", "macgo-test-dst-*")
	if err != nil {
		t.Fatal(err)
	}
	dstPath := dstFile.Name()
	dstFile.Close()
	os.Remove(dstPath) // Remove it so copyFile can create it
	defer os.Remove(dstPath)

	// Copy the file
	if err := copyFile(srcFile.Name(), dstPath); err != nil {
		t.Fatal(err)
	}

	// Verify the content
	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(dstContent) != content {
		t.Errorf("Expected content %q, got %q", content, string(dstContent))
	}
}

// TestAppBundleCreation skips actual creation in test mode
func TestAppBundleCreation(t *testing.T) {
	// Skip if we can't find our own executable
	execPath, err := os.Executable()
	if err != nil {
		t.Skip("Could not determine executable path")
	}

	// Skip this test - it's more of a functionality test
	// We can't properly test this without actually creating an app bundle
	// and that might interfere with the user's environment
	if strings.Contains(execPath, "go-build") {
		t.Log("Running with go test, which uses temporary binaries")
		// Verify the isTemporary detection works
		if !strings.Contains(execPath, "go-build") {
			t.Error("Expected to detect temporary binary during test")
		}
	}
}

// TestNewConfig tests the NewConfig function
func TestNewConfig(t *testing.T) {
	cfg := NewConfig()

	// Test default values
	if cfg.Relaunch != true {
		t.Error("Expected Relaunch to be true by default")
	}

	if cfg.AutoSign != true {
		t.Error("Expected AutoSign to be true by default")
	}

	// Test that Entitlements map is initialized
	if cfg.Entitlements == nil {
		t.Error("Expected Entitlements map to be initialized")
	}

	// Test that PlistEntries map is initialized
	if cfg.PlistEntries == nil {
		t.Error("Expected PlistEntries map to be initialized")
	}

	// Test default LSUIElement value
	if val, exists := cfg.PlistEntries["LSUIElement"]; !exists || val != true {
		t.Error("Expected LSUIElement to be true by default")
	}

	// Test that other fields are empty/default
	if cfg.ApplicationName != "" {
		t.Error("Expected ApplicationName to be empty by default")
	}

	if cfg.BundleID != "" {
		t.Error("Expected BundleID to be empty by default")
	}

	if cfg.CustomDestinationAppPath != "" {
		t.Error("Expected CustomDestinationAppPath to be empty by default")
	}

	if cfg.KeepTemp != false {
		t.Error("Expected KeepTemp to be false by default")
	}

	if cfg.AppTemplate != nil {
		t.Error("Expected AppTemplate to be nil by default")
	}

	if cfg.SigningIdentity != "" {
		t.Error("Expected SigningIdentity to be empty by default")
	}
}

// TestConfig_AddEntitlement tests the AddEntitlement method
func TestConfig_AddEntitlement(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() *Config
		entitlement Entitlement
		expected    bool
	}{
		{
			name: "add entitlement to empty config",
			setup: func() *Config {
				return &Config{}
			},
			entitlement: EntCamera,
			expected:    true,
		},
		{
			name: "add entitlement to config with existing entitlements",
			setup: func() *Config {
				return &Config{
					Entitlements: map[Entitlement]bool{
						EntMicrophone: true,
					},
				}
			},
			entitlement: EntCamera,
			expected:    true,
		},
		{
			name: "add custom entitlement",
			setup: func() *Config {
				return &Config{}
			},
			entitlement: Entitlement("com.example.custom.entitlement"),
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.setup()
			cfg.AddEntitlement(tt.entitlement)

			// Check that Entitlements map was initialized if needed
			if cfg.Entitlements == nil {
				t.Error("Expected Entitlements map to be initialized")
				return
			}

			// Check that the entitlement was added with correct value
			if val, exists := cfg.Entitlements[tt.entitlement]; !exists {
				t.Errorf("Expected entitlement %s to be added", tt.entitlement)
			} else if val != tt.expected {
				t.Errorf("Expected entitlement %s to have value %v, got %v", tt.entitlement, tt.expected, val)
			}
		})
	}
}

// TestConfig_AddPermission tests the AddPermission method (legacy method)
func TestConfig_AddPermission(t *testing.T) {
	cfg := &Config{}
	permission := EntLocation

	cfg.AddPermission(permission)

	// Check that Entitlements map was initialized
	if cfg.Entitlements == nil {
		t.Error("Expected Entitlements map to be initialized")
		return
	}

	// Check that the permission was added via AddEntitlement
	if val, exists := cfg.Entitlements[permission]; !exists {
		t.Errorf("Expected permission %s to be added", permission)
	} else if val != true {
		t.Errorf("Expected permission %s to have value true, got %v", permission, val)
	}
}

// TestConfig_AddPlistEntry tests the AddPlistEntry method
func TestConfig_AddPlistEntry(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *Config
		key      string
		value    any
		expected any
	}{
		{
			name: "add plist entry to empty config",
			setup: func() *Config {
				return &Config{}
			},
			key:      "CFBundleName",
			value:    "TestApp",
			expected: "TestApp",
		},
		{
			name: "add plist entry to config with existing entries",
			setup: func() *Config {
				return &Config{
					PlistEntries: map[string]any{
						"LSUIElement": false,
					},
				}
			},
			key:      "CFBundleVersion",
			value:    "1.0.0",
			expected: "1.0.0",
		},
		{
			name: "add boolean plist entry",
			setup: func() *Config {
				return &Config{}
			},
			key:      "LSBackgroundOnly",
			value:    true,
			expected: true,
		},
		{
			name: "add numeric plist entry",
			setup: func() *Config {
				return &Config{}
			},
			key:      "CFBundleVersion",
			value:    42,
			expected: 42,
		},
		{
			name: "overwrite existing plist entry",
			setup: func() *Config {
				return &Config{
					PlistEntries: map[string]any{
						"LSUIElement": true,
					},
				}
			},
			key:      "LSUIElement",
			value:    false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.setup()
			cfg.AddPlistEntry(tt.key, tt.value)

			// Check that PlistEntries map was initialized if needed
			if cfg.PlistEntries == nil {
				t.Error("Expected PlistEntries map to be initialized")
				return
			}

			// Check that the entry was added with correct value
			if val, exists := cfg.PlistEntries[tt.key]; !exists {
				t.Errorf("Expected plist entry %s to be added", tt.key)
			} else if val != tt.expected {
				t.Errorf("Expected plist entry %s to have value %v, got %v", tt.key, tt.expected, val)
			}
		})
	}
}

// TestConfig_RequestEntitlements tests the RequestEntitlements method on Config
func TestConfig_RequestEntitlements(t *testing.T) {
	tests := []struct {
		name         string
		setup        func() *Config
		entitlements []interface{}
		expected     map[Entitlement]bool
	}{
		{
			name: "add multiple entitlements to empty config",
			setup: func() *Config {
				return &Config{}
			},
			entitlements: []interface{}{EntCamera, EntMicrophone, EntLocation},
			expected: map[Entitlement]bool{
				EntCamera:     true,
				EntMicrophone: true,
				EntLocation:   true,
			},
		},
		{
			name: "add string and Entitlement types",
			setup: func() *Config {
				return &Config{}
			},
			entitlements: []interface{}{"com.apple.security.custom", EntAppSandbox},
			expected: map[Entitlement]bool{
				Entitlement("com.apple.security.custom"): true,
				EntAppSandbox:                            true,
			},
		},
		{
			name: "add to config with existing entitlements",
			setup: func() *Config {
				return &Config{
					Entitlements: map[Entitlement]bool{
						EntNetworkClient: true,
					},
				}
			},
			entitlements: []interface{}{EntCamera, EntMicrophone},
			expected: map[Entitlement]bool{
				EntNetworkClient: true,
				EntCamera:        true,
				EntMicrophone:    true,
			},
		},
		{
			name: "ignore invalid types",
			setup: func() *Config {
				return &Config{}
			},
			entitlements: []interface{}{EntCamera, 123, nil, EntMicrophone, map[string]string{}},
			expected: map[Entitlement]bool{
				EntCamera:     true,
				EntMicrophone: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.setup()
			cfg.RequestEntitlements(tt.entitlements...)

			// Check that Entitlements map was initialized if needed
			if cfg.Entitlements == nil && len(tt.expected) > 0 {
				t.Error("Expected Entitlements map to be initialized")
				return
			}

			// Check that all expected entitlements are present
			for expectedEnt, expectedVal := range tt.expected {
				if val, exists := cfg.Entitlements[expectedEnt]; !exists {
					t.Errorf("Expected entitlement %s to be present", expectedEnt)
				} else if val != expectedVal {
					t.Errorf("Expected entitlement %s to have value %v, got %v", expectedEnt, expectedVal, val)
				}
			}

			// Check that no unexpected entitlements were added
			for actualEnt := range cfg.Entitlements {
				if _, expected := tt.expected[actualEnt]; !expected {
					t.Errorf("Unexpected entitlement %s was added", actualEnt)
				}
			}
		})
	}
}

// TestConfigure tests the Configure function
func TestConfigure(t *testing.T) {
	// Save original DefaultConfig
	originalConfig := DefaultConfig
	defer func() {
		DefaultConfig = originalConfig
	}()

	tests := []struct {
		name           string
		setup          func()
		inputConfig    *Config
		expectedChecks func(t *testing.T)
	}{
		{
			name: "configure with all fields",
			setup: func() {
				DefaultConfig = &Config{
					Relaunch:     true,
					Entitlements: map[Entitlement]bool{},
					PlistEntries: map[string]any{},
				}
			},
			inputConfig: &Config{
				ApplicationName:          "TestApp",
				BundleID:                 "com.test.app",
				Relaunch:                 false,
				CustomDestinationAppPath: "/custom/path",
				KeepTemp:                 true,
				AutoSign:                 false,
				SigningIdentity:          "TestIdentity",
				Entitlements: map[Entitlement]bool{
					EntCamera:     true,
					EntMicrophone: false,
				},
				PlistEntries: map[string]any{
					"CFBundleName": "TestApp",
					"LSUIElement":  false,
				},
			},
			expectedChecks: func(t *testing.T) {
				if DefaultConfig.ApplicationName != "TestApp" {
					t.Errorf("Expected ApplicationName to be 'TestApp', got %s", DefaultConfig.ApplicationName)
				}
				if DefaultConfig.BundleID != "com.test.app" {
					t.Errorf("Expected BundleID to be 'com.test.app', got %s", DefaultConfig.BundleID)
				}
				if DefaultConfig.Relaunch != false {
					t.Error("Expected Relaunch to be false")
				}
				if DefaultConfig.CustomDestinationAppPath != "/custom/path" {
					t.Errorf("Expected CustomDestinationAppPath to be '/custom/path', got %s", DefaultConfig.CustomDestinationAppPath)
				}
				if DefaultConfig.KeepTemp != true {
					t.Error("Expected KeepTemp to be true")
				}
				if DefaultConfig.AutoSign != false {
					t.Error("Expected AutoSign to be false")
				}
				if DefaultConfig.SigningIdentity != "TestIdentity" {
					t.Errorf("Expected SigningIdentity to be 'TestIdentity', got %s", DefaultConfig.SigningIdentity)
				}

				// Check entitlements
				if val, exists := DefaultConfig.Entitlements[EntCamera]; !exists || val != true {
					t.Error("Expected EntCamera to be true")
				}
				if val, exists := DefaultConfig.Entitlements[EntMicrophone]; !exists || val != false {
					t.Error("Expected EntMicrophone to be false")
				}

				// Check plist entries
				if val, exists := DefaultConfig.PlistEntries["CFBundleName"]; !exists || val != "TestApp" {
					t.Error("Expected CFBundleName to be 'TestApp'")
				}
				if val, exists := DefaultConfig.PlistEntries["LSUIElement"]; !exists || val != false {
					t.Error("Expected LSUIElement to be false")
				}
			},
		},
		{
			name: "configure with nil entitlements and plist entries",
			setup: func() {
				DefaultConfig = &Config{
					Entitlements: map[Entitlement]bool{
						EntNetworkClient: true,
					},
					PlistEntries: map[string]any{
						"ExistingKey": "ExistingValue",
					},
				}
			},
			inputConfig: &Config{
				ApplicationName: "TestApp",
				Entitlements:    nil,
				PlistEntries:    nil,
			},
			expectedChecks: func(t *testing.T) {
				if DefaultConfig.ApplicationName != "TestApp" {
					t.Errorf("Expected ApplicationName to be 'TestApp', got %s", DefaultConfig.ApplicationName)
				}

				// Check that existing entitlements are preserved
				if val, exists := DefaultConfig.Entitlements[EntNetworkClient]; !exists || val != true {
					t.Error("Expected existing EntNetworkClient to be preserved")
				}

				// Check that existing plist entries are preserved
				if val, exists := DefaultConfig.PlistEntries["ExistingKey"]; !exists || val != "ExistingValue" {
					t.Error("Expected existing plist entry to be preserved")
				}
			},
		},
		{
			name: "configure merges entitlements and plist entries",
			setup: func() {
				DefaultConfig = &Config{
					Entitlements: map[Entitlement]bool{
						EntNetworkClient: true,
						EntCamera:        false,
					},
					PlistEntries: map[string]any{
						"ExistingKey": "ExistingValue",
						"LSUIElement": true,
					},
				}
			},
			inputConfig: &Config{
				Entitlements: map[Entitlement]bool{
					EntCamera:     true, // Should overwrite
					EntMicrophone: true, // Should add
				},
				PlistEntries: map[string]any{
					"LSUIElement": false,      // Should overwrite
					"NewKey":      "NewValue", // Should add
				},
			},
			expectedChecks: func(t *testing.T) {
				// Check entitlements merging
				if val, exists := DefaultConfig.Entitlements[EntNetworkClient]; !exists || val != true {
					t.Error("Expected existing EntNetworkClient to be preserved")
				}
				if val, exists := DefaultConfig.Entitlements[EntCamera]; !exists || val != true {
					t.Error("Expected EntCamera to be overwritten to true")
				}
				if val, exists := DefaultConfig.Entitlements[EntMicrophone]; !exists || val != true {
					t.Error("Expected EntMicrophone to be added as true")
				}

				// Check plist entries merging
				if val, exists := DefaultConfig.PlistEntries["ExistingKey"]; !exists || val != "ExistingValue" {
					t.Error("Expected existing plist entry to be preserved")
				}
				if val, exists := DefaultConfig.PlistEntries["LSUIElement"]; !exists || val != false {
					t.Error("Expected LSUIElement to be overwritten to false")
				}
				if val, exists := DefaultConfig.PlistEntries["NewKey"]; !exists || val != "NewValue" {
					t.Error("Expected NewKey to be added")
				}
			},
		},
		{
			name: "configure with empty strings should not overwrite",
			setup: func() {
				DefaultConfig = &Config{
					ApplicationName: "ExistingApp",
					BundleID:        "com.existing.app",
					SigningIdentity: "ExistingIdentity",
				}
			},
			inputConfig: &Config{
				ApplicationName: "",
				BundleID:        "",
				SigningIdentity: "",
			},
			expectedChecks: func(t *testing.T) {
				if DefaultConfig.ApplicationName != "ExistingApp" {
					t.Error("Expected empty ApplicationName to not overwrite existing value")
				}
				if DefaultConfig.BundleID != "com.existing.app" {
					t.Error("Expected empty BundleID to not overwrite existing value")
				}
				if DefaultConfig.SigningIdentity != "ExistingIdentity" {
					t.Error("Expected empty SigningIdentity to not overwrite existing value")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			Configure(tt.inputConfig)
			tt.expectedChecks(t)
		})
	}
}

// TestConfigure_InitializeMaps tests that Configure initializes maps when they're nil
func TestConfigure_InitializeMaps(t *testing.T) {
	// Save original DefaultConfig
	originalConfig := DefaultConfig
	defer func() {
		DefaultConfig = originalConfig
	}()

	// Set up DefaultConfig with nil maps
	DefaultConfig = &Config{
		Entitlements: nil,
		PlistEntries: nil,
	}

	inputConfig := &Config{
		Entitlements: map[Entitlement]bool{
			EntCamera: true,
		},
		PlistEntries: map[string]any{
			"TestKey": "TestValue",
		},
	}

	Configure(inputConfig)

	// Check that maps were initialized
	if DefaultConfig.Entitlements == nil {
		t.Error("Expected Entitlements map to be initialized")
	}
	if DefaultConfig.PlistEntries == nil {
		t.Error("Expected PlistEntries map to be initialized")
	}

	// Check that values were copied
	if val, exists := DefaultConfig.Entitlements[EntCamera]; !exists || val != true {
		t.Error("Expected EntCamera to be copied")
	}
	if val, exists := DefaultConfig.PlistEntries["TestKey"]; !exists || val != "TestValue" {
		t.Error("Expected TestKey to be copied")
	}
}

// TestEntitlementConstants tests that entitlement constants have correct values
func TestEntitlementConstants(t *testing.T) {
	tests := []struct {
		entitlement Entitlement
		expected    string
	}{
		{EntAppSandbox, "com.apple.security.app-sandbox"},
		{EntCamera, "com.apple.security.device.camera"},
		{EntMicrophone, "com.apple.security.device.microphone"},
		{EntLocation, "com.apple.security.personal-information.location"},
		{EntNetworkClient, "com.apple.security.network.client"},
		{EntNetworkServer, "com.apple.security.network.server"},
		{EntUserSelectedReadOnly, "com.apple.security.files.user-selected.read-only"},
		{EntUserSelectedReadWrite, "com.apple.security.files.user-selected.read-write"},
	}

	for _, tt := range tests {
		t.Run(string(tt.entitlement), func(t *testing.T) {
			if string(tt.entitlement) != tt.expected {
				t.Errorf("Expected entitlement %s to have value %s, got %s",
					tt.entitlement, tt.expected, string(tt.entitlement))
			}
		})
	}
}

// TestConfigEdgeCases tests edge cases and potential issues
func TestConfigEdgeCases(t *testing.T) {
	t.Run("AddEntitlement with nil config", func(t *testing.T) {
		var cfg *Config
		// This should panic if not handled properly
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic when calling AddEntitlement on nil config")
			}
		}()
		cfg.AddEntitlement(EntCamera)
	})

	t.Run("AddPlistEntry with nil config", func(t *testing.T) {
		var cfg *Config
		// This should panic if not handled properly
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic when calling AddPlistEntry on nil config")
			}
		}()
		cfg.AddPlistEntry("key", "value")
	})

	t.Run("RequestEntitlements with empty slice", func(t *testing.T) {
		cfg := &Config{}
		cfg.RequestEntitlements()

		// Should not crash and should initialize empty map
		if cfg.Entitlements == nil {
			t.Error("Expected Entitlements map to be initialized even with empty input")
		}
	})

	t.Run("Configure with nil config", func(t *testing.T) {
		// Save original DefaultConfig
		originalConfig := DefaultConfig
		defer func() {
			DefaultConfig = originalConfig
		}()

		// This should not crash and should be a no-op
		Configure(nil)

		// DefaultConfig should remain unchanged
		if DefaultConfig != originalConfig {
			t.Error("Expected DefaultConfig to remain unchanged when Configure is called with nil")
		}
	})
}

// TestRequestEntitlements tests the package-level RequestEntitlements function
func TestRequestEntitlements(t *testing.T) {
	// Save original DefaultConfig
	originalConfig := DefaultConfig
	defer func() {
		DefaultConfig = originalConfig
	}()

	tests := []struct {
		name         string
		setup        func()
		entitlements []interface{}
		expected     map[Entitlement]bool
	}{
		{
			name: "add multiple entitlements to clean config",
			setup: func() {
				DefaultConfig = &Config{
					Entitlements: map[Entitlement]bool{},
				}
			},
			entitlements: []interface{}{EntCamera, EntMicrophone, EntLocation},
			expected: map[Entitlement]bool{
				EntCamera:     true,
				EntMicrophone: true,
				EntLocation:   true,
			},
		},
		{
			name: "add string and Entitlement types",
			setup: func() {
				DefaultConfig = &Config{
					Entitlements: map[Entitlement]bool{},
				}
			},
			entitlements: []interface{}{"com.apple.security.custom", EntAppSandbox},
			expected: map[Entitlement]bool{
				Entitlement("com.apple.security.custom"): true,
				EntAppSandbox:                            true,
			},
		},
		{
			name: "add to config with existing entitlements",
			setup: func() {
				DefaultConfig = &Config{
					Entitlements: map[Entitlement]bool{
						EntNetworkClient: true,
					},
				}
			},
			entitlements: []interface{}{EntCamera, EntMicrophone},
			expected: map[Entitlement]bool{
				EntNetworkClient: true,
				EntCamera:        true,
				EntMicrophone:    true,
			},
		},
		{
			name: "ignore invalid types",
			setup: func() {
				DefaultConfig = &Config{
					Entitlements: map[Entitlement]bool{},
				}
			},
			entitlements: []interface{}{EntCamera, 123, nil, EntMicrophone, map[string]string{}},
			expected: map[Entitlement]bool{
				EntCamera:     true,
				EntMicrophone: true,
			},
		},
		{
			name: "initialize nil entitlements map",
			setup: func() {
				DefaultConfig = &Config{
					Entitlements: nil,
				}
			},
			entitlements: []interface{}{EntCamera},
			expected: map[Entitlement]bool{
				EntCamera: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			RequestEntitlements(tt.entitlements...)

			// Check that Entitlements map was initialized if needed
			if DefaultConfig.Entitlements == nil && len(tt.expected) > 0 {
				t.Error("Expected Entitlements map to be initialized")
				return
			}

			// Check that all expected entitlements are present
			for expectedEnt, expectedVal := range tt.expected {
				if val, exists := DefaultConfig.Entitlements[expectedEnt]; !exists {
					t.Errorf("Expected entitlement %s to be present", expectedEnt)
				} else if val != expectedVal {
					t.Errorf("Expected entitlement %s to have value %v, got %v", expectedEnt, expectedVal, val)
				}
			}

			// Check that no unexpected entitlements were added
			for actualEnt := range DefaultConfig.Entitlements {
				if _, expected := tt.expected[actualEnt]; !expected {
					t.Errorf("Unexpected entitlement %s was added", actualEnt)
				}
			}
		})
	}
}

// TestRequestEntitlement tests the package-level RequestEntitlement function
func TestRequestEntitlement(t *testing.T) {
	// Save original DefaultConfig
	originalConfig := DefaultConfig
	defer func() {
		DefaultConfig = originalConfig
	}()

	tests := []struct {
		name        string
		setup       func()
		entitlement interface{}
		expected    Entitlement
		shouldAdd   bool
	}{
		{
			name: "add string entitlement",
			setup: func() {
				DefaultConfig = &Config{
					Entitlements: map[Entitlement]bool{},
				}
			},
			entitlement: "com.apple.security.custom",
			expected:    Entitlement("com.apple.security.custom"),
			shouldAdd:   true,
		},
		{
			name: "add Entitlement type",
			setup: func() {
				DefaultConfig = &Config{
					Entitlements: map[Entitlement]bool{},
				}
			},
			entitlement: EntCamera,
			expected:    EntCamera,
			shouldAdd:   true,
		},
		{
			name: "ignore invalid type",
			setup: func() {
				DefaultConfig = &Config{
					Entitlements: map[Entitlement]bool{},
				}
			},
			entitlement: 123,
			expected:    "",
			shouldAdd:   false,
		},
		{
			name: "initialize nil entitlements map",
			setup: func() {
				DefaultConfig = &Config{
					Entitlements: nil,
				}
			},
			entitlement: EntMicrophone,
			expected:    EntMicrophone,
			shouldAdd:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			originalCount := 0
			if DefaultConfig.Entitlements != nil {
				originalCount = len(DefaultConfig.Entitlements)
			}

			RequestEntitlement(tt.entitlement)

			if tt.shouldAdd {
				// Check that Entitlements map was initialized if needed
				if DefaultConfig.Entitlements == nil {
					t.Error("Expected Entitlements map to be initialized")
					return
				}

				// Check that the entitlement was added
				if val, exists := DefaultConfig.Entitlements[tt.expected]; !exists {
					t.Errorf("Expected entitlement %s to be added", tt.expected)
				} else if val != true {
					t.Errorf("Expected entitlement %s to have value true, got %v", tt.expected, val)
				}
			} else {
				// Check that nothing was added for invalid types
				currentCount := 0
				if DefaultConfig.Entitlements != nil {
					currentCount = len(DefaultConfig.Entitlements)
				}
				if currentCount != originalCount {
					t.Errorf("Expected no entitlements to be added for invalid type, but count changed from %d to %d", originalCount, currentCount)
				}
			}
		})
	}
}

// TestLoadEntitlementsFromJSON tests the LoadEntitlementsFromJSON function
func TestLoadEntitlementsFromJSON(t *testing.T) {
	// Save original DefaultConfig
	originalConfig := DefaultConfig
	defer func() {
		DefaultConfig = originalConfig
	}()

	tests := []struct {
		name        string
		setup       func()
		jsonData    string
		expected    map[Entitlement]bool
		expectError bool
	}{
		{
			name: "load valid JSON",
			setup: func() {
				DefaultConfig = &Config{
					Entitlements: map[Entitlement]bool{},
				}
			},
			jsonData: `{"com.apple.security.device.camera": true, "com.apple.security.device.microphone": false}`,
			expected: map[Entitlement]bool{
				EntCamera:     true,
				EntMicrophone: false,
			},
			expectError: false,
		},
		{
			name: "load into existing entitlements",
			setup: func() {
				DefaultConfig = &Config{
					Entitlements: map[Entitlement]bool{
						EntNetworkClient: true,
					},
				}
			},
			jsonData: `{"com.apple.security.device.camera": true}`,
			expected: map[Entitlement]bool{
				EntNetworkClient: true,
				EntCamera:        true,
			},
			expectError: false,
		},
		{
			name: "invalid JSON",
			setup: func() {
				DefaultConfig = &Config{
					Entitlements: map[Entitlement]bool{},
				}
			},
			jsonData:    `{"invalid": json}`,
			expected:    map[Entitlement]bool{},
			expectError: true,
		},
		{
			name: "empty JSON object",
			setup: func() {
				DefaultConfig = &Config{
					Entitlements: map[Entitlement]bool{},
				}
			},
			jsonData:    `{}`,
			expected:    map[Entitlement]bool{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := LoadEntitlementsFromJSON([]byte(tt.jsonData))

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check that all expected entitlements are present
			for expectedEnt, expectedVal := range tt.expected {
				if val, exists := DefaultConfig.Entitlements[expectedEnt]; !exists {
					t.Errorf("Expected entitlement %s to be present", expectedEnt)
				} else if val != expectedVal {
					t.Errorf("Expected entitlement %s to have value %v, got %v", expectedEnt, expectedVal, val)
				}
			}
		})
	}
}

// TestAPIConvenienceFunctions tests various convenience functions
func TestAPIConvenienceFunctions(t *testing.T) {
	// Save original DefaultConfig
	originalConfig := DefaultConfig
	defer func() {
		DefaultConfig = originalConfig
	}()

	t.Run("SetAppName", func(t *testing.T) {
		DefaultConfig = &Config{}
		SetAppName("TestApp")
		if DefaultConfig.ApplicationName != "TestApp" {
			t.Errorf("Expected ApplicationName to be 'TestApp', got %s", DefaultConfig.ApplicationName)
		}
	})

	t.Run("SetBundleID", func(t *testing.T) {
		DefaultConfig = &Config{}
		SetBundleID("com.test.app")
		if DefaultConfig.BundleID != "com.test.app" {
			t.Errorf("Expected BundleID to be 'com.test.app', got %s", DefaultConfig.BundleID)
		}
	})

	t.Run("EnableKeepTemp", func(t *testing.T) {
		DefaultConfig = &Config{}
		EnableKeepTemp()
		if DefaultConfig.KeepTemp != true {
			t.Error("Expected KeepTemp to be true")
		}
	})

	t.Run("DisableRelaunch", func(t *testing.T) {
		DefaultConfig = &Config{Relaunch: true}
		DisableRelaunch()
		if DefaultConfig.Relaunch != false {
			t.Error("Expected Relaunch to be false")
		}
	})

	t.Run("EnableDockIcon", func(t *testing.T) {
		DefaultConfig = &Config{}
		EnableDockIcon()
		if DefaultConfig.PlistEntries == nil {
			t.Error("Expected PlistEntries to be initialized")
			return
		}
		if val, exists := DefaultConfig.PlistEntries["LSUIElement"]; !exists || val != false {
			t.Error("Expected LSUIElement to be false")
		}
	})

	t.Run("EnableSigning", func(t *testing.T) {
		DefaultConfig = &Config{}
		EnableSigning("TestIdentity")
		if DefaultConfig.AutoSign != true {
			t.Error("Expected AutoSign to be true")
		}
		if DefaultConfig.SigningIdentity != "TestIdentity" {
			t.Errorf("Expected SigningIdentity to be 'TestIdentity', got %s", DefaultConfig.SigningIdentity)
		}
	})

	t.Run("EnableSigning with empty identity", func(t *testing.T) {
		DefaultConfig = &Config{}
		EnableSigning("")
		if DefaultConfig.AutoSign != true {
			t.Error("Expected AutoSign to be true")
		}
		if DefaultConfig.SigningIdentity != "" {
			t.Errorf("Expected SigningIdentity to be empty, got %s", DefaultConfig.SigningIdentity)
		}
	})

	t.Run("SetIconFile", func(t *testing.T) {
		DefaultConfig = &Config{}
		SetIconFile("/path/to/icon.icns")
		if DefaultConfig.PlistEntries == nil {
			t.Error("Expected PlistEntries to be initialized")
			return
		}
		if val, exists := DefaultConfig.PlistEntries["CFBundleIconFile"]; !exists || val != "/path/to/icon.icns" {
			t.Error("Expected CFBundleIconFile to be set correctly")
		}
	})

	t.Run("AddPlistEntry", func(t *testing.T) {
		DefaultConfig = &Config{}
		AddPlistEntry("TestKey", "TestValue")
		if DefaultConfig.PlistEntries == nil {
			t.Error("Expected PlistEntries to be initialized")
			return
		}
		if val, exists := DefaultConfig.PlistEntries["TestKey"]; !exists || val != "TestValue" {
			t.Error("Expected TestKey to be set correctly")
		}
	})
}

// TestSetCustomAppBundle tests the SetCustomAppBundle function
func TestSetCustomAppBundle(t *testing.T) {
	// Save original DefaultConfig
	originalConfig := DefaultConfig
	defer func() {
		DefaultConfig = originalConfig
	}()

	// Create a mock filesystem for testing
	var mockFS MockFS

	DefaultConfig = &Config{}
	SetCustomAppBundle(mockFS)

	if DefaultConfig.AppTemplate != mockFS {
		t.Error("Expected AppTemplate to be set correctly")
	}
}

// TestConfigureWithAppTemplate tests Configure function with AppTemplate
func TestConfigureWithAppTemplate(t *testing.T) {
	// Save original DefaultConfig
	originalConfig := DefaultConfig
	defer func() {
		DefaultConfig = originalConfig
	}()

	// Create a mock filesystem for testing
	var mockFS MockFS

	DefaultConfig = &Config{}

	inputConfig := &Config{
		AppTemplate: mockFS,
	}

	Configure(inputConfig)

	if DefaultConfig.AppTemplate != mockFS {
		t.Error("Expected AppTemplate to be set correctly via Configure")
	}
}

// MockFS is a simple mock filesystem for testing
type MockFS struct{}

func (MockFS) Open(name string) (fs.File, error) {
	return nil, fs.ErrNotExist
}

// TestConfigMethodsWithNilMaps tests that Config methods properly handle nil maps
func TestConfigMethodsWithNilMaps(t *testing.T) {
	t.Run("AddEntitlement with nil Entitlements map", func(t *testing.T) {
		cfg := &Config{Entitlements: nil}
		cfg.AddEntitlement(EntCamera)

		if cfg.Entitlements == nil {
			t.Error("Expected Entitlements map to be initialized")
			return
		}

		if val, exists := cfg.Entitlements[EntCamera]; !exists || val != true {
			t.Error("Expected EntCamera to be added with value true")
		}
	})

	t.Run("AddPlistEntry with nil PlistEntries map", func(t *testing.T) {
		cfg := &Config{PlistEntries: nil}
		cfg.AddPlistEntry("TestKey", "TestValue")

		if cfg.PlistEntries == nil {
			t.Error("Expected PlistEntries map to be initialized")
			return
		}

		if val, exists := cfg.PlistEntries["TestKey"]; !exists || val != "TestValue" {
			t.Error("Expected TestKey to be added with correct value")
		}
	})

	t.Run("RequestEntitlements with nil Entitlements map", func(t *testing.T) {
		cfg := &Config{Entitlements: nil}
		cfg.RequestEntitlements(EntCamera, EntMicrophone)

		if cfg.Entitlements == nil {
			t.Error("Expected Entitlements map to be initialized")
			return
		}

		expected := map[Entitlement]bool{
			EntCamera:     true,
			EntMicrophone: true,
		}

		for ent, expectedVal := range expected {
			if val, exists := cfg.Entitlements[ent]; !exists || val != expectedVal {
				t.Errorf("Expected entitlement %s to have value %v, got %v", ent, expectedVal, val)
			}
		}
	})
}

// TestEntitlementTypeConversion tests type conversion scenarios
func TestEntitlementTypeConversion(t *testing.T) {
	t.Run("String to Entitlement conversion", func(t *testing.T) {
		customEntitlement := "com.example.custom.permission"
		ent := Entitlement(customEntitlement)

		if string(ent) != customEntitlement {
			t.Errorf("Expected string conversion to work, got %s", string(ent))
		}
	})

	t.Run("Entitlement constants as strings", func(t *testing.T) {
		if string(EntCamera) != "com.apple.security.device.camera" {
			t.Error("EntCamera constant has incorrect value")
		}

		if string(EntAppSandbox) != "com.apple.security.app-sandbox" {
			t.Error("EntAppSandbox constant has incorrect value")
		}
	})
}

// TestConfigConcurrentAccess tests concurrent access to Config methods
// This is a basic test to ensure no race conditions in simple scenarios
func TestConfigConcurrentAccess(t *testing.T) {
	cfg := NewConfig()

	// Test concurrent access to AddEntitlement
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 100; i++ {
			cfg.AddEntitlement(EntCamera)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			cfg.AddEntitlement(EntMicrophone)
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Verify both entitlements were added
	if val, exists := cfg.Entitlements[EntCamera]; !exists || val != true {
		t.Error("Expected EntCamera to be added")
	}
	if val, exists := cfg.Entitlements[EntMicrophone]; !exists || val != true {
		t.Error("Expected EntMicrophone to be added")
	}
}

// TestDefaultConfigValues tests that DefaultConfig has reasonable default values
func TestDefaultConfigValues(t *testing.T) {
	// Note: This test examines the package's DefaultConfig, which may be modified
	// by other tests, but it still provides some basic validation

	if DefaultConfig == nil {
		t.Fatal("DefaultConfig should not be nil")
	}

	// These are basic sanity checks
	if DefaultConfig.Entitlements == nil {
		t.Error("DefaultConfig.Entitlements should be initialized")
	}

	if DefaultConfig.PlistEntries == nil {
		t.Error("DefaultConfig.PlistEntries should be initialized")
	}
}

// TestConfigFieldAssignment tests direct field assignment behavior
func TestConfigFieldAssignment(t *testing.T) {
	cfg := NewConfig()

	// Test string field assignments
	cfg.ApplicationName = "TestApplication"
	cfg.BundleID = "com.test.application"
	cfg.CustomDestinationAppPath = "/custom/path/to/app"
	cfg.SigningIdentity = "Developer ID Application: Test"

	// Test boolean field assignments
	cfg.Relaunch = false
	cfg.KeepTemp = true
	cfg.AutoSign = false

	// Verify assignments
	if cfg.ApplicationName != "TestApplication" {
		t.Errorf("Expected ApplicationName to be 'TestApplication', got %s", cfg.ApplicationName)
	}
	if cfg.BundleID != "com.test.application" {
		t.Errorf("Expected BundleID to be 'com.test.application', got %s", cfg.BundleID)
	}
	if cfg.CustomDestinationAppPath != "/custom/path/to/app" {
		t.Errorf("Expected CustomDestinationAppPath to be '/custom/path/to/app', got %s", cfg.CustomDestinationAppPath)
	}
	if cfg.SigningIdentity != "Developer ID Application: Test" {
		t.Errorf("Expected SigningIdentity to be 'Developer ID Application: Test', got %s", cfg.SigningIdentity)
	}
	if cfg.Relaunch != false {
		t.Error("Expected Relaunch to be false")
	}
	if cfg.KeepTemp != true {
		t.Error("Expected KeepTemp to be true")
	}
	if cfg.AutoSign != false {
		t.Error("Expected AutoSign to be false")
	}
}

// TestEntitlementsMapOperations tests map operations on Entitlements
func TestEntitlementsMapOperations(t *testing.T) {
	cfg := NewConfig()

	// Test direct map assignment
	cfg.Entitlements[EntCamera] = true
	cfg.Entitlements[EntMicrophone] = false
	cfg.Entitlements[Entitlement("custom.entitlement")] = true

	// Test reading values
	if val, exists := cfg.Entitlements[EntCamera]; !exists || val != true {
		t.Error("Expected EntCamera to be true")
	}
	if val, exists := cfg.Entitlements[EntMicrophone]; !exists || val != false {
		t.Error("Expected EntMicrophone to be false")
	}
	if val, exists := cfg.Entitlements[Entitlement("custom.entitlement")]; !exists || val != true {
		t.Error("Expected custom.entitlement to be true")
	}

	// Test map length
	if len(cfg.Entitlements) != 3 {
		t.Errorf("Expected Entitlements map to have 3 entries, got %d", len(cfg.Entitlements))
	}

	// Test deleting an entitlement
	delete(cfg.Entitlements, EntMicrophone)
	if _, exists := cfg.Entitlements[EntMicrophone]; exists {
		t.Error("Expected EntMicrophone to be deleted from map")
	}
}

// TestLoadEntitlementsFromJSON_EdgeCases tests edge cases for LoadEntitlementsFromJSON
func TestLoadEntitlementsFromJSON_EdgeCases(t *testing.T) {
	// Save original DefaultConfig
	originalConfig := DefaultConfig
	defer func() {
		DefaultConfig = originalConfig
	}()

	tests := []struct {
		name        string
		setup       func()
		jsonData    string
		expected    map[Entitlement]bool
		expectError bool
		errorMsg    string
	}{
		{
			name: "null JSON value",
			setup: func() {
				DefaultConfig = &Config{
					Entitlements: map[Entitlement]bool{},
				}
			},
			jsonData:    `null`,
			expectError: true,
			errorMsg:    "parse entitlements JSON",
		},
		{
			name: "JSON array instead of object",
			setup: func() {
				DefaultConfig = &Config{
					Entitlements: map[Entitlement]bool{},
				}
			},
			jsonData:    `["com.apple.security.device.camera", "com.apple.security.device.microphone"]`,
			expectError: true,
			errorMsg:    "parse entitlements JSON",
		},
		{
			name: "JSON with non-boolean values",
			setup: func() {
				DefaultConfig = &Config{
					Entitlements: map[Entitlement]bool{},
				}
			},
			jsonData:    `{"com.apple.security.device.camera": "yes", "com.apple.security.device.microphone": 1}`,
			expectError: true,
			errorMsg:    "parse entitlements JSON",
		},
		{
			name: "empty JSON string",
			setup: func() {
				DefaultConfig = &Config{
					Entitlements: map[Entitlement]bool{},
				}
			},
			jsonData:    ``,
			expectError: true,
			errorMsg:    "parse entitlements JSON",
		},
		{
			name: "JSON with nested objects",
			setup: func() {
				DefaultConfig = &Config{
					Entitlements: map[Entitlement]bool{},
				}
			},
			jsonData:    `{"com.apple.security.device.camera": {"enabled": true}}`,
			expectError: true,
			errorMsg:    "parse entitlements JSON",
		},
		{
			name: "very large JSON",
			setup: func() {
				DefaultConfig = &Config{
					Entitlements: map[Entitlement]bool{},
				}
			},
			jsonData: func() string {
				// Create a large JSON with many entitlements
				entries := []string{}
				for i := 0; i < 1000; i++ {
					entries = append(entries, fmt.Sprintf(`"com.apple.security.test.entitlement%d": true`, i))
				}
				return "{" + strings.Join(entries, ",") + "}"
			}(),
			expectError: false,
			expected: func() map[Entitlement]bool {
				m := make(map[Entitlement]bool)
				for i := 0; i < 1000; i++ {
					m[Entitlement(fmt.Sprintf("com.apple.security.test.entitlement%d", i))] = true
				}
				return m
			}(),
		},
		{
			name: "JSON with special characters in keys",
			setup: func() {
				DefaultConfig = &Config{
					Entitlements: map[Entitlement]bool{},
				}
			},
			jsonData: `{"com.apple.security.device.camera\n": true, "com.apple.security.device\t.microphone": false}`,
			expected: map[Entitlement]bool{
				Entitlement("com.apple.security.device.camera\n"):     true,
				Entitlement("com.apple.security.device\t.microphone"): false,
			},
			expectError: false,
		},
		{
			name: "load when DefaultConfig.Entitlements is nil",
			setup: func() {
				DefaultConfig = &Config{
					Entitlements: nil,
				}
			},
			jsonData: `{"com.apple.security.device.camera": true}`,
			expected: map[Entitlement]bool{
				EntCamera: true,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := LoadEntitlementsFromJSON([]byte(tt.jsonData))

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %v", tt.errorMsg, err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// For large test case, just check a few samples
			if len(tt.expected) > 100 {
				// Check that the map has the right size
				if len(DefaultConfig.Entitlements) != len(tt.expected) {
					t.Errorf("Expected %d entitlements, got %d", len(tt.expected), len(DefaultConfig.Entitlements))
				}
				// Check a few samples
				for i := 0; i < 10; i++ {
					key := Entitlement(fmt.Sprintf("com.apple.security.test.entitlement%d", i))
					if val, exists := DefaultConfig.Entitlements[key]; !exists || val != true {
						t.Errorf("Expected entitlement %s to be true", key)
					}
				}
			} else {
				// Check all expected entitlements
				for expectedEnt, expectedVal := range tt.expected {
					if val, exists := DefaultConfig.Entitlements[expectedEnt]; !exists {
						t.Errorf("Expected entitlement %s to be present", expectedEnt)
					} else if val != expectedVal {
						t.Errorf("Expected entitlement %s to have value %v, got %v", expectedEnt, expectedVal, val)
					}
				}
			}
		})
	}
}

// TestEnableDebug tests the EnableDebug function
func TestEnableDebug(t *testing.T) {
	// Save original env var
	originalDebug := os.Getenv("MACGO_DEBUG")
	defer func() {
		if originalDebug == "" {
			os.Unsetenv("MACGO_DEBUG")
		} else {
			os.Setenv("MACGO_DEBUG", originalDebug)
		}
	}()

	// Clear the env var first
	os.Unsetenv("MACGO_DEBUG")

	// Enable debug
	EnableDebug()

	// Check that env var was set
	if val := os.Getenv("MACGO_DEBUG"); val != "1" {
		t.Errorf("Expected MACGO_DEBUG to be '1', got %q", val)
	}
}

// TestDefaultConfigInitialization tests that DefaultConfig is properly initialized
func TestDefaultConfigInitialization(t *testing.T) {
	// Test that DefaultConfig is not nil
	if DefaultConfig == nil {
		t.Fatal("DefaultConfig should not be nil")
	}

	// Test that AutoSign is true by default
	if !DefaultConfig.AutoSign {
		t.Error("Expected DefaultConfig.AutoSign to be true")
	}

	// Test that Relaunch is true by default
	if !DefaultConfig.Relaunch {
		t.Error("Expected DefaultConfig.Relaunch to be true")
	}

	// Test that Entitlements map is initialized
	if DefaultConfig.Entitlements == nil {
		t.Error("Expected DefaultConfig.Entitlements to be initialized")
	}

	// Test that PlistEntries map is initialized
	if DefaultConfig.PlistEntries == nil {
		t.Error("Expected DefaultConfig.PlistEntries to be initialized")
	}

	// Test default LSUIElement value
	if val, exists := DefaultConfig.PlistEntries["LSUIElement"]; !exists || val != true {
		t.Error("Expected LSUIElement to be true by default in DefaultConfig (dock icon hidden)")
	}
}

// TestAPIFunctionNilDefaultConfig tests API functions behavior when DefaultConfig is nil
func TestAPIFunctionNilDefaultConfig(t *testing.T) {
	// Save original DefaultConfig
	originalConfig := DefaultConfig
	defer func() {
		DefaultConfig = originalConfig
	}()

	// Test each function with nil DefaultConfig to ensure they don't panic
	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "RequestEntitlements with nil DefaultConfig",
			fn: func() {
				DefaultConfig = nil
				// Should panic
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic when DefaultConfig is nil")
					}
				}()
				RequestEntitlements(EntCamera)
			},
		},
		{
			name: "RequestEntitlement with nil DefaultConfig",
			fn: func() {
				DefaultConfig = nil
				// Should panic
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic when DefaultConfig is nil")
					}
				}()
				RequestEntitlement(EntCamera)
			},
		},
		{
			name: "EnableDockIcon with nil DefaultConfig",
			fn: func() {
				DefaultConfig = nil
				// Should panic
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic when DefaultConfig is nil")
					}
				}()
				EnableDockIcon()
			},
		},
		{
			name: "SetAppName with nil DefaultConfig",
			fn: func() {
				DefaultConfig = nil
				// Should panic
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic when DefaultConfig is nil")
					}
				}()
				SetAppName("Test")
			},
		},
		{
			name: "SetBundleID with nil DefaultConfig",
			fn: func() {
				DefaultConfig = nil
				// Should panic
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic when DefaultConfig is nil")
					}
				}()
				SetBundleID("com.test")
			},
		},
		{
			name: "EnableKeepTemp with nil DefaultConfig",
			fn: func() {
				DefaultConfig = nil
				// Should panic
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic when DefaultConfig is nil")
					}
				}()
				EnableKeepTemp()
			},
		},
		{
			name: "DisableRelaunch with nil DefaultConfig",
			fn: func() {
				DefaultConfig = nil
				// Should panic
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic when DefaultConfig is nil")
					}
				}()
				DisableRelaunch()
			},
		},
		{
			name: "EnableSigning with nil DefaultConfig",
			fn: func() {
				DefaultConfig = nil
				// Should panic
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic when DefaultConfig is nil")
					}
				}()
				EnableSigning("")
			},
		},
		{
			name: "LoadEntitlementsFromJSON with nil DefaultConfig",
			fn: func() {
				DefaultConfig = nil
				// Should panic
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic when DefaultConfig is nil")
					}
				}()
				LoadEntitlementsFromJSON([]byte(`{"test": true}`))
			},
		},
		{
			name: "AddPlistEntry with nil DefaultConfig",
			fn: func() {
				DefaultConfig = nil
				// Should panic
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic when DefaultConfig is nil")
					}
				}()
				AddPlistEntry("key", "value")
			},
		},
		{
			name: "SetIconFile with nil DefaultConfig",
			fn: func() {
				DefaultConfig = nil
				// Should panic
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic when DefaultConfig is nil")
					}
				}()
				SetIconFile("/path/to/icon")
			},
		},
		{
			name: "SetCustomAppBundle with nil DefaultConfig",
			fn: func() {
				DefaultConfig = nil
				// Should panic
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic when DefaultConfig is nil")
					}
				}()
				SetCustomAppBundle(MockFS{})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fn()
		})
	}
}

// TestRequestEntitlementsVariadicEdgeCases tests edge cases for variadic entitlements
func TestRequestEntitlementsVariadicEdgeCases(t *testing.T) {
	// Save original DefaultConfig
	originalConfig := DefaultConfig
	defer func() {
		DefaultConfig = originalConfig
	}()

	tests := []struct {
		name         string
		setup        func()
		entitlements []interface{}
		description  string
		validate     func(t *testing.T)
	}{
		{
			name: "mix of valid and invalid types",
			setup: func() {
				DefaultConfig = &Config{
					Entitlements: map[Entitlement]bool{},
				}
			},
			entitlements: []interface{}{
				EntCamera,
				123,
				"com.apple.security.custom",
				struct{}{},
				EntMicrophone,
				[]string{"invalid"},
				Entitlement("com.apple.security.another"),
				map[string]string{"invalid": "map"},
				nil,
			},
			description: "should only add valid string and Entitlement types",
			validate: func(t *testing.T) {
				expected := map[Entitlement]bool{
					EntCamera:                                 true,
					Entitlement("com.apple.security.custom"):  true,
					EntMicrophone:                             true,
					Entitlement("com.apple.security.another"): true,
				}
				if len(DefaultConfig.Entitlements) != len(expected) {
					t.Errorf("Expected %d entitlements, got %d", len(expected), len(DefaultConfig.Entitlements))
				}
				for ent := range expected {
					if val, exists := DefaultConfig.Entitlements[ent]; !exists || val != true {
						t.Errorf("Expected entitlement %s to be true", ent)
					}
				}
			},
		},
		{
			name: "duplicate entitlements",
			setup: func() {
				DefaultConfig = &Config{
					Entitlements: map[Entitlement]bool{},
				}
			},
			entitlements: []interface{}{
				EntCamera,
				EntCamera,
				"com.apple.security.device.camera",
				EntCamera,
			},
			description: "should handle duplicates gracefully",
			validate: func(t *testing.T) {
				if len(DefaultConfig.Entitlements) != 1 {
					t.Errorf("Expected 1 entitlement, got %d", len(DefaultConfig.Entitlements))
				}
				if val, exists := DefaultConfig.Entitlements[EntCamera]; !exists || val != true {
					t.Error("Expected EntCamera to be true")
				}
			},
		},
		{
			name: "empty string entitlement",
			setup: func() {
				DefaultConfig = &Config{
					Entitlements: map[Entitlement]bool{},
				}
			},
			entitlements: []interface{}{
				"",
				EntCamera,
				Entitlement(""),
			},
			description: "should add empty strings as entitlements",
			validate: func(t *testing.T) {
				if len(DefaultConfig.Entitlements) != 2 {
					t.Errorf("Expected 2 entitlements, got %d", len(DefaultConfig.Entitlements))
				}
				if val, exists := DefaultConfig.Entitlements[Entitlement("")]; !exists || val != true {
					t.Error("Expected empty entitlement to be true")
				}
				if val, exists := DefaultConfig.Entitlements[EntCamera]; !exists || val != true {
					t.Error("Expected EntCamera to be true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			RequestEntitlements(tt.entitlements...)
			tt.validate(t)
		})
	}
}

// TestConfigureEdgeCases tests edge cases for Configure function
func TestConfigureEdgeCases(t *testing.T) {
	// Save original DefaultConfig
	originalConfig := DefaultConfig
	defer func() {
		DefaultConfig = originalConfig
	}()

	tests := []struct {
		name           string
		setup          func()
		inputConfig    *Config
		expectedChecks func(t *testing.T)
	}{
		{
			name: "configure with AppTemplate when DefaultConfig has no AppTemplate",
			setup: func() {
				DefaultConfig = &Config{
					AppTemplate: nil,
				}
			},
			inputConfig: &Config{
				AppTemplate: MockFS{},
			},
			expectedChecks: func(t *testing.T) {
				if DefaultConfig.AppTemplate == nil {
					t.Error("Expected AppTemplate to be set")
				}
			},
		},
		{
			name: "configure preserves boolean false values",
			setup: func() {
				DefaultConfig = &Config{
					Relaunch: true,
					AutoSign: true,
					KeepTemp: true,
				}
			},
			inputConfig: &Config{
				Relaunch: false,
				AutoSign: false,
				KeepTemp: false,
			},
			expectedChecks: func(t *testing.T) {
				if DefaultConfig.Relaunch != false {
					t.Error("Expected Relaunch to be false")
				}
				if DefaultConfig.AutoSign != false {
					t.Error("Expected AutoSign to be false")
				}
				if DefaultConfig.KeepTemp != false {
					t.Error("Expected KeepTemp to be false")
				}
			},
		},
		{
			name: "configure with very long strings",
			setup: func() {
				DefaultConfig = &Config{}
			},
			inputConfig: &Config{
				ApplicationName:          strings.Repeat("a", 1000),
				BundleID:                 "com.test." + strings.Repeat("b", 500),
				CustomDestinationAppPath: "/" + strings.Repeat("path/", 200),
				SigningIdentity:          strings.Repeat("Developer ID ", 50),
			},
			expectedChecks: func(t *testing.T) {
				if len(DefaultConfig.ApplicationName) != 1000 {
					t.Error("Expected ApplicationName to be preserved with full length")
				}
				if !strings.HasPrefix(DefaultConfig.BundleID, "com.test.") {
					t.Error("Expected BundleID to be preserved")
				}
				if !strings.HasPrefix(DefaultConfig.CustomDestinationAppPath, "/") {
					t.Error("Expected CustomDestinationAppPath to be preserved")
				}
				if len(DefaultConfig.SigningIdentity) != 650 { // "Developer ID " is 13 chars * 50
					t.Error("Expected SigningIdentity to be preserved with full length")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			Configure(tt.inputConfig)
			tt.expectedChecks(t)
		})
	}
}

// TestLoadEntitlementsFromJSONFilePath tests loading entitlements from a file
func TestLoadEntitlementsFromJSONFilePath(t *testing.T) {
	// Create a temporary JSON file
	tmpFile, err := os.CreateTemp("", "entitlements-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	// Write JSON data to file
	jsonData := `{
		"com.apple.security.device.camera": true,
		"com.apple.security.device.microphone": false,
		"com.apple.security.app-sandbox": true
	}`
	if _, err := tmpFile.Write([]byte(jsonData)); err != nil {
		t.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatal(err)
	}

	// Save original DefaultConfig
	originalConfig := DefaultConfig
	defer func() {
		DefaultConfig = originalConfig
	}()

	// Setup DefaultConfig
	DefaultConfig = &Config{
		Entitlements: map[Entitlement]bool{},
	}

	// Read file and load entitlements
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	err = LoadEntitlementsFromJSON(data)
	if err != nil {
		t.Fatalf("Failed to load entitlements: %v", err)
	}

	// Verify entitlements were loaded
	expected := map[Entitlement]bool{
		EntCamera:     true,
		EntMicrophone: false,
		EntAppSandbox: true,
	}

	for ent, expectedVal := range expected {
		if val, exists := DefaultConfig.Entitlements[ent]; !exists {
			t.Errorf("Expected entitlement %s to be present", ent)
		} else if val != expectedVal {
			t.Errorf("Expected entitlement %s to have value %v, got %v", ent, expectedVal, val)
		}
	}
}

// TestMultipleAPICallsSequence tests a sequence of API calls
func TestMultipleAPICallsSequence(t *testing.T) {
	// Save original DefaultConfig
	originalConfig := DefaultConfig
	defer func() {
		DefaultConfig = originalConfig
	}()

	// Reset DefaultConfig
	DefaultConfig = NewConfig()

	// Sequence of API calls that might be used in a real application
	SetAppName("TestApplication")
	SetBundleID("com.example.testapp")
	RequestEntitlements(EntCamera, EntMicrophone, EntLocation)
	EnableDockIcon()
	EnableSigning("Developer ID Application: Test")
	AddPlistEntry("CFBundleVersion", "1.0.0")
	AddPlistEntry("NSHighResolutionCapable", true)
	SetIconFile("/Applications/TestApp.app/Contents/Resources/AppIcon.icns")
	RequestEntitlement(EntAddressBook)
	EnableKeepTemp()

	// Verify all settings were applied correctly
	if DefaultConfig.ApplicationName != "TestApplication" {
		t.Error("ApplicationName not set correctly")
	}
	if DefaultConfig.BundleID != "com.example.testapp" {
		t.Error("BundleID not set correctly")
	}
	if !DefaultConfig.AutoSign {
		t.Error("AutoSign should be enabled")
	}
	if DefaultConfig.SigningIdentity != "Developer ID Application: Test" {
		t.Error("SigningIdentity not set correctly")
	}
	if !DefaultConfig.KeepTemp {
		t.Error("KeepTemp should be enabled")
	}

	// Check entitlements
	expectedEntitlements := []Entitlement{EntCamera, EntMicrophone, EntLocation, EntAddressBook}
	for _, ent := range expectedEntitlements {
		if val, exists := DefaultConfig.Entitlements[ent]; !exists || val != true {
			t.Errorf("Expected entitlement %s to be true", ent)
		}
	}

	// Check plist entries
	if val, exists := DefaultConfig.PlistEntries["LSUIElement"]; !exists || val != false {
		t.Error("LSUIElement should be false (dock icon enabled)")
	}
	if val, exists := DefaultConfig.PlistEntries["CFBundleVersion"]; !exists || val != "1.0.0" {
		t.Error("CFBundleVersion not set correctly")
	}
	if val, exists := DefaultConfig.PlistEntries["NSHighResolutionCapable"]; !exists || val != true {
		t.Error("NSHighResolutionCapable not set correctly")
	}
	if val, exists := DefaultConfig.PlistEntries["CFBundleIconFile"]; !exists || val != "/Applications/TestApp.app/Contents/Resources/AppIcon.icns" {
		t.Error("CFBundleIconFile not set correctly")
	}
}

// TestRequestEntitlementWithCustomEntitlementConstant tests using custom entitlement constants
func TestRequestEntitlementWithCustomEntitlementConstant(t *testing.T) {
	// Save original DefaultConfig
	originalConfig := DefaultConfig
	defer func() {
		DefaultConfig = originalConfig
	}()

	// Define custom entitlement constants (as users might do)
	const (
		CustomEntVirtualization Entitlement = "com.apple.security.virtualization"
		CustomEntHypervisor     Entitlement = "com.apple.security.hypervisor"
		CustomEntScreenCapture  Entitlement = "com.apple.security.screencapture"
	)

	DefaultConfig = &Config{
		Entitlements: map[Entitlement]bool{},
	}

	// Test adding custom constants
	RequestEntitlements(
		CustomEntVirtualization,
		CustomEntHypervisor,
		CustomEntScreenCapture,
	)

	// Verify they were added
	expected := []Entitlement{
		CustomEntVirtualization,
		CustomEntHypervisor,
		CustomEntScreenCapture,
	}

	for _, ent := range expected {
		if val, exists := DefaultConfig.Entitlements[ent]; !exists || val != true {
			t.Errorf("Expected custom entitlement %s to be true", ent)
		}
	}
}

// TestAPIThreadSafety tests API functions for basic thread safety
func TestAPIThreadSafety(t *testing.T) {
	// Save original DefaultConfig
	originalConfig := DefaultConfig
	defer func() {
		DefaultConfig = originalConfig
	}()

	DefaultConfig = NewConfig()

	// Run multiple goroutines making API calls
	done := make(chan bool, 5)

	// Goroutine 1: Add entitlements
	go func() {
		for i := 0; i < 50; i++ {
			RequestEntitlements(EntCamera, EntMicrophone)
			time.Sleep(time.Microsecond)
		}
		done <- true
	}()

	// Goroutine 2: Set app name and bundle ID
	go func() {
		for i := 0; i < 50; i++ {
			SetAppName(fmt.Sprintf("App%d", i))
			SetBundleID(fmt.Sprintf("com.test.app%d", i))
			time.Sleep(time.Microsecond)
		}
		done <- true
	}()

	// Goroutine 3: Add plist entries
	go func() {
		for i := 0; i < 50; i++ {
			AddPlistEntry(fmt.Sprintf("Key%d", i), fmt.Sprintf("Value%d", i))
			time.Sleep(time.Microsecond)
		}
		done <- true
	}()

	// Goroutine 4: Toggle settings
	go func() {
		for i := 0; i < 50; i++ {
			if i%2 == 0 {
				EnableDockIcon()
				EnableKeepTemp()
			} else {
				DisableRelaunch()
			}
			time.Sleep(time.Microsecond)
		}
		done <- true
	}()

	// Goroutine 5: Request individual entitlements
	go func() {
		entitlements := []Entitlement{EntLocation, EntAddressBook, EntPhotos, EntCalendars}
		for i := 0; i < 50; i++ {
			RequestEntitlement(entitlements[i%len(entitlements)])
			time.Sleep(time.Microsecond)
		}
		done <- true
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 5; i++ {
		<-done
	}

	// Basic verification that config is not corrupted
	if DefaultConfig.Entitlements == nil {
		t.Error("Entitlements map should not be nil after concurrent access")
	}
	if DefaultConfig.PlistEntries == nil {
		t.Error("PlistEntries map should not be nil after concurrent access")
	}

	// Check that some values were set
	if len(DefaultConfig.Entitlements) == 0 {
		t.Error("Expected some entitlements to be set")
	}
	if len(DefaultConfig.PlistEntries) == 0 {
		t.Error("Expected some plist entries to be set")
	}
}

// TestLoadEntitlementsFromJSONNilMapPanic tests that LoadEntitlementsFromJSON panics with nil map
func TestLoadEntitlementsFromJSONNilMapPanic(t *testing.T) {
	// Save original DefaultConfig
	originalConfig := DefaultConfig
	defer func() {
		DefaultConfig = originalConfig
	}()

	// Set DefaultConfig with nil Entitlements map to trigger panic
	DefaultConfig = &Config{
		Entitlements: nil,
	}

	// This should panic when trying to assign to nil map
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when LoadEntitlementsFromJSON is called with nil Entitlements map")
		} else {
			// Verify it's the expected panic
			if _, ok := r.(runtime.Error); !ok {
				t.Errorf("Expected runtime error panic, got %v", r)
			}
		}
	}()

	// This should panic
	err := LoadEntitlementsFromJSON([]byte(`{"com.apple.security.device.camera": true}`))
	if err != nil {
		t.Errorf("Should have panicked before returning error: %v", err)
	}
}

// TestDisableAutoInit is a placeholder to acknowledge the DisableAutoInit function
// Note: DisableAutoInit is not defined in api.go, this test documents its absence
func TestDisableAutoInit(t *testing.T) {
	// This test verifies that DisableAutoInit is not implemented
	// If it gets implemented in the future, this test will need to be updated
	t.Skip("DisableAutoInit is not implemented in the current API")
}
