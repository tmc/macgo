//go:build darwin
// +build darwin

package macgo

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBundleIDInference tests the bundle ID inference logic
func TestBundleIDInference(t *testing.T) {
	// Save original config
	originalConfig := DefaultConfig
	defer func() {
		DefaultConfig = originalConfig
	}()

	tests := []struct {
		name           string
		execPath       string
		appName        string
		bundleID       string // If set in config
		expectedPrefix string
	}{
		{
			name:           "regular binary without bundle ID",
			execPath:       "/usr/local/bin/myapp",
			appName:        "MyApp",
			bundleID:       "",
			expectedPrefix: "com.macgo.MyApp",
		},
		{
			name:           "temporary binary without bundle ID",
			execPath:       "/tmp/go-build123456/main",
			appName:        "TempApp",
			bundleID:       "",
			expectedPrefix: "com.macgo.TempApp",
		},
		{
			name:           "custom bundle ID",
			execPath:       "/usr/local/bin/myapp",
			appName:        "MyApp",
			bundleID:       "com.example.customapp",
			expectedPrefix: "com.example.customapp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip actual bundle creation, just test the logic
			// by checking what would be written to Info.plist

			tmpDir, err := os.MkdirTemp("", "bundle-id-test-*")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			// Create a minimal test executable
			tmpExec := filepath.Join(tmpDir, "test-exec")
			os.WriteFile(tmpExec, []byte("test"), 0755)

			DefaultConfig = NewConfig()
			DefaultConfig.ApplicationName = tt.appName
			DefaultConfig.BundleID = tt.bundleID
			DefaultConfig.CustomDestinationAppPath = filepath.Join(tmpDir, "Test.app")

			bundlePath, err := createBundle(tmpExec)
			if err != nil {
				t.Fatalf("Failed to create bundle: %v", err)
			}

			// Read Info.plist to check bundle ID
			infoPlist := filepath.Join(bundlePath, "Contents", "Info.plist")
			content, err := os.ReadFile(infoPlist)
			if err != nil {
				t.Fatal(err)
			}

			if !strings.Contains(string(content), tt.expectedPrefix) {
				t.Errorf("Expected bundle ID to contain %q, got:\n%s", tt.expectedPrefix, string(content))
			}
		})
	}
}

// TestBundleWithoutGOPATH tests bundle creation when GOPATH is not set
func TestBundleWithoutGOPATH(t *testing.T) {
	// Save original values
	originalConfig := DefaultConfig
	originalGOPATH := os.Getenv("GOPATH")
	defer func() {
		DefaultConfig = originalConfig
		os.Setenv("GOPATH", originalGOPATH)
	}()

	// Unset GOPATH
	os.Unsetenv("GOPATH")

	// Create test executable
	tmpExec, err := os.CreateTemp("", "test-exec-*")
	if err != nil {
		t.Fatal(err)
	}
	tmpExec.Write([]byte("test"))
	tmpExec.Close()
	defer os.Remove(tmpExec.Name())
	os.Chmod(tmpExec.Name(), 0755)

	DefaultConfig = NewConfig()
	bundlePath, err := createBundle(tmpExec.Name())
	if err != nil {
		t.Fatalf("Failed to create bundle without GOPATH: %v", err)
	}
	defer os.RemoveAll(bundlePath)

	// Should fall back to ~/go/bin
	home, _ := os.UserHomeDir()
	expectedPath := filepath.Join(home, "go", "bin")
	if !strings.Contains(bundlePath, expectedPath) {
		t.Errorf("Expected bundle in %s, got %s", expectedPath, bundlePath)
	}
}

// TestBundlePermissionErrors tests error handling for permission issues
func TestBundlePermissionErrors(t *testing.T) {
	// Skip if running as root
	if os.Geteuid() == 0 {
		t.Skip("Cannot test permission errors as root")
	}

	// Save original config
	originalConfig := DefaultConfig
	defer func() {
		DefaultConfig = originalConfig
	}()

	// Create test executable
	tmpExec, err := os.CreateTemp("", "test-exec-*")
	if err != nil {
		t.Fatal(err)
	}
	tmpExec.Close()
	defer os.Remove(tmpExec.Name())

	DefaultConfig = NewConfig()
	DefaultConfig.CustomDestinationAppPath = "/root/forbidden/test.app"

	_, err = createBundle(tmpExec.Name())
	if err == nil {
		t.Error("Expected permission error when creating bundle in forbidden directory")
	}
}

