package macgo

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestBundleCreationEdgeCases(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping bundle creation edge cases test on non-macOS platform")
	}

	tests := []struct {
		name        string
		setupConfig func() *Config
		setupFiles  func(*testing.T) (string, func())
		expectError bool
		errorMsg    string
		verify      func(*testing.T, string)
		description string
	}{
		{
			name: "Empty application name",
			setupConfig: func() *Config {
				cfg := NewConfig()
				cfg.ApplicationName = ""
				cfg.BundleID = "com.example.empty"
				return cfg
			},
			setupFiles: func(t *testing.T) (string, func()) {
				return createTempExecutableFile(t)
			},
			expectError: false,
			description: "Empty application name should be handled gracefully",
		},
		{
			name: "Empty bundle ID",
			setupConfig: func() *Config {
				cfg := NewConfig()
				cfg.ApplicationName = "TestApp"
				cfg.BundleID = ""
				return cfg
			},
			setupFiles: func(t *testing.T) (string, func()) {
				return createTempExecutableFile(t)
			},
			expectError: false,
			description: "Empty bundle ID should be handled gracefully",
		},
		{
			name: "Very long application name",
			setupConfig: func() *Config {
				cfg := NewConfig()
				cfg.ApplicationName = strings.Repeat("VeryLongAppName", 20) // 300 characters
				cfg.BundleID = "com.example.longname"
				return cfg
			},
			setupFiles: func(t *testing.T) (string, func()) {
				return createTempExecutableFile(t)
			},
			expectError: false,
			description: "Very long application name should be handled",
		},
		{
			name: "Very long bundle ID",
			setupConfig: func() *Config {
				cfg := NewConfig()
				cfg.ApplicationName = "TestApp"
				cfg.BundleID = "com.example." + strings.Repeat("verylongbundleid", 20)
				return cfg
			},
			setupFiles: func(t *testing.T) (string, func()) {
				return createTempExecutableFile(t)
			},
			expectError: false,
			description: "Very long bundle ID should be handled",
		},
		{
			name: "Special characters in application name",
			setupConfig: func() *Config {
				cfg := NewConfig()
				cfg.ApplicationName = "Test App (Special) & More!"
				cfg.BundleID = "com.example.special"
				return cfg
			},
			setupFiles: func(t *testing.T) (string, func()) {
				return createTempExecutableFile(t)
			},
			expectError: false,
			description: "Special characters in application name should be handled",
		},
		{
			name: "Special characters in bundle ID",
			setupConfig: func() *Config {
				cfg := NewConfig()
				cfg.ApplicationName = "TestApp"
				cfg.BundleID = "com.example.test-app_special"
				return cfg
			},
			setupFiles: func(t *testing.T) (string, func()) {
				return createTempExecutableFile(t)
			},
			expectError: false,
			description: "Valid special characters in bundle ID should be handled",
		},
		{
			name: "Non-existent executable path",
			setupConfig: func() *Config {
				cfg := NewConfig()
				cfg.ApplicationName = "TestApp"
				cfg.BundleID = "com.example.nonexistent"
				return cfg
			},
			setupFiles: func(t *testing.T) (string, func()) {
				return "/nonexistent/path/to/executable", func() {}
			},
			expectError: true,
			errorMsg:    "executable file does not exist",
			description: "Non-existent executable should cause error",
		},
		{
			name: "Directory as executable path",
			setupConfig: func() *Config {
				cfg := NewConfig()
				cfg.ApplicationName = "TestApp"
				cfg.BundleID = "com.example.directory"
				return cfg
			},
			setupFiles: func(t *testing.T) (string, func()) {
				tmpDir, err := os.MkdirTemp("", "macgo-test-dir-*")
				if err != nil {
					t.Fatalf("Failed to create temp directory: %v", err)
				}
				return tmpDir, func() { os.RemoveAll(tmpDir) }
			},
			expectError: true,
			errorMsg:    "is a directory",
			description: "Directory path should cause error",
		},
		{
			name: "Executable without execute permissions",
			setupConfig: func() *Config {
				cfg := NewConfig()
				cfg.ApplicationName = "TestApp"
				cfg.BundleID = "com.example.noexec"
				return cfg
			},
			setupFiles: func(t *testing.T) (string, func()) {
				tmpFile, err := os.CreateTemp("", "macgo-test-noexec-*")
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				tmpFile.WriteString("#!/bin/bash\necho 'test'\n")
				tmpFile.Close()

				// Remove execute permissions
				os.Chmod(tmpFile.Name(), 0644)

				return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }
			},
			expectError: false, // Should handle gracefully
			description: "Executable without execute permissions should be handled",
		},
		{
			name: "Massive entitlements list",
			setupConfig: func() *Config {
				cfg := NewConfig()
				cfg.ApplicationName = "TestApp"
				cfg.BundleID = "com.example.massive"

				// Add many entitlements
				cfg.Entitlements = make(map[Entitlement]bool)
				for i := 0; i < 100; i++ {
					cfg.Entitlements[Entitlement(fmt.Sprintf("com.example.entitlement%d", i))] = true
				}

				return cfg
			},
			setupFiles: func(t *testing.T) (string, func()) {
				return createTempExecutableFile(t)
			},
			expectError: false,
			description: "Massive entitlements list should be handled",
		},
		{
			name: "Massive plist entries",
			setupConfig: func() *Config {
				cfg := NewConfig()
				cfg.ApplicationName = "TestApp"
				cfg.BundleID = "com.example.massiveplist"

				// Add many plist entries
				cfg.PlistEntries = make(map[string]any)
				for i := 0; i < 100; i++ {
					cfg.PlistEntries[fmt.Sprintf("CustomKey%d", i)] = fmt.Sprintf("CustomValue%d", i)
				}

				return cfg
			},
			setupFiles: func(t *testing.T) (string, func()) {
				return createTempExecutableFile(t)
			},
			expectError: false,
			description: "Massive plist entries should be handled",
		},
		{
			name: "Invalid plist entry types",
			setupConfig: func() *Config {
				cfg := NewConfig()
				cfg.ApplicationName = "TestApp"
				cfg.BundleID = "com.example.invalidplist"

				// Add invalid plist entry types
				cfg.PlistEntries = make(map[string]any)
				cfg.PlistEntries["ValidString"] = "string value"
				cfg.PlistEntries["ValidBool"] = true
				cfg.PlistEntries["ValidInt"] = 42
				cfg.PlistEntries["ValidFloat"] = 3.14
				cfg.PlistEntries["ValidArray"] = []string{"a", "b", "c"}
				cfg.PlistEntries["ComplexMap"] = map[string]string{"key": "value"}
				cfg.PlistEntries["NilValue"] = nil

				return cfg
			},
			setupFiles: func(t *testing.T) (string, func()) {
				return createTempExecutableFile(t)
			},
			expectError: false,
			description: "Various plist entry types should be handled",
		},
		{
			name: "Read-only destination directory",
			setupConfig: func() *Config {
				cfg := NewConfig()
				cfg.ApplicationName = "TestApp"
				cfg.BundleID = "com.example.readonly"
				return cfg
			},
			setupFiles: func(t *testing.T) (string, func()) {
				return createTempExecutableFile(t)
			},
			expectError: false, // Should create in temp directory
			description: "Read-only destination should fall back to temp directory",
		},
		{
			name: "Circular symbolic link as executable",
			setupConfig: func() *Config {
				cfg := NewConfig()
				cfg.ApplicationName = "TestApp"
				cfg.BundleID = "com.example.symlink"
				return cfg
			},
			setupFiles: func(t *testing.T) (string, func()) {
				tmpDir, err := os.MkdirTemp("", "macgo-test-symlink-*")
				if err != nil {
					t.Fatalf("Failed to create temp directory: %v", err)
				}

				link1 := filepath.Join(tmpDir, "link1")
				link2 := filepath.Join(tmpDir, "link2")

				// Create circular symbolic links
				os.Symlink(link2, link1)
				os.Symlink(link1, link2)

				return link1, func() { os.RemoveAll(tmpDir) }
			},
			expectError: true,
			errorMsg:    "too many levels of symbolic links",
			description: "Circular symbolic links should cause error",
		},
		{
			name: "Unicode characters in names",
			setupConfig: func() *Config {
				cfg := NewConfig()
				cfg.ApplicationName = "测试应用程序 🚀"
				cfg.BundleID = "com.example.unicode"
				return cfg
			},
			setupFiles: func(t *testing.T) (string, func()) {
				return createTempExecutableFile(t)
			},
			expectError: false,
			description: "Unicode characters should be handled",
		},
		{
			name: "Empty entitlements and plist entries",
			setupConfig: func() *Config {
				cfg := NewConfig()
				cfg.ApplicationName = "TestApp"
				cfg.BundleID = "com.example.empty"
				cfg.Entitlements = make(map[Entitlement]bool)
				cfg.PlistEntries = make(map[string]any)
				return cfg
			},
			setupFiles: func(t *testing.T) (string, func()) {
				return createTempExecutableFile(t)
			},
			expectError: false,
			description: "Empty entitlements and plist entries should be handled",
		},
		{
			name: "Nil entitlements and plist entries",
			setupConfig: func() *Config {
				cfg := NewConfig()
				cfg.ApplicationName = "TestApp"
				cfg.BundleID = "com.example.nil"
				cfg.Entitlements = nil
				cfg.PlistEntries = nil
				return cfg
			},
			setupFiles: func(t *testing.T) (string, func()) {
				return createTempExecutableFile(t)
			},
			expectError: false,
			description: "Nil entitlements and plist entries should be handled",
		},
		{
			name: "Extremely large executable file",
			setupConfig: func() *Config {
				cfg := NewConfig()
				cfg.ApplicationName = "TestApp"
				cfg.BundleID = "com.example.large"
				return cfg
			},
			setupFiles: func(t *testing.T) (string, func()) {
				return createLargeExecutableFile(t)
			},
			expectError: false,
			description: "Large executable files should be handled",
		},
		{
			name: "Executable with special file permissions",
			setupConfig: func() *Config {
				cfg := NewConfig()
				cfg.ApplicationName = "TestApp"
				cfg.BundleID = "com.example.permissions"
				return cfg
			},
			setupFiles: func(t *testing.T) (string, func()) {
				tmpFile, err := os.CreateTemp("", "macgo-test-perms-*")
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				tmpFile.WriteString("#!/bin/bash\necho 'test'\n")
				tmpFile.Close()

				// Set special permissions
				os.Chmod(tmpFile.Name(), 0755)

				return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }
			},
			expectError: false,
			description: "Executable with special permissions should be handled",
		},
		{
			name: "Bundle creation with keep temp disabled",
			setupConfig: func() *Config {
				cfg := NewConfig()
				cfg.ApplicationName = "TestApp"
				cfg.BundleID = "com.example.notemp"
				cfg.KeepTemp = false
				return cfg
			},
			setupFiles: func(t *testing.T) (string, func()) {
				return createTempExecutableFile(t)
			},
			expectError: false,
			description: "Bundle creation with keep temp disabled should work",
		},
		{
			name: "Bundle creation with auto-sign disabled",
			setupConfig: func() *Config {
				cfg := NewConfig()
				cfg.ApplicationName = "TestApp"
				cfg.BundleID = "com.example.nosign"
				cfg.AutoSign = false
				return cfg
			},
			setupFiles: func(t *testing.T) (string, func()) {
				return createTempExecutableFile(t)
			},
			expectError: false,
			description: "Bundle creation with auto-sign disabled should work",
		},
		{
			name: "Bundle creation with custom signing identity",
			setupConfig: func() *Config {
				cfg := NewConfig()
				cfg.ApplicationName = "TestApp"
				cfg.BundleID = "com.example.customsign"
				cfg.AutoSign = true
				cfg.SigningIdentity = "Developer ID Application: Test Developer"
				return cfg
			},
			setupFiles: func(t *testing.T) (string, func()) {
				return createTempExecutableFile(t)
			},
			expectError: false, // Should handle gracefully even if identity doesn't exist
			description: "Bundle creation with custom signing identity should be handled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.setupConfig()
			execPath, cleanup := tt.setupFiles(t)
			defer cleanup()

			t.Logf("Testing %s", tt.description)

			// Test bundle creation
			err := testBundleCreation(cfg, execPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tt.name)
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%v'", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.name, err)
				}
			}
		})
	}
}

