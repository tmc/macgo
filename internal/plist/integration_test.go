package plist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestIntegrationInfoAndEntitlements tests the complete workflow of generating
// both Info.plist and entitlements.plist files as would be used by macgo.
func TestIntegrationInfoAndEntitlements(t *testing.T) {
	tempDir := t.TempDir()

	// Test scenario: Camera and Microphone app with sandbox
	appName := "Test Camera App"
	bundleID := "com.example.testcameraapp"
	execName := "testcameraapp"
	version := "2.1.0"

	// Create Info.plist
	infoCfg := InfoPlistConfig{
		AppName:  appName,
		BundleID: bundleID,
		ExecName: execName,
		Version:  version,
		CustomKeys: map[string]interface{}{
			"LSUIElement":                  false, // Show in dock for this example
			"NSCameraUsageDescription":     "This app needs camera access to capture photos.",
			"NSMicrophoneUsageDescription": "This app needs microphone access for audio recording.",
		},
	}

	infoPath := filepath.Join(tempDir, "Info.plist")
	if err := WriteInfoPlist(infoPath, infoCfg); err != nil {
		t.Fatalf("Failed to write Info.plist: %v", err)
	}

	// Create entitlements.plist
	entCfg := EntitlementsConfig{
		Permissions: []Permission{Camera, Microphone, Sandbox, Files},
		Custom:      []string{"com.apple.security.device.bluetooth"},
		AppGroups:   []string{"group.com.example.shared"},
	}

	entPath := filepath.Join(tempDir, "entitlements.plist")
	if err := WriteEntitlements(entPath, entCfg); err != nil {
		t.Fatalf("Failed to write entitlements.plist: %v", err)
	}

	// Verify both files exist
	if _, err := os.Stat(infoPath); os.IsNotExist(err) {
		t.Fatal("Info.plist was not created")
	}
	if _, err := os.Stat(entPath); os.IsNotExist(err) {
		t.Fatal("entitlements.plist was not created")
	}

	// Read and verify Info.plist content
	infoContent, err := os.ReadFile(infoPath)
	if err != nil {
		t.Fatalf("Failed to read Info.plist: %v", err)
	}

	infoStr := string(infoContent)
	infoChecks := []string{
		`<string>Test Camera App</string>`,
		`<string>com.example.testcameraapp</string>`,
		`<string>testcameraapp</string>`,
		`<string>2.1.0</string>`,
		`<key>LSUIElement</key>`,
		`<false/>`,
		`<key>NSCameraUsageDescription</key>`,
		`<string>This app needs camera access to capture photos.</string>`,
		`<key>NSMicrophoneUsageDescription</key>`,
		`<string>This app needs microphone access for audio recording.</string>`,
	}

	for _, check := range infoChecks {
		if !strings.Contains(infoStr, check) {
			t.Errorf("Info.plist missing expected content: %s", check)
		}
	}

	// Read and verify entitlements.plist content
	entContent, err := os.ReadFile(entPath)
	if err != nil {
		t.Fatalf("Failed to read entitlements.plist: %v", err)
	}

	entStr := string(entContent)
	entChecks := []string{
		`<key>com.apple.security.device.camera</key>`,
		`<key>com.apple.security.device.microphone</key>`,
		`<key>com.apple.security.app-sandbox</key>`,
		`<key>com.apple.security.files.user-selected.read-only</key>`,
		`<key>com.apple.security.device.bluetooth</key>`,
		`<key>com.apple.security.application-groups</key>`,
		`<string>group.com.example.shared</string>`,
		`<true/>`,
	}

	for _, check := range entChecks {
		if !strings.Contains(entStr, check) {
			t.Errorf("entitlements.plist missing expected content: %s", check)
		}
	}

	t.Logf("Successfully created Info.plist (%d bytes) and entitlements.plist (%d bytes)",
		len(infoContent), len(entContent))
}

// TestIntegrationRealWorldScenarios tests various real-world app scenarios.
func TestIntegrationRealWorldScenarios(t *testing.T) {
	scenarios := []struct {
		name        string
		appName     string
		permissions []Permission
		custom      []string
		appGroups   []string
	}{
		{
			name:        "Media Editor",
			appName:     "Video Editor Pro",
			permissions: []Permission{Camera, Microphone, Files, Sandbox},
			custom:      []string{"com.apple.security.device.usb"},
		},
		{
			name:        "Web Server",
			appName:     "Local Web Server",
			permissions: []Permission{Network, Files},
		},
		{
			name:        "Location Tracker",
			appName:     "GPS Tracker",
			permissions: []Permission{Location, Files, Sandbox},
		},
		{
			name:        "Shared Data App",
			appName:     "Team Sync",
			permissions: []Permission{Sandbox, Files},
			appGroups:   []string{"group.com.example.teamsync"},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			tempDir := t.TempDir()

			// Generate bundle ID from app name
			bundleID := GenerateDefaultBundleID(scenario.appName)

			// Create Info.plist
			infoCfg := InfoPlistConfig{
				AppName:  scenario.appName,
				BundleID: bundleID,
				ExecName: strings.ToLower(strings.ReplaceAll(scenario.appName, " ", "")),
				Version:  "1.0.0",
			}

			infoPath := filepath.Join(tempDir, "Info.plist")
			if err := WriteInfoPlist(infoPath, infoCfg); err != nil {
				t.Fatalf("Failed to write Info.plist for %s: %v", scenario.name, err)
			}

			// Create entitlements.plist if permissions exist
			if len(scenario.permissions) > 0 || len(scenario.custom) > 0 || len(scenario.appGroups) > 0 {
				entCfg := EntitlementsConfig{
					Permissions: scenario.permissions,
					Custom:      scenario.custom,
					AppGroups:   scenario.appGroups,
				}

				entPath := filepath.Join(tempDir, "entitlements.plist")
				if err := WriteEntitlements(entPath, entCfg); err != nil {
					t.Fatalf("Failed to write entitlements.plist for %s: %v", scenario.name, err)
				}

				// Verify entitlements file was created
				if _, err := os.Stat(entPath); os.IsNotExist(err) {
					t.Errorf("entitlements.plist was not created for %s", scenario.name)
				}
			}

			// Verify Info.plist was created and contains app name
			infoContent, err := os.ReadFile(infoPath)
			if err != nil {
				t.Fatalf("Failed to read Info.plist for %s: %v", scenario.name, err)
			}

			if !strings.Contains(string(infoContent), EscapeXML(scenario.appName)) {
				t.Errorf("Info.plist for %s doesn't contain escaped app name", scenario.name)
			}
		})
	}
}

// TestIntegrationErrorHandling tests error scenarios in the integration workflow.
func TestIntegrationErrorHandling(t *testing.T) {
	// Test invalid Info.plist config
	invalidInfoCfg := InfoPlistConfig{
		AppName: "", // Missing required field
	}

	err := WriteInfoPlist("/tmp/invalid-info.plist", invalidInfoCfg)
	if err == nil {
		t.Error("Expected error for invalid Info.plist config, got nil")
	}

	// Test invalid app groups
	invalidEntCfg := EntitlementsConfig{
		AppGroups: []string{"invalid-group"}, // Should start with "group."
	}

	err = ValidateAppGroups(invalidEntCfg.AppGroups)
	if err == nil {
		t.Error("Expected error for invalid app groups, got nil")
	}

	// Test invalid permissions
	invalidPerms := []Permission{"unknown-permission"}
	err = ValidatePermissions(invalidPerms)
	if err == nil {
		t.Error("Expected error for invalid permissions, got nil")
	}
}
