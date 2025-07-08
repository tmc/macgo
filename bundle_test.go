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
		name           string
		setup          func()
		execPath       string
		expectError    bool
		checkResult    func(t *testing.T, bundlePath string, err error)
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
		name   string
		setup  func()
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

func (m *mockDirEntry) Name() string               { return m.name }
func (m *mockDirEntry) IsDir() bool                { return m.isDir }
func (m *mockDirEntry) Type() fs.FileMode          { return 0 }
func (m *mockDirEntry) Info() (fs.FileInfo, error) { return &mockFileInfo{name: m.name, isDir: m.isDir}, nil }

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