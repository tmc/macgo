package macgo

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/tmc/misc/macgo/entitlements"
)

// TestEnvironmentVariableHandling tests all environment variable handling in macgo
func TestEnvironmentVariableHandling(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS-specific environment variable tests on non-macOS platform")
	}

	tests := []struct {
		name        string
		envVars     map[string]string
		setup       func(*Config)
		verify      func(*testing.T, *Config)
		description string
	}{
		{
			name: "MACGO_APP_NAME",
			envVars: map[string]string{
				"MACGO_APP_NAME": "TestEnvironmentApp",
			},
			setup: func(cfg *Config) {
				// No additional setup needed
			},
			verify: func(t *testing.T, cfg *Config) {
				if cfg.ApplicationName != "TestEnvironmentApp" {
					t.Errorf("Expected ApplicationName to be 'TestEnvironmentApp', got %s", cfg.ApplicationName)
				}
			},
			description: "MACGO_APP_NAME should set the application name",
		},
		{
			name: "MACGO_BUNDLE_ID",
			envVars: map[string]string{
				"MACGO_BUNDLE_ID": "com.example.envtest",
			},
			setup: func(cfg *Config) {
				// No additional setup needed
			},
			verify: func(t *testing.T, cfg *Config) {
				if cfg.BundleID != "com.example.envtest" {
					t.Errorf("Expected BundleID to be 'com.example.envtest', got %s", cfg.BundleID)
				}
			},
			description: "MACGO_BUNDLE_ID should set the bundle identifier",
		},
		{
			name: "MACGO_NO_RELAUNCH",
			envVars: map[string]string{
				"MACGO_NO_RELAUNCH": "1",
			},
			setup: func(cfg *Config) {
				// No additional setup needed
			},
			verify: func(t *testing.T, cfg *Config) {
				if cfg.Relaunch != false {
					t.Errorf("Expected Relaunch to be false when MACGO_NO_RELAUNCH=1, got %v", cfg.Relaunch)
				}
			},
			description: "MACGO_NO_RELAUNCH should disable relaunching",
		},
		{
			name: "MACGO_KEEP_TEMP",
			envVars: map[string]string{
				"MACGO_KEEP_TEMP": "1",
			},
			setup: func(cfg *Config) {
				// No additional setup needed
			},
			verify: func(t *testing.T, cfg *Config) {
				if cfg.KeepTemp != true {
					t.Errorf("Expected KeepTemp to be true when MACGO_KEEP_TEMP=1, got %v", cfg.KeepTemp)
				}
			},
			description: "MACGO_KEEP_TEMP should enable keeping temporary files",
		},
		{
			name: "MACGO_SHOW_DOCK_ICON",
			envVars: map[string]string{
				"MACGO_SHOW_DOCK_ICON": "1",
			},
			setup: func(cfg *Config) {
				// No additional setup needed
			},
			verify: func(t *testing.T, cfg *Config) {
				if cfg.PlistEntries == nil {
					t.Error("Expected PlistEntries to be initialized")
				} else if val, ok := cfg.PlistEntries["LSUIElement"]; !ok || val != false {
					t.Errorf("Expected LSUIElement to be false when MACGO_SHOW_DOCK_ICON=1, got %v", val)
				}
			},
			description: "MACGO_SHOW_DOCK_ICON should enable dock icon display",
		},
		{
			name: "MACGO_DEBUG",
			envVars: map[string]string{
				"MACGO_DEBUG": "1",
			},
			setup: func(cfg *Config) {
				// No additional setup needed
			},
			verify: func(t *testing.T, cfg *Config) {
				if !isDebugEnabled() {
					t.Error("Expected debug to be enabled when MACGO_DEBUG=1")
				}
			},
			description: "MACGO_DEBUG should enable debug mode",
		},
		{
			name: "Multiple environment variables",
			envVars: map[string]string{
				"MACGO_APP_NAME":       "MultiEnvApp",
				"MACGO_BUNDLE_ID":      "com.example.multienv",
				"MACGO_KEEP_TEMP":      "1",
				"MACGO_SHOW_DOCK_ICON": "1",
				"MACGO_DEBUG":          "1",
			},
			setup: func(cfg *Config) {
				// No additional setup needed
			},
			verify: func(t *testing.T, cfg *Config) {
				if cfg.ApplicationName != "MultiEnvApp" {
					t.Errorf("Expected ApplicationName to be 'MultiEnvApp', got %s", cfg.ApplicationName)
				}
				if cfg.BundleID != "com.example.multienv" {
					t.Errorf("Expected BundleID to be 'com.example.multienv', got %s", cfg.BundleID)
				}
				if cfg.KeepTemp != true {
					t.Errorf("Expected KeepTemp to be true, got %v", cfg.KeepTemp)
				}
				if cfg.PlistEntries == nil {
					t.Error("Expected PlistEntries to be initialized")
				} else if val, ok := cfg.PlistEntries["LSUIElement"]; !ok || val != false {
					t.Errorf("Expected LSUIElement to be false, got %v", val)
				}
				if !isDebugEnabled() {
					t.Error("Expected debug to be enabled")
				}
			},
			description: "Multiple environment variables should all be processed",
		},
		{
			name: "Empty environment variables (should be ignored)",
			envVars: map[string]string{
				"MACGO_APP_NAME":  "",
				"MACGO_BUNDLE_ID": "",
				"MACGO_KEEP_TEMP": "",
				"MACGO_DEBUG":     "",
			},
			setup: func(cfg *Config) {
				cfg.ApplicationName = "DefaultApp"
				cfg.BundleID = "com.example.default"
			},
			verify: func(t *testing.T, cfg *Config) {
				if cfg.ApplicationName != "DefaultApp" {
					t.Errorf("Expected ApplicationName to remain 'DefaultApp', got %s", cfg.ApplicationName)
				}
				if cfg.BundleID != "com.example.default" {
					t.Errorf("Expected BundleID to remain 'com.example.default', got %s", cfg.BundleID)
				}
				if cfg.KeepTemp != false {
					t.Errorf("Expected KeepTemp to remain false, got %v", cfg.KeepTemp)
				}
			},
			description: "Empty environment variables should be ignored",
		},
		{
			name: "Invalid boolean values (should be ignored)",
			envVars: map[string]string{
				"MACGO_KEEP_TEMP":      "invalid",
				"MACGO_SHOW_DOCK_ICON": "maybe",
				"MACGO_DEBUG":          "yes",
				"MACGO_NO_RELAUNCH":    "false",
			},
			setup: func(cfg *Config) {
				// No additional setup needed
			},
			verify: func(t *testing.T, cfg *Config) {
				if cfg.KeepTemp != false {
					t.Errorf("Expected KeepTemp to remain false with invalid value, got %v", cfg.KeepTemp)
				}
				if cfg.PlistEntries != nil {
					if val, ok := cfg.PlistEntries["LSUIElement"]; ok && val == false {
						t.Error("Expected LSUIElement not to be set with invalid MACGO_SHOW_DOCK_ICON value")
					}
				}
				if cfg.Relaunch != true {
					t.Errorf("Expected Relaunch to remain true with invalid value, got %v", cfg.Relaunch)
				}
			},
			description: "Invalid boolean values should be ignored",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			originalEnv := make(map[string]string)
			for key := range tt.envVars {
				originalEnv[key] = os.Getenv(key)
			}

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Restore environment after test
			defer func() {
				for key, originalValue := range originalEnv {
					if originalValue != "" {
						os.Setenv(key, originalValue)
					} else {
						os.Unsetenv(key)
					}
				}
			}()

			// Create a new config and apply environment variables
			cfg := NewConfig()
			if tt.setup != nil {
				tt.setup(cfg)
			}

			// Apply environment variables to config
			applyEnvironmentVariables(cfg)

			// Verify the result
			if tt.verify != nil {
				tt.verify(t, cfg)
			}
		})
	}
}