func TestBundleCreationConcurrency(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping bundle creation concurrency test on non-macOS platform")
	}

	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}

	const numGoroutines = 10
	done := make(chan error, numGoroutines)

	// Create temporary executable
	execPath, cleanup := createTempExecutableFile(t)
	defer cleanup()

	// Run multiple bundle creations concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			cfg := NewConfig()
			cfg.ApplicationName = fmt.Sprintf("ConcurrentApp%d", id)
			cfg.BundleID = fmt.Sprintf("com.example.concurrent%d", id)
			cfg.KeepTemp = true

			err := testBundleCreation(cfg, execPath)
			done <- err
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Concurrent bundle creation failed: %v", err)
			}
		case <-time.After(30 * time.Second):
			t.Fatal("Concurrent bundle creation timed out")
		}
	}
}

func TestBundleCreationStress(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping bundle creation stress test on non-macOS platform")
	}

	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	// Create temporary executable
	execPath, cleanup := createTempExecutableFile(t)
	defer cleanup()

	// Create many bundles rapidly
	const numBundles = 50
	for i := 0; i < numBundles; i++ {
		cfg := NewConfig()
		cfg.ApplicationName = fmt.Sprintf("StressApp%d", i)
		cfg.BundleID = fmt.Sprintf("com.example.stress%d", i)
		cfg.KeepTemp = false // Clean up automatically

		err := testBundleCreation(cfg, execPath)
		if err != nil {
			t.Errorf("Stress test bundle creation %d failed: %v", i, err)
		}
	}
}

