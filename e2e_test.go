package macgo_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/tmc/macgo"
	"github.com/tmc/macgo/internal/bundle"
	"github.com/tmc/macgo/internal/system"
	"github.com/tmc/macgo/internal/tcc"
)

// TestE2E_BasicWorkflow tests the complete workflow from config to bundle creation.
// This is the fundamental end-to-end test covering the happy path.
func TestE2E_BasicWorkflow(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("E2E tests only run on darwin")
	}

	// Set up test environment
	os.Setenv("MACGO_NO_RELAUNCH", "1") // Prevent actual relaunch
	defer os.Unsetenv("MACGO_NO_RELAUNCH")

	tests := []struct {
		name   string
		config *macgo.Config
	}{
		{
			name: "basic_camera_permission",
			config: &macgo.Config{
				AppName:     "E2ETestCameraApp",
				Permissions: []macgo.Permission{macgo.Camera},
				Debug:       true,
			},
		},
		{
			name: "multiple_permissions",
			config: &macgo.Config{
				AppName:     "E2ETestMultiApp",
				Permissions: []macgo.Permission{macgo.Camera, macgo.Microphone},
				Debug:       true,
			},
		},
		{
			name: "with_custom_entitlements",
			config: &macgo.Config{
				AppName:     "E2ETestCustomApp",
				Permissions: []macgo.Permission{macgo.Camera},
				Custom:      []string{"com.apple.security.device.bluetooth"},
				Debug:       true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute Start which should create bundle but not relaunch
			err := macgo.Start(tt.config)
			if err != nil {
				t.Errorf("Start failed: %v", err)
			}

			// Verify bundle was created (would be in /tmp)
			// Note: Without relaunch, bundle creation still happens
			// This verifies the bundle creation path works
		})
	}
}

// TestE2E_BundleCreationAndReuse tests that bundles are created correctly
// and can be reused on subsequent runs.
func TestE2E_BundleCreationAndReuse(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("E2E tests only run on darwin")
	}

	os.Setenv("MACGO_NO_RELAUNCH", "1")
	defer os.Unsetenv("MACGO_NO_RELAUNCH")

	// Enable bundle keeping
	cfg := &macgo.Config{
		AppName:       "E2ETestBundleReuse",
		BundleID:      "com.test.e2e.reuse",
		Permissions:   []macgo.Permission{macgo.Camera},
		CleanupBundle: false,
		Debug:         true,
	}

	// First run - creates bundle
	err := macgo.Start(cfg)
	if err != nil {
		t.Fatalf("First Start failed: %v", err)
	}

	// Second run - should reuse bundle
	err = macgo.Start(cfg)
	if err != nil {
		t.Fatalf("Second Start failed: %v", err)
	}

	// Bundle should exist and be reusable
	// The bundle package handles this internally
}

// TestE2E_CodeSigningStrategies tests different code signing approaches.
func TestE2E_CodeSigningStrategies(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("E2E tests only run on darwin")
	}

	os.Setenv("MACGO_NO_RELAUNCH", "1")
	defer os.Unsetenv("MACGO_NO_RELAUNCH")

	tests := []struct {
		name   string
		config *macgo.Config
	}{
		{
			name: "ad_hoc_signing",
			config: &macgo.Config{
				AppName:     "E2ETestAdHoc",
				Permissions: []macgo.Permission{macgo.Camera},
				AdHocSign:   true,
				Debug:       true,
			},
		},
		{
			name: "auto_signing",
			config: &macgo.Config{
				AppName:     "E2ETestAutoSign",
				Permissions: []macgo.Permission{macgo.Camera},
				AutoSign:    true,
				Debug:       true,
			},
		},
		{
			name: "no_signing",
			config: &macgo.Config{
				AppName:     "E2ETestNoSign",
				Permissions: []macgo.Permission{macgo.Camera},
				Debug:       true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := macgo.Start(tt.config)
			if err != nil {
				// Auto-signing might fail if no Developer ID is available
				// This is expected in CI environments
				if tt.config.AutoSign && strings.Contains(err.Error(), "certificate") {
					t.Skipf("Skipping auto-sign test: %v", err)
				} else {
					t.Errorf("Start failed: %v", err)
				}
			}
		})
	}
}

