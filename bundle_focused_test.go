// +build darwin

package macgo

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestBundleChecksum tests the checksum calculation function
func TestBundleChecksum(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "macgo-checksum-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	testContent := "test content for checksum verification"
	if _, err := tmpFile.Write([]byte(testContent)); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	// Calculate checksum
	hash, err := checksum(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to calculate checksum: %v", err)
	}

	// Verify it's a valid SHA256 hash (64 hex chars)
	if len(hash) != 64 {
		t.Errorf("Expected 64 character hash, got %d", len(hash))
	}

	// Verify correctness by calculating expected hash
	h := sha256.New()
	h.Write([]byte(testContent))
	expected := hex.EncodeToString(h.Sum(nil))
	
	if hash != expected {
		t.Errorf("Expected hash %s, got %s", expected, hash)
	}

	// Test with non-existent file
	_, err = checksum("/non/existent/file")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

// TestBundleCopyFileFunction tests the copyFile function
func TestBundleCopyFileFunction(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "macgo-copy-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create source file
	srcPath := filepath.Join(tmpDir, "source.txt")
	testContent := "test file content for copy operation"
	if err := os.WriteFile(srcPath, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test successful copy
	dstPath := filepath.Join(tmpDir, "destination.txt")
	if err := copyFile(srcPath, dstPath); err != nil {
		t.Fatalf("Failed to copy file: %v", err)
	}

	// Verify content
	copiedContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(copiedContent) != testContent {
		t.Error("Copied content does not match original")
	}

	// Verify permissions (should be 0755)
	info, err := os.Stat(dstPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0755 {
		t.Errorf("Expected permissions 0755, got %v", info.Mode().Perm())
	}

	// Test copy of non-existent file
	err = copyFile("/non/existent/source", filepath.Join(tmpDir, "fail.txt"))
	if err == nil {
		t.Error("Expected error when copying non-existent file")
	}
}

// TestBundleWritePlist tests the writePlist function
func TestBundleWritePlist(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "macgo-plist-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test basic plist writing
	plistPath := filepath.Join(tmpDir, "test.plist")
	data := map[string]any{
		"CFBundleName":       "TestApp",
		"CFBundleIdentifier": "com.test.app",
		"LSUIElement":        true,
		"NumberValue":        42,
		"FloatValue":         3.14,
	}

	err = writePlist(plistPath, data)
	if err != nil {
		t.Fatalf("Failed to write plist: %v", err)
	}

	// Read and verify content
	content, err := os.ReadFile(plistPath)
	if err != nil {
		t.Fatal(err)
	}

	plistStr := string(content)
	
	// Check for expected content
	checks := []string{
		"<?xml version=\"1.0\" encoding=\"UTF-8\"?>",
		"<!DOCTYPE plist",
		"<key>CFBundleName</key>",
		"<string>TestApp</string>",
		"<key>LSUIElement</key>",
		"<true/>",
		"<key>NumberValue</key>",
		"<integer>42</integer>",
		"<key>FloatValue</key>",
		"<real>3.14</real>",
	}

	for _, check := range checks {
		if !strings.Contains(plistStr, check) {
			t.Errorf("Expected plist to contain %q", check)
		}
	}

	// Test with entitlements
	entPath := filepath.Join(tmpDir, "entitlements.plist")
	entData := map[Entitlement]any{
		EntCamera:     true,
		EntMicrophone: false,
		EntAppSandbox: true,
	}

	err = writePlist(entPath, entData)
	if err != nil {
		t.Fatalf("Failed to write entitlements plist: %v", err)
	}

	entContent, _ := os.ReadFile(entPath)
	entStr := string(entContent)

	// Check entitlement values
	if !strings.Contains(entStr, "com.apple.security.device.camera") {
		t.Error("Missing camera entitlement")
	}
	if !strings.Contains(entStr, "<true/>") {
		t.Error("Missing true value for camera")
	}
	if !strings.Contains(entStr, "com.apple.security.device.microphone") {
		t.Error("Missing microphone entitlement")  
	}
	if !strings.Contains(entStr, "<false/>") {
		t.Error("Missing false value for microphone")
	}
}

// TestBundleCheckExisting tests the checkExisting function
func TestBundleCheckExisting(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "macgo-existing-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test executable
	execPath := filepath.Join(tmpDir, "test-binary")
	execContent := []byte("test executable content")
	if err := os.WriteFile(execPath, execContent, 0755); err != nil {
		t.Fatal(err)
	}

	// Test 1: App bundle doesn't exist
	appPath := filepath.Join(tmpDir, "TestApp.app")
	result := checkExisting(appPath, execPath)
	if result {
		t.Error("Expected false when app bundle doesn't exist")
	}

	// Test 2: Create app bundle but without executable
	bundleExecPath := filepath.Join(appPath, "Contents", "MacOS", "test-binary")
	if err := os.MkdirAll(filepath.Dir(bundleExecPath), 0755); err != nil {
		t.Fatal(err)
	}
	
	result = checkExisting(appPath, execPath)
	if result {
		t.Error("Expected false when bundle executable doesn't exist")
	}

	// Test 3: Add executable with same content
	if err := copyFile(execPath, bundleExecPath); err != nil {
		t.Fatal(err)
	}
	
	result = checkExisting(appPath, execPath)
	if !result {
		t.Error("Expected true when bundle exists with same content")
	}

	// Test 4: Change executable content
	if err := os.WriteFile(execPath, []byte("modified content"), 0755); err != nil {
		t.Fatal(err)
	}
	
	result = checkExisting(appPath, execPath)
	if result {
		t.Error("Expected false when executable content differs")
	}

	// Verify app bundle was removed after content change
	if _, err := os.Stat(appPath); !os.IsNotExist(err) {
		t.Error("Expected app bundle to be removed when content differs")
	}
}

// TestBundleCreationStructure tests the structure created by createBundle
func TestBundleCreationStructure(t *testing.T) {
	// Skip if we can't create bundles in test environment
	if os.Getenv("MACGO_SKIP_BUNDLE_TESTS") == "1" {
		t.Skip("Skipping bundle creation test")
	}

	// Save and restore DefaultConfig
	originalConfig := DefaultConfig
	defer func() {
		DefaultConfig = originalConfig
	}()

	// Create temporary executable
	tmpExec, err := os.CreateTemp("", "test-exec-*")
	if err != nil {
		t.Fatal(err)
	}
	tmpExec.Write([]byte("#!/bin/sh\necho test"))
	tmpExec.Close()
	defer os.Remove(tmpExec.Name())
	os.Chmod(tmpExec.Name(), 0755)

	// Test with custom destination
	tmpDir, err := os.MkdirTemp("", "macgo-bundle-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	DefaultConfig = NewConfig()
	DefaultConfig.CustomDestinationAppPath = filepath.Join(tmpDir, "TestApp.app")
	DefaultConfig.ApplicationName = "TestApp"
	DefaultConfig.BundleID = "com.test.testapp"
	DefaultConfig.Entitlements = map[Entitlement]bool{
		EntCamera: true,
	}

	bundlePath, err := createBundle(tmpExec.Name())
	if err != nil {
		t.Fatalf("Failed to create bundle: %v", err)
	}

	// Verify bundle structure
	expectedPaths := []string{
		"Contents",
		"Contents/MacOS",
		filepath.Join("Contents", "MacOS", filepath.Base(tmpExec.Name())),
		"Contents/Info.plist",
		"Contents/entitlements.plist",
	}

	for _, path := range expectedPaths {
		fullPath := filepath.Join(bundlePath, path)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("Expected path %s to exist in bundle", path)
		}
	}

	// Verify Info.plist content
	infoPlistPath := filepath.Join(bundlePath, "Contents", "Info.plist")
	infoPlistContent, err := os.ReadFile(infoPlistPath)
	if err != nil {
		t.Fatal(err)
	}

	infoPlistStr := string(infoPlistContent)
	if !strings.Contains(infoPlistStr, "TestApp") {
		t.Error("Info.plist should contain application name")
	}
	if !strings.Contains(infoPlistStr, "com.test.testapp") {
		t.Error("Info.plist should contain bundle ID")
	}

	// Verify entitlements
	entPlistPath := filepath.Join(bundlePath, "Contents", "entitlements.plist")
	entPlistContent, err := os.ReadFile(entPlistPath)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(entPlistContent), "com.apple.security.device.camera") {
		t.Error("Entitlements should contain camera permission")
	}
}

// TestCreatePipeFunction tests the createPipe function
func TestCreatePipeFunction(t *testing.T) {
	// Skip on non-Darwin platforms
	if os.Getenv("GOOS") == "linux" || os.Getenv("GOOS") == "windows" {
		t.Skip("Named pipes test only runs on macOS")
	}

	pipeName, err := createPipe("test-pipe")
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	defer os.Remove(pipeName)

	// Verify the pipe was created
	info, err := os.Stat(pipeName)
	if err != nil {
		t.Fatalf("Failed to stat pipe: %v", err)
	}

	// Check if it's a named pipe (FIFO)
	if info.Mode()&os.ModeNamedPipe == 0 {
		t.Error("Created file is not a named pipe")
	}
}

// TestPipeIOContext tests the pipeIOContext function with real pipes
func TestPipeIOContext(t *testing.T) {
	// Skip on non-Darwin platforms
	if os.Getenv("GOOS") == "linux" || os.Getenv("GOOS") == "windows" {
		t.Skip("Named pipes test only runs on macOS")
	}

	// Create a named pipe
	pipeName, err := createPipe("test-io-pipe")
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	defer os.Remove(pipeName)

	testData := "Hello from pipe test"

	// Test writing to pipe
	t.Run("write to pipe", func(t *testing.T) {
		done := make(chan error, 1)
		
		// Reader goroutine
		go func() {
			f, err := os.OpenFile(pipeName, os.O_RDONLY, 0)
			if err != nil {
				done <- err
				return
			}
			defer f.Close()

			buf := make([]byte, len(testData))
			n, err := io.ReadFull(f, buf)
			if err != nil {
				done <- err
				return
			}
			if n != len(testData) || string(buf) != testData {
				done <- os.ErrInvalid
				return
			}
			done <- nil
		}()

		// Writer using pipeIO
		go func() {
			reader := strings.NewReader(testData)
			pipeIO(pipeName, reader, nil)
		}()

		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Pipe IO failed: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("Pipe IO timed out")
		}
	})
}

