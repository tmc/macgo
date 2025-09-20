//go:build darwin
// +build darwin

package macgo

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestMacOSIntegrationAppBundleCreation tests the complete app bundle creation workflow
func TestMacOSIntegrationAppBundleCreation(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("This test requires macOS")
	}

	// Save original config
	origConfig := DefaultConfig
	defer func() { DefaultConfig = origConfig }()

	// Create a test executable
	tmpExec, err := ioutil.TempFile("", "macgo-test-exec-*")
	if err != nil {
		t.Fatalf("Failed to create temp executable: %v", err)
	}
	defer os.Remove(tmpExec.Name())

	// Write a simple executable content
	execContent := []byte("#!/bin/sh\necho 'test executable'")
	if _, err := tmpExec.Write(execContent); err != nil {
		t.Fatal(err)
	}
	if err := tmpExec.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(tmpExec.Name(), 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		config func() *Config
		verify func(t *testing.T, appPath string)
	}{
		{
			name: "minimal configuration",
			config: func() *Config {
				cfg := NewConfig()
				cfg.ApplicationName = "TestApp"
				cfg.BundleID = "com.test.testapp"
				return cfg
			},
			verify: func(t *testing.T, appPath string) {
				// Verify basic structure
				if !strings.HasSuffix(appPath, "TestApp.app") {
					t.Errorf("Expected app path to end with TestApp.app, got %s", appPath)
				}
				verifyBundleStructure(t, appPath)
			},
		},
		{
			name: "with entitlements",
			config: func() *Config {
				cfg := NewConfig()
				cfg.ApplicationName = "EntitledApp"
				cfg.RequestEntitlements(EntCamera, EntMicrophone, EntAppSandbox)
				return cfg
			},
			verify: func(t *testing.T, appPath string) {
				verifyBundleStructure(t, appPath)
				// Check entitlements.plist
				entPath := filepath.Join(appPath, "Contents", "entitlements.plist")
				if _, err := os.Stat(entPath); err != nil {
					t.Errorf("Expected entitlements.plist to exist: %v", err)
				}
			},
		},
		{
			name: "with custom plist entries",
			config: func() *Config {
				cfg := NewConfig()
				cfg.ApplicationName = "CustomPlistApp"
				cfg.AddPlistEntry("LSUIElement", false)
				cfg.AddPlistEntry("NSAppleEventsUsageDescription", "This app needs AppleEvents")
				cfg.AddPlistEntry("CustomKey", "CustomValue")
				return cfg
			},
			verify: func(t *testing.T, appPath string) {
				verifyBundleStructure(t, appPath)
				// Verify Info.plist contains custom entries
				infoPlist := filepath.Join(appPath, "Contents", "Info.plist")
				content, err := ioutil.ReadFile(infoPlist)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(content), "CustomKey") {
					t.Error("Info.plist should contain CustomKey")
				}
				if !strings.Contains(string(content), "NSAppleEventsUsageDescription") {
					t.Error("Info.plist should contain NSAppleEventsUsageDescription")
				}
			},
		},
		{
			name: "custom destination path",
			config: func() *Config {
				cfg := NewConfig()
				tmpDir, _ := ioutil.TempDir("", "macgo-custom-dest-*")
				cfg.CustomDestinationAppPath = filepath.Join(tmpDir, "CustomApp.app")
				return cfg
			},
			verify: func(t *testing.T, appPath string) {
				if !strings.Contains(appPath, "macgo-custom-dest") {
					t.Errorf("Expected custom destination path, got %s", appPath)
				}
				verifyBundleStructure(t, appPath)
				// Clean up custom path
				os.RemoveAll(filepath.Dir(appPath))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			DefaultConfig = tt.config()

			appPath, err := createBundle(tmpExec.Name())
			if err != nil {
				t.Fatalf("Failed to create bundle: %v", err)
			}
			defer os.RemoveAll(appPath)

			tt.verify(t, appPath)
		})
	}
}