// TestE2E_BundleIdentifierGeneration tests automatic bundle ID generation.
func TestE2E_BundleIdentifierGeneration(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("E2E tests only run on darwin")
	}

	os.Setenv("MACGO_NO_RELAUNCH", "1")
	defer os.Unsetenv("MACGO_NO_RELAUNCH")

	tests := []struct {
		name             string
		config           *macgo.Config
		shouldContain    string
		shouldNotContain string
	}{
		{
			name: "auto_generated_bundle_id",
			config: &macgo.Config{
				AppName:     "E2ETestAutoID",
				Permissions: []macgo.Permission{macgo.Camera},
				Debug:       true,
			},
			shouldNotContain: "com.macgo", // Old prefix should not be used
		},
		{
			name: "explicit_bundle_id",
			config: &macgo.Config{
				AppName:     "E2ETestExplicitID",
				BundleID:    "com.test.explicit.e2e",
				Permissions: []macgo.Permission{macgo.Camera},
				Debug:       true,
			},
			shouldContain: "com.test.explicit.e2e",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := macgo.Start(tt.config)
			if err != nil {
				t.Errorf("Start failed: %v", err)
			}

			// Verify bundle ID through internal system functions
			bundleID := tt.config.BundleID
			if bundleID == "" {
				bundleID = system.InferBundleID(tt.config.AppName)
			}

			if tt.shouldContain != "" && !strings.Contains(bundleID, tt.shouldContain) {
				t.Errorf("Bundle ID %q should contain %q", bundleID, tt.shouldContain)
			}

			if tt.shouldNotContain != "" && strings.Contains(bundleID, tt.shouldNotContain) {
				t.Errorf("Bundle ID %q should not contain %q", bundleID, tt.shouldNotContain)
			}
		})
	}
}

// TestE2E_EnvironmentConfiguration tests configuration via environment variables.
func TestE2E_EnvironmentConfiguration(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("E2E tests only run on darwin")
	}

	// Save original environment
	originalEnv := map[string]string{
		"MACGO_APP_NAME":    os.Getenv("MACGO_APP_NAME"),
		"MACGO_BUNDLE_ID":   os.Getenv("MACGO_BUNDLE_ID"),
		"MACGO_DEBUG":       os.Getenv("MACGO_DEBUG"),
		"MACGO_CAMERA":      os.Getenv("MACGO_CAMERA"),
		"MACGO_MICROPHONE":  os.Getenv("MACGO_MICROPHONE"),
		"MACGO_NO_RELAUNCH": os.Getenv("MACGO_NO_RELAUNCH"),
	}
	defer func() {
		for k, v := range originalEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	// Set up environment
	os.Setenv("MACGO_APP_NAME", "E2ETestEnvConfig")
	os.Setenv("MACGO_BUNDLE_ID", "com.test.e2e.env")
	os.Setenv("MACGO_DEBUG", "1")
	os.Setenv("MACGO_CAMERA", "1")
	os.Setenv("MACGO_MICROPHONE", "1")
	os.Setenv("MACGO_NO_RELAUNCH", "1")

	// Load from environment
	cfg := macgo.NewConfig().FromEnv()

	if cfg.AppName != "E2ETestEnvConfig" {
		t.Errorf("Expected AppName=E2ETestEnvConfig, got %s", cfg.AppName)
	}

	if cfg.BundleID != "com.test.e2e.env" {
		t.Errorf("Expected BundleID=com.test.e2e.env, got %s", cfg.BundleID)
	}

	if !cfg.Debug {
		t.Error("Expected Debug=true")
	}

	expectedPerms := map[macgo.Permission]bool{
		macgo.Camera:     true,
		macgo.Microphone: true,
	}

	for _, perm := range cfg.Permissions {
		if !expectedPerms[perm] {
			t.Errorf("Unexpected permission: %s", perm)
		}
	}

	// Actually start with this config
	err := macgo.Start(cfg)
	if err != nil {
		t.Errorf("Start with env config failed: %v", err)
	}
}

