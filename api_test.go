package macgo

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"sync"
	"testing"
	"testing/fstest"
)

// Helper function to reset DefaultConfig state between tests
func resetDefaultConfig() {
	configMutex.Lock()
	defer configMutex.Unlock()
	
	DefaultConfig = &Config{
		ApplicationName:  "",
		BundleID:        "",
		Entitlements:    nil,
		PlistEntries:    nil,
		AppTemplate:     nil,
		Relaunch:        true,
		KeepTemp:        false,
		AutoSign:        false,
		SigningIdentity: "",
	}
}

func TestRequestEntitlements(t *testing.T) {
	resetDefaultConfig()
	
	tests := []struct {
		name         string
		entitlements []interface{}
		expected     map[Entitlement]bool
	}{
		{
			name:         "Single string entitlement",
			entitlements: []interface{}{"com.apple.security.camera"},
			expected: map[Entitlement]bool{
				"com.apple.security.camera": true,
			},
		},
		{
			name:         "Single Entitlement type",
			entitlements: []interface{}{EntCamera},
			expected: map[Entitlement]bool{
				EntCamera: true,
			},
		},
		{
			name: "Multiple mixed entitlements",
			entitlements: []interface{}{
				EntCamera,
				"com.apple.security.microphone",
				EntAppSandbox,
			},
			expected: map[Entitlement]bool{
				EntCamera:     true,
				"com.apple.security.microphone": true,
				EntAppSandbox: true,
			},
		},
		{
			name:         "Invalid type (should be skipped)",
			entitlements: []interface{}{123, EntCamera, "valid"},
			expected: map[Entitlement]bool{
				EntCamera: true,
				"valid":   true,
			},
		},
		{
			name:         "Empty entitlements",
			entitlements: []interface{}{},
			expected:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetDefaultConfig()
			
			RequestEntitlements(tt.entitlements...)
			
			if !reflect.DeepEqual(DefaultConfig.Entitlements, tt.expected) {
				t.Errorf("Expected entitlements %v, got %v", tt.expected, DefaultConfig.Entitlements)
			}
		})
	}
}

func TestRequestEntitlement(t *testing.T) {
	resetDefaultConfig()
	
	tests := []struct {
		name        string
		entitlement interface{}
		expected    map[Entitlement]bool
	}{
		{
			name:        "String entitlement",
			entitlement: "com.apple.security.camera",
			expected: map[Entitlement]bool{
				"com.apple.security.camera": true,
			},
		},
		{
			name:        "Entitlement type",
			entitlement: EntMicrophone,
			expected: map[Entitlement]bool{
				EntMicrophone: true,
			},
		},
		{
			name:        "Invalid type (should be ignored)",
			entitlement: 123,
			expected:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetDefaultConfig()
			
			RequestEntitlement(tt.entitlement)
			
			if !reflect.DeepEqual(DefaultConfig.Entitlements, tt.expected) {
				t.Errorf("Expected entitlements %v, got %v", tt.expected, DefaultConfig.Entitlements)
			}
		})
	}
}

func TestEnableDockIcon(t *testing.T) {
	resetDefaultConfig()
	
	EnableDockIcon()
	
	if DefaultConfig.PlistEntries == nil {
		t.Error("Expected PlistEntries to be initialized")
	}
	
	if val, ok := DefaultConfig.PlistEntries["LSUIElement"]; !ok || val != false {
		t.Errorf("Expected LSUIElement to be false, got %v", val)
	}
}

func TestSetAppName(t *testing.T) {
	resetDefaultConfig()
	
	testName := "TestApplication"
	SetAppName(testName)
	
	if DefaultConfig.ApplicationName != testName {
		t.Errorf("Expected ApplicationName %s, got %s", testName, DefaultConfig.ApplicationName)
	}
}

func TestSetBundleID(t *testing.T) {
	resetDefaultConfig()
	
	testBundleID := "com.example.testapp"
	SetBundleID(testBundleID)
	
	if DefaultConfig.BundleID != testBundleID {
		t.Errorf("Expected BundleID %s, got %s", testBundleID, DefaultConfig.BundleID)
	}
}

func TestEnableKeepTemp(t *testing.T) {
	resetDefaultConfig()
	
	// Should be false by default
	if DefaultConfig.KeepTemp {
		t.Error("Expected KeepTemp to be false by default")
	}
	
	EnableKeepTemp()
	
	if !DefaultConfig.KeepTemp {
		t.Error("Expected KeepTemp to be true after EnableKeepTemp()")
	}
}

