package bundle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tmc/misc/macgo/internal/system"
)

func TestBundle_isBundleUpToDate(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a simple test executable
	testExecPath := filepath.Join(tmpDir, "test-exec")
	testContent := []byte("#!/bin/bash\necho 'test executable'\n")
	if err := os.WriteFile(testExecPath, testContent, 0755); err != nil {
		t.Fatalf("Failed to create test executable: %v", err)
	}

	// Create bundle with debug enabled
	config := &Config{
		AppName: "TestApp",
		Debug:   true,
	}
	bundle, err := New(testExecPath, config)
	if err != nil {
		t.Fatalf("Failed to create bundle: %v", err)
	}

	t.Run("bundle_not_created", func(t *testing.T) {
		// Before creating bundle, should return false
		if bundle.isBundleUpToDate() {
			t.Error("Expected false for non-existent bundle")
		}
	})

	// Create the bundle
	if err := bundle.Create(); err != nil {
		t.Fatalf("Failed to create bundle: %v", err)
	}

	t.Run("bundle_up_to_date", func(t *testing.T) {
		// After creating bundle, should return true (identical hashes)
		if !bundle.isBundleUpToDate() {
			t.Error("Expected true for newly created bundle")
		}
	})

	t.Run("source_binary_changed", func(t *testing.T) {
		// Modify the source executable
		modifiedContent := []byte("#!/bin/bash\necho 'modified test executable'\n")
		if err := os.WriteFile(testExecPath, modifiedContent, 0755); err != nil {
			t.Fatalf("Failed to modify test executable: %v", err)
		}

		// Should return false now (different hashes)
		if bundle.isBundleUpToDate() {
			t.Error("Expected false after modifying source binary")
		}
	})

	t.Run("bundle_executable_missing", func(t *testing.T) {
		// Remove the executable from the bundle
		execName := filepath.Base(bundle.appName)
		bundleExecPath := filepath.Join(bundle.Path, "Contents", "MacOS", execName)
		if err := os.Remove(bundleExecPath); err != nil {
			t.Fatalf("Failed to remove bundle executable: %v", err)
		}

		// Should return false when bundle executable is missing
		if bundle.isBundleUpToDate() {
			t.Error("Expected false when bundle executable is missing")
		}
	})
}

func TestBundle_SHA256_Integration(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a test executable
	testExecPath := filepath.Join(tmpDir, "integration-test")
	originalContent := []byte("#!/bin/bash\necho 'original version'\n")
	if err := os.WriteFile(testExecPath, originalContent, 0755); err != nil {
		t.Fatalf("Failed to create test executable: %v", err)
	}

	// Set GOPATH to our temp directory to control where bundles are created
	originalGOPATH := os.Getenv("GOPATH")
	defer func() {
		if originalGOPATH == "" {
			os.Unsetenv("GOPATH")
		} else {
			os.Setenv("GOPATH", originalGOPATH)
		}
	}()
	os.Setenv("GOPATH", tmpDir)

	// Create the bin directory
	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	config := &Config{
		AppName:    "TestApp",
		Debug:      true,
		KeepBundle: &[]bool{true}[0], // Explicit true to test reuse logic
	}

	// First creation
	bundle1, err := New(testExecPath, config)
	if err != nil {
		t.Fatalf("Failed to create bundle: %v", err)
	}

	if err := bundle1.Create(); err != nil {
		t.Fatalf("Failed to create bundle: %v", err)
	}

	bundleDir := bundle1.Path
	t.Logf("Bundle created at: %s", bundleDir)

	// Verify bundle exists
	if _, err := os.Stat(bundleDir); err != nil {
		t.Fatalf("Bundle directory not created: %v", err)
	}

	// Second creation with same binary should reuse
	bundle2, err := New(testExecPath, config)
	if err != nil {
		t.Fatalf("Failed to create second bundle instance: %v", err)
	}

	if err := bundle2.Create(); err != nil {
		t.Fatalf("Failed on second Create() call: %v", err)
	}

	// Verify it was reused (same path)
	if bundle2.Path != bundleDir {
		t.Errorf("Expected bundle path %s, got %s", bundleDir, bundle2.Path)
	}

	// Modify the source binary
	modifiedContent := []byte("#!/bin/bash\necho 'modified version'\n")
	if err := os.WriteFile(testExecPath, modifiedContent, 0755); err != nil {
		t.Fatalf("Failed to modify test executable: %v", err)
	}

	// Third creation should recreate the bundle due to binary change
	bundle3, err := New(testExecPath, config)
	if err != nil {
		t.Fatalf("Failed to create third bundle instance: %v", err)
	}

	if err := bundle3.Create(); err != nil {
		t.Fatalf("Failed on third Create() call after binary change: %v", err)
	}

	// Verify the bundle executable now has the modified content
	execName := filepath.Base(bundle3.appName)
	bundleExecPath := filepath.Join(bundle3.Path, "Contents", "MacOS", execName)
	bundleContent, err := os.ReadFile(bundleExecPath)
	if err != nil {
		t.Fatalf("Failed to read bundle executable: %v", err)
	}

	if string(bundleContent) != string(modifiedContent) {
		t.Errorf("Bundle executable content doesn't match modified source.\nExpected: %q\nGot: %q",
			string(modifiedContent), string(bundleContent))
	}
}

func TestSHA256_Comparison(t *testing.T) {
	// Test that our SHA256 comparison actually works
	tmpDir := t.TempDir()

	// Create two identical files
	file1 := filepath.Join(tmpDir, "file1")
	file2 := filepath.Join(tmpDir, "file2")
	content := []byte("test content for SHA256")

	if err := os.WriteFile(file1, content, 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	if err := os.WriteFile(file2, content, 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	// Calculate hashes
	hash1, err := system.CalculateFileSHA256(file1)
	if err != nil {
		t.Fatalf("Failed to calculate hash for file1: %v", err)
	}
	hash2, err := system.CalculateFileSHA256(file2)
	if err != nil {
		t.Fatalf("Failed to calculate hash for file2: %v", err)
	}

	// Identical files should have identical hashes
	if hash1 != hash2 {
		t.Errorf("Identical files have different hashes: %q vs %q", hash1, hash2)
	}

	// Modify one file
	modifiedContent := []byte("modified test content for SHA256")
	if err := os.WriteFile(file2, modifiedContent, 0644); err != nil {
		t.Fatalf("Failed to modify file2: %v", err)
	}

	// Recalculate hash for modified file
	hash2Modified, err := system.CalculateFileSHA256(file2)
	if err != nil {
		t.Fatalf("Failed to calculate hash for modified file2: %v", err)
	}

	// Different files should have different hashes
	if hash1 == hash2Modified {
		t.Errorf("Different files have identical hashes: %q", hash1)
	}
}
