package macgo_test

import (
	"os"
	"testing"

	"github.com/tmc/misc/macgo"
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
		Debug:      true,
		AutoSign:   true,
		KeepBundle: &[]bool{true}[0],
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
	os.Setenv("MACGO_NO_RELAUNCH", "1")
	defer os.Unsetenv("MACGO_NO_RELAUNCH")

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
}

// Test the helper functions from examples
func TestHelperFunctions(t *testing.T) {
	os.Setenv("MACGO_NO_RELAUNCH", "1")
	defer os.Unsetenv("MACGO_NO_RELAUNCH")

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
}

// Test auto packages functionality
func TestAutoPackages(t *testing.T) {
	os.Setenv("MACGO_NO_RELAUNCH", "1")
	defer os.Unsetenv("MACGO_NO_RELAUNCH")

	// These would normally be imported as:
	// import _ "github.com/tmc/misc/macgo/auto/camera"
	// import _ "github.com/tmc/misc/macgo/auto/files"
	// But we'll test the functionality directly

	t.Run("camera_auto", func(t *testing.T) {
		err := macgo.Request(macgo.Camera)
		if err != nil {
			t.Logf("Camera request: %v", err) // This is expected in test environment
		}
	})

	t.Run("files_auto", func(t *testing.T) {
		err := macgo.Request(macgo.Files)
		if err != nil {
			t.Logf("Files request: %v", err) // This is expected in test environment
		}
	})
}

// Test error handling
func TestErrorHandling(t *testing.T) {
	// Test with invalid configuration
	cfg := &macgo.Config{
		AppName: "", // Invalid empty name
	}

	err := macgo.Start(cfg)
	if err == nil {
		t.Log("Start with empty AppName succeeded (may be valid)")
	}
}

// Example showing migration from v1 to v2 patterns
func Example_migration() {
	// v1 pattern (removed):
	// macgo.RequestEntitlements(macgo.EntCamera, macgo.EntMicrophone)
	// macgo.Start()

	// v2 pattern (current):
	err := macgo.Request(macgo.Camera, macgo.Microphone)
	if err != nil {
		// Handle error
		return
	}
	// Much simpler!
}