func TestDisableRelaunch(t *testing.T) {
	resetDefaultConfig()
	
	// Should be true by default
	if !DefaultConfig.Relaunch {
		t.Error("Expected Relaunch to be true by default")
	}
	
	DisableRelaunch()
	
	if DefaultConfig.Relaunch {
		t.Error("Expected Relaunch to be false after DisableRelaunch()")
	}
}

func TestEnableDebug(t *testing.T) {
	// Save original value
	originalDebug := os.Getenv("MACGO_DEBUG")
	defer func() {
		if originalDebug != "" {
			os.Setenv("MACGO_DEBUG", originalDebug)
		} else {
			os.Unsetenv("MACGO_DEBUG")
		}
	}()
	
	EnableDebug()
	
	if os.Getenv("MACGO_DEBUG") != "1" {
		t.Error("Expected MACGO_DEBUG environment variable to be set to '1'")
	}
}

func TestSetCustomAppBundle(t *testing.T) {
	resetDefaultConfig()
	
	// Create a test filesystem
	testFS := fstest.MapFS{
		"Contents/Info.plist": &fstest.MapFile{
			Data: []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<plist version=\"1.0\">\n</plist>"),
		},
		"Contents/MacOS/app": &fstest.MapFile{
			Data: []byte("#!/bin/bash\necho 'test app'\n"),
		},
	}
	
	SetCustomAppBundle(testFS)
	
	if DefaultConfig.AppTemplate == nil {
		t.Error("Expected AppTemplate to be set")
	}
	
	// Verify the filesystem can be read
	if _, err := DefaultConfig.AppTemplate.Open("Contents/Info.plist"); err != nil {
		t.Errorf("Expected to be able to read from AppTemplate: %v", err)
	}
}

func TestEnableSigning(t *testing.T) {
	resetDefaultConfig()
	
	tests := []struct {
		name             string
		identity         string
		expectedAutoSign bool
		expectedIdentity string
	}{
		{
			name:             "With identity",
			identity:         "Developer ID Application: Example Corp",
			expectedAutoSign: true,
			expectedIdentity: "Developer ID Application: Example Corp",
		},
		{
			name:             "Without identity (ad-hoc)",
			identity:         "",
			expectedAutoSign: true,
			expectedIdentity: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetDefaultConfig()
			
			EnableSigning(tt.identity)
			
			if DefaultConfig.AutoSign != tt.expectedAutoSign {
				t.Errorf("Expected AutoSign %v, got %v", tt.expectedAutoSign, DefaultConfig.AutoSign)
			}
			
			if DefaultConfig.SigningIdentity != tt.expectedIdentity {
				t.Errorf("Expected SigningIdentity %s, got %s", tt.expectedIdentity, DefaultConfig.SigningIdentity)
			}
		})
	}
}

func TestLoadEntitlementsFromJSON(t *testing.T) {
	resetDefaultConfig()
	
	tests := []struct {
		name        string
		jsonData    string
		expected    map[Entitlement]bool
		expectError bool
	}{
		{
			name:     "Valid JSON",
			jsonData: `{"com.apple.security.camera": true, "com.apple.security.microphone": false}`,
			expected: map[Entitlement]bool{
				"com.apple.security.camera":    true,
				"com.apple.security.microphone": false,
			},
			expectError: false,
		},
		{
			name:        "Invalid JSON",
			jsonData:    `{"com.apple.security.camera": true`,
			expected:    nil,
			expectError: true,
		},
		{
			name:        "Empty JSON",
			jsonData:    `{}`,
			expected:    nil,
			expectError: false,
		},
		{
			name:     "Mixed types (should work)",
			jsonData: `{"com.apple.security.camera": true, "com.apple.security.microphone": false, "com.apple.security.sandbox": true}`,
			expected: map[Entitlement]bool{
				"com.apple.security.camera":    true,
				"com.apple.security.microphone": false,
				"com.apple.security.sandbox":   true,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetDefaultConfig()
			
			err := LoadEntitlementsFromJSON([]byte(tt.jsonData))
			
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			
			if !tt.expectError && !reflect.DeepEqual(DefaultConfig.Entitlements, tt.expected) {
				t.Errorf("Expected entitlements %v, got %v", tt.expected, DefaultConfig.Entitlements)
			}
		})
	}
}