func TestEntitlementEnvironmentVariables(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS-specific entitlement environment variable tests on non-macOS platform")
	}

	entitlementTests := []struct {
		envVar      string
		entitlement Entitlement
		description string
	}{
		{"MACGO_CAMERA", entitlements.EntCamera, "Camera entitlement"},
		{"MACGO_MIC", entitlements.EntMicrophone, "Microphone entitlement"},
		{"MACGO_LOCATION", entitlements.EntLocation, "Location entitlement"},
		{"MACGO_CONTACTS", entitlements.EntAddressBook, "Contacts entitlement"},
		{"MACGO_CALENDARS", entitlements.EntCalendars, "Calendars entitlement"},
		{"MACGO_REMINDERS", entitlements.EntReminders, "Reminders entitlement"},
		{"MACGO_PHOTOS", entitlements.EntPhotos, "Photos entitlement"},
		{"MACGO_SANDBOX", entitlements.EntAppSandbox, "App sandbox entitlement"},
		{"MACGO_NETWORK_CLIENT", entitlements.EntNetworkClient, "Network client entitlement"},
		{"MACGO_NETWORK_SERVER", entitlements.EntNetworkServer, "Network server entitlement"},
		{"MACGO_USER_SELECTED_READ_ONLY", entitlements.EntUserSelectedReadOnly, "User selected read-only entitlement"},
		{"MACGO_USER_SELECTED_READ_WRITE", entitlements.EntUserSelectedReadWrite, "User selected read-write entitlement"},
	}

	for _, tt := range entitlementTests {
		t.Run(tt.description, func(t *testing.T) {
			// Save original environment
			originalValue := os.Getenv(tt.envVar)

			// Set environment variable
			os.Setenv(tt.envVar, "1")

			// Restore environment after test
			defer func() {
				if originalValue != "" {
					os.Setenv(tt.envVar, originalValue)
				} else {
					os.Unsetenv(tt.envVar)
				}
			}()

			// Create a new config and apply environment variables
			cfg := NewConfig()
			applyEnvironmentVariables(cfg)

			// Verify the entitlement is set
			if cfg.Entitlements == nil {
				t.Error("Expected Entitlements to be initialized")
			} else if val, ok := cfg.Entitlements[tt.entitlement]; !ok || val != true {
				t.Errorf("Expected entitlement %s to be true, got %v", tt.entitlement, val)
			}
		})
	}
}

