package macgo_test

import (
	"os"
	"strings"
	"testing"

	"github.com/tmc/misc/macgo/helpers/bundle"
)

func TestEnvironmentPrefixes(t *testing.T) {
	// Save original environment
	originalAppPrefix := os.Getenv("MACGO_APP_NAME_PREFIX")
	originalBundlePrefix := os.Getenv("MACGO_BUNDLE_ID_PREFIX")

	defer func() {
		if originalAppPrefix == "" {
			os.Unsetenv("MACGO_APP_NAME_PREFIX")
		} else {
			os.Setenv("MACGO_APP_NAME_PREFIX", originalAppPrefix)
		}
		if originalBundlePrefix == "" {
			os.Unsetenv("MACGO_BUNDLE_ID_PREFIX")
		} else {
			os.Setenv("MACGO_BUNDLE_ID_PREFIX", originalBundlePrefix)
		}
	}()

	t.Run("app_name_prefix", func(t *testing.T) {
		os.Setenv("MACGO_APP_NAME_PREFIX", "Dev-")
		defer os.Unsetenv("MACGO_APP_NAME_PREFIX")

		result := bundle.CleanAppName("MyApp")
		expected := "Dev-MyApp"

		if result != expected {
			t.Errorf("CleanAppName with MACGO_APP_NAME_PREFIX=Dev- got %q, want %q", result, expected)
		}
	})

	t.Run("bundle_id_prefix", func(t *testing.T) {
		os.Setenv("MACGO_BUNDLE_ID_PREFIX", "development")
		defer os.Unsetenv("MACGO_BUNDLE_ID_PREFIX")

		result := bundle.InferBundleID("MyApp")

		if !strings.HasPrefix(result, "development.") {
			t.Errorf("InferBundleID with MACGO_BUNDLE_ID_PREFIX=development got %q, expected to start with 'development.'", result)
		}
	})

	t.Run("bundle_id_prefix_with_dot", func(t *testing.T) {
		os.Setenv("MACGO_BUNDLE_ID_PREFIX", "test.")
		defer os.Unsetenv("MACGO_BUNDLE_ID_PREFIX")

		result := bundle.InferBundleID("MyApp")

		if !strings.HasPrefix(result, "test.") {
			t.Errorf("InferBundleID with MACGO_BUNDLE_ID_PREFIX=test. got %q, expected to start with 'test.'", result)
		}

		// Should not have double dots
		if strings.Contains(result, "..") {
			t.Errorf("InferBundleID result %q should not contain consecutive dots", result)
		}
	})

	t.Run("empty_prefixes", func(t *testing.T) {
		os.Unsetenv("MACGO_APP_NAME_PREFIX")
		os.Unsetenv("MACGO_BUNDLE_ID_PREFIX")

		appResult := bundle.CleanAppName("MyApp")
		bundleResult := bundle.InferBundleID("MyApp")

		if appResult != "MyApp" {
			t.Errorf("CleanAppName without prefix got %q, want MyApp", appResult)
		}

		// Bundle result should not start with any added prefix
		// (it may still have prefixes from module path inference, which is fine)
		if bundleResult == "" {
			t.Error("InferBundleID should return a valid bundle ID")
		}
	})
}

func Example_environmentPrefixes() {
	// Example showing how to use environment variables for prefixes

	// Set app name prefix
	os.Setenv("MACGO_APP_NAME_PREFIX", "Dev-")
	appName := bundle.CleanAppName("MyApp")
	// appName is now "Dev-MyApp"
	_ = appName

	// Set bundle ID prefix
	os.Setenv("MACGO_BUNDLE_ID_PREFIX", "development")
	bundleID := bundle.InferBundleID("MyApp")
	// bundleID now starts with "development." followed by the inferred ID
	_ = bundleID

	// Clean up
	os.Unsetenv("MACGO_APP_NAME_PREFIX")
	os.Unsetenv("MACGO_BUNDLE_ID_PREFIX")
}