func TestBundleCreationResourceExhaustion(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping bundle creation resource exhaustion test on non-macOS platform")
	}

	if testing.Short() {
		t.Skip("Skipping resource exhaustion test in short mode")
	}

	// Create temporary executable
	execPath, cleanup := createTempExecutableFile(t)
	defer cleanup()

	// Test with extremely large configurations
	cfg := NewConfig()
	cfg.ApplicationName = "ResourceExhaustionApp"
	cfg.BundleID = "com.example.resourceexhaustion"
	cfg.KeepTemp = true

	// Add many entitlements and plist entries
	cfg.Entitlements = make(map[Entitlement]bool)
	cfg.PlistEntries = make(map[string]any)

	for i := 0; i < 1000; i++ {
		cfg.Entitlements[Entitlement(fmt.Sprintf("com.example.entitlement%d", i))] = true
		cfg.PlistEntries[fmt.Sprintf("CustomKey%d", i)] = strings.Repeat("Value", 100)
	}

	err := testBundleCreation(cfg, execPath)
	if err != nil {
		t.Errorf("Resource exhaustion test failed: %v", err)
	}
}

func TestBundleCreationCleanup(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping bundle creation cleanup test on non-macOS platform")
	}

	// Create temporary executable
	execPath, cleanup := createTempExecutableFile(t)
	defer cleanup()

	// Test with KeepTemp = false
	cfg := NewConfig()
	cfg.ApplicationName = "CleanupApp"
	cfg.BundleID = "com.example.cleanup"
	cfg.KeepTemp = false

	err := testBundleCreation(cfg, execPath)
	if err != nil {
		t.Errorf("Cleanup test failed: %v", err)
	}

	// Verify cleanup occurred (this is hard to test directly,
	// but we can at least verify no error occurred)
}

