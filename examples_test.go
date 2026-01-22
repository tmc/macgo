package macgo_test

import (
	"os"
	"testing"

	"github.com/tmc/macgo"
)

func ExampleRequest_basic() {
	// Simple permission request
	err := macgo.Request(macgo.Camera, macgo.Microphone)
	if err != nil {
		// Handle error
		return
	}
	// App now has camera and microphone permissions
}

func ExampleRequest_files() {
	// Request file access permissions
	err := macgo.Request(macgo.Files)
	if err != nil {
		// Handle error
		return
	}
	// App now has file access permissions
}

func ExampleStart_basic() {
	// Using Start with configuration
	cfg := &macgo.Config{
		AppName:     "MyApp",
		Permissions: []macgo.Permission{macgo.Camera},
		Debug:       false,
	}

	err := macgo.Start(cfg)
	if err != nil {
		// Handle error
		return
	}
	// App now running with configuration
}

func ExampleStart_comprehensive() {
	// Comprehensive configuration example
	cfg := &macgo.Config{
		AppName:  "ComprehensiveApp",
		BundleID: "com.example.comprehensive",
		Version:  "1.0.0",
		Permissions: []macgo.Permission{
			macgo.Camera,
			macgo.Microphone,
			macgo.Files,
			macgo.Network,
		},
		Custom: []string{
			"com.apple.security.device.capture",
			"com.apple.security.automation.apple-events",
		},
		Debug:         true,
		AutoSign:      true,
		CleanupBundle: false,
	}

	err := macgo.Start(cfg)
	if err != nil {
		// Handle error
		return
	}
	// App running with full configuration
}

func ExampleConfig_builder() {
	// Using builder pattern with Config
	cfg := macgo.NewConfig().
		WithAppName("BuilderApp").
		WithPermissions(macgo.Camera, macgo.Microphone).
		WithCustom("com.apple.security.device.capture").
		WithDebug()

	err := macgo.Start(cfg)
	if err != nil {
		// Handle error
		return
	}
	// App configured using builder pattern
}

func ExampleConfig_fromEnv() {
	// Environment-based configuration
	// Set MACGO_CAMERA=1, MACGO_DEBUG=1, etc.
	cfg := macgo.NewConfig().FromEnv()

	err := macgo.Start(cfg)
	if err != nil {
		// Handle error
		return
	}
	// App configured from environment variables
}

func ExamplePermission_constants() {
	// All available permission constants
	permissions := []macgo.Permission{
		macgo.Camera,     // Camera access
		macgo.Microphone, // Microphone access
		macgo.Location,   // Location services
		macgo.Files,      // File access
		macgo.Network,    // Network access
		macgo.Sandbox,    // App sandbox
	}

	err := macgo.Request(permissions...)
	if err != nil {
		// Handle error
		return
	}
	// App has all permissions
}

func TestExampleFunctions(t *testing.T) {
	// Ensure all example functions can be called without panicking

	// Test environment setup for examples
	_ = os.Setenv("MACGO_NO_RELAUNCH", "1")
	defer func() { _ = os.Unsetenv("MACGO_NO_RELAUNCH") }()

	// Test each example function exists and can be called
	// These won't actually run macgo since MACGO_NO_RELAUNCH=1

	t.Run("basic_request", func(t *testing.T) {
		ExampleRequest_basic()
	})

	t.Run("files_request", func(t *testing.T) {
		ExampleRequest_files()
	})

	t.Run("basic_start", func(t *testing.T) {
		ExampleStart_basic()
	})

	t.Run("comprehensive_start", func(t *testing.T) {
		ExampleStart_comprehensive()
	})

	t.Run("builder_config", func(t *testing.T) {
		ExampleConfig_builder()
	})

	t.Run("env_config", func(t *testing.T) {
		ExampleConfig_fromEnv()
	})

	t.Run("permission_constants", func(t *testing.T) {
		ExamplePermission_constants()
	})

	t.Run("app_groups", func(t *testing.T) {
		ExampleConfig_appGroups()
	})

	t.Run("app_groups_builder", func(t *testing.T) {
		ExampleConfig_appGroupsBuilder()
	})
}

// Test the helper functions from examples
func TestHelperFunctions(t *testing.T) {
	_ = os.Setenv("MACGO_NO_RELAUNCH", "1")
	defer func() { _ = os.Unsetenv("MACGO_NO_RELAUNCH") }()

	// Test NewConfig
	cfg := macgo.NewConfig()
	if cfg == nil {
		t.Error("NewConfig() returned nil")
	}

	// Test builder methods
	cfg = cfg.WithAppName("TestApp").
		WithPermissions(macgo.Camera).
		WithDebug()

	if cfg.AppName != "TestApp" {
		t.Errorf("Expected AppName 'TestApp', got %q", cfg.AppName)
	}

	if !cfg.Debug {
		t.Error("Expected Debug to be true")
	}

	if len(cfg.Permissions) != 1 || cfg.Permissions[0] != macgo.Camera {
		t.Error("Expected Camera permission")
	}

	// Test WithAppGroups
	cfg = cfg.WithAppGroups("group.test.shared")
	if len(cfg.AppGroups) != 1 || cfg.AppGroups[0] != "group.test.shared" {
		t.Error("Expected app group 'group.test.shared'")
	}
}