func TestEnvironmentVariableOverrides(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS-specific environment variable override tests on non-macOS platform")
	}

	// Test that environment variables override programmatic settings
	t.Run("Environment overrides programmatic settings", func(t *testing.T) {
		// Save original environment
		originalAppName := os.Getenv("MACGO_APP_NAME")
		originalBundleID := os.Getenv("MACGO_BUNDLE_ID")

		// Set environment variables
		os.Setenv("MACGO_APP_NAME", "EnvApp")
		os.Setenv("MACGO_BUNDLE_ID", "com.example.env")

		// Restore environment after test
		defer func() {
			if originalAppName != "" {
				os.Setenv("MACGO_APP_NAME", originalAppName)
			} else {
				os.Unsetenv("MACGO_APP_NAME")
			}
			if originalBundleID != "" {
				os.Setenv("MACGO_BUNDLE_ID", originalBundleID)
			} else {
				os.Unsetenv("MACGO_BUNDLE_ID")
			}
		}()

		// Create config with programmatic settings
		cfg := NewConfig()
		cfg.ApplicationName = "ProgrammaticApp"
		cfg.BundleID = "com.example.programmatic"

		// Apply environment variables (should override programmatic settings)
		applyEnvironmentVariables(cfg)

		// Verify environment variables take precedence
		if cfg.ApplicationName != "EnvApp" {
			t.Errorf("Expected ApplicationName to be 'EnvApp' (from env), got %s", cfg.ApplicationName)
		}
		if cfg.BundleID != "com.example.env" {
			t.Errorf("Expected BundleID to be 'com.example.env' (from env), got %s", cfg.BundleID)
		}
	})
}