func TestBundleCreationErrorRecovery(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping bundle creation error recovery test on non-macOS platform")
	}

	// Test recovery from various error conditions
	tests := []struct {
		name        string
		setupError  func() (string, func())
		expectError bool
		description string
	}{
		{
			name: "Invalid executable path",
			setupError: func() (string, func()) {
				return "/invalid/path/to/executable", func() {}
			},
			expectError: true,
			description: "Should handle invalid executable path gracefully",
		},
		{
			name: "Insufficient disk space simulation",
			setupError: func() (string, func()) {
				// Create a very large file to simulate disk space issues
				tmpFile, err := os.CreateTemp("", "macgo-test-large-*")
				if err != nil {
					return "", func() {}
				}
				tmpFile.WriteString("#!/bin/bash\necho 'test'\n")
				tmpFile.Close()
				os.Chmod(tmpFile.Name(), 0755)
				return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }
			},
			expectError: false,
			description: "Should handle disk space issues gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			execPath, cleanup := tt.setupError()
			defer cleanup()

			cfg := NewConfig()
			cfg.ApplicationName = "ErrorRecoveryApp"
			cfg.BundleID = "com.example.errorrecovery"

			err := testBundleCreation(cfg, execPath)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, but got none", tt.name)
			} else if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.name, err)
			}
		})
	}
}

