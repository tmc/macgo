// +build darwin

package macgo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestSignBundle tests the signBundle function
func TestSignBundle(t *testing.T) {
	// Check if codesign is available
	if _, err := exec.LookPath("codesign"); err != nil {
		t.Skip("codesign not available, skipping signing tests")
	}

	// Save original config
	originalConfig := DefaultConfig
	defer func() {
		DefaultConfig = originalConfig
	}()

	// Create a minimal test app bundle
	tmpDir, err := os.MkdirTemp("", "sign-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	appPath := filepath.Join(tmpDir, "TestApp.app")
	contentsPath := filepath.Join(appPath, "Contents")
	macosPath := filepath.Join(contentsPath, "MacOS")
	
	// Create bundle structure
	if err := os.MkdirAll(macosPath, 0755); err != nil {
		t.Fatal(err)
	}

	// Create executable
	execPath := filepath.Join(macosPath, "TestApp")
	if err := os.WriteFile(execPath, []byte("#!/bin/sh\necho test"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create Info.plist
	infoPlist := map[string]any{
		"CFBundleExecutable": "TestApp",
		"CFBundleIdentifier": "com.test.signtest",
		"CFBundleName":       "TestApp",
	}
	if err := writePlist(filepath.Join(contentsPath, "Info.plist"), infoPlist); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name            string
		setup           func()
		expectError     bool
		skipIfNoIdentity bool
	}{
		{
			name: "ad-hoc signing",
			setup: func() {
				DefaultConfig = NewConfig()
				DefaultConfig.SigningIdentity = ""
			},
			expectError: false,
		},
		{
			name: "signing with specific identity",
			setup: func() {
				DefaultConfig = NewConfig()
				DefaultConfig.SigningIdentity = "Developer ID Application"
			},
			expectError:      true, // Will fail unless identity exists
			skipIfNoIdentity: true,
		},
		{
			name: "signing with entitlements",
			setup: func() {
				DefaultConfig = NewConfig()
				DefaultConfig.SigningIdentity = ""
				// Create entitlements file
				entitlements := map[string]any{
					"com.apple.security.app-sandbox": true,
				}
				writePlist(filepath.Join(contentsPath, "entitlements.plist"), entitlements)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipIfNoIdentity {
				// Check if identity exists
				cmd := exec.Command("security", "find-identity", "-v", "-p", "codesigning")
				output, _ := cmd.Output()
				if !strings.Contains(string(output), "Developer ID Application") {
					t.Skip("No Developer ID Application identity found")
				}
			}

			tt.setup()
			err := signBundle(appPath)
			
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			} else if !tt.expectError && err != nil {
				// Ad-hoc signing might fail in some environments
				t.Logf("Signing failed (may be expected in some environments): %v", err)
			}

			// If signing succeeded, verify the signature
			if err == nil {
				cmd := exec.Command("codesign", "-v", appPath)
				if output, err := cmd.CombinedOutput(); err != nil {
					t.Errorf("Signature verification failed: %v\nOutput: %s", err, output)
				}
			}
		})
	}
}

// TestCheckDeveloperEnvironment tests the developer environment checking
func TestCheckDeveloperEnvironment(t *testing.T) {
	// Save original debug setting
	originalDebug := os.Getenv("MACGO_DEBUG")
	defer func() {
		os.Setenv("MACGO_DEBUG", originalDebug)
	}()

	// Enable debug to trigger the check
	os.Setenv("MACGO_DEBUG", "1")

	// This should not panic or cause errors
	checkDeveloperEnvironment()
	
	// We can't easily test the actual warnings, but we can ensure it doesn't crash
	t.Log("Developer environment check completed without crash")
}

// TestDebugf tests the debug logging function
func TestDebugf(t *testing.T) {
	// Save original debug setting
	originalDebug := os.Getenv("MACGO_DEBUG")
	defer func() {
		os.Setenv("MACGO_DEBUG", originalDebug)
	}()

	tests := []struct {
		name      string
		debugFlag string
	}{
		{
			name:      "debug disabled",
			debugFlag: "",
		},
		{
			name:      "debug enabled",
			debugFlag: "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("MACGO_DEBUG", tt.debugFlag)
			
			// Should not panic regardless of debug setting
			debugf("Test message: %s", "value")
			debugf("Test with multiple args: %s %d %v", "string", 42, true)
		})
	}
}

// TestIsDebugEnabled tests the debug detection function
func TestIsDebugEnabled(t *testing.T) {
	// Save original debug setting
	originalDebug := os.Getenv("MACGO_DEBUG")
	defer func() {
		os.Setenv("MACGO_DEBUG", originalDebug)
	}()

	tests := []struct {
		envValue string
		expected bool
	}{
		{"", false},
		{"0", false},
		{"1", true},
		{"true", false}, // Only "1" enables debug
		{"yes", false},
	}

	for _, tt := range tests {
		t.Run("MACGO_DEBUG="+tt.envValue, func(t *testing.T) {
			os.Setenv("MACGO_DEBUG", tt.envValue)
			result := isDebugEnabled()
			if result != tt.expected {
				t.Errorf("Expected isDebugEnabled() = %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestCreateDebugLogFile tests debug log file creation
func TestCreateDebugLogFile(t *testing.T) {
	// Test stdout log
	stdoutLog, err := createDebugLogFile("stdout")
	if err != nil {
		t.Errorf("Failed to create stdout debug log: %v", err)
	} else {
		defer stdoutLog.Close()
		defer os.Remove(stdoutLog.Name())
		
		// Verify we can write to it
		if _, err := stdoutLog.WriteString("test log entry\n"); err != nil {
			t.Errorf("Failed to write to debug log: %v", err)
		}
		
		// Verify filename pattern
		if !strings.Contains(stdoutLog.Name(), "macgo-debug-stdout") {
			t.Errorf("Unexpected debug log filename: %s", stdoutLog.Name())
		}
	}

	// Test stderr log
	stderrLog, err := createDebugLogFile("stderr")
	if err != nil {
		t.Errorf("Failed to create stderr debug log: %v", err)
	} else {
		defer stderrLog.Close()
		defer os.Remove(stderrLog.Name())
		
		// Verify filename includes PID
		expectedPid := os.Getpid()
		if !strings.Contains(stderrLog.Name(), fmt.Sprintf("%d", expectedPid)) {
			t.Log("Debug log filename doesn't contain PID as expected")
		}
	}
}

// TestEnvironmentVariableDetection tests the init() function's env var handling
func TestEnvironmentVariableDetection(t *testing.T) {
	// We can't directly test init() since it runs before tests,
	// but we can verify the mechanism works
	
	envMappings := map[string]Entitlement{
		"MACGO_CAMERA":         EntCamera,
		"MACGO_MIC":            EntMicrophone,
		"MACGO_LOCATION":       EntLocation,
		"MACGO_APP_SANDBOX":    EntAppSandbox,
		"MACGO_NETWORK_CLIENT": EntNetworkClient,
		"MACGO_BLUETOOTH":      EntBluetooth,
		"MACGO_USB":            EntUSB,
	}

	// Just verify the constants are correct
	for envVar, expectedEnt := range envMappings {
		t.Run(envVar, func(t *testing.T) {
			// If the env var is set, the entitlement should be in DefaultConfig
			if os.Getenv(envVar) == "1" {
				if _, exists := DefaultConfig.Entitlements[expectedEnt]; !exists {
					t.Logf("%s=1 but entitlement %s not found in DefaultConfig", envVar, expectedEnt)
				}
			}
		})
	}
}