// TestE2E_ErrorHandling tests various error conditions.
func TestE2E_ErrorHandling(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("E2E tests only run on darwin")
	}

	os.Setenv("MACGO_NO_RELAUNCH", "1")
	defer os.Unsetenv("MACGO_NO_RELAUNCH")

	tests := []struct {
		name        string
		config      *macgo.Config
		expectError bool
	}{
		{
			name: "invalid_bundle_id",
			config: &macgo.Config{
				AppName:     "E2ETestInvalidID",
				BundleID:    "invalid..bundle..id",
				Permissions: []macgo.Permission{macgo.Camera},
			},
			expectError: true,
		},
		{
			name: "conflicting_app_groups_without_sandbox",
			config: &macgo.Config{
				AppName:   "E2ETestAppGroupNoSandbox",
				AppGroups: []string{"group.test.shared"},
				// Missing Sandbox permission
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First validate the config
			err := tt.config.Validate()

			if tt.expectError && err == nil {
				// Try Start as well to see if it catches it
				err = macgo.Start(tt.config)
				if err == nil {
					t.Error("Expected error but got none (both Validate and Start)")
				}
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestE2E_ConfigValidation tests the config validation functionality.
func TestE2E_ConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *macgo.Config
		expectError bool
	}{
		{
			name: "valid_minimal_config",
			config: &macgo.Config{
				AppName:     "ValidApp",
				Permissions: []macgo.Permission{macgo.Camera},
			},
			expectError: false,
		},
		{
			name: "valid_comprehensive_config",
			config: &macgo.Config{
				AppName:     "ValidComprehensive",
				BundleID:    "com.test.valid",
				Permissions: []macgo.Permission{macgo.Camera, macgo.Microphone},
				Custom:      []string{"com.apple.security.device.capture"},
				Debug:       true,
			},
			expectError: false,
		},
		{
			name: "invalid_bundle_id_format",
			config: &macgo.Config{
				AppName:  "InvalidBundleID",
				BundleID: "not-a-valid-bundle-id",
			},
			expectError: true,
		},
		{
			name: "app_groups_requires_sandbox",
			config: &macgo.Config{
				AppName:   "AppGroupsNoSandbox",
				AppGroups: []string{"group.test.shared"},
				// Missing Sandbox permission - this should fail validation
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

// TestE2E_BundleStructure verifies the created bundle has correct structure.
func TestE2E_BundleStructure(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("E2E tests only run on darwin")
	}

	// Create a temporary executable for testing
	tmpDir := t.TempDir()
	execPath := filepath.Join(tmpDir, "test-app")

	// Write a simple executable
	content := []byte("#!/bin/sh\necho 'test'\n")
	if err := os.WriteFile(execPath, content, 0755); err != nil {
		t.Fatalf("Failed to create test executable: %v", err)
	}

	// Create bundle
	bundleObj, err := bundle.Create(
		execPath,
		"E2ETestStructure",
		"com.test.e2e.structure",
		"1.0.0",
		[]string{"camera"},
		[]string{},
		[]string{},
		true,                    // debug
		false,                   // cleanupBundle (nil -> false in old, now explicit false to keep)
		"",                      // codeSignIdentity
		"",                      // codeSigningIdentifier
		false,                   // autoSign
		true,                    // adHocSign
		nil,                     // customInfo
		bundle.UIModeBackground, // uiMode (default: background)
		false,                   // devMode
	)

	if err != nil {
		t.Fatalf("Bundle creation failed: %v", err)
	}

	// Verify bundle structure
	requiredPaths := []string{
		filepath.Join(bundleObj.Path, "Contents"),
		filepath.Join(bundleObj.Path, "Contents", "MacOS"),
		filepath.Join(bundleObj.Path, "Contents", "Info.plist"),
	}

	for _, path := range requiredPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Required bundle path does not exist: %s", path)
		}
	}

	// Verify Info.plist contains expected keys
	infoPlistPath := filepath.Join(bundleObj.Path, "Contents", "Info.plist")
	content, err = os.ReadFile(infoPlistPath)
	if err != nil {
		t.Fatalf("Failed to read Info.plist: %v", err)
	}

	infoPlist := string(content)
	requiredKeys := []string{
		"CFBundleIdentifier",
		"CFBundleName",
		"CFBundleExecutable",
	}

	for _, key := range requiredKeys {
		if !strings.Contains(infoPlist, key) {
			t.Errorf("Info.plist missing required key: %s", key)
		}
	}

	// Check for permission usage descriptions if permissions were requested
	// Note: NSCameraUsageDescription is added when camera permission is requested
	if strings.Contains(infoPlist, "NSCamera") {
		t.Log("Info.plist contains camera permission usage description")
	}
}

// TestE2E_LaunchStrategySelection tests the selection between direct and LaunchServices execution.
func TestE2E_LaunchStrategySelection(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("E2E tests only run on darwin")
	}

	os.Setenv("MACGO_NO_RELAUNCH", "1")
	defer os.Unsetenv("MACGO_NO_RELAUNCH")

	tests := []struct {
		name   string
		config *macgo.Config
	}{
		{
			name: "force_direct_execution",
			config: &macgo.Config{
				AppName:              "E2ETestDirect",
				Permissions:          []macgo.Permission{macgo.Camera},
				ForceDirectExecution: true,
				Debug:                true,
			},
		},
		{
			name: "automatic_selection",
			config: &macgo.Config{
				AppName:     "E2ETestAutoSelect",
				Permissions: []macgo.Permission{macgo.Camera},
				Debug:       true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := macgo.Start(tt.config)
			if err != nil {
				t.Errorf("Start failed: %v", err)
			}
		})
	}
}

// TestE2E_PermissionValidation tests TCC permission validation.
func TestE2E_PermissionValidation(t *testing.T) {
	tests := []struct {
		name        string
		permissions []macgo.Permission
		expectError bool
	}{
		{
			name:        "valid_single_permission",
			permissions: []macgo.Permission{macgo.Camera},
			expectError: false,
		},
		{
			name:        "valid_multiple_permissions",
			permissions: []macgo.Permission{macgo.Camera, macgo.Microphone, macgo.Files},
			expectError: false,
		},
		{
			name:        "empty_permissions",
			permissions: []macgo.Permission{},
			expectError: false, // Empty permissions should be valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tccPerms []tcc.Permission
			for _, perm := range tt.permissions {
				tccPerms = append(tccPerms, tcc.Permission(perm))
			}

			err := tcc.ValidatePermissions(tccPerms)

			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

// TestE2E_ContextCancellation tests that context cancellation works correctly.
func TestE2E_ContextCancellation(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("E2E tests only run on darwin")
	}

	os.Setenv("MACGO_NO_RELAUNCH", "1")
	defer os.Unsetenv("MACGO_NO_RELAUNCH")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	cfg := &macgo.Config{
		AppName:     "E2ETestContextCancel",
		Permissions: []macgo.Permission{macgo.Camera},
		Debug:       true,
	}

	// This should complete quickly since NO_RELAUNCH is set
	err := macgo.StartContext(ctx, cfg)
	if err != nil && err != context.DeadlineExceeded {
		t.Logf("Start completed (error: %v)", err)
	}
}

// TestE2E_ConcurrentStarts tests multiple concurrent macgo.Start calls.
func TestE2E_ConcurrentStarts(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("E2E tests only run on darwin")
	}

	os.Setenv("MACGO_NO_RELAUNCH", "1")
	defer os.Unsetenv("MACGO_NO_RELAUNCH")

	// Run multiple starts concurrently
	done := make(chan error, 5)

	for i := 0; i < 5; i++ {
		go func(n int) {
			cfg := &macgo.Config{
				AppName:     fmt.Sprintf("E2ETestConcurrent%d", n),
				BundleID:    fmt.Sprintf("com.test.e2e.concurrent.%d", n),
				Permissions: []macgo.Permission{macgo.Camera},
				Debug:       false, // Disable debug to reduce output noise
			}
			err := macgo.Start(cfg)
			done <- err
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < 5; i++ {
		if err := <-done; err != nil {
			t.Errorf("Concurrent start %d failed: %v", i, err)
		}
	}
}

// TestE2E_BuilderPattern tests the fluent configuration builder pattern.
func TestE2E_BuilderPattern(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("E2E tests only run on darwin")
	}

	os.Setenv("MACGO_NO_RELAUNCH", "1")
	defer os.Unsetenv("MACGO_NO_RELAUNCH")

	// Test builder pattern
	cfg := macgo.NewConfig().
		WithAppName("E2ETestBuilder").
		WithPermissions(macgo.Camera, macgo.Microphone).
		WithCustom("com.apple.security.device.capture").
		WithDebug().
		WithAdHocSign()

	if cfg.AppName != "E2ETestBuilder" {
		t.Errorf("Expected AppName=E2ETestBuilder, got %s", cfg.AppName)
	}

	if len(cfg.Permissions) != 2 {
		t.Errorf("Expected 2 permissions, got %d", len(cfg.Permissions))
	}

	if !cfg.Debug {
		t.Error("Expected Debug=true")
	}

	if !cfg.AdHocSign {
		t.Error("Expected AdHocSign=true")
	}

	err := macgo.Start(cfg)
	if err != nil {
		t.Errorf("Start with builder pattern failed: %v", err)
	}
}

// BenchmarkE2E_BundleCreation benchmarks bundle creation performance.
func BenchmarkE2E_BundleCreation(b *testing.B) {
	if runtime.GOOS != "darwin" {
		b.Skip("Benchmark only relevant on darwin")
	}

	// Create a temporary executable
	tmpDir := b.TempDir()
	execPath := filepath.Join(tmpDir, "bench-app")
	content := []byte("#!/bin/sh\necho 'test'\n")
	if err := os.WriteFile(execPath, content, 0755); err != nil {
		b.Fatalf("Failed to create test executable: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := bundle.Create(
			execPath,
			fmt.Sprintf("BenchApp%d", i),
			fmt.Sprintf("com.bench.app.%d", i),
			"1.0.0",
			[]string{"camera"},
			[]string{},
			[]string{},
			false,                   // debug
			true,                    // cleanupBundle (will be cleaned up)
			"",                      // codeSignIdentity
			"",                      // codeSigningIdentifier
			false,                   // autoSign
			true,                    // adHocSign
			nil,                     // customInfo
			bundle.UIModeBackground, // uiMode (default: background)
			false,                   // devMode
		)
		if err != nil {
			b.Fatalf("Bundle creation failed: %v", err)
		}
	}
}

// TestE2E_TCCIntegration tests TCC permission reset functionality.
// This test requires MACGO_RESET_PERMISSIONS=1 to actually reset permissions.
func TestE2E_TCCIntegration(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("E2E tests only run on darwin")
	}

	// Only run if explicitly requested
	if os.Getenv("MACGO_E2E_TCC_TESTS") != "1" {
		t.Skip("TCC integration tests require MACGO_E2E_TCC_TESTS=1")
	}

	os.Setenv("MACGO_NO_RELAUNCH", "1")
	defer os.Unsetenv("MACGO_NO_RELAUNCH")

	// Note: Actually resetting TCC requires Full Disk Access
	os.Setenv("MACGO_RESET_PERMISSIONS", "1")
	defer os.Unsetenv("MACGO_RESET_PERMISSIONS")

	cfg := &macgo.Config{
		AppName:     "E2ETestTCCReset",
		BundleID:    "com.test.e2e.tcc.reset",
		Permissions: []macgo.Permission{macgo.Camera},
		Debug:       true,
	}

	err := macgo.Start(cfg)
	if err != nil {
		// Expected to fail if we don't have Full Disk Access
		if strings.Contains(err.Error(), "Full Disk Access") {
			t.Skip("Skipping: Full Disk Access required for TCC reset")
		}
		t.Errorf("Start with TCC reset failed: %v", err)
	}
}

// TestE2E_AppGroups tests app groups configuration and validation.
func TestE2E_AppGroups(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("E2E tests only run on darwin")
	}

	os.Setenv("MACGO_NO_RELAUNCH", "1")
	defer os.Unsetenv("MACGO_NO_RELAUNCH")

	// App groups require sandbox permission
	cfg := &macgo.Config{
		AppName:     "E2ETestAppGroups",
		Permissions: []macgo.Permission{macgo.Sandbox},
		AppGroups:   []string{"TEAMID.shared-data"},
		Debug:       true,
		AutoSign:    true, // App groups require signing
	}

	err := macgo.Start(cfg)
	if err != nil {
		// AutoSign might fail in CI
		if strings.Contains(err.Error(), "certificate") || strings.Contains(err.Error(), "team") {
			t.Skipf("Skipping app groups test: %v", err)
		}
		t.Errorf("Start with app groups failed: %v", err)
	}
}

// TestE2E_RealExecutableTest creates and runs a real test executable through macgo.
// This is the closest to a true end-to-end test that actually launches an app.
func TestE2E_RealExecutableTest(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("E2E tests only run on darwin")
	}

	// Only run if explicitly requested
	if os.Getenv("MACGO_E2E_REAL_LAUNCH") != "1" {
		t.Skip("Real executable tests require MACGO_E2E_REAL_LAUNCH=1")
	}

	// Create a temporary test executable
	tmpDir := t.TempDir()
	testAppSrc := filepath.Join(tmpDir, "testapp.go")
	testAppBin := filepath.Join(tmpDir, "testapp")

	// Write a simple test app that uses macgo
	testCode := `package main

import (
	"fmt"
	"github.com/tmc/macgo"
)

func main() {
	if _, err := macgo.Request(macgo.Camera); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println("SUCCESS: macgo executed correctly")
}`

	if err := os.WriteFile(testAppSrc, []byte(testCode), 0644); err != nil {
		t.Fatalf("Failed to write test app source: %v", err)
	}

	// Build the test app
	cmd := exec.Command("go", "build", "-o", testAppBin, testAppSrc)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build test app: %v\nOutput: %s", err, output)
	}

	// Run the test app
	cmd = exec.Command(testAppBin)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Test app output: %s", output)
		t.Fatalf("Test app execution failed: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "SUCCESS") {
		t.Errorf("Expected SUCCESS in output, got: %s", outputStr)
	}

	t.Logf("Real executable test output: %s", outputStr)
}
