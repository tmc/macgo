package system

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCalculateFileSHA256(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := []byte("test content for SHA256 calculation")

	// Create test file
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Calculate SHA256
	hash, err := CalculateFileSHA256(testFile)
	if err != nil {
		t.Fatalf("Failed to calculate SHA256: %v", err)
	}

	// Verify hash format (should be 64 hex characters)
	if len(hash) != 64 {
		t.Errorf("Expected hash length 64, got %d", len(hash))
	}

	// Calculate again to ensure consistency
	hash2, err := CalculateFileSHA256(testFile)
	if err != nil {
		t.Fatalf("Failed to calculate SHA256 second time: %v", err)
	}

	if hash != hash2 {
		t.Errorf("Hash is not consistent: %q vs %q", hash, hash2)
	}

	// Test with empty file path
	_, err = CalculateFileSHA256("")
	if err == nil {
		t.Error("Expected error for empty file path")
	}

	// Test with non-existent file
	_, err = CalculateFileSHA256("/non/existent/file")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestCompareFileSHA256(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	content := []byte("identical content")

	// Create two identical files
	if err := os.WriteFile(file1, content, 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	if err := os.WriteFile(file2, content, 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	// Get hash of first file
	hash1, err := CalculateFileSHA256(file1)
	if err != nil {
		t.Fatalf("Failed to calculate hash for file1: %v", err)
	}

	// Compare with second file (should match)
	match, err := CompareFileSHA256(file2, hash1)
	if err != nil {
		t.Fatalf("Failed to compare hashes: %v", err)
	}
	if !match {
		t.Error("Expected files with identical content to have matching hashes")
	}

	// Modify second file
	if err := os.WriteFile(file2, []byte("different content"), 0644); err != nil {
		t.Fatalf("Failed to modify file2: %v", err)
	}

	// Compare again (should not match)
	match, err = CompareFileSHA256(file2, hash1)
	if err != nil {
		t.Fatalf("Failed to compare hashes after modification: %v", err)
	}
	if match {
		t.Error("Expected files with different content to have different hashes")
	}

	// Test with invalid hash
	match, err = CompareFileSHA256(file1, "invalid_hash")
	if err != nil {
		t.Fatalf("Failed to compare with invalid hash: %v", err)
	}
	if match {
		t.Error("Expected mismatch with invalid hash")
	}
}