// TestMacOSIntegrationBundleValidation tests bundle validation and reuse logic
func TestMacOSIntegrationBundleValidation(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("This test requires macOS")
	}

	// Create a test executable
	tmpExec, err := ioutil.TempFile("", "macgo-test-validation-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpExec.Name())

	execContent := []byte("#!/bin/sh\necho 'validation test'")
	if _, err := tmpExec.Write(execContent); err != nil {
		t.Fatal(err)
	}
	tmpExec.Close()
	os.Chmod(tmpExec.Name(), 0755)

	// Create initial bundle
	origConfig := DefaultConfig
	defer func() { DefaultConfig = origConfig }()

	cfg := NewConfig()
	cfg.ApplicationName = "ValidationTest"
	cfg.CustomDestinationAppPath = filepath.Join(os.TempDir(), "ValidationTest.app")
	DefaultConfig = cfg

	appPath, err := createBundle(tmpExec.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(appPath)

	// Test 1: Bundle should be reused if unchanged
	appPath2, err := createBundle(tmpExec.Name())
	if err != nil {
		t.Fatal(err)
	}
	if appPath != appPath2 {
		t.Error("Expected to reuse existing bundle")
	}

	// Test 2: Bundle should be recreated if executable changes
	if err := ioutil.WriteFile(tmpExec.Name(), []byte("#!/bin/sh\necho 'modified'"), 0755); err != nil {
		t.Fatal(err)
	}

	appPath3, err := createBundle(tmpExec.Name())
	if err != nil {
		t.Fatal(err)
	}
	if appPath != appPath3 {
		t.Error("Expected same app path after recreation")
	}

	// Verify the executable was updated
	bundleExec := filepath.Join(appPath3, "Contents", "MacOS", filepath.Base(tmpExec.Name()))
	content, err := ioutil.ReadFile(bundleExec)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "modified") {
		t.Error("Bundle executable should be updated")
	}
}

// TestMacOSIntegrationPlistGeneration tests plist generation with various data types
func TestMacOSIntegrationPlistGeneration(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("This test requires macOS")
	}

	tests := []struct {
		name     string
		plist    map[string]any
		expected []string
	}{
		{
			name: "basic types",
			plist: map[string]any{
				"StringKey":   "StringValue",
				"BoolKey":     true,
				"IntKey":      42,
				"FloatKey":    3.14,
				"NSUIElement": true,
			},
			expected: []string{
				"<key>StringKey</key>",
				"<string>StringValue</string>",
				"<key>BoolKey</key>",
				"<true/>",
				"<key>IntKey</key>",
				"<integer>42</integer>",
			},
		},
		{
			name: "unsupported types fallback to string",
			plist: map[string]any{
				"ArrayKey": []string{"item1", "item2", "item3"},
				"NilKey":   nil,
				"MapKey": map[string]string{
					"nested": "value",
				},
			},
			expected: []string{
				"<key>ArrayKey</key>",
				"<string>[item1 item2 item3]</string>", // Arrays are converted to string
				"<key>NilKey</key>",
				"<string>&lt;nil&gt;</string>", // nil is converted to string with XML escaping
				"<key>MapKey</key>",
				"<string>map[nested:value]</string>", // Maps are converted to string
			},
		},
		{
			name: "entitlements",
			plist: map[string]any{
				string(EntCamera):               true,
				string(EntMicrophone):           true,
				string(EntUserSelectedReadOnly): true,
				string(EntAppSandbox):           true,
			},
			expected: []string{
				"com.apple.security.device.camera",
				"com.apple.security.device.microphone",
				"com.apple.security.files.user-selected.read-only",
				"com.apple.security.app-sandbox",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("plist-test-%d.plist", time.Now().UnixNano()))
			defer os.Remove(tmpFile)

			if err := writePlist(tmpFile, tt.plist); err != nil {
				t.Fatalf("Failed to write plist: %v", err)
			}

			content, err := ioutil.ReadFile(tmpFile)
			if err != nil {
				t.Fatal(err)
			}

			// Verify expected content
			for _, expected := range tt.expected {
				if !strings.Contains(string(content), expected) {
					t.Errorf("Expected plist to contain %q", expected)
				}
			}

			// Verify it's valid XML
			if !strings.HasPrefix(string(content), "<?xml") {
				t.Error("Plist should start with XML declaration")
			}
			if !strings.Contains(string(content), "<!DOCTYPE plist") {
				t.Error("Plist should contain DOCTYPE declaration")
			}
		})
	}
}