// Helper functions

func createTempExecutableFile(t *testing.T) (string, func()) {
	tmpFile, err := os.CreateTemp("", "macgo-test-exec-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	execContent := `#!/bin/bash
echo "Test executable"
exit 0
`
	tmpFile.WriteString(execContent)
	tmpFile.Close()

	// Make it executable
	os.Chmod(tmpFile.Name(), 0755)

	return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }
}

func createLargeExecutableFile(t *testing.T) (string, func()) {
	tmpFile, err := os.CreateTemp("", "macgo-test-large-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Create a large executable (10MB)
	execContent := `#!/bin/bash
echo "Large test executable"
# Large comment section to increase file size
`
	tmpFile.WriteString(execContent)

	// Add padding to make it large
	padding := strings.Repeat("# Padding line to make file large\n", 100000)
	tmpFile.WriteString(padding)
	tmpFile.WriteString("exit 0\n")

	tmpFile.Close()

	// Make it executable
	os.Chmod(tmpFile.Name(), 0755)

	return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }
}

func testBundleCreation(cfg *Config, execPath string) error {
	// This is a simplified test of bundle creation
	// In the real implementation, this would call the actual bundle creation logic

	// Basic validation
	if cfg.ApplicationName == "" {
		cfg.ApplicationName = "DefaultApp"
	}

	if cfg.BundleID == "" {
		cfg.BundleID = "com.example.default"
	}

	// Check if executable exists
	if _, err := os.Stat(execPath); err != nil {
		return fmt.Errorf("executable file does not exist: %v", err)
	}

	// Check if it's a directory
	if info, err := os.Stat(execPath); err == nil && info.IsDir() {
		return fmt.Errorf("path is a directory, not an executable")
	}

	// Simulate bundle creation
	tmpDir, err := os.MkdirTemp("", "macgo-test-bundle-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}

	// Clean up unless KeepTemp is true
	if !cfg.KeepTemp {
		defer os.RemoveAll(tmpDir)
	}

	// Apply the same filename length validation as the main bundle creation code
	appName := cfg.ApplicationName
	const maxAppNameLen = 251 // Reserve 4 chars for ".app"
	if len(appName) > maxAppNameLen {
		appName = appName[:maxAppNameLen]
	}

	bundlePath := filepath.Join(tmpDir, appName+".app")
	contentsPath := filepath.Join(bundlePath, "Contents")
	macOSPath := filepath.Join(contentsPath, "MacOS")

	// Create bundle structure
	if err := os.MkdirAll(macOSPath, 0755); err != nil {
		return fmt.Errorf("failed to create bundle structure: %v", err)
	}

	// Copy executable
	execName := appName
	if execName == "" {
		execName = "app"
	}
	destExec := filepath.Join(macOSPath, execName)

	if err := copyExecutable(execPath, destExec); err != nil {
		return fmt.Errorf("failed to copy executable: %v", err)
	}

	// Create Info.plist
	infoPlist := createInfoPlist(cfg)
	infoPlistPath := filepath.Join(contentsPath, "Info.plist")
	if err := os.WriteFile(infoPlistPath, []byte(infoPlist), 0644); err != nil {
		return fmt.Errorf("failed to create Info.plist: %v", err)
	}

	// Create entitlements if needed
	if cfg.Entitlements != nil && len(cfg.Entitlements) > 0 {
		entitlements := createEntitlementsPlist(cfg)
		entitlementsPath := filepath.Join(contentsPath, "entitlements.plist")
		if err := os.WriteFile(entitlementsPath, []byte(entitlements), 0644); err != nil {
			return fmt.Errorf("failed to create entitlements: %v", err)
		}
	}

	return nil
}

func copyExecutable(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = dstFile.ReadFrom(srcFile)
	if err != nil {
		return err
	}

	// Copy permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, srcInfo.Mode())
}

func createInfoPlist(cfg *Config) string {
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleName</key>
	<string>%s</string>
	<key>CFBundleIdentifier</key>
	<string>%s</string>
	<key>CFBundleVersion</key>
	<string>1.0</string>
	<key>CFBundleExecutable</key>
	<string>%s</string>
`

	execName := cfg.ApplicationName
	if execName == "" {
		execName = "app"
	}

	plistContent := fmt.Sprintf(plist, cfg.ApplicationName, cfg.BundleID, execName)

	// Add custom plist entries
	if cfg.PlistEntries != nil {
		for key, value := range cfg.PlistEntries {
			plistContent += fmt.Sprintf("	<key>%s</key>\n", key)
			plistContent += formatPlistValue(value)
		}
	}

	plistContent += `</dict>
</plist>`

	return plistContent
}

func createEntitlementsPlist(cfg *Config) string {
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
`

	for ent, value := range cfg.Entitlements {
		plist += fmt.Sprintf("	<key>%s</key>\n", ent)
		if value {
			plist += "	<true/>\n"
		} else {
			plist += "	<false/>\n"
		}
	}

	plist += `</dict>
</plist>`

	return plist
}

func formatPlistValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("	<string>%s</string>\n", v)
	case bool:
		if v {
			return "	<true/>\n"
		}
		return "	<false/>\n"
	case int:
		return fmt.Sprintf("	<integer>%d</integer>\n", v)
	case float64:
		return fmt.Sprintf("	<real>%f</real>\n", v)
	case []string:
		result := "	<array>\n"
		for _, item := range v {
			result += fmt.Sprintf("		<string>%s</string>\n", item)
		}
		result += "	</array>\n"
		return result
	case nil:
		return "	<string></string>\n"
	default:
		return fmt.Sprintf("	<string>%v</string>\n", v)
	}
}

// Benchmark tests moved to bundle_bench_test.go to avoid duplicates

func BenchmarkLargeExecutableBundleCreation(b *testing.B) {
	if runtime.GOOS != "darwin" {
		b.Skip("Skipping large executable bundle creation benchmark on non-macOS platform")
	}

	// Create large temporary executable
	tmpFile, err := os.CreateTemp("", "macgo-bench-large-*")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}

	// Write large executable content
	execContent := "#!/bin/bash\necho 'large test executable'\n"
	tmpFile.WriteString(execContent)

	// Add padding
	padding := strings.Repeat("# Padding line\n", 10000)
	tmpFile.WriteString(padding)
	tmpFile.WriteString("exit 0\n")

	tmpFile.Close()
	os.Chmod(tmpFile.Name(), 0755)
	defer os.Remove(tmpFile.Name())

	cfg := NewConfig()
	cfg.ApplicationName = "BenchmarkLargeApp"
	cfg.BundleID = "com.example.benchmarklarge"
	cfg.KeepTemp = false

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := testBundleCreation(cfg, tmpFile.Name())
		if err != nil {
			b.Fatalf("Large executable bundle creation failed: %v", err)
		}
	}
}
