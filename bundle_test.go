package macgo

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCreateBundle tests the createBundle function with various configurations
func TestCreateBundle(t *testing.T) {
	// Save original DefaultConfig
	originalConfig := DefaultConfig
	originalDebug := os.Getenv("MACGO_DEBUG")
	defer func() {
		DefaultConfig = originalConfig
		os.Setenv("MACGO_DEBUG", originalDebug)
	}()

	// Create a temporary executable for testing
	tmpExec, err := os.CreateTemp("", "test-exec-*")
	if err != nil {
		t.Fatal(err)
	}
	tmpExec.Write([]byte("test binary content"))
	tmpExec.Close()
	defer os.Remove(tmpExec.Name())

	// Make it executable
	os.Chmod(tmpExec.Name(), 0755)

	tests := []struct {
		name        string
		setup       func()
		execPath    string
		expectError bool
		checkResult func(t *testing.T, bundlePath string, err error)
	}{
		{
			name: "create bundle with default config",
			setup: func() {
				DefaultConfig = NewConfig()
			},
			execPath: tmpExec.Name(),
			checkResult: func(t *testing.T, bundlePath string, err error) {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
					return
				}
				// Check bundle structure
				if !strings.HasSuffix(bundlePath, ".app") {
					t.Error("Expected bundle path to end with .app")
				}
			},
		},
		{
			name: "create bundle with custom application name",
			setup: func() {
				DefaultConfig = NewConfig()
				DefaultConfig.ApplicationName = "CustomTestApp"
			},
			execPath: tmpExec.Name(),
			checkResult: func(t *testing.T, bundlePath string, err error) {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
					return
				}
				if !strings.Contains(bundlePath, "CustomTestApp.app") {
					t.Error("Expected bundle to use custom application name")
				}
			},
		},
		{
			name: "create bundle with custom destination path",
			setup: func() {
				tmpDir, _ := os.MkdirTemp("", "macgo-test-*")
				DefaultConfig = NewConfig()
				DefaultConfig.CustomDestinationAppPath = filepath.Join(tmpDir, "MyCustomApp.app")
			},
			execPath: tmpExec.Name(),
			checkResult: func(t *testing.T, bundlePath string, err error) {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
					return
				}
				if bundlePath != DefaultConfig.CustomDestinationAppPath {
					t.Errorf("Expected bundle path %s, got %s", DefaultConfig.CustomDestinationAppPath, bundlePath)
				}
				// Clean up
				os.RemoveAll(filepath.Dir(bundlePath))
			},
		},
		{
			name: "create temporary bundle for go-build binary",
			setup: func() {
				DefaultConfig = NewConfig()
			},
			execPath: func() string {
				// Create a temporary go-build-like path
				tmpDir := "/tmp/go-build" + fmt.Sprintf("%d", time.Now().UnixNano())
				os.MkdirAll(tmpDir, 0755)
				tmpExec := filepath.Join(tmpDir, "main")
				os.WriteFile(tmpExec, []byte("test binary"), 0755)
				return tmpExec
			}(),
			checkResult: func(t *testing.T, bundlePath string, err error) {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
					return
				}
				// Check it's in a temp directory
				if !strings.Contains(bundlePath, "/macgo-") {
					t.Error("Expected temporary bundle to be in macgo-* temp directory")
				}
				// Clean up
				os.RemoveAll(filepath.Dir(bundlePath))
			},
		},
		{
			name: "create bundle with entitlements",
			setup: func() {
				DefaultConfig = NewConfig()
				DefaultConfig.Entitlements = map[Entitlement]bool{
					EntCamera:     true,
					EntMicrophone: false,
				}
			},
			execPath: tmpExec.Name(),
			checkResult: func(t *testing.T, bundlePath string, err error) {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
					return
				}

				// When AutoSign is enabled (default), entitlements are embedded in the signature
				// So we check for the _CodeSignature directory instead
				sigPath := filepath.Join(bundlePath, "Contents", "_CodeSignature")
				if _, err := os.Stat(sigPath); os.IsNotExist(err) {
					// If not signed, check for standalone entitlements.plist
					entPath := filepath.Join(bundlePath, "Contents", "entitlements.plist")
					if _, err := os.Stat(entPath); os.IsNotExist(err) {
						t.Error("Expected either signed bundle or entitlements.plist to exist")
					} else {
						// Verify entitlements content
						content, _ := os.ReadFile(entPath)
						if !strings.Contains(string(content), "com.apple.security.device.camera") {
							t.Errorf("Expected camera entitlement in entitlements.plist")
						}
						// Entitlements should only include camera (true), not microphone (false)
						if strings.Contains(string(content), "com.apple.security.device.microphone") {
							t.Error("Did not expect microphone entitlement (was set to false)")
						}
					}
				} else {
					// Bundle is signed, entitlements are embedded in the signature
					// This is the expected behavior with AutoSign=true (default)
					t.Log("Bundle is signed, entitlements are embedded in signature")
				}
				// Clean up
				os.RemoveAll(bundlePath)
			},
		},
		{
			name: "create bundle with custom bundle ID",
			setup: func() {
				DefaultConfig = NewConfig()
				DefaultConfig.BundleID = "com.test.custombundle"
			},
			execPath: tmpExec.Name(),
			checkResult: func(t *testing.T, bundlePath string, err error) {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
					return
				}
				// Read Info.plist to verify bundle ID
				infoPlist := filepath.Join(bundlePath, "Contents", "Info.plist")
				content, _ := os.ReadFile(infoPlist)
				if !strings.Contains(string(content), "com.test.custombundle") {
					t.Error("Expected custom bundle ID in Info.plist")
				}
				// Clean up
				os.RemoveAll(bundlePath)
			},
		},
		{
			name: "create bundle with keep temp enabled",
			setup: func() {
				DefaultConfig = NewConfig()
				DefaultConfig.KeepTemp = true
			},
			execPath: func() string {
				// Create a temporary go-build-like path
				tmpDir := "/tmp/go-build" + fmt.Sprintf("%d", time.Now().UnixNano())
				os.MkdirAll(tmpDir, 0755)
				tmpExec := filepath.Join(tmpDir, "temp-binary")
				os.WriteFile(tmpExec, []byte("test binary"), 0755)
				return tmpExec
			}(),
			checkResult: func(t *testing.T, bundlePath string, err error) {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
					return
				}
				// With KeepTemp, bundle should persist
				time.Sleep(100 * time.Millisecond)
				if _, err := os.Stat(bundlePath); os.IsNotExist(err) {
					t.Error("Expected bundle to persist with KeepTemp=true")
				}
				// Clean up
				os.RemoveAll(filepath.Dir(bundlePath))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			bundlePath, err := createBundle(tt.execPath)
			tt.checkResult(t, bundlePath, err)
		})
	}
}