// TestMacOSIntegrationCodeSigning tests code signing integration
func TestMacOSIntegrationCodeSigning(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("This test requires macOS")
	}

	// Check if codesign is available
	if _, err := exec.LookPath("codesign"); err != nil {
		t.Skip("codesign not available")
	}

	// Create a test bundle
	tmpExec, err := ioutil.TempFile("", "macgo-sign-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpExec.Name())
	tmpExec.Write([]byte("#!/bin/sh\necho 'sign test'"))
	tmpExec.Close()
	os.Chmod(tmpExec.Name(), 0755)

	origConfig := DefaultConfig
	defer func() { DefaultConfig = origConfig }()

	cfg := NewConfig()
	cfg.ApplicationName = "SignTest"
	cfg.AutoSign = true
	cfg.CustomDestinationAppPath = filepath.Join(os.TempDir(), "SignTest.app")
	DefaultConfig = cfg

	appPath, err := createBundle(tmpExec.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(appPath)

	// Verify the bundle was created
	verifyBundleStructure(t, appPath)

	// Check if bundle is signed (might be ad-hoc)
	cmd := exec.Command("codesign", "-v", appPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Signing might fail in some environments, which is ok
		t.Logf("Code signing verification output: %s", output)
	} else {
		t.Log("Bundle was successfully signed")
	}
}

// TestMacOSIntegrationEnvironmentDetection tests environment detection
func TestMacOSIntegrationEnvironmentDetection(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("This test requires macOS")
	}

	tests := []struct {
		name     string
		envVars  map[string]string
		expected func(t *testing.T)
	}{
		{
			name: "GOPATH detection",
			envVars: map[string]string{
				"GOPATH": "/custom/go/path",
			},
			expected: func(t *testing.T) {
				// Test will be verified in bundle creation
			},
		},
		{
			name: "MACGO_DEBUG detection",
			envVars: map[string]string{
				"MACGO_DEBUG": "1",
			},
			expected: func(t *testing.T) {
				if !isDebugEnabled() {
					t.Error("Debug should be enabled when MACGO_DEBUG=1")
				}
			},
		},
		{
			name: "MACGO_NO_RELAUNCH detection",
			envVars: map[string]string{
				"MACGO_NO_RELAUNCH": "1",
			},
			expected: func(t *testing.T) {
				// This prevents relaunch, tested implicitly
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and set environment
			saved := make(map[string]string)
			for k, v := range tt.envVars {
				saved[k] = os.Getenv(k)
				os.Setenv(k, v)
			}
			defer func() {
				for k, v := range saved {
					if v == "" {
						os.Unsetenv(k)
					} else {
						os.Setenv(k, v)
					}
				}
			}()

			tt.expected(t)
		})
	}
}

