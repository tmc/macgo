package bundle

import (
	"os"
	"strings"
	"testing"
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
			bundleID := InferBundleID(tt.appName)

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

func TestEnvironmentBasedInference(t *testing.T) {
	// Test that environment variables affect fallback bundle ID inference
	originalUser := os.Getenv("LOGNAME")
	defer func() {
		if originalUser == "" {
			_ = os.Unsetenv("LOGNAME")
		} else {
			_ = os.Setenv("LOGNAME", originalUser)
		}
	}()

	// Set a test username
	_ = os.Setenv("LOGNAME", "testuser")

	// Test the fallback function directly, which uses environment variables
	bundleID := InferFallbackBundleID("TestApp")

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