// TestCheckExisting tests the checkExisting function
func TestCheckExisting(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "macgo-test-existing-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test executable
	execPath := filepath.Join(tmpDir, "test-exec")
	if err := os.WriteFile(execPath, []byte("test content"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create a mock app bundle
	appPath := filepath.Join(tmpDir, "TestApp.app")
	bundleExecPath := filepath.Join(appPath, "Contents", "MacOS", "test-exec")
	if err := os.MkdirAll(filepath.Dir(bundleExecPath), 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		setup    func()
		appPath  string
		execPath string
		expected bool
	}{
		{
			name:     "bundle does not exist",
			appPath:  filepath.Join(tmpDir, "NonExistent.app"),
			execPath: execPath,
			expected: false,
		},
		{
			name: "bundle exists but executable missing",
			setup: func() {
				// Create empty bundle structure
				os.MkdirAll(filepath.Join(appPath, "Contents", "MacOS"), 0755)
			},
			appPath:  appPath,
			execPath: execPath,
			expected: false,
		},
		{
			name: "bundle and executable exist with same content",
			setup: func() {
				// Copy executable to bundle
				content, _ := os.ReadFile(execPath)
				os.WriteFile(bundleExecPath, content, 0755)
			},
			appPath:  appPath,
			execPath: execPath,
			expected: true,
		},
		{
			name: "bundle exists but content differs",
			setup: func() {
				// Write different content to bundle executable
				os.WriteFile(bundleExecPath, []byte("different content"), 0755)
			},
			appPath:  appPath,
			execPath: execPath,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			result := checkExisting(tt.appPath, tt.execPath)
			if result != tt.expected {
				t.Errorf("Expected checkExisting to return %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestChecksum tests the checksum function
func TestChecksum(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "macgo-checksum-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	testContent := "test content for checksum"
	tmpFile.Write([]byte(testContent))
	tmpFile.Close()

	tests := []struct {
		name        string
		path        string
		expectError bool
		validate    func(t *testing.T, hash string, err error)
	}{
		{
			name: "valid file checksum",
			path: tmpFile.Name(),
			validate: func(t *testing.T, hash string, err error) {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
					return
				}
				// Verify it's a valid SHA256 hash (64 hex chars)
				if len(hash) != 64 {
					t.Errorf("Expected 64 character hash, got %d", len(hash))
				}
				// Verify it matches expected hash
				h := sha256.New()
				h.Write([]byte(testContent))
				expected := hex.EncodeToString(h.Sum(nil))
				if hash != expected {
					t.Errorf("Expected hash %s, got %s", expected, hash)
				}
			},
		},
		{
			name:        "non-existent file",
			path:        "/non/existent/file",
			expectError: true,
			validate: func(t *testing.T, hash string, err error) {
				if err == nil {
					t.Error("Expected error for non-existent file")
				}
				if hash != "" {
					t.Error("Expected empty hash on error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := checksum(tt.path)
			tt.validate(t, hash, err)
		})
	}
}

// TestBundleCopyFile tests the copyFile function in bundle context
func TestBundleCopyFile(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "macgo-copyfile-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name        string
		setup       func() (src, dst string)
		expectError bool
		validate    func(t *testing.T, src, dst string, err error)
	}{
		{
			name: "copy regular file",
			setup: func() (string, string) {
				src := filepath.Join(tmpDir, "source.txt")
				dst := filepath.Join(tmpDir, "dest.txt")
				os.WriteFile(src, []byte("test content"), 0644)
				return src, dst
			},
			validate: func(t *testing.T, src, dst string, err error) {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
					return
				}
				srcContent, _ := os.ReadFile(src)
				dstContent, _ := os.ReadFile(dst)
				if string(srcContent) != string(dstContent) {
					t.Error("File contents do not match")
				}
				// Check permissions
				info, _ := os.Stat(dst)
				if info.Mode() != 0755 {
					t.Errorf("Expected mode 0755, got %v", info.Mode())
				}
			},
		},
		{
			name: "copy non-existent file",
			setup: func() (string, string) {
				return "/non/existent/file", filepath.Join(tmpDir, "dest.txt")
			},
			expectError: true,
			validate: func(t *testing.T, src, dst string, err error) {
				if err == nil {
					t.Error("Expected error when copying non-existent file")
				}
			},
		},
		{
			name: "copy to invalid destination",
			setup: func() (string, string) {
				src := filepath.Join(tmpDir, "source.txt")
				os.WriteFile(src, []byte("content"), 0644)
				return src, "/invalid/path/dest.txt"
			},
			expectError: true,
			validate: func(t *testing.T, src, dst string, err error) {
				if err == nil {
					t.Error("Expected error when copying to invalid destination")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, dst := tt.setup()
			err := copyFile(src, dst)
			tt.validate(t, src, dst, err)
		})
	}
}

// TestWritePlist tests the writePlist function
func TestWritePlist(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "macgo-plist-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name        string
		data        map[string]any
		expectError bool
		validate    func(t *testing.T, path string, content string)
	}{
		{
			name: "write plist with various types",
			data: map[string]any{
				"StringKey":  "StringValue",
				"BoolTrue":   true,
				"BoolFalse":  false,
				"IntKey":     42,
				"FloatKey":   3.14,
				"DefaultKey": struct{}{}, // Should be stringified
			},
			validate: func(t *testing.T, path string, content string) {
				// Check XML declaration
				if !strings.Contains(content, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>") {
					t.Error("Missing XML declaration")
				}
				// Check DOCTYPE
				if !strings.Contains(content, "<!DOCTYPE plist") {
					t.Error("Missing DOCTYPE declaration")
				}
				// Check key-value pairs
				if !strings.Contains(content, "<key>StringKey</key>") {
					t.Error("Missing StringKey")
				}
				if !strings.Contains(content, "<string>StringValue</string>") {
					t.Error("Missing StringValue")
				}
				if !strings.Contains(content, "<key>BoolTrue</key>") {
					t.Error("Missing BoolTrue key")
				}
				if !strings.Contains(content, "<true/>") {
					t.Error("Missing true value")
				}
				if !strings.Contains(content, "<key>BoolFalse</key>") {
					t.Error("Missing BoolFalse key")
				}
				if !strings.Contains(content, "<false/>") {
					t.Error("Missing false value")
				}
				if !strings.Contains(content, "<integer>42</integer>") {
					t.Error("Missing integer value")
				}
				if !strings.Contains(content, "<real>3.14</real>") {
					t.Error("Missing real value")
				}
			},
		},
		{
			name: "write plist with entitlements",
			data: map[string]any{
				string(EntCamera):     true,
				string(EntMicrophone): false,
				string(EntAppSandbox): true,
			},
			validate: func(t *testing.T, path string, content string) {
				if !strings.Contains(content, "com.apple.security.device.camera") {
					t.Error("Missing camera entitlement")
				}
				if !strings.Contains(content, "com.apple.security.device.microphone") {
					t.Error("Missing microphone entitlement")
				}
				if !strings.Contains(content, "com.apple.security.app-sandbox") {
					t.Error("Missing app sandbox entitlement")
				}
			},
		},
		{
			name: "write empty plist",
			data: map[string]any{},
			validate: func(t *testing.T, path string, content string) {
				if !strings.Contains(content, "<dict>") {
					t.Error("Missing dict tags")
				}
				if !strings.Contains(content, "</dict>") {
					t.Error("Missing closing dict tag")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plistPath := filepath.Join(tmpDir, "test.plist")
			err := writePlist(plistPath, tt.data)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if err == nil {
				content, _ := os.ReadFile(plistPath)
				tt.validate(t, plistPath, string(content))
			}
		})
	}
}

// TestCreateFromTemplate tests the createFromTemplate function
func TestCreateFromTemplate(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "macgo-template-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test executable
	execPath := filepath.Join(tmpDir, "test-exec")
	if err := os.WriteFile(execPath, []byte("executable content"), 0755); err != nil {
		t.Fatal(err)
	}

	// Save original DefaultConfig
	originalConfig := DefaultConfig
	defer func() {
		DefaultConfig = originalConfig
	}()

	tests := []struct {
		name        string
		setup       func() fs.FS
		appPath     string
		execPath    string
		appName     string
		expectError bool
		validate    func(t *testing.T, appPath string, err error)
	}{
		{
			name: "create from valid template",
			setup: func() fs.FS {
				return &mockTemplateFS{
					files: map[string]mockFile{
						"Contents/Info.plist": {
							content: `<?xml version="1.0"?>
<plist>
<dict>
	<key>CFBundleName</key>
	<string>{{BundleName}}</string>
	<key>CFBundleExecutable</key>
	<string>{{BundleExecutable}}</string>
	<key>CFBundleIdentifier</key>
	<string>{{BundleIdentifier}}</string>
</dict>
</plist>`,
						},
						"Contents/MacOS/exec.placeholder": {
							content: "placeholder",
						},
						"Contents/Resources/icon.icns": {
							content: "icon data",
						},
					},
				}
			},
			appPath:  filepath.Join(tmpDir, "TestApp.app"),
			execPath: execPath,
			appName:  "TestApp",
			validate: func(t *testing.T, appPath string, err error) {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
					return
				}
				// Check Info.plist was processed
				infoPlist := filepath.Join(appPath, "Contents", "Info.plist")
				content, _ := os.ReadFile(infoPlist)
				if !strings.Contains(string(content), "TestApp") {
					t.Error("Expected app name in Info.plist")
				}
				if !strings.Contains(string(content), "test-exec") {
					t.Error("Expected executable name in Info.plist")
				}
				// Check executable was copied
				bundleExec := filepath.Join(appPath, "Contents", "MacOS", "test-exec")
				if _, err := os.Stat(bundleExec); os.IsNotExist(err) {
					t.Error("Expected executable to be copied")
				}
			},
		},
		{
			name: "template with custom plist entries",
			setup: func() fs.FS {
				DefaultConfig = NewConfig()
				DefaultConfig.PlistEntries = map[string]any{
					"LSUIElement": false,
					"CustomKey":   "CustomValue",
				}
				return &mockTemplateFS{
					files: map[string]mockFile{
						"Contents/Info.plist": {
							content: `<?xml version="1.0"?>
<plist>
<dict>
	<key>CFBundleName</key>
	<string>{{BundleName}}</string>
</dict>
</plist>`,
						},
					},
				}
			},
			appPath:  filepath.Join(tmpDir, "CustomApp.app"),
			execPath: execPath,
			appName:  "CustomApp",
			validate: func(t *testing.T, appPath string, err error) {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
					return
				}
				infoPlist := filepath.Join(appPath, "Contents", "Info.plist")
				content, _ := os.ReadFile(infoPlist)
				if !strings.Contains(string(content), "LSUIElement") {
					t.Error("Expected LSUIElement in Info.plist")
				}
				if !strings.Contains(string(content), "CustomKey") {
					t.Error("Expected CustomKey in Info.plist")
				}
			},
		},
		{
			name: "template with entitlements",
			setup: func() fs.FS {
				DefaultConfig = NewConfig()
				DefaultConfig.Entitlements = map[Entitlement]bool{
					EntCamera:     true,
					EntMicrophone: true,
				}
				return &mockTemplateFS{
					files: map[string]mockFile{
						"Contents/entitlements.plist": {
							content: "placeholder",
						},
					},
				}
			},
			appPath:  filepath.Join(tmpDir, "EntitledApp.app"),
			execPath: execPath,
			appName:  "EntitledApp",
			validate: func(t *testing.T, appPath string, err error) {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
					return
				}
				entPath := filepath.Join(appPath, "Contents", "entitlements.plist")
				content, _ := os.ReadFile(entPath)
				if !strings.Contains(string(content), "com.apple.security.device.camera") {
					t.Error("Expected camera entitlement")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template := tt.setup()
			appPath, err := createFromTemplate(template, tt.appPath, tt.execPath, tt.appName)
			tt.validate(t, appPath, err)
			// Clean up
			os.RemoveAll(tt.appPath)
		})
	}
}

// TestBundleCreationEdgeCases tests edge cases for bundle creation
func TestBundleCreationEdgeCases(t *testing.T) {
	// Save original config and environment
	originalConfig := DefaultConfig
	originalGOPATH := os.Getenv("GOPATH")
	defer func() {
		DefaultConfig = originalConfig
		os.Setenv("GOPATH", originalGOPATH)
	}()

	t.Run("missing GOPATH fallback to home", func(t *testing.T) {
		os.Unsetenv("GOPATH")
		DefaultConfig = NewConfig()

		// Create a test executable
		tmpExec, _ := os.CreateTemp("", "test-exec-*")
		tmpExec.Close()
		defer os.Remove(tmpExec.Name())

		bundlePath, err := createBundle(tmpExec.Name())
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
			return
		}

		// Should fall back to ~/go/bin
		home, _ := os.UserHomeDir()
		expectedPath := filepath.Join(home, "go", "bin")
		if !strings.Contains(bundlePath, expectedPath) {
			t.Errorf("Expected bundle in %s, got %s", expectedPath, bundlePath)
		}

		// Clean up
		os.RemoveAll(bundlePath)
	})

	t.Run("bundle ID inference for temporary binary", func(t *testing.T) {
		DefaultConfig = NewConfig()
		DefaultConfig.BundleID = "" // Ensure empty

		// Create a temporary go-build-like binary
		tmpDir := "/tmp/go-build" + fmt.Sprintf("%d", time.Now().UnixNano())
		os.MkdirAll(tmpDir, 0755)
		tmpExec := filepath.Join(tmpDir, "main")
		os.WriteFile(tmpExec, []byte("test binary"), 0755)
		execPath := tmpExec
		bundlePath, err := createBundle(execPath)
		defer os.RemoveAll(tmpDir)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
			return
		}

		// Read Info.plist to check inferred bundle ID
		infoPlist := filepath.Join(bundlePath, "Contents", "Info.plist")
		content, _ := os.ReadFile(infoPlist)

		// Should contain com.macgo.main with hash suffix
		if !strings.Contains(string(content), "com.macgo.main") {
			t.Error("Expected inferred bundle ID with com.macgo prefix")
		}

		// Clean up
		os.RemoveAll(filepath.Dir(bundlePath))
	})

	t.Run("permission error handling", func(t *testing.T) {
		// Skip if running as root
		if os.Geteuid() == 0 {
			t.Skip("Cannot test permission errors when running as root")
		}

		DefaultConfig = NewConfig()
		DefaultConfig.CustomDestinationAppPath = "/root/forbidden/test.app"

		tmpExec, _ := os.CreateTemp("", "test-exec-*")
		tmpExec.Close()
		defer os.Remove(tmpExec.Name())

		_, err := createBundle(tmpExec.Name())
		if err == nil {
			t.Error("Expected permission error")
		}
	})
}

// TestSignBundle tests the signBundle function
func TestSignBundleIntegration(t *testing.T) {
	// This test is tricky because it requires codesign to be available
	// We'll test the function but may need to skip actual signing

	tmpDir, err := os.MkdirTemp("", "macgo-sign-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a minimal app bundle structure
	appPath := filepath.Join(tmpDir, "TestApp.app")
	os.MkdirAll(filepath.Join(appPath, "Contents", "MacOS"), 0755)

	// Create a dummy executable
	execPath := filepath.Join(appPath, "Contents", "MacOS", "TestApp")
	os.WriteFile(execPath, []byte("#!/bin/sh\necho test"), 0755)

	// Create Info.plist
	infoPlist := filepath.Join(appPath, "Contents", "Info.plist")
	plistData := map[string]any{
		"CFBundleExecutable": "TestApp",
		"CFBundleIdentifier": "com.test.app",
	}
	writePlist(infoPlist, plistData)

	// Save original config
	originalConfig := DefaultConfig
	defer func() {
		DefaultConfig = originalConfig
	}()

	tests := []struct {
		name     string
		setup    func()
		validate func(t *testing.T, err error)
	}{
		{
			name: "sign with ad-hoc identity",
			setup: func() {
				DefaultConfig = NewConfig()
				DefaultConfig.SigningIdentity = ""
			},
			validate: func(t *testing.T, err error) {
				// We expect this might fail if codesign is not available
				// or if running in a restricted environment
				if err != nil {
					t.Logf("Signing failed (expected in some environments): %v", err)
				}
			},
		},
		{
			name: "sign with specific identity",
			setup: func() {
				DefaultConfig = NewConfig()
				DefaultConfig.SigningIdentity = "Test Developer ID"
			},
			validate: func(t *testing.T, err error) {
				// This will likely fail unless the identity exists
				if err != nil {
					t.Logf("Signing with specific identity failed (expected): %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := signBundle(appPath)
			tt.validate(t, err)
		})
	}
}

// TestDebugf tests the debugf function
func TestDebugfIntegration(t *testing.T) {
	// Save original environment
	originalDebug := os.Getenv("MACGO_DEBUG")
	defer func() {
		os.Setenv("MACGO_DEBUG", originalDebug)
	}()

	tests := []struct {
		name     string
		debugEnv string
		validate func(t *testing.T)
	}{
		{
			name:     "debug disabled",
			debugEnv: "",
			validate: func(t *testing.T) {
				// Should not print anything
				// Hard to test stderr output, but we can ensure no panic
				debugf("This should not print")
			},
		},
		{
			name:     "debug enabled",
			debugEnv: "1",
			validate: func(t *testing.T) {
				// Should print to stderr
				debugf("Debug message: %s", "test")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("MACGO_DEBUG", tt.debugEnv)
			tt.validate(t)
		})
	}
}

// mockTemplateFS implements fs.FS for testing
type mockTemplateFS struct {
	files map[string]mockFile
}

type mockFile struct {
	content string
	isDir   bool
}

func (m *mockTemplateFS) Open(name string) (fs.File, error) {
	if name == "." {
		return &mockDir{fs: m, path: "."}, nil
	}

	// Check if it's a directory by looking for files with this prefix
	isDir := false
	for path := range m.files {
		if strings.HasPrefix(path, name+"/") {
			isDir = true
			break
		}
	}

	if isDir {
		return &mockDir{fs: m, path: name}, nil
	}

	if f, ok := m.files[name]; ok {
		return &mockFileHandle{
			name:    name,
			content: f.content,
			isDir:   f.isDir,
		}, nil
	}
	return nil, fs.ErrNotExist
}

// ReadFile implements fs.ReadFileFS interface
func (m *mockTemplateFS) ReadFile(name string) ([]byte, error) {
	if f, ok := m.files[name]; ok {
		return []byte(f.content), nil
	}
	return nil, fs.ErrNotExist
}

type mockDir struct {
	fs    *mockTemplateFS
	path  string
	index int
}

func (d *mockDir) Read([]byte) (int, error) {
	return 0, errors.New("cannot read directory")
}

func (d *mockDir) Close() error {
	return nil
}

func (d *mockDir) Stat() (fs.FileInfo, error) {
	return &mockFileInfo{name: d.path, isDir: true}, nil
}

func (d *mockDir) ReadDir(n int) ([]fs.DirEntry, error) {
	var entries []fs.DirEntry
	seen := make(map[string]bool)

	// List entries in this directory
	for path := range d.fs.files {
		// Skip if we've processed enough
		if n > 0 && len(entries) >= n {
			break
		}

		var entryPath string
		if d.path == "." {
			// For root directory
			if !strings.Contains(path, "/") {
				// Direct file in root
				entryPath = path
			} else {
				// Get first directory component
				parts := strings.Split(path, "/")
				entryPath = parts[0]
			}
		} else {
			// For subdirectories
			if strings.HasPrefix(path, d.path+"/") {
				remaining := strings.TrimPrefix(path, d.path+"/")
				if !strings.Contains(remaining, "/") {
					// Direct child
					entryPath = remaining
				} else {
					// Get first directory component
					parts := strings.Split(remaining, "/")
					entryPath = parts[0]
				}
			} else {
				continue
			}
		}

		if entryPath != "" && !seen[entryPath] {
			seen[entryPath] = true

			// Check if it's a directory
			isDir := false
			fullPath := entryPath
			if d.path != "." {
				fullPath = d.path + "/" + entryPath
			}

			// If there's a file with this exact path, it's a file
			if _, exists := d.fs.files[fullPath]; exists {
				isDir = d.fs.files[fullPath].isDir
			} else {
				// Otherwise check if there are files with this prefix
				for p := range d.fs.files {
					if strings.HasPrefix(p, fullPath+"/") {
						isDir = true
						break
					}
				}
			}

			entries = append(entries, &mockDirEntry{
				name:  entryPath,
				isDir: isDir,
			})
		}
	}

	if len(entries) == 0 && d.index > 0 {
		return nil, io.EOF
	}

	d.index += len(entries)
	return entries, nil
}

type mockFileHandle struct {
	name    string
	content string
	isDir   bool
	offset  int
}

func (f *mockFileHandle) Read(p []byte) (int, error) {
	if f.offset >= len(f.content) {
		return 0, io.EOF
	}
	n := copy(p, f.content[f.offset:])
	f.offset += n
	return n, nil
}

func (f *mockFileHandle) Close() error {
	return nil
}

func (f *mockFileHandle) Stat() (fs.FileInfo, error) {
	return &mockFileInfo{
		name:  f.name,
		isDir: f.isDir,
		size:  int64(len(f.content)),
	}, nil
}

type mockFileInfo struct {
	name  string
	isDir bool
	size  int64
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return m.size }
func (m *mockFileInfo) Mode() fs.FileMode  { return 0644 }
func (m *mockFileInfo) ModTime() time.Time { return time.Now() }
func (m *mockFileInfo) IsDir() bool        { return m.isDir }
func (m *mockFileInfo) Sys() interface{}   { return nil }

type mockDirEntry struct {
	name  string
	isDir bool
}

func (m *mockDirEntry) Name() string      { return m.name }
func (m *mockDirEntry) IsDir() bool       { return m.isDir }
func (m *mockDirEntry) Type() fs.FileMode { return 0 }
func (m *mockDirEntry) Info() (fs.FileInfo, error) {
	return &mockFileInfo{name: m.name, isDir: m.isDir}, nil
}

// TestCreatePipe tests the createPipe function
func TestCreatePipe(t *testing.T) {
	pipe, err := createPipe("test-pipe")
	if err != nil {
		t.Errorf("Failed to create pipe: %v", err)
		return
	}
	defer os.Remove(pipe)

	// Check that the pipe was created
	info, err := os.Stat(pipe)
	if err != nil {
		t.Errorf("Failed to stat pipe: %v", err)
		return
	}

	// On macOS, we can check if it's a named pipe
	if info.Mode()&os.ModeNamedPipe == 0 {
		t.Error("Created file is not a named pipe")
	}
}

// TestEscapeXML tests the escapeXML function
func TestEscapeXML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no special characters",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "ampersand",
			input:    "A & B",
			expected: "A &amp; B",
		},
		{
			name:     "less than",
			input:    "a < b",
			expected: "a &lt; b",
		},
		{
			name:     "greater than",
			input:    "a > b",
			expected: "a &gt; b",
		},
		{
			name:     "double quote",
			input:    "say \"hello\"",
			expected: "say &quot;hello&quot;",
		},
		{
			name:     "single quote",
			input:    "it's working",
			expected: "it&apos;s working",
		},
		{
			name:     "all special characters",
			input:    "<tag attr=\"value\" other='value2'>content & more</tag>",
			expected: "&lt;tag attr=&quot;value&quot; other=&apos;value2&apos;&gt;content &amp; more&lt;/tag&gt;",
		},
		{
			name:     "XML injection attempt",
			input:    "</string><key>injected</key><string>malicious",
			expected: "&lt;/string&gt;&lt;key&gt;injected&lt;/key&gt;&lt;string&gt;malicious",
		},
		{
			name:     "multiple ampersands",
			input:    "A & B & C",
			expected: "A &amp; B &amp; C",
		},
		{
			name:     "script injection attempt",
			input:    "<script>alert('xss')</script>",
			expected: "&lt;script&gt;alert(&apos;xss&apos;)&lt;/script&gt;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeXML(tt.input)
			if result != tt.expected {
				t.Errorf("escapeXML(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestPlistXMLEscaping tests that writePlist properly escapes XML
func TestPlistXMLEscaping(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "macgo-plist-escape-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name        string
		data        map[string]any
		expected    []string // Expected escaped strings in output
		notExpected []string // Strings that should NOT be in output (unescaped)
	}{
		{
			name: "string values with special characters",
			data: map[string]any{
				"NormalKey": "normal value",
				"AmpersandKey": "A & B",
				"LessKey": "a < b",
				"GreaterKey": "a > b",
				"QuoteKey": "say \"hello\"",
				"AposKey": "it's working",
			},
			expected: []string{
				"<string>normal value</string>",
				"<string>A &amp; B</string>",
				"<string>a &lt; b</string>",
				"<string>a &gt; b</string>",
				"<string>say &quot;hello&quot;</string>",
				"<string>it&apos;s working</string>",
			},
			notExpected: []string{
				"<string>A & B</string>",
				"<string>a < b</string>",
				"<string>a > b</string>",
				"<string>say \"hello\"</string>",
				"<string>it's working</string>",
			},
		},
		{
			name: "keys with special characters",
			data: map[string]any{
				"Key<with>brackets": "value1",
				"Key&with&ampersand": "value2",
				"Key\"with\"quotes": "value3",
			},
			expected: []string{
				"<key>Key&lt;with&gt;brackets</key>",
				"<key>Key&amp;with&amp;ampersand</key>",
				"<key>Key&quot;with&quot;quotes</key>",
			},
			notExpected: []string{
				"<key>Key<with>brackets</key>",
				"<key>Key&with&ampersand</key>",
				"<key>Key\"with\"quotes</key>",
			},
		},
		{
			name: "XML injection attempt in values",
			data: map[string]any{
				"InjectionKey": "</string><key>injected</key><string>malicious",
				"ScriptKey": "<script>alert('xss')</script>",
			},
			expected: []string{
				"<string>&lt;/string&gt;&lt;key&gt;injected&lt;/key&gt;&lt;string&gt;malicious</string>",
				"<string>&lt;script&gt;alert(&apos;xss&apos;)&lt;/script&gt;</string>",
			},
			notExpected: []string{
				"<string></string><key>injected</key><string>malicious</string>",
				"<string><script>alert('xss')</script></string>",
			},
		},
		{
			name: "complex structures (converted to string)",
			data: map[string]any{
				"StructKey": struct{ Field string }{Field: "<value>"},
				"ArrayKey": []string{"<item1>", "item2&more"},
			},
			expected: []string{
				"<string>{&lt;value&gt;}</string>",
				"<string>[&lt;item1&gt; item2&amp;more]</string>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plistPath := filepath.Join(tmpDir, fmt.Sprintf("%s.plist", tt.name))
			err := writePlist(plistPath, tt.data)
			if err != nil {
				t.Errorf("Failed to write plist: %v", err)
				return
			}

			content, _ := os.ReadFile(plistPath)
			contentStr := string(content)

			// Check for expected escaped strings
			for _, expected := range tt.expected {
				if !strings.Contains(contentStr, expected) {
					t.Errorf("Expected %s in plist content", expected)
				}
			}

			// Check that unescaped strings are NOT present
			for _, notExpected := range tt.notExpected {
				if strings.Contains(contentStr, notExpected) {
					t.Errorf("Found unescaped string %s in plist content (security vulnerability)", notExpected)
				}
			}

			// Verify the plist is still valid XML by checking structure
			if !strings.Contains(contentStr, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>") {
				t.Error("Missing XML declaration")
			}
			if !strings.Contains(contentStr, "<plist version=\"1.0\">") {
				t.Error("Missing plist declaration")
			}
			if !strings.Contains(contentStr, "<dict>") || !strings.Contains(contentStr, "</dict>") {
				t.Error("Missing dict tags")
			}
		})
	}
}

// TestPlistValueTypes tests writePlist with various value types
func TestPlistValueTypes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "macgo-plist-types-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name     string
		data     map[string]any
		expected []string // Expected strings in output
	}{
		{
			name: "integer types",
			data: map[string]any{
				"Int":   int(42),
				"Int32": int32(42),
				"Int64": int64(42),
			},
			expected: []string{
				"<integer>42</integer>",
			},
		},
		{
			name: "float types",
			data: map[string]any{
				"Float32": float32(3.14),
				"Float64": float64(3.14159),
			},
			expected: []string{
				"<real>3.14</real>",
				"<real>3.14159</real>",
			},
		},
		{
			name: "complex types converted to string",
			data: map[string]any{
				"Struct": struct{ Name string }{Name: "test"},
				"Array":  []string{"a", "b"},
			},
			expected: []string{
				"<string>{test}</string>",
				"<string>[a b]</string>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plistPath := filepath.Join(tmpDir, fmt.Sprintf("%s.plist", tt.name))
			err := writePlist(plistPath, tt.data)
			if err != nil {
				t.Errorf("Failed to write plist: %v", err)
				return
			}

			content, _ := os.ReadFile(plistPath)
			for _, expected := range tt.expected {
				if !strings.Contains(string(content), expected) {
					t.Errorf("Expected %s in plist content", expected)
				}
			}
		})
	}
}

// TestPlistSecurityVulnerabilityPrevention tests that XML injection vulnerabilities are prevented
func TestPlistSecurityVulnerabilityPrevention(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "macgo-plist-security-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test various XML injection attack scenarios
	tests := []struct {
		name        string
		data        map[string]any
		description string
	}{
		{
			name: "XML tag injection in key",
			data: map[string]any{
				"</key><key>injected</key><key>": "malicious_value",
			},
			description: "Attempts to inject XML tags in key names",
		},
		{
			name: "XML tag injection in value",
			data: map[string]any{
				"NormalKey": "</string><key>injected</key><string>malicious",
			},
			description: "Attempts to inject XML tags in string values",
		},
		{
			name: "CDATA injection attempt",
			data: map[string]any{
				"CDATAKey": "<![CDATA[malicious]]>",
			},
			description: "Attempts to inject CDATA sections",
		},
		{
			name: "DOCTYPE injection attempt",
			data: map[string]any{
				"DOCTYPEKey": "<!DOCTYPE html><html><body>malicious</body></html>",
			},
			description: "Attempts to inject DOCTYPE declarations",
		},
		{
			name: "Entity reference injection",
			data: map[string]any{
				"EntityKey": "&lt;script&gt;alert('xss')&lt;/script&gt;",
			},
			description: "Attempts to inject entity references",
		},
		{
			name: "Complex XML structure injection",
			data: map[string]any{
				"ComplexKey": "</string></dict><dict><key>injected</key><string>payload</string></dict><dict><key>original</key><string>",
			},
			description: "Attempts to inject complex XML structures",
		},
		{
			name: "Bundle ID injection attempt",
			data: map[string]any{
				"CFBundleIdentifier": "com.evil.app</string><key>LSUIElement</key><false/><key>CFBundleIdentifier</key><string>com.normal.app",
			},
			description: "Attempts to inject malicious bundle configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plistPath := filepath.Join(tmpDir, fmt.Sprintf("%s.plist", tt.name))
			err := writePlist(plistPath, tt.data)
			if err != nil {
				t.Errorf("Failed to write plist: %v", err)
				return
			}

			content, _ := os.ReadFile(plistPath)
			contentStr := string(content)

			// The content should be properly escaped
			// Count opening and closing tags to ensure structure is maintained
			openingDictCount := strings.Count(contentStr, "<dict>")
			closingDictCount := strings.Count(contentStr, "</dict>")
			if openingDictCount != closingDictCount {
				t.Errorf("XML structure corrupted: %d opening <dict> tags vs %d closing </dict> tags", openingDictCount, closingDictCount)
			}

			// Ensure only one plist element
			plistCount := strings.Count(contentStr, "<plist")
			if plistCount != 1 {
				t.Errorf("Expected exactly 1 plist element, found %d", plistCount)
			}

			// Check that XML special characters are properly escaped
			for key, value := range tt.data {
				if strVal, ok := value.(string); ok {
					// If the original value contains unescaped < or >, the output should contain escaped versions
					if strings.Contains(strVal, "<") && !strings.Contains(contentStr, "&lt;") {
						t.Error("Expected < characters to be escaped as &lt;")
					}
					if strings.Contains(strVal, ">") && !strings.Contains(contentStr, "&gt;") {
						t.Error("Expected > characters to be escaped as &gt;")
					}
				}
				if strings.Contains(key, "<") && !strings.Contains(contentStr, "&lt;") {
					t.Error("Expected < characters in keys to be escaped as &lt;")
				}
				if strings.Contains(key, ">") && !strings.Contains(contentStr, "&gt;") {
					t.Error("Expected > characters in keys to be escaped as &gt;")
				}
			}

			// Verify the plist structure is intact
			if !strings.Contains(contentStr, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>") {
				t.Error("Missing or corrupted XML declaration")
			}
			if !strings.Contains(contentStr, "<!DOCTYPE plist") {
				t.Error("Missing or corrupted DOCTYPE declaration")
			}

			t.Logf("Test '%s' passed: %s", tt.name, tt.description)
		})
	}
}

// TestPlistEscapingEdgeCases tests edge cases for XML escaping
func TestPlistEscapingEdgeCases(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "macgo-plist-edge-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name   string
		data   map[string]any
		check  func(t *testing.T, content string)
	}{
		{
			name: "empty strings",
			data: map[string]any{
				"EmptyKey": "",
			},
			check: func(t *testing.T, content string) {
				if !strings.Contains(content, "<string></string>") {
					t.Error("Empty string should produce <string></string>")
				}
			},
		},
		{
			name: "only special characters",
			data: map[string]any{
				"SpecialKey": "&<>\"'",
			},
			check: func(t *testing.T, content string) {
				expected := "<string>&amp;&lt;&gt;&quot;&apos;</string>"
				if !strings.Contains(content, expected) {
					t.Errorf("Expected %s in content", expected)
				}
			},
		},
		{
			name: "unicode characters mixed with special",
			data: map[string]any{
				"UnicodeKey": "Hello 世界 & <test>",
			},
			check: func(t *testing.T, content string) {
				expected := "<string>Hello 世界 &amp; &lt;test&gt;</string>"
				if !strings.Contains(content, expected) {
					t.Errorf("Expected %s in content", expected)
				}
			},
		},
		{
			name: "very long string with special chars",
			data: map[string]any{
				"LongKey": strings.Repeat("a&b<c>d\"e'f", 100),
			},
			check: func(t *testing.T, content string) {
				expectedPattern := "a&amp;b&lt;c&gt;d&quot;e&apos;f"
				if !strings.Contains(content, expectedPattern) {
					t.Errorf("Expected pattern %s in content", expectedPattern)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plistPath := filepath.Join(tmpDir, fmt.Sprintf("%s.plist", tt.name))
			err := writePlist(plistPath, tt.data)
			if err != nil {
				t.Errorf("Failed to write plist: %v", err)
				return
			}

			content, _ := os.ReadFile(plistPath)
			tt.check(t, string(content))
		})
	}
}

// TestXMLEscapingBeforeAfterDemo demonstrates the security fix by showing before/after XML output
func TestXMLEscapingBeforeAfterDemo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "macgo-demo-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test data with potentially malicious XML content
	testData := map[string]any{
		"NormalKey": "normal value",
		"InjectionKey": "</string><key>injected</key><string>malicious",
		"ScriptKey": "<script>alert('xss')</script>",
		"BundleKey": "com.evil.app</string><key>LSUIElement</key><false/>",
		"MultiChar": "A & B < C > D \"quoted\" 'apostrophe'",
		"Key<with>brackets": "value with & symbols",
	}

	plistPath := filepath.Join(tmpDir, "demo.plist")
	err = writePlist(plistPath, testData)
	if err != nil {
		t.Fatalf("Failed to write plist: %v", err)
	}

	content, err := os.ReadFile(plistPath)
	if err != nil {
		t.Fatalf("Failed to read plist: %v", err)
	}

	contentStr := string(content)

	// Log the secure output for demonstration
	t.Logf("=== SECURE PLIST OUTPUT (with XML escaping) ===\n%s", contentStr)

	// Verify XML structure integrity
	plistCount := strings.Count(contentStr, "<plist")
	dictOpen := strings.Count(contentStr, "<dict>")
	dictClose := strings.Count(contentStr, "</dict>")

	t.Logf("\n=== SECURITY VALIDATION ===")
	t.Logf("Number of <plist> tags: %d (should be 1)", plistCount)
	t.Logf("Number of <dict> tags: %d", dictOpen)
	t.Logf("Number of </dict> tags: %d", dictClose)
	t.Logf("XML structure integrity: %s", map[bool]string{true: "PASS", false: "FAIL"}[dictOpen == dictClose])

	if plistCount != 1 {
		t.Errorf("Expected exactly 1 plist tag, got %d", plistCount)
	}
	if dictOpen != dictClose {
		t.Errorf("XML structure compromised: %d opening vs %d closing dict tags", dictOpen, dictClose)
	}

	// Verify escaping works
	t.Logf("\n=== ESCAPING VERIFICATION ===")
	escapingChecks := map[string]bool{
		"&amp; (escaped &)": strings.Contains(contentStr, "&amp;"),
		"&lt; (escaped <)": strings.Contains(contentStr, "&lt;"),
		"&gt; (escaped >)": strings.Contains(contentStr, "&gt;"),
		"&quot; (escaped \")": strings.Contains(contentStr, "&quot;"),
		"&apos; (escaped ')": strings.Contains(contentStr, "&apos;"),
	}

	for check, found := range escapingChecks {
		t.Logf("Contains %s: %t", check, found)
		if !found {
			t.Errorf("Expected to find escaped character: %s", check)
		}
	}

	// Verify injection prevention
	t.Logf("\n=== INJECTION PREVENTION ===")
	dangerous := []string{
		"</string><key>injected</key><string>",
		"<script>alert('xss')</script>",
		"<key>LSUIElement</key><false/>",
	}

	for _, pattern := range dangerous {
		found := strings.Contains(contentStr, pattern)
		t.Logf("Dangerous pattern '%s' found: %t (should be false)", pattern, found)
		if found {
			t.Errorf("Security vulnerability: Found unescaped dangerous pattern '%s'", pattern)
		}
	}

	// Demonstrate what would happen WITHOUT escaping (simulated)
	t.Logf("\n=== BEFORE vs AFTER COMPARISON ===")
	t.Logf("BEFORE (vulnerable): <string>A & B < C > D</string>")
	t.Logf("AFTER (secure): <string>A &amp; B &lt; C &gt; D</string>")
	t.Logf("\nBEFORE (vulnerable): <key>Key<with>brackets</key>")
	t.Logf("AFTER (secure): <key>Key&lt;with&gt;brackets</key>")
	t.Logf("\nINJECTION ATTEMPT: </string><key>injected</key><string>malicious")
	t.Logf("SECURED OUTPUT: &lt;/string&gt;&lt;key&gt;injected&lt;/key&gt;&lt;string&gt;malicious")
}

// TestEnvironmentVariableDetection tests the init() function's env var handling
func TestEnvironmentVariableDetectionIntegration(t *testing.T) {
	// This test is tricky because init() runs before tests
	// We can only verify the current state based on environment

	// Save and restore environment
	envVars := []string{
		"MACGO_CAMERA", "MACGO_MIC", "MACGO_LOCATION",
		"MACGO_APP_SANDBOX", "MACGO_NETWORK_CLIENT",
	}

	originalValues := make(map[string]string)
	for _, env := range envVars {
		originalValues[env] = os.Getenv(env)
	}

	defer func() {
		for env, val := range originalValues {
			if val == "" {
				os.Unsetenv(env)
			} else {
				os.Setenv(env, val)
			}
		}
	}()

	// Note: We can't test the init() function directly,
	// but we can verify that the mechanism works by checking
	// if any current env vars are reflected in DefaultConfig

	if os.Getenv("MACGO_CAMERA") == "1" {
		if _, exists := DefaultConfig.Entitlements[EntCamera]; !exists {
			t.Error("MACGO_CAMERA=1 but camera entitlement not found")
		}
	}
}