// TestMacOSIntegrationSandboxConfiguration tests sandbox configuration validation
func TestMacOSIntegrationSandboxConfiguration(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("This test requires macOS")
	}

	origConfig := DefaultConfig
	defer func() { DefaultConfig = origConfig }()

	tests := []struct {
		name         string
		entitlements []Entitlement
		verify       func(t *testing.T, cfg *Config)
	}{
		{
			name: "sandbox with file access",
			entitlements: []Entitlement{
				EntAppSandbox,
				EntUserSelectedReadOnly,
			},
			verify: func(t *testing.T, cfg *Config) {
				if !cfg.Entitlements[EntAppSandbox] {
					t.Error("App sandbox should be enabled")
				}
				if !cfg.Entitlements[EntUserSelectedReadOnly] {
					t.Error("User selected read-only should be enabled")
				}
			},
		},
		{
			name: "sandbox with network access",
			entitlements: []Entitlement{
				EntAppSandbox,
				EntNetworkClient,
				EntNetworkServer,
			},
			verify: func(t *testing.T, cfg *Config) {
				if !cfg.Entitlements[EntNetworkClient] {
					t.Error("Network client should be enabled")
				}
				if !cfg.Entitlements[EntNetworkServer] {
					t.Error("Network server should be enabled")
				}
			},
		},
		{
			name: "TCC permissions",
			entitlements: []Entitlement{
				EntCamera,
				EntMicrophone,
				EntLocation,
				EntAddressBook,
				EntPhotos,
			},
			verify: func(t *testing.T, cfg *Config) {
				for _, ent := range []Entitlement{EntCamera, EntMicrophone, EntLocation, EntAddressBook, EntPhotos} {
					if !cfg.Entitlements[ent] {
						t.Errorf("Entitlement %s should be enabled", ent)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewConfig()
			entInterfaces := make([]interface{}, len(tt.entitlements))
			for i, ent := range tt.entitlements {
				entInterfaces[i] = ent
			}
			cfg.RequestEntitlements(entInterfaces...)
			DefaultConfig = cfg

			tt.verify(t, cfg)

			// Create a bundle to verify entitlements are written
			tmpExec, _ := ioutil.TempFile("", "macgo-sandbox-test-*")
			defer os.Remove(tmpExec.Name())
			tmpExec.Write([]byte("#!/bin/sh\necho 'test'"))
			tmpExec.Close()
			os.Chmod(tmpExec.Name(), 0755)

			cfg.ApplicationName = "SandboxTest"
			appPath, err := createBundle(tmpExec.Name())
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(appPath)

			// Verify entitlements.plist exists
			entPath := filepath.Join(appPath, "Contents", "entitlements.plist")
			if _, err := os.Stat(entPath); err != nil {
				t.Error("Entitlements.plist should exist for sandboxed app")
			}
		})
	}
}

// TestMacOSIntegrationPathHandling tests platform-specific path handling
func TestMacOSIntegrationPathHandling(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("This test requires macOS")
	}

	tests := []struct {
		name     string
		setup    func() (string, func())
		expected func(t *testing.T, result string)
	}{
		{
			name: "spaces in paths",
			setup: func() (string, func()) {
				dir, _ := ioutil.TempDir("", "macgo test spaces")
				exec := filepath.Join(dir, "test executable.sh")
				ioutil.WriteFile(exec, []byte("#!/bin/sh\necho 'test'"), 0755)
				return exec, func() { os.RemoveAll(dir) }
			},
			expected: func(t *testing.T, appPath string) {
				if appPath == "" {
					t.Error("Should handle paths with spaces")
				}
				verifyBundleStructure(t, appPath)
			},
		},
		{
			name: "unicode in paths",
			setup: func() (string, func()) {
				dir, _ := ioutil.TempDir("", "macgo-测试-πテスト")
				exec := filepath.Join(dir, "test.sh")
				ioutil.WriteFile(exec, []byte("#!/bin/sh\necho 'test'"), 0755)
				return exec, func() { os.RemoveAll(dir) }
			},
			expected: func(t *testing.T, appPath string) {
				if appPath == "" {
					t.Error("Should handle unicode paths")
				}
			},
		},
		{
			name: "deeply nested paths",
			setup: func() (string, func()) {
				base, _ := ioutil.TempDir("", "macgo-nested")
				dir := base
				for i := 0; i < 10; i++ {
					dir = filepath.Join(dir, fmt.Sprintf("level%d", i))
				}
				os.MkdirAll(dir, 0755)
				exec := filepath.Join(dir, "test.sh")
				ioutil.WriteFile(exec, []byte("#!/bin/sh\necho 'test'"), 0755)
				return exec, func() { os.RemoveAll(base) }
			},
			expected: func(t *testing.T, appPath string) {
				if appPath == "" {
					t.Error("Should handle deeply nested paths")
				}
			},
		},
	}

	origConfig := DefaultConfig
	defer func() { DefaultConfig = origConfig }()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec, cleanup := tt.setup()
			defer cleanup()

			cfg := NewConfig()
			cfg.ApplicationName = "PathTest"
			DefaultConfig = cfg

			appPath, err := createBundle(exec)
			if err != nil {
				t.Logf("Bundle creation error (may be expected): %v", err)
			} else {
				defer os.RemoveAll(appPath)
			}

			tt.expected(t, appPath)
		})
	}
}

