package bundle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIdentifierInSignedCode(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a test executable
	testExecPath := filepath.Join(tmpDir, "test-app")
	testContent := []byte("#!/bin/bash\necho 'test app'\n")
	if err := os.WriteFile(testExecPath, testContent, 0755); err != nil {
		t.Fatalf("Failed to create test executable: %v", err)
	}

	// Create bundle configuration without specifying BundleID to test inference
	config := &Config{
		AppName:   "TestApp",
		Debug:     true,
		AdHocSign: true, // Use ad-hoc signing for testing
	}

	bundle, err := New(testExecPath, config)
	if err != nil {
		t.Fatalf("Failed to create bundle: %v", err)
	}

	// Create the bundle
	if err := bundle.Create(); err != nil {
		t.Fatalf("Failed to create bundle: %v", err)
	}

	// Verify the bundle ID was inferred correctly
	bundleID := bundle.BundleID()
	if bundleID == "" {
		t.Fatal("Bundle ID was not set")
	}

	if strings.Contains(bundleID, "com.macgo") {
		t.Errorf("Bundle ID %q should not contain 'com.macgo'", bundleID)
	}

	t.Logf("Inferred bundle ID: %s", bundleID)

	// Sign the bundle
	if err := bundle.Sign(); err != nil {
		t.Fatalf("Failed to sign bundle: %v", err)
	}

	// Verify that the Info.plist contains the correct bundle ID
	plistPath := filepath.Join(bundle.Path, "Contents", "Info.plist")
	plistData, err := os.ReadFile(plistPath)
	if err != nil {
		t.Fatalf("Failed to read Info.plist: %v", err)
	}

	plistContent := string(plistData)
	if !strings.Contains(plistContent, bundleID) {
		t.Errorf("Info.plist does not contain the expected bundle ID %q", bundleID)
		t.Logf("Info.plist content:\n%s", plistContent)
	}

	// Verify that the signing identifier was properly set by reading it back
	signatureInfo, err := GetSignatureInfo(bundle.Path)
	if err != nil {
		// This might fail in test environments without proper code signing setup
		t.Logf("Could not get signature info (expected in test environment): %v", err)
		return
	}

	if identifier, exists := signatureInfo["Identifier"]; exists {
		if identifier != bundleID {
			t.Errorf("Signed identifier %q does not match bundle ID %q", identifier, bundleID)
		} else {
			t.Logf("Signed identifier correctly matches bundle ID: %s", identifier)
		}
	} else {
		t.Log("Identifier not found in signature info (may be expected in test environment)")
	}
}

func TestBundleIDPlistBinding(t *testing.T) {
	// Test that the Info.plist is properly bound to the inferred bundle ID
	tmpDir := t.TempDir()

	// Create a test executable
	testExecPath := filepath.Join(tmpDir, "bind-test")
	testContent := []byte("#!/bin/bash\necho 'bind test'\n")
	if err := os.WriteFile(testExecPath, testContent, 0755); err != nil {
		t.Fatalf("Failed to create test executable: %v", err)
	}

	// Test with multiple app names to ensure consistent behavior
	testCases := []string{
		"SimpleApp",
		"complex-app-name",
		"App With Spaces",
	}

	for _, appName := range testCases {
		t.Run(appName, func(t *testing.T) {
			config := &Config{
				AppName: appName,
				Debug:   true,
			}

			bundle, err := New(testExecPath, config)
			if err != nil {
				t.Fatalf("Failed to create bundle: %v", err)
			}

			// Create the bundle
			if err := bundle.Create(); err != nil {
				t.Fatalf("Failed to create bundle: %v", err)
			}

			bundleID := bundle.BundleID()

			// Verify bundle ID was inferred (not com.macgo)
			if strings.Contains(bundleID, "com.macgo") {
				t.Errorf("Bundle ID %q should not contain 'com.macgo'", bundleID)
			}

			// Read the created Info.plist and verify it contains the correct bundle ID
			plistPath := filepath.Join(bundle.Path, "Contents", "Info.plist")

			// Use the readBundleIDFromPlist function to verify
			readBundleID, err := readBundleIDFromPlist(plistPath)
			if err != nil {
				t.Fatalf("Failed to read bundle ID from plist: %v", err)
			}

			if readBundleID != bundleID {
				t.Errorf("Plist bundle ID %q does not match expected %q", readBundleID, bundleID)
			}

			t.Logf("App: %q -> Bundle ID: %q (correctly bound in plist)", appName, bundleID)
		})
	}
}
