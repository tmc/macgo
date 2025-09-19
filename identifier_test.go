package macgo

import (
	"os"
	"strings"
	"testing"

	"github.com/tmc/misc/macgo/helpers/bundle"
	bundlePkg "github.com/tmc/misc/macgo/internal/bundle"
)

func TestIdentifierPopulation(t *testing.T) {
	// Test that bundle IDs are correctly inferred and populated
	tests := []struct {
		name    string
		appName string
	}{
		{
			name:    "simple app name",
			appName: "TestApp",
		},
		{
			name:    "app with hyphens",
			appName: "test-app",
		},
		{
			name:    "app with spaces",
			appName: "Test App",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test direct bundle ID inference
			bundleID := bundle.InferBundleID(tt.appName)

			// Should not be empty
			if bundleID == "" {
				t.Errorf("InferBundleID(%q) returned empty string", tt.appName)
			}

			// Should contain at least one dot (valid bundle ID format)
			if !strings.Contains(bundleID, ".") {
				t.Errorf("InferBundleID(%q) = %q, should contain at least one dot", tt.appName, bundleID)
			}

			// Should NOT contain com.macgo anymore
			if strings.Contains(bundleID, "com.macgo") {
				t.Errorf("InferBundleID(%q) = %q, should not contain 'com.macgo'", tt.appName, bundleID)
			}

			// Should be a reasonable length
			if len(bundleID) < 3 {
				t.Errorf("InferBundleID(%q) = %q, too short", tt.appName, bundleID)
			}

			t.Logf("InferBundleID(%q) = %q", tt.appName, bundleID)
		})
	}
}

func TestBundleIDInferenceIntegration(t *testing.T) {
	// Test that Config properly uses inferred bundle IDs when none is provided
	cfg := &Config{
		AppName: "IntegrationTest",
		Debug:   true,
	}

	// Create a temporary executable for testing
	tmpDir := t.TempDir()
	execPath := tmpDir + "/integration-test"
	if err := os.WriteFile(execPath, []byte("#!/bin/bash\necho test\n"), 0755); err != nil {
		t.Fatalf("Failed to create test executable: %v", err)
	}

	// Test bundle creation to see if identifier gets populated
	bundleConfig := &bundlePkg.Config{
		AppName: cfg.AppName,
		Debug:   cfg.Debug,
	}

	testBundle, err := bundlePkg.New(execPath, bundleConfig)
	if err != nil {
		t.Fatalf("Failed to create bundle: %v", err)
	}

	bundleID := testBundle.BundleID()

	// Verify the bundle ID was properly inferred
	if bundleID == "" {
		t.Error("Bundle ID was not populated")
	}

	if strings.Contains(bundleID, "com.macgo") {
		t.Errorf("Bundle ID %q should not contain 'com.macgo'", bundleID)
	}

	if !strings.Contains(bundleID, ".") {
		t.Errorf("Bundle ID %q should contain at least one dot", bundleID)
	}

	t.Logf("Inferred bundle ID: %s", bundleID)
}

func TestEnvironmentBasedInference(t *testing.T) {
	// Test that environment variables affect bundle ID inference
	originalUser := os.Getenv("LOGNAME")
	defer func() {
		if originalUser == "" {
			os.Unsetenv("LOGNAME")
		} else {
			os.Setenv("LOGNAME", originalUser)
		}
	}()

	// Set a test username
	os.Setenv("LOGNAME", "testuser")

	bundleID := bundle.InferBundleID("TestApp")

	// Should contain the username in some form
	if !strings.Contains(bundleID, "testuser") {
		t.Errorf("Bundle ID %q should contain username 'testuser'", bundleID)
	}

	// Should not contain com.macgo
	if strings.Contains(bundleID, "com.macgo") {
		t.Errorf("Bundle ID %q should not contain 'com.macgo'", bundleID)
	}

	t.Logf("Environment-based bundle ID: %s", bundleID)
}