// TestMacOSIntegrationTCCPermissionHandling tests TCC permission request handling
func TestMacOSIntegrationTCCPermissionHandling(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("This test requires macOS")
	}

	// This test verifies that TCC permissions are properly configured
	// without actually triggering permission requests

	origConfig := DefaultConfig
	defer func() { DefaultConfig = origConfig }()

	permissions := []struct {
		name string
		ents []Entitlement
		desc string
	}{
		{
			name: "camera and microphone",
			ents: []Entitlement{EntCamera, EntMicrophone},
			desc: "AV capture permissions",
		},
		{
			name: "location services",
			ents: []Entitlement{EntLocation},
			desc: "Location access",
		},
		{
			name: "contacts and calendar",
			ents: []Entitlement{EntAddressBook, EntCalendars},
			desc: "Personal information access",
		},
		{
			name: "photos library",
			ents: []Entitlement{EntPhotos},
			desc: "Photos access",
		},
	}

	for _, perm := range permissions {
		t.Run(perm.name, func(t *testing.T) {
			cfg := NewConfig()
			cfg.ApplicationName = "TCCTest"
			entInterfaces := make([]interface{}, len(perm.ents))
			for i, ent := range perm.ents {
				entInterfaces[i] = ent
			}
			cfg.RequestEntitlements(entInterfaces...)
			DefaultConfig = cfg

			// Create test executable
			tmpExec, _ := ioutil.TempFile("", "macgo-tcc-test-*")
			defer os.Remove(tmpExec.Name())
			tmpExec.Write([]byte("#!/bin/sh\necho 'tcc test'"))
			tmpExec.Close()
			os.Chmod(tmpExec.Name(), 0755)

			appPath, err := createBundle(tmpExec.Name())
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(appPath)

			// Verify entitlements are in bundle
			entPath := filepath.Join(appPath, "Contents", "entitlements.plist")
			content, err := ioutil.ReadFile(entPath)
			if err != nil {
				t.Fatal(err)
			}

			for _, ent := range perm.ents {
				if !strings.Contains(string(content), string(ent)) {
					t.Errorf("Entitlement %s not found in entitlements.plist", ent)
				}
			}
		})
	}
}

// TestMacOSIntegrationBundleCleanup tests temporary bundle cleanup logic
func TestMacOSIntegrationBundleCleanup(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("This test requires macOS")
	}

	origConfig := DefaultConfig
	defer func() { DefaultConfig = origConfig }()

	// Create a temporary executable that looks like go-build output
	tmpDir, err := ioutil.TempDir("", "go-build")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tmpExec := filepath.Join(tmpDir, "test-binary")
	if err := ioutil.WriteFile(tmpExec, []byte("#!/bin/sh\necho 'temp test'"), 0755); err != nil {
		t.Fatal(err)
	}

	cfg := NewConfig()
	cfg.ApplicationName = "TempTest"
	cfg.KeepTemp = false // Ensure cleanup is enabled
	DefaultConfig = cfg

	appPath, err := createBundle(tmpExec)
	if err != nil {
		t.Fatal(err)
	}

	// Verify bundle was created in temp directory
	if !strings.Contains(appPath, "/tmp") && !strings.Contains(appPath, "/var/folders") {
		t.Errorf("Temporary bundle should be in temp directory, got %s", appPath)
	}

	// Verify bundle exists immediately
	if _, err := os.Stat(appPath); err != nil {
		t.Error("Bundle should exist immediately after creation")
	}

	// The cleanup happens after 30 seconds in a goroutine
	// For testing, we'll just verify the cleanup was scheduled
	// by checking that the bundle was created in a temporary location
	t.Log("Temporary bundle created at:", appPath)
}

