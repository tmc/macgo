package bundle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tmc/macgo/internal/system"
)

func TestValidateCodeSignIdentity(t *testing.T) {
	tests := []struct {
		name     string
		identity string
		wantErr  bool
	}{
		{
			name:     "empty identity",
			identity: "",
			wantErr:  true,
		},
		{
			name:     "ad-hoc identity",
			identity: "-",
			wantErr:  false,
		},
		// Note: Testing with actual identities would require them to be present
		// in the keychain, so we focus on the basic validation logic
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCodeSignIdentity(tt.identity)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCodeSignIdentity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReadBundleIDFromPlist(t *testing.T) {
	// Create a temporary bundle structure for testing
	tempDir, err := os.MkdirTemp("", "bundle-read-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create a test bundle structure
	bundlePath := filepath.Join(tempDir, "TestApp.app")
	contentsDir := filepath.Join(bundlePath, "Contents")
	if err := os.MkdirAll(contentsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a test Info.plist
	plistPath := filepath.Join(contentsDir, "Info.plist")
	plistContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleIdentifier</key>
	<string>com.example.test</string>
	<key>CFBundleName</key>
	<string>TestApp</string>
</dict>
</plist>`

	if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test reading the bundle ID using system.GetBundleID
	bundleID := system.GetBundleID(bundlePath)
	expectedBundleID := "com.example.test"
	if bundleID != expectedBundleID {
		t.Errorf("system.GetBundleID() = %q, want %q", bundleID, expectedBundleID)
	}

	// Test with non-existent bundle
	nonExistentBundleID := system.GetBundleID("/nonexistent/path.app")
	if nonExistentBundleID != "" {
		t.Error("system.GetBundleID() should return empty string for non-existent bundle")
	}
}

func TestCodeSignBundle_AdHoc(t *testing.T) {
	// Create a temporary bundle structure for testing
	tempDir, err := os.MkdirTemp("", "codesign-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create minimal bundle structure
	bundlePath := filepath.Join(tempDir, "TestApp.app")
	contentsDir := filepath.Join(bundlePath, "Contents")
	macosDir := filepath.Join(contentsDir, "MacOS")

	if err := os.MkdirAll(macosDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a dummy executable
	execPath := filepath.Join(macosDir, "TestApp")
	if err := os.WriteFile(execPath, []byte("#!/bin/bash\necho test"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create Info.plist
	plistPath := filepath.Join(contentsDir, "Info.plist")
	plistContent := `<?xml version="1.0"?>
<plist version="1.0">
<dict>
	<key>CFBundleIdentifier</key>
	<string>com.example.test</string>
</dict>
</plist>`
	if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test ad-hoc signing
	config := &Config{
		CodeSignIdentity: "-",
		Debug:            false,
	}

	err = codeSignBundle(bundlePath, config)
	// This test will only work on macOS with codesign available
	// On other platforms or CI environments, we expect it to fail
	if err != nil {
		// Skip if codesign is not available
		if strings.Contains(err.Error(), "executable file not found") ||
			strings.Contains(err.Error(), "no such file or directory") {
			t.Skipf("codesign not available: %v", err)
		}
		// Otherwise, it's a real error in our code
		t.Logf("codeSignBundle failed (may be expected in test environment): %v", err)
	}
}

func TestFindDeveloperID(t *testing.T) {
	// This function depends on the security command and actual certificates
	// In a test environment, we can only test that it doesn't crash
	identity := findDeveloperID(true) // Enable debug output

	// We can't assert much about the result since it depends on the system
	// But we can check that it returns a string (could be empty)
	t.Logf("findDeveloperID returned: %q", identity)

	// Test with debug disabled
	identity2 := findDeveloperID(false)
	t.Logf("findDeveloperID (no debug) returned: %q", identity2)
}

func TestListAvailableIdentities(t *testing.T) {
	identities, err := listAvailableIdentities()
	if err != nil {
		// Skip if security command is not available
		if strings.Contains(err.Error(), "executable file not found") ||
			strings.Contains(err.Error(), "no such file or directory") {
			t.Skipf("security command not available: %v", err)
		}
		t.Errorf("listAvailableIdentities() failed: %v", err)
		return
	}

	// We can't assert specific identities, but we can check the format
	t.Logf("Found %d identities", len(identities))
	for i, identity := range identities {
		if identity == "" {
			t.Errorf("Identity %d is empty", i)
		}
		t.Logf("Identity %d: %s", i, identity)
	}
}

func TestVerifySignature(t *testing.T) {
	// Create a temporary bundle for testing
	tempDir, err := os.MkdirTemp("", "verify-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	bundlePath := filepath.Join(tempDir, "TestApp.app")
	contentsDir := filepath.Join(bundlePath, "Contents")
	macosDir := filepath.Join(contentsDir, "MacOS")

	if err := os.MkdirAll(macosDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a dummy executable
	execPath := filepath.Join(macosDir, "TestApp")
	if err := os.WriteFile(execPath, []byte("#!/bin/bash\necho test"), 0755); err != nil {
		t.Fatal(err)
	}

	// Test verification on unsigned bundle (should fail)
	err = VerifySignature(bundlePath)
	if err == nil {
		t.Error("VerifySignature() should fail for unsigned bundle")
	}

	// Test with non-existent bundle
	err = VerifySignature("/nonexistent/bundle.app")
	if err == nil {
		t.Error("VerifySignature() should fail for non-existent bundle")
	}
}

func TestGetSignatureInfo(t *testing.T) {
	// Create a temporary bundle for testing
	tempDir, err := os.MkdirTemp("", "siginfo-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	bundlePath := filepath.Join(tempDir, "TestApp.app")
	contentsDir := filepath.Join(bundlePath, "Contents")
	macosDir := filepath.Join(contentsDir, "MacOS")

	if err := os.MkdirAll(macosDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a dummy executable
	execPath := filepath.Join(macosDir, "TestApp")
	if err := os.WriteFile(execPath, []byte("#!/bin/bash\necho test"), 0755); err != nil {
		t.Fatal(err)
	}

	// Test getting signature info (should fail for unsigned bundle)
	_, err = GetSignatureInfo(bundlePath)
	if err == nil {
		t.Error("GetSignatureInfo() should fail for unsigned bundle")
	} else {
		t.Logf("Expected error for unsigned bundle: %v", err)
	}

	// Test with non-existent bundle
	_, err = GetSignatureInfo("/nonexistent/bundle.app")
	if err == nil {
		t.Error("GetSignatureInfo() should fail for non-existent bundle")
	}
}

// TestCodeSignBundle_Config tests the configuration parsing for code signing
func TestCodeSignBundle_Config(t *testing.T) {
	// Create a temporary bundle structure
	tempDir, err := os.MkdirTemp("", "config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	bundlePath := filepath.Join(tempDir, "TestApp.app")
	contentsDir := filepath.Join(bundlePath, "Contents")
	macosDir := filepath.Join(contentsDir, "MacOS")

	if err := os.MkdirAll(macosDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create Info.plist with bundle ID
	plistPath := filepath.Join(contentsDir, "Info.plist")
	plistContent := `<?xml version="1.0"?>
<plist version="1.0">
<dict>
	<key>CFBundleIdentifier</key>
	<string>com.example.testconfig</string>
</dict>
</plist>`
	if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create entitlements file
	entPath := filepath.Join(contentsDir, "entitlements.plist")
	entContent := `<?xml version="1.0"?>
<plist version="1.0">
<dict>
	<key>com.apple.security.app-sandbox</key>
	<true/>
</dict>
</plist>`
	if err := os.WriteFile(entPath, []byte(entContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create dummy executable
	execPath := filepath.Join(macosDir, "TestApp")
	if err := os.WriteFile(execPath, []byte("#!/bin/bash\necho test"), 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		config *Config
	}{
		{
			name: "with custom identifier",
			config: &Config{
				CodeSignIdentity:      "-",
				CodeSigningIdentifier: "com.custom.identifier",
				Debug:                 true,
			},
		},
		{
			name: "using bundle ID as identifier",
			config: &Config{
				CodeSignIdentity:      "-",
				CodeSigningIdentifier: "",
				Debug:                 true,
			},
		},
		{
			name: "with entitlements",
			config: &Config{
				CodeSignIdentity: "TestIdentity",
				Debug:            true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := codeSignBundle(bundlePath, tt.config)
			// We expect this to fail in test environments where codesign isn't available
			// or the identity doesn't exist. We're mainly testing that the function
			// handles different configurations without panicking.
			if err != nil {
				t.Logf("codeSignBundle failed (expected in test environment): %v", err)
			}
		})
	}
}