func TestLoadEntitlementsFromJSONMerging(t *testing.T) {
	resetDefaultConfig()
	
	// Add some initial entitlements
	RequestEntitlements(EntCamera, EntMicrophone)
	
	// Load additional entitlements from JSON
	jsonData := `{"com.apple.security.sandbox": true, "com.apple.security.network.client": false}`
	err := LoadEntitlementsFromJSON([]byte(jsonData))
	
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	expected := map[Entitlement]bool{
		EntCamera:                            true,
		EntMicrophone:                        true,
		"com.apple.security.sandbox":        true,
		"com.apple.security.network.client": false,
	}
	
	if !reflect.DeepEqual(DefaultConfig.Entitlements, expected) {
		t.Errorf("Expected merged entitlements %v, got %v", expected, DefaultConfig.Entitlements)
	}
}

func TestAddPlistEntry(t *testing.T) {
	resetDefaultConfig()
	
	tests := []struct {
		name  string
		key   string
		value interface{}
	}{
		{
			name:  "String value",
			key:   "CFBundleName",
			value: "Test App",
		},
		{
			name:  "Boolean value",
			key:   "LSUIElement",
			value: true,
		},
		{
			name:  "Integer value",
			key:   "CFBundleVersion",
			value: 1,
		},
		{
			name:  "Array value",
			key:   "UTExportedTypeDeclarations",
			value: []string{"com.example.type1", "com.example.type2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetDefaultConfig()
			
			AddPlistEntry(tt.key, tt.value)
			
			if DefaultConfig.PlistEntries == nil {
				t.Error("Expected PlistEntries to be initialized")
			}
			
			if val, ok := DefaultConfig.PlistEntries[tt.key]; !ok || !reflect.DeepEqual(val, tt.value) {
				t.Errorf("Expected PlistEntries[%s] = %v, got %v", tt.key, tt.value, val)
			}
		})
	}
}

func TestSetIconFile(t *testing.T) {
	resetDefaultConfig()
	
	testIconPath := "/path/to/icon.icns"
	SetIconFile(testIconPath)
	
	if DefaultConfig.PlistEntries == nil {
		t.Error("Expected PlistEntries to be initialized")
	}
	
	if val, ok := DefaultConfig.PlistEntries["CFBundleIconFile"]; !ok || val != testIconPath {
		t.Errorf("Expected CFBundleIconFile to be %s, got %v", testIconPath, val)
	}
}

func TestConfigRequestEntitlements(t *testing.T) {
	config := &Config{}
	
	entitlements := []interface{}{
		EntCamera,
		"com.apple.security.microphone",
		EntAppSandbox,
	}
	
	config.RequestEntitlements(entitlements...)
	
	expected := map[Entitlement]bool{
		EntCamera:                         true,
		"com.apple.security.microphone":  true,
		EntAppSandbox:                     true,
	}
	
	if !reflect.DeepEqual(config.Entitlements, expected) {
		t.Errorf("Expected entitlements %v, got %v", expected, config.Entitlements)
	}
}

func TestConfigRequestEntitlementsWithExisting(t *testing.T) {
	config := &Config{
		Entitlements: map[Entitlement]bool{
			EntMicrophone: true,
		},
	}
	
	config.RequestEntitlements(EntCamera, EntAppSandbox)
	
	expected := map[Entitlement]bool{
		EntMicrophone: true,
		EntCamera:     true,
		EntAppSandbox: true,
	}
	
	if !reflect.DeepEqual(config.Entitlements, expected) {
		t.Errorf("Expected entitlements %v, got %v", expected, config.Entitlements)
	}
}

func TestConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent access test in short mode")
	}
	
	resetDefaultConfig()
	
	const numGoroutines = 10
	const numOperations = 100
	
	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	
	// Run multiple goroutines that access the default config concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			
			for j := 0; j < numOperations; j++ {
				switch j % 6 {
				case 0:
					RequestEntitlements(fmt.Sprintf("com.example.test%d.%d", id, j))
				case 1:
					SetAppName(fmt.Sprintf("TestApp%d", id))
				case 2:
					SetBundleID(fmt.Sprintf("com.example.test%d", id))
				case 3:
					AddPlistEntry(fmt.Sprintf("TestKey%d", id), fmt.Sprintf("TestValue%d", id))
				case 4:
					EnableDockIcon()
				case 5:
					EnableKeepTemp()
				}
			}
		}(i)
	}
	
	wg.Wait()
	
	// Verify that the config is in a consistent state
	if DefaultConfig.Entitlements == nil {
		t.Error("Expected entitlements to be initialized after concurrent access")
	}
	
	if DefaultConfig.PlistEntries == nil {
		t.Error("Expected plist entries to be initialized after concurrent access")
	}
}