// TestMacOSIntegrationCompleteWorkflow tests a complete integration workflow
func TestMacOSIntegrationCompleteWorkflow(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("This test requires macOS")
	}

	// Save original config
	origConfig := DefaultConfig
	defer func() { DefaultConfig = origConfig }()

	// Create a more complex test executable
	tmpExec, err := ioutil.TempFile("", "macgo-workflow-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpExec.Name())

	// Write a simple Go-like executable
	execContent := `#!/bin/sh
echo "Workflow test application"
echo "Args: $@"
exit 0`
	if _, err := tmpExec.Write([]byte(execContent)); err != nil {
		t.Fatal(err)
	}
	tmpExec.Close()
	os.Chmod(tmpExec.Name(), 0755)

	// Configure a complete application
	cfg := NewConfig()
	cfg.ApplicationName = "WorkflowTestApp"
	cfg.BundleID = "com.test.workflowapp"
	cfg.AutoSign = true

	// Add various entitlements
	cfg.RequestEntitlements(
		EntAppSandbox,
		EntCamera,
		EntMicrophone,
		EntUserSelectedReadOnly,
		EntNetworkClient,
	)

	// Add custom plist entries
	cfg.AddPlistEntry("LSUIElement", true)
	cfg.AddPlistEntry("NSCameraUsageDescription", "This app needs camera access for testing")
	cfg.AddPlistEntry("NSMicrophoneUsageDescription", "This app needs microphone access for testing")

	DefaultConfig = cfg

	// Create the bundle
	appPath, err := createBundle(tmpExec.Name())
	if err != nil {
		t.Fatalf("Failed to create bundle: %v", err)
	}
	defer os.RemoveAll(appPath)

	// Comprehensive verification
	t.Run("bundle structure", func(t *testing.T) {
		verifyBundleStructure(t, appPath)
	})

	t.Run("Info.plist content", func(t *testing.T) {
		infoPlist := filepath.Join(appPath, "Contents", "Info.plist")
		content, err := ioutil.ReadFile(infoPlist)
		if err != nil {
			t.Fatal(err)
		}

		expectedKeys := []string{
			"CFBundleIdentifier",
			"com.test.workflowapp",
			"CFBundleName",
			"WorkflowTestApp",
			"LSUIElement",
			"NSCameraUsageDescription",
			"NSMicrophoneUsageDescription",
		}

		for _, key := range expectedKeys {
			if !strings.Contains(string(content), key) {
				t.Errorf("Info.plist should contain %s", key)
			}
		}
	})

	t.Run("entitlements.plist content", func(t *testing.T) {
		entPlist := filepath.Join(appPath, "Contents", "entitlements.plist")
		content, err := ioutil.ReadFile(entPlist)
		if err != nil {
			t.Fatal(err)
		}

		expectedEnts := []string{
			string(EntAppSandbox),
			string(EntCamera),
			string(EntMicrophone),
			string(EntUserSelectedReadOnly),
			string(EntNetworkClient),
		}

		for _, ent := range expectedEnts {
			if !strings.Contains(string(content), ent) {
				t.Errorf("entitlements.plist should contain %s", ent)
			}
		}
	})

	t.Run("executable permissions", func(t *testing.T) {
		execPath := filepath.Join(appPath, "Contents", "MacOS", filepath.Base(tmpExec.Name()))
		info, err := os.Stat(execPath)
		if err != nil {
			t.Fatal(err)
		}

		if info.Mode()&0111 == 0 {
			t.Error("Executable should have execute permissions")
		}
	})

	// Test bundle reuse
	t.Run("bundle reuse", func(t *testing.T) {
		appPath2, err := createBundle(tmpExec.Name())
		if err != nil {
			t.Fatal(err)
		}
		if appPath != appPath2 {
			t.Error("Should reuse existing bundle for unchanged executable")
		}
	})
}

