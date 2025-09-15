package bundle

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/tmc/misc/macgo"
)

// createDirectoryStructure creates the standard macOS app bundle directory structure.
func (c *Creator) createDirectoryStructure(bundlePath string) error {
	dirs := []string{
		filepath.Join(bundlePath, "Contents"),
		filepath.Join(bundlePath, "Contents", "MacOS"),
		filepath.Join(bundlePath, "Contents", "Resources"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	return nil
}

// copyExecutable copies the executable to the bundle and sets appropriate permissions.
func (c *Creator) copyExecutable(execPath, bundlePath string, cfg *macgo.Config) error {
	appName := c.getApplicationName(cfg, execPath)
	destPath := filepath.Join(bundlePath, "Contents", "MacOS", appName)

	// Open source file
	src, err := os.Open(execPath)
	if err != nil {
		return fmt.Errorf("open source executable: %w", err)
	}
	defer src.Close()

	// Create destination file
	dst, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create destination executable: %w", err)
	}
	defer dst.Close()

	// Copy file content
	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copy executable content: %w", err)
	}

	// Set executable permissions
	if err := os.Chmod(destPath, 0755); err != nil {
		return fmt.Errorf("set executable permissions: %w", err)
	}

	return nil
}

// copyFile copies a file from src to dst with the given permissions.
func copyFile(src, dst string, perm os.FileMode) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source file %s: %w", src, err)
	}
	defer sourceFile.Close()

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create destination file %s: %w", dst, err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("copy file content: %w", err)
	}

	if err := os.Chmod(dst, perm); err != nil {
		return fmt.Errorf("set file permissions: %w", err)
	}

	return nil
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// removeFile removes a file if it exists.
func removeFile(path string) error {
	if !fileExists(path) {
		return nil
	}
	return os.Remove(path)
}

// ensureDir ensures a directory exists, creating it if necessary.
func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// calculateFileChecksum calculates SHA256 checksum of a file.
// This function is extracted from the original bundle.go checksum function.
func calculateFileChecksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file for checksum: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("calculate checksum: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// copyFileSimple copies a file from src to dst.
// This function is extracted from the original bundle.go copyFile function.
func copyFileSimple(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read source file: %w", err)
	}
	return os.WriteFile(dst, data, 0755)
}