func TestAPIFunctionChaining(t *testing.T) {
	resetDefaultConfig()
	
	// Test that API functions can be called in sequence
	RequestEntitlements(EntCamera, EntMicrophone)
	SetAppName("ChainedApp")
	SetBundleID("com.example.chained")
	EnableDockIcon()
	EnableKeepTemp()
	DisableRelaunch()
	AddPlistEntry("CustomKey", "CustomValue")
	SetIconFile("/path/to/icon.icns")
	
	// Verify all settings were applied
	expectedEntitlements := map[Entitlement]bool{
		EntCamera:     true,
		EntMicrophone: true,
	}
	
	if !reflect.DeepEqual(DefaultConfig.Entitlements, expectedEntitlements) {
		t.Errorf("Expected entitlements %v, got %v", expectedEntitlements, DefaultConfig.Entitlements)
	}
	
	if DefaultConfig.ApplicationName != "ChainedApp" {
		t.Errorf("Expected ApplicationName 'ChainedApp', got %s", DefaultConfig.ApplicationName)
	}
	
	if DefaultConfig.BundleID != "com.example.chained" {
		t.Errorf("Expected BundleID 'com.example.chained', got %s", DefaultConfig.BundleID)
	}
	
	if !DefaultConfig.KeepTemp {
		t.Error("Expected KeepTemp to be true")
	}
	
	if DefaultConfig.Relaunch {
		t.Error("Expected Relaunch to be false")
	}
	
	expectedPlistEntries := map[string]interface{}{
		"LSUIElement":        false,
		"CustomKey":          "CustomValue",
		"CFBundleIconFile":   "/path/to/icon.icns",
	}
	
	if !reflect.DeepEqual(DefaultConfig.PlistEntries, expectedPlistEntries) {
		t.Errorf("Expected PlistEntries %v, got %v", expectedPlistEntries, DefaultConfig.PlistEntries)
	}
}

func TestEdgeCases(t *testing.T) {
	resetDefaultConfig()
	
	// Test empty string values
	SetAppName("")
	SetBundleID("")
	
	if DefaultConfig.ApplicationName != "" {
		t.Errorf("Expected empty ApplicationName, got %s", DefaultConfig.ApplicationName)
	}
	
	if DefaultConfig.BundleID != "" {
		t.Errorf("Expected empty BundleID, got %s", DefaultConfig.BundleID)
	}
	
	// Test nil values
	AddPlistEntry("NilTest", nil)
	
	if val, ok := DefaultConfig.PlistEntries["NilTest"]; !ok || val != nil {
		t.Errorf("Expected nil value for NilTest, got %v", val)
	}
	
	// Test very long strings
	longString := string(make([]byte, 10000))
	SetAppName(longString)
	
	if DefaultConfig.ApplicationName != longString {
		t.Error("Expected long string to be set correctly")
	}
}

// Benchmark tests
func BenchmarkRequestEntitlements(b *testing.B) {
	resetDefaultConfig()
	
	entitlements := []interface{}{
		EntCamera,
		EntMicrophone,
		EntAppSandbox,
		"com.apple.security.network.client",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RequestEntitlements(entitlements...)
	}
}

func BenchmarkSetAppName(b *testing.B) {
	resetDefaultConfig()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SetAppName(fmt.Sprintf("TestApp%d", i))
	}
}

func BenchmarkAddPlistEntry(b *testing.B) {
	resetDefaultConfig()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		AddPlistEntry(fmt.Sprintf("Key%d", i), fmt.Sprintf("Value%d", i))
	}
}

func BenchmarkLoadEntitlementsFromJSON(b *testing.B) {
	resetDefaultConfig()
	
	jsonData := []byte(`{
		"com.apple.security.camera": true,
		"com.apple.security.microphone": true,
		"com.apple.security.sandbox": true,
		"com.apple.security.network.client": true
	}`)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LoadEntitlementsFromJSON(jsonData)
	}
}

func BenchmarkConcurrentAccess(b *testing.B) {
	resetDefaultConfig()
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			RequestEntitlements(EntCamera)
			SetAppName("BenchmarkApp")
			AddPlistEntry("BenchKey", "BenchValue")
		}
	})
}