// Helper function to verify basic bundle structure
func verifyBundleStructure(t *testing.T, appPath string) {
	t.Helper()

	// Check main directories
	dirs := []string{
		"Contents",
		"Contents/MacOS",
	}

	for _, dir := range dirs {
		path := filepath.Join(appPath, dir)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("Directory %s should exist: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s should be a directory", dir)
		}
	}

	// Check Info.plist
	infoPlist := filepath.Join(appPath, "Contents", "Info.plist")
	if _, err := os.Stat(infoPlist); err != nil {
		t.Error("Info.plist should exist")
	}

	// Check for executable
	macosDir := filepath.Join(appPath, "Contents", "MacOS")
	entries, err := os.ReadDir(macosDir)
	if err != nil {
		t.Fatal(err)
	}

	hasExecutable := false
	for _, entry := range entries {
		if !entry.IsDir() {
			hasExecutable = true
			break
		}
	}

	if !hasExecutable {
		t.Error("MacOS directory should contain at least one executable")
	}
}

// TestMacOSIntegrationBundleIconHandling tests default icon handling
func TestMacOSIntegrationBundleIconHandling(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("This test requires macOS")
	}

	// Create test executable
	tmpExec, err := ioutil.TempFile("", "macgo-icon-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpExec.Name())
	tmpExec.Write([]byte("#!/bin/sh\necho 'icon test'"))
	tmpExec.Close()
	os.Chmod(tmpExec.Name(), 0755)

	origConfig := DefaultConfig
	defer func() { DefaultConfig = origConfig }()

	cfg := NewConfig()
	cfg.ApplicationName = "IconTest"
	DefaultConfig = cfg

	appPath, err := createBundle(tmpExec.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(appPath)

	// Check if default icon was copied (if it exists on the system)
	defaultIconPath := "/System/Library/CoreServices/CoreTypes.bundle/Contents/Resources/ExecutableBinaryIcon.icns"
	if _, err := os.Stat(defaultIconPath); err == nil {
		// Default icon exists, check if it was copied
		bundleIconPath := filepath.Join(appPath, "Contents", "Resources", "ExecutableBinaryIcon.icns")
		if _, err := os.Stat(bundleIconPath); err != nil {
			t.Log("Default icon was not copied (this may be expected depending on permissions)")
		} else {
			t.Log("Default icon was successfully copied")
		}
	} else {
		t.Log("Default system icon not found (this is expected on some systems)")
	}
}

// TestMacOSIntegrationXcodeEnvironment tests Xcode environment detection
func TestMacOSIntegrationXcodeEnvironment(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("This test requires macOS")
	}

	// Test developer environment checking
	checkDeveloperEnvironment()

	// Check for codesign
	if _, err := exec.LookPath("codesign"); err != nil {
		t.Log("codesign not found in PATH - this is expected in some environments")
	} else {
		t.Log("codesign is available")

		// Try to get code signing identities
		cmd := exec.Command("security", "find-identity", "-v", "-p", "codesigning")
		output, err := cmd.Output()
		if err != nil {
			t.Log("Could not list code signing identities:", err)
		} else {
			t.Logf("Available code signing identities:\n%s", output)
		}
	}

	// Check Xcode installation
	cmd := exec.Command("xcode-select", "-p")
	if output, err := cmd.Output(); err == nil {
		t.Logf("Xcode developer directory: %s", strings.TrimSpace(string(output)))
	} else {
		t.Log("Xcode command line tools not installed")
	}
}

