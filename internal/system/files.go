// Package system provides internal system-level utilities for macgo.
package system

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CopyFile copies a file from src to dst.
// It handles error cases gracefully and preserves file permissions.
func CopyFile(src, dst string) error {
	if src == "" {
		return fmt.Errorf("source file path cannot be empty")
	}
	if dst == "" {
		return fmt.Errorf("destination file path cannot be empty")
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer func() { _ = srcFile.Close() }()

	// Get source file info for permissions
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get source file info: %w", err)
	}

	// Create destination directory if it doesn't exist
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer func() { _ = dstFile.Close() }()

	// Copy the file content
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// Preserve permissions
	if err := dstFile.Chmod(srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}

// IsInAppBundle checks if we're already running inside an app bundle.
// This is determined by checking if the executable path contains ".app/Contents/MacOS/".
func IsInAppBundle() bool {
	execPath, err := os.Executable()
	if err != nil {
		return false
	}
	return strings.Contains(execPath, ".app/Contents/MacOS/")
}

// SafeWriteFile writes data to a file safely by writing to a temporary file first,
// then moving it to the final location. This prevents partial writes in case of errors.
func SafeWriteFile(filename string, data []byte, perm os.FileMode) error {
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}

	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create temporary file in the same directory
	tmpFile, err := os.CreateTemp(dir, filepath.Base(filename)+".tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	tmpName := tmpFile.Name()

	// Clean up temp file on error
	defer func() {
		if tmpFile != nil {
			_ = tmpFile.Close()
			_ = os.Remove(tmpName)
		}
	}()

	// Write data to temp file
	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}

	// Set permissions
	if err := tmpFile.Chmod(perm); err != nil {
		return fmt.Errorf("failed to set temporary file permissions: %w", err)
	}

	// Close temp file
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}
	tmpFile = nil // Prevent cleanup

	// Atomically move temp file to final location
	if err := os.Rename(tmpName, filename); err != nil {
		_ = os.Remove(tmpName) // Clean up on error
		return fmt.Errorf("failed to move temporary file to final location: %w", err)
	}

	return nil
}

// FileExists checks if a file exists and is not a directory.
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// DirExists checks if a directory exists.
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// EnsureDir creates a directory and all parent directories if they don't exist.
func EnsureDir(path string, perm os.FileMode) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}
	return os.MkdirAll(path, perm)
}

// GetBundleExecutablePath constructs the path to the executable inside an app bundle.
func GetBundleExecutablePath(bundlePath, execName string) string {
	return filepath.Join(bundlePath, "Contents", "MacOS", execName)
}

// GetBundleContentsPath constructs the path to the Contents directory in an app bundle.
func GetBundleContentsPath(bundlePath string) string {
	return filepath.Join(bundlePath, "Contents")
}

// GetBundleInfoPlistPath constructs the path to the Info.plist in an app bundle.
func GetBundleInfoPlistPath(bundlePath string) string {
	return filepath.Join(bundlePath, "Contents", "Info.plist")
}

// GetBundleEntitlementsPath constructs the path to the entitlements.plist in an app bundle.
func GetBundleEntitlementsPath(bundlePath string) string {
	return filepath.Join(bundlePath, "Contents", "entitlements.plist")
}

// IsAppBundle checks if the given path appears to be an app bundle.
func IsAppBundle(path string) bool {
	if !strings.HasSuffix(path, ".app") {
		return false
	}
	contentsPath := GetBundleContentsPath(path)
	return DirExists(contentsPath)
}

// GetBundleID extracts the bundle identifier from an app bundle's Info.plist
func GetBundleID(bundlePath string) string {
	if bundlePath == "" || !strings.HasSuffix(bundlePath, ".app") {
		return ""
	}

	plistPath := filepath.Join(bundlePath, "Contents", "Info.plist")
	data, err := os.ReadFile(plistPath)
	if err != nil {
		return ""
	}

	// Simple extraction of CFBundleIdentifier
	// Look for the key and then the next string value
	lines := strings.Split(string(data), "\n")
	foundKey := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if foundKey && strings.HasPrefix(line, "<string>") && strings.HasSuffix(line, "</string>") {
			bundleID := strings.TrimPrefix(line, "<string>")
			bundleID = strings.TrimSuffix(bundleID, "</string>")
			return bundleID
		}
		if strings.Contains(line, "CFBundleIdentifier") {
			foundKey = true
		} else if foundKey && !strings.HasPrefix(line, "<string>") {
			// Key was found but next line isn't a string value
			foundKey = false
		}
	}

	return ""
}

// CalculateFileSHA256 calculates the SHA256 hash of a file.
// Returns the hexadecimal string representation of the hash.
func CalculateFileSHA256(filePath string) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer func() { _ = file.Close() }()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to read file %s for hashing: %w", filePath, err)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// CompareFileSHA256 compares the SHA256 hash of a file with an expected hash.
// Returns true if they match, false otherwise.
func CompareFileSHA256(filePath, expectedHash string) (bool, error) {
	actualHash, err := CalculateFileSHA256(filePath)
	if err != nil {
		return false, err
	}
	return actualHash == expectedHash, nil
}