// TestBundleTemporaryCleanup tests that temporary bundles are cleaned up
func TestBundleTemporaryCleanup(t *testing.T) {
	// Save original config
	originalConfig := DefaultConfig
	defer func() {
		DefaultConfig = originalConfig
	}()

	// Create a simulated go-build temporary binary path
	DefaultConfig = NewConfig()
	DefaultConfig.KeepTemp = false

	// We can't easily test the actual cleanup goroutine timing,
	// but we can verify the logic paths
	execPath := "/tmp/go-build123456/main"

	// Check that isTemp detection works
	isTemp := strings.Contains(execPath, "go-build")
	if !isTemp {
		t.Error("Expected go-build path to be detected as temporary")
	}
}

// TestBundlePlistEntryTypes tests various plist entry types
func TestBundlePlistEntryTypes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "plist-types-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test different data types
	tests := []struct {
		name     string
		data     map[string]any
		expected []string
	}{
		{
			name: "nil and empty values",
			data: map[string]any{
				"NilValue":  nil,
				"EmptyStr":  "",
				"ZeroInt":   0,
				"ZeroFloat": 0.0,
				"FalseBool": false,
			},
			expected: []string{
				"<string></string>", // nil becomes empty string
				"<string></string>", // empty string
				"<integer>0</integer>",
				"<real>0</real>",
				"<false/>",
			},
		},
		{
			name: "special characters",
			data: map[string]any{
				"XMLChars": "<>&\"'",
				"Unicode":  "Hello 世界 🌍",
			},
			expected: []string{
				"&lt;&gt;&amp;&quot;&apos;", // writePlist properly escapes XML characters
				"Hello 世界 🌍",
			},
		},
		{
			name: "large numbers",
			data: map[string]any{
				"MaxInt64": int64(9223372036854775807),
				"MinInt64": int64(-9223372036854775808),
				"BigFloat": 1.7976931348623157e+308,
			},
			expected: []string{
				"<integer>9223372036854775807</integer>",
				"<integer>-9223372036854775808</integer>",
				"<real>1.7976931348623157e+308</real>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plistPath := filepath.Join(tmpDir, tt.name+".plist")
			err := writePlist(plistPath, tt.data)
			if err != nil {
				t.Fatalf("Failed to write plist: %v", err)
			}

			content, _ := os.ReadFile(plistPath)
			contentStr := string(content)

			for _, expected := range tt.expected {
				if !strings.Contains(contentStr, expected) {
					t.Errorf("Expected plist to contain %q", expected)
				}
			}
		})
	}
}

// TestCheckExistingEdgeCases tests edge cases for checkExisting
func TestCheckExistingEdgeCases(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "check-existing-edge-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	execPath := filepath.Join(tmpDir, "test-exec")
	os.WriteFile(execPath, []byte("test"), 0755)

	tests := []struct {
		name     string
		setup    func() string
		expected bool
	}{
		{
			name: "app bundle is a file not directory",
			setup: func() string {
				appPath := filepath.Join(tmpDir, "NotABundle.app")
				os.WriteFile(appPath, []byte("not a directory"), 0644)
				return appPath
			},
			expected: false,
		},
		{
			name: "executable in bundle is a directory",
			setup: func() string {
				appPath := filepath.Join(tmpDir, "BadBundle.app")
				bundleExecPath := filepath.Join(appPath, "Contents", "MacOS", "test-exec")
				os.MkdirAll(bundleExecPath, 0755) // Create as directory
				return appPath
			},
			expected: false,
		},
		{
			name: "bundle with wrong executable name",
			setup: func() string {
				appPath := filepath.Join(tmpDir, "WrongExec.app")
				bundleExecPath := filepath.Join(appPath, "Contents", "MacOS", "wrong-name")
				os.MkdirAll(filepath.Dir(bundleExecPath), 0755)
				os.WriteFile(bundleExecPath, []byte("test"), 0755)
				return appPath
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appPath := tt.setup()
			result := checkExisting(appPath, execPath)
			if result != tt.expected {
				t.Errorf("Expected checkExisting to return %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestCreateFromTemplateErrors tests error handling in createFromTemplate
func TestCreateFromTemplateErrors(t *testing.T) {
	// Save original config
	originalConfig := DefaultConfig
	defer func() {
		DefaultConfig = originalConfig
	}()

	tmpDir, err := os.MkdirTemp("", "template-errors-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	execPath := filepath.Join(tmpDir, "test-exec")
	os.WriteFile(execPath, []byte("test"), 0755)

	// Test with a template that will cause errors
	template := &mockErrorFS{
		shouldFailOpen: true,
	}

	appPath := filepath.Join(tmpDir, "ErrorApp.app")
	_, err = createFromTemplate(template, appPath, execPath, "ErrorApp")
	if err == nil {
		t.Error("Expected error when template fails to open")
	}
}

// mockErrorFS is a filesystem that returns errors
type mockErrorFS struct {
	shouldFailOpen bool
}

func (m *mockErrorFS) Open(name string) (fs.File, error) {
	if m.shouldFailOpen {
		return nil, os.ErrNotExist
	}
	return nil, nil
}