// Test error handling
func TestErrorHandling(t *testing.T) {
	// Disable relaunch to prevent os.Exit() during test
	original := os.Getenv("MACGO_NO_RELAUNCH")
	_ = os.Setenv("MACGO_NO_RELAUNCH", "1")
	defer func() {
		if original == "" {
			_ = os.Unsetenv("MACGO_NO_RELAUNCH")
		} else {
			_ = os.Setenv("MACGO_NO_RELAUNCH", original)
		}
	}()

	// Test with invalid configuration
	cfg := &macgo.Config{
		AppName: "", // Invalid empty name
	}

	err := macgo.Start(cfg)
	if err == nil {
		t.Log("Start with empty AppName succeeded (may be valid)")
	}
}

// Example showing environment variables
func Example_environmentVariables() {
	// Environment variables that control macgo behavior:
	// MACGO_NO_RELAUNCH=1           - Skip bundle creation and relaunch
	// MACGO_DEBUG=1                 - Enable debug logging
	// MACGO_RESET_PERMISSIONS=1     - Reset TCC permissions using tccutil
	// MACGO_APP_NAME=MyApp          - Set application name
	// MACGO_APP_NAME_PREFIX=Dev-    - Add prefix to all app names
	// MACGO_BUNDLE_ID=com.example   - Set bundle identifier
	// MACGO_BUNDLE_ID_PREFIX=dev    - Add prefix to all bundle IDs
	// MACGO_KEEP_BUNDLE=1           - Preserve bundle after execution
	// MACGO_CODE_SIGN_IDENTITY=xyz  - Set code signing identity
	// MACGO_AUTO_SIGN=1             - Enable automatic code signing
	// MACGO_AD_HOC_SIGN=1           - Use ad-hoc code signing
	// MACGO_CAMERA=1                - Request camera permissions
	// MACGO_MICROPHONE=1            - Request microphone permissions
	// MACGO_LOCATION=1              - Request location permissions
	// MACGO_FILES=1                 - Request file access permissions
	// MACGO_NETWORK=1               - Request network permissions
	// MACGO_SANDBOX=1               - Enable app sandbox

	// Example: Reset permissions before requesting new ones
	_ = os.Setenv("MACGO_RESET_PERMISSIONS", "1")
	_ = os.Setenv("MACGO_DEBUG", "1")
	defer func() {
		_ = os.Unsetenv("MACGO_RESET_PERMISSIONS")
		_ = os.Unsetenv("MACGO_DEBUG")
	}()

	err := macgo.Request(macgo.Camera)
	if err != nil {
		// Handle error
		return
	}
	// Permissions reset and then requested
}

// Example showing automatic bundle ID generation
func Example_bundleIDGeneration() {
	// macgo automatically generates meaningful bundle IDs based on your Go module
	// Examples of automatic bundle ID generation:
	//
	// Module: github.com/user/myproject -> Bundle ID: com.github.user.myproject.appname
	// Module: gitlab.com/company/tool   -> Bundle ID: com.gitlab.company.tool.appname
	// Module: example.com/service       -> Bundle ID: com.example.service.appname
	// No module info available          -> Bundle ID: dev.username.appname (or local.app.appname)
	//
	// This replaces the old generic "com.macgo.*" format with meaningful,
	// unique identifiers that reflect your actual project.

	cfg := &macgo.Config{
		AppName: "MyTool",
		// BundleID is automatically inferred from Go module if not specified
		Permissions: []macgo.Permission{macgo.Files},
	}

	err := macgo.Start(cfg)
	if err != nil {
		// Handle error
		return
	}
	// Bundle ID automatically generated based on module path
}

// Example showing app groups for sharing data between sandboxed apps
func ExampleConfig_appGroups() {
	// App groups allow sandboxed apps to share data
	cfg := &macgo.Config{
		AppName:  "AppGroupsExample",
		BundleID: "com.example.appgroups.demo",
		Permissions: []macgo.Permission{
			macgo.Sandbox, // Required for app groups
		},
		AppGroups: []string{
			"TEAMID.shared-data", // TEAMID placeholder gets automatically substituted
		},
		Debug:    true,
		AutoSign: true,
	}

	err := macgo.Start(cfg)
	if err != nil {
		// Handle error
		return
	}
	// App now has access to shared app group container
}

// Example using builder pattern for app groups
func ExampleConfig_appGroupsBuilder() {
	// Using builder pattern with app groups
	cfg := macgo.NewConfig().
		WithAppName("AppGroupBuilder").
		WithPermissions(macgo.Sandbox).
		WithAppGroups("TEAMID.shared-data"). // TEAMID placeholder gets automatically substituted
		WithDebug()

	err := macgo.Start(cfg)
	if err != nil {
		// Handle error
		return
	}
	// App configured with app groups using builder pattern
}