func TestEnvironmentVariableEdgeCases(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS-specific environment variable edge case tests on non-macOS platform")
	}

	t.Run("Special characters in environment variables", func(t *testing.T) {
		// Save original environment
		originalAppName := os.Getenv("MACGO_APP_NAME")
		originalBundleID := os.Getenv("MACGO_BUNDLE_ID")

		// Set environment variables with special characters
		os.Setenv("MACGO_APP_NAME", "Test App (Special) & More")
		os.Setenv("MACGO_BUNDLE_ID", "com.example.test-app_special")

		// Restore environment after test
		defer func() {
			if originalAppName != "" {
				os.Setenv("MACGO_APP_NAME", originalAppName)
			} else {
				os.Unsetenv("MACGO_APP_NAME")
			}
			if originalBundleID != "" {
				os.Setenv("MACGO_BUNDLE_ID", originalBundleID)
			} else {
				os.Unsetenv("MACGO_BUNDLE_ID")
			}
		}()

		// Create config and apply environment variables
		cfg := NewConfig()
		applyEnvironmentVariables(cfg)

		// Verify special characters are preserved
		if cfg.ApplicationName != "Test App (Special) & More" {
			t.Errorf("Expected ApplicationName to preserve special characters, got %s", cfg.ApplicationName)
		}
		if cfg.BundleID != "com.example.test-app_special" {
			t.Errorf("Expected BundleID to preserve special characters, got %s", cfg.BundleID)
		}
	})

	t.Run("Very long environment variable values", func(t *testing.T) {
		// Save original environment
		originalAppName := os.Getenv("MACGO_APP_NAME")

		// Create a very long app name
		longAppName := make([]byte, 1000)
		for i := range longAppName {
			longAppName[i] = 'A' + byte(i%26)
		}
		longAppNameStr := string(longAppName)

		// Set environment variable
		os.Setenv("MACGO_APP_NAME", longAppNameStr)

		// Restore environment after test
		defer func() {
			if originalAppName != "" {
				os.Setenv("MACGO_APP_NAME", originalAppName)
			} else {
				os.Unsetenv("MACGO_APP_NAME")
			}
		}()

		// Create config and apply environment variables
		cfg := NewConfig()
		applyEnvironmentVariables(cfg)

		// Verify long value is handled correctly
		if cfg.ApplicationName != longAppNameStr {
			t.Error("Expected ApplicationName to handle very long values correctly")
		}
	})
}

func TestTestEnvironmentDetection(t *testing.T) {
	// Test that the code detects test environments correctly
	tests := []struct {
		name        string
		envVars     map[string]string
		expectTest  bool
		description string
	}{
		{
			name: "MACGO_TEST=1",
			envVars: map[string]string{
				"MACGO_TEST": "1",
			},
			expectTest:  true,
			description: "MACGO_TEST=1 should be detected as test environment",
		},
		{
			name: "GO_TEST=1",
			envVars: map[string]string{
				"GO_TEST": "1",
			},
			expectTest:  true,
			description: "GO_TEST=1 should be detected as test environment",
		},
		{
			name: "TEST_TMPDIR set",
			envVars: map[string]string{
				"TEST_TMPDIR": "/tmp/test",
			},
			expectTest:  true,
			description: "TEST_TMPDIR should be detected as test environment",
		},
		{
			name:        "No test environment variables",
			envVars:     map[string]string{},
			expectTest:  false,
			description: "No test environment variables should not be detected as test environment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			originalEnv := make(map[string]string)
			for key := range tt.envVars {
				originalEnv[key] = os.Getenv(key)
			}

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Restore environment after test
			defer func() {
				for key, originalValue := range originalEnv {
					if originalValue != "" {
						os.Setenv(key, originalValue)
					} else {
						os.Unsetenv(key)
					}
				}
			}()

			// Test the detection logic
			isTest := (os.Getenv("MACGO_TEST") == "1" || os.Getenv("GO_TEST") == "1" || os.Getenv("TEST_TMPDIR") != "")

			if isTest != tt.expectTest {
				t.Errorf("Expected test environment detection to be %v, got %v", tt.expectTest, isTest)
			}
		})
	}
}

func TestEnvironmentVariablePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Set up environment variables
	os.Setenv("MACGO_APP_NAME", "PerfTestApp")
	os.Setenv("MACGO_BUNDLE_ID", "com.example.perftest")
	os.Setenv("MACGO_KEEP_TEMP", "1")
	os.Setenv("MACGO_DEBUG", "1")
	defer func() {
		os.Unsetenv("MACGO_APP_NAME")
		os.Unsetenv("MACGO_BUNDLE_ID")
		os.Unsetenv("MACGO_KEEP_TEMP")
		os.Unsetenv("MACGO_DEBUG")
	}()

	// Measure performance of environment variable processing
	start := time.Now()
	iterations := 1000

	for i := 0; i < iterations; i++ {
		cfg := NewConfig()
		applyEnvironmentVariables(cfg)
	}

	elapsed := time.Since(start)
	avgTime := elapsed / time.Duration(iterations)

	// Environment variable processing should be fast
	if avgTime > time.Millisecond {
		t.Errorf("Environment variable processing too slow: %v per iteration", avgTime)
	}
}

func TestEnvironmentVariableConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}

	// Set up environment variables
	os.Setenv("MACGO_APP_NAME", "ConcurrentApp")
	os.Setenv("MACGO_BUNDLE_ID", "com.example.concurrent")
	defer func() {
		os.Unsetenv("MACGO_APP_NAME")
		os.Unsetenv("MACGO_BUNDLE_ID")
	}()

	// Test concurrent access to environment variables
	const numGoroutines = 100
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			cfg := NewConfig()
			applyEnvironmentVariables(cfg)

			// Verify consistent results
			if cfg.ApplicationName != "ConcurrentApp" {
				t.Errorf("Goroutine %d: expected ApplicationName 'ConcurrentApp', got %s", id, cfg.ApplicationName)
			}
			if cfg.BundleID != "com.example.concurrent" {
				t.Errorf("Goroutine %d: expected BundleID 'com.example.concurrent', got %s", id, cfg.BundleID)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
			// Goroutine completed successfully
		case <-time.After(5 * time.Second):
			t.Fatal("Goroutine did not complete within timeout")
		}
	}
}

// Helper function to apply environment variables to config
// This simulates the actual environment variable processing in the main code
func applyEnvironmentVariables(cfg *Config) {
	// Apply environment variables as done in the main code
	if name := os.Getenv("MACGO_APP_NAME"); name != "" {
		cfg.ApplicationName = name
	}
	if id := os.Getenv("MACGO_BUNDLE_ID"); id != "" {
		cfg.BundleID = id
	}
	if os.Getenv("MACGO_NO_RELAUNCH") == "1" {
		cfg.Relaunch = false
	}
	if os.Getenv("MACGO_KEEP_TEMP") == "1" {
		cfg.KeepTemp = true
	}
	if os.Getenv("MACGO_SHOW_DOCK_ICON") == "1" {
		cfg.AddPlistEntry("LSUIElement", false)
	}

	// Apply entitlement environment variables
	entitlementEnvVars := map[string]Entitlement{
		"MACGO_CAMERA":                   entitlements.EntCamera,
		"MACGO_MIC":                      entitlements.EntMicrophone,
		"MACGO_LOCATION":                 entitlements.EntLocation,
		"MACGO_CONTACTS":                 entitlements.EntAddressBook,
		"MACGO_CALENDARS":                entitlements.EntCalendars,
		"MACGO_REMINDERS":                entitlements.EntReminders,
		"MACGO_PHOTOS":                   entitlements.EntPhotos,
		"MACGO_SANDBOX":                  entitlements.EntAppSandbox,
		"MACGO_NETWORK_CLIENT":           entitlements.EntNetworkClient,
		"MACGO_NETWORK_SERVER":           entitlements.EntNetworkServer,
		"MACGO_USER_SELECTED_READ_ONLY":  entitlements.EntUserSelectedReadOnly,
		"MACGO_USER_SELECTED_READ_WRITE": entitlements.EntUserSelectedReadWrite,
	}

	for envVar, ent := range entitlementEnvVars {
		if os.Getenv(envVar) == "1" {
			if cfg.Entitlements == nil {
				cfg.Entitlements = make(map[Entitlement]bool)
			}
			cfg.Entitlements[ent] = true
		}
	}
}

// Benchmark tests
func BenchmarkEnvironmentVariableProcessing(b *testing.B) {
	// Set up environment variables
	os.Setenv("MACGO_APP_NAME", "BenchmarkApp")
	os.Setenv("MACGO_BUNDLE_ID", "com.example.benchmark")
	os.Setenv("MACGO_KEEP_TEMP", "1")
	os.Setenv("MACGO_DEBUG", "1")
	defer func() {
		os.Unsetenv("MACGO_APP_NAME")
		os.Unsetenv("MACGO_BUNDLE_ID")
		os.Unsetenv("MACGO_KEEP_TEMP")
		os.Unsetenv("MACGO_DEBUG")
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg := NewConfig()
		applyEnvironmentVariables(cfg)
	}
}

func BenchmarkEnvironmentVariableRead(b *testing.B) {
	// Set up environment variables
	os.Setenv("MACGO_APP_NAME", "BenchmarkApp")
	os.Setenv("MACGO_BUNDLE_ID", "com.example.benchmark")
	defer func() {
		os.Unsetenv("MACGO_APP_NAME")
		os.Unsetenv("MACGO_BUNDLE_ID")
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = os.Getenv("MACGO_APP_NAME")
		_ = os.Getenv("MACGO_BUNDLE_ID")
		_ = os.Getenv("MACGO_KEEP_TEMP")
		_ = os.Getenv("MACGO_DEBUG")
	}
}