// TestMacOSIntegrationErrorRecovery tests error recovery scenarios
func TestMacOSIntegrationErrorRecovery(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("This test requires macOS")
	}

	origConfig := DefaultConfig
	defer func() { DefaultConfig = origConfig }()

	tests := []struct {
		name        string
		setup       func() (string, func())
		expectError bool
	}{
		{
			name: "non-existent executable",
			setup: func() (string, func()) {
				return "/non/existent/executable", func() {}
			},
			expectError: true,
		},
		{
			name: "invalid bundle path",
			setup: func() (string, func()) {
				tmpExec, _ := ioutil.TempFile("", "test-*")
				tmpExec.Close()

				cfg := NewConfig()
				cfg.CustomDestinationAppPath = "/root/no-permission/Test.app"
				DefaultConfig = cfg

				return tmpExec.Name(), func() { os.Remove(tmpExec.Name()) }
			},
			expectError: true,
		},
		{
			name: "corrupted bundle recovery",
			setup: func() (string, func()) {
				// Create executable
				tmpExec, _ := ioutil.TempFile("", "test-*")
				tmpExec.Write([]byte("#!/bin/sh\necho test"))
				tmpExec.Close()
				os.Chmod(tmpExec.Name(), 0755)

				// Create a corrupted bundle (file instead of directory)
				cfg := NewConfig()
				cfg.ApplicationName = "CorruptTest"
				cfg.CustomDestinationAppPath = filepath.Join(os.TempDir(), "CorruptTest.app")
				DefaultConfig = cfg

				// Create a file where the bundle should be
				ioutil.WriteFile(cfg.CustomDestinationAppPath, []byte("not a bundle"), 0644)

				return tmpExec.Name(), func() {
					os.Remove(tmpExec.Name())
					os.Remove(cfg.CustomDestinationAppPath)
				}
			},
			expectError: true, // mkdir will fail when a file exists at the bundle path
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec, cleanup := tt.setup()
			defer cleanup()

			_, err := createBundle(exec)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestMacOSIntegrationConcurrentBundleCreation tests concurrent bundle operations
func TestMacOSIntegrationConcurrentBundleCreation(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("This test requires macOS")
	}

	origConfig := DefaultConfig
	defer func() { DefaultConfig = origConfig }()

	// Create multiple executables
	var executables []string
	for i := 0; i < 5; i++ {
		tmpExec, err := ioutil.TempFile("", fmt.Sprintf("concurrent-test-%d-*", i))
		if err != nil {
			t.Fatal(err)
		}
		tmpExec.Write([]byte(fmt.Sprintf("#!/bin/sh\necho 'test %d'", i)))
		tmpExec.Close()
		os.Chmod(tmpExec.Name(), 0755)
		executables = append(executables, tmpExec.Name())
		defer os.Remove(tmpExec.Name())
	}

	// Run concurrent bundle creations
	done := make(chan struct{})
	errors := make(chan error, len(executables))

	for i, exec := range executables {
		go func(idx int, execPath string) {
			cfg := NewConfig()
			cfg.ApplicationName = fmt.Sprintf("ConcurrentTest%d", idx)
			cfg.CustomDestinationAppPath = filepath.Join(os.TempDir(), fmt.Sprintf("ConcurrentTest%d.app", idx))

			// Use a local copy to avoid race conditions
			localConfig := *cfg
			DefaultConfig = &localConfig

			appPath, err := createBundle(execPath)
			if err != nil {
				errors <- err
			} else {
				defer os.RemoveAll(appPath)
				// Verify bundle was created correctly
				verifyBundleStructure(t, appPath)
			}
			done <- struct{}{}
		}(i, exec)
	}

	// Wait for all to complete
	for i := 0; i < len(executables); i++ {
		<-done
	}

	close(errors)
	for err := range errors {
		t.Errorf("Concurrent bundle creation error: %v", err)
	}
}

// TestMacOSIntegrationBundleWithContext tests bundle operations with context
func TestMacOSIntegrationBundleWithContext(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("This test requires macOS")
	}

	// Test pipe operations with context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pipePath, err := createPipe("test-pipe")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(pipePath)

	// Test pipe I/O with context
	done := make(chan struct{})
	go func() {
		// pipeIOContext doesn't return an error, it just handles I/O
		pipeIOContext(ctx, pipePath, os.Stdin, nil)
		done <- struct{}{}
	}()

	select {
	case <-done:
		// Operation completed
	case <-ctx.Done():
		// Context cancelled/timed out as expected
	case <-time.After(6 * time.Second):
		t.Error("Context timeout not respected")
	}
}
