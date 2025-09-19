package plist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteEntitlements(t *testing.T) {
	tempDir := t.TempDir()
	entPath := filepath.Join(tempDir, "entitlements.plist")

	cfg := EntitlementsConfig{
		Permissions: []Permission{Camera, Microphone, Sandbox},
		Custom:      []string{"com.apple.security.device.bluetooth"},
		AppGroups:   []string{"group.com.example.shared", "group.com.example.data"},
	}

	err := WriteEntitlements(entPath, cfg)
	if err != nil {
		t.Fatalf("WriteEntitlements failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(entPath); os.IsNotExist(err) {
		t.Fatal("entitlements.plist file was not created")
	}

	// Read and verify content
	content, err := os.ReadFile(entPath)
	if err != nil {
		t.Fatalf("Failed to read entitlements file: %v", err)
	}

	contentStr := string(content)

	// Check for required XML elements
	requiredElements := []string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<!DOCTYPE plist`,
		`<plist version="1.0">`,
		`<dict>`,
		`<key>com.apple.security.device.camera</key>`,
		`<true/>`,
		`<key>com.apple.security.device.microphone</key>`,
		`<true/>`,
		`<key>com.apple.security.app-sandbox</key>`,
		`<true/>`,
		`<key>com.apple.security.device.bluetooth</key>`,
		`<true/>`,
		`<key>com.apple.security.application-groups</key>`,
		`<array>`,
		`<string>group.com.example.shared</string>`,
		`<string>group.com.example.data</string>`,
		`</array>`,
		`</dict>`,
		`</plist>`,
	}

	for _, element := range requiredElements {
		if !strings.Contains(contentStr, element) {
			t.Errorf("Missing required element: %s", element)
		}
	}
}

func TestWriteEntitlementsEmpty(t *testing.T) {
	tempDir := t.TempDir()
	entPath := filepath.Join(tempDir, "entitlements.plist")

	cfg := EntitlementsConfig{
		Permissions: []Permission{},
		Custom:      []string{},
		AppGroups:   []string{},
	}

	err := WriteEntitlements(entPath, cfg)
	if err != nil {
		t.Fatalf("WriteEntitlements failed: %v", err)
	}

	// File should not be created for empty entitlements
	if _, err := os.Stat(entPath); !os.IsNotExist(err) {
		t.Error("entitlements.plist file should not be created for empty config")
	}
}

func TestPermissionToEntitlement(t *testing.T) {
	tests := []struct {
		name       string
		permission Permission
		expected   string
	}{
		{
			name:       "camera permission",
			permission: Camera,
			expected:   "com.apple.security.device.camera",
		},
		{
			name:       "microphone permission",
			permission: Microphone,
			expected:   "com.apple.security.device.microphone",
		},
		{
			name:       "location permission",
			permission: Location,
			expected:   "com.apple.security.personal-information.location",
		},
		{
			name:       "sandbox permission",
			permission: Sandbox,
			expected:   "com.apple.security.app-sandbox",
		},
		{
			name:       "files permission",
			permission: Files,
			expected:   "com.apple.security.files.user-selected.read-only",
		},
		{
			name:       "network permission",
			permission: Network,
			expected:   "com.apple.security.network.client",
		},
		{
			name:       "unknown permission",
			permission: Permission("unknown"),
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := permissionToEntitlement(tt.permission)
			if result != tt.expected {
				t.Errorf("permissionToEntitlement(%v) = %q, want %q", tt.permission, result, tt.expected)
			}
		})
	}
}

func TestGetAvailablePermissions(t *testing.T) {
	permissions := GetAvailablePermissions()

	expectedPermissions := []Permission{
		Camera,
		Microphone,
		Location,
		Files,
		Network,
		Sandbox,
	}

	if len(permissions) != len(expectedPermissions) {
		t.Errorf("Expected %d permissions, got %d", len(expectedPermissions), len(permissions))
	}

	// Check that all expected permissions are present
	permMap := make(map[Permission]bool)
	for _, p := range permissions {
		permMap[p] = true
	}

	for _, expected := range expectedPermissions {
		if !permMap[expected] {
			t.Errorf("Missing expected permission: %v", expected)
		}
	}
}

func TestPermissionDescription(t *testing.T) {
	tests := []struct {
		name       string
		permission Permission
		shouldContain string
	}{
		{
			name:          "camera description",
			permission:    Camera,
			shouldContain: "Camera access",
		},
		{
			name:          "microphone description",
			permission:    Microphone,
			shouldContain: "Microphone access",
		},
		{
			name:          "location description",
			permission:    Location,
			shouldContain: "Location services",
		},
		{
			name:          "files description",
			permission:    Files,
			shouldContain: "file access",
		},
		{
			name:          "network description",
			permission:    Network,
			shouldContain: "Network client",
		},
		{
			name:          "sandbox description",
			permission:    Sandbox,
			shouldContain: "sandbox",
		},
		{
			name:          "unknown permission",
			permission:    Permission("unknown"),
			shouldContain: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PermissionDescription(tt.permission)
			if !strings.Contains(strings.ToLower(result), strings.ToLower(tt.shouldContain)) {
				t.Errorf("PermissionDescription(%v) = %q, should contain %q", tt.permission, result, tt.shouldContain)
			}
		})
	}
}

func TestValidatePermissions(t *testing.T) {
	tests := []struct {
		name        string
		permissions []Permission
		shouldErr   bool
		errorMsg    string
	}{
		{
			name:        "valid permissions",
			permissions: []Permission{Camera, Microphone, Files},
			shouldErr:   false,
		},
		{
			name:        "empty permissions",
			permissions: []Permission{},
			shouldErr:   false,
		},
		{
			name:        "unknown permission",
			permissions: []Permission{Camera, Permission("unknown")},
			shouldErr:   true,
			errorMsg:    "unknown permission: unknown",
		},
		{
			name:        "all valid permissions",
			permissions: GetAvailablePermissions(),
			shouldErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePermissions(tt.permissions)
			if tt.shouldErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestValidateAppGroups(t *testing.T) {
	tests := []struct {
		name      string
		appGroups []string
		shouldErr bool
		errorMsg  string
	}{
		{
			name:      "valid app groups",
			appGroups: []string{"group.com.example.shared", "group.com.mycompany.data"},
			shouldErr: false,
		},
		{
			name:      "empty app groups",
			appGroups: []string{},
			shouldErr: false,
		},
		{
			name:      "missing group prefix",
			appGroups: []string{"com.example.shared"},
			shouldErr: true,
			errorMsg:  "app group identifier must start with 'group.'",
		},
		{
			name:      "too short identifier",
			appGroups: []string{"group."},
			shouldErr: true,
			errorMsg:  "app group identifier too short",
		},
		{
			name:      "mixed valid and invalid",
			appGroups: []string{"group.com.example.shared", "invalid.group"},
			shouldErr: true,
			errorMsg:  "app group identifier must start with 'group.'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAppGroups(tt.appGroups)
			if tt.shouldErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestGenerateEntitlementsContentOnlyPermissions(t *testing.T) {
	cfg := EntitlementsConfig{
		Permissions: []Permission{Camera, Files},
	}

	content := generateEntitlementsContent(cfg)

	// Should contain XML structure
	if !strings.Contains(content, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("Content should contain XML header")
	}

	// Should contain camera and files entitlements
	expectedEntitlements := []string{
		`<key>com.apple.security.device.camera</key>`,
		`<key>com.apple.security.files.user-selected.read-only</key>`,
		`<true/>`,
	}

	for _, expected := range expectedEntitlements {
		if !strings.Contains(content, expected) {
			t.Errorf("Content should contain: %s", expected)
		}
	}

	// Should not contain app groups
	if strings.Contains(content, "application-groups") {
		t.Error("Content should not contain app groups")
	}
}

func TestGenerateEntitlementsContentOnlyCustom(t *testing.T) {
	cfg := EntitlementsConfig{
		Custom: []string{"com.apple.security.device.bluetooth", "com.apple.security.device.usb"},
	}

	content := generateEntitlementsContent(cfg)

	expectedEntitlements := []string{
		`<key>com.apple.security.device.bluetooth</key>`,
		`<key>com.apple.security.device.usb</key>`,
		`<true/>`,
	}

	for _, expected := range expectedEntitlements {
		if !strings.Contains(content, expected) {
			t.Errorf("Content should contain: %s", expected)
		}
	}
}

func TestGenerateEntitlementsContentOnlyAppGroups(t *testing.T) {
	cfg := EntitlementsConfig{
		AppGroups: []string{"group.com.example.shared"},
	}

	content := generateEntitlementsContent(cfg)

	expectedElements := []string{
		`<key>com.apple.security.application-groups</key>`,
		`<array>`,
		`<string>group.com.example.shared</string>`,
		`</array>`,
	}

	for _, expected := range expectedElements {
		if !strings.Contains(content, expected) {
			t.Errorf("Content should contain: %s", expected)
		}
	}
}

func TestEntitlementsXMLEscaping(t *testing.T) {
	tempDir := t.TempDir()
	entPath := filepath.Join(tempDir, "entitlements.plist")

	cfg := EntitlementsConfig{
		Custom:    []string{"com.apple.security.test&<>\"'"},
		AppGroups: []string{"group.com.example.test&<>\"'"},
	}

	err := WriteEntitlements(entPath, cfg)
	if err != nil {
		t.Fatalf("WriteEntitlements failed: %v", err)
	}

	content, err := os.ReadFile(entPath)
	if err != nil {
		t.Fatalf("Failed to read entitlements file: %v", err)
	}

	contentStr := string(content)

	// Check that special characters are properly escaped
	expectedEscapes := []string{
		`<key>com.apple.security.test&amp;&lt;&gt;&quot;&#39;</key>`,
		`<string>group.com.example.test&amp;&lt;&gt;&quot;&#39;</string>`,
	}

	for _, escape := range expectedEscapes {
		if !strings.Contains(contentStr, escape) {
			t.Errorf("Missing escaped content: %s", escape)
		}
	}

	// Check that unescaped characters are not present
	forbiddenStrings := []string{
		`<key>com.apple.security.test&<>"'</key>`,
		`<string>group.com.example.test&<>"'</string>`,
	}

	for _, forbidden := range forbiddenStrings {
		if strings.Contains(contentStr, forbidden) {
			t.Errorf("Found unescaped content: %s", forbidden)
		}
	}
}