// Package security provides security validation and path sanitization functionality.
// This package implements security-critical operations with a focus on preventing
// directory traversal attacks and ensuring safe file operations.
package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tmc/misc/macgo"
)

// Validator implements the PathValidator interface.
// It provides secure path validation and sanitization operations.
type Validator struct{}

// NewValidator creates a new security Validator.
func NewValidator() *Validator {
	return &Validator{}
}

// Validate checks if a path is safe to use.
// It performs various security checks to prevent directory traversal and other attacks.
func (v *Validator) Validate(path string) error {
	if path == "" {
		return fmt.Errorf("security: path cannot be empty")
	}

	// Check for directory traversal attempts
	if strings.Contains(path, "..") {
		return fmt.Errorf("security: path contains directory traversal: %s", path)
	}

	// Check for null bytes (can be used to bypass security checks)
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("security: path contains null byte: %s", path)
	}

	// Check for suspicious characters
	suspicious := []string{"|", "&", ";", "$", "`", "\"", "'", "\\", "<", ">"}
	for _, char := range suspicious {
		if strings.Contains(path, char) {
			return fmt.Errorf("security: path contains suspicious character '%s': %s", char, path)
		}
	}

	// Ensure path is within reasonable length limits
	if len(path) > 4096 {
		return fmt.Errorf("security: path too long (%d bytes): %s", len(path), path)
	}

	// Check if path exists and is accessible
	if _, err := os.Stat(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("security: path not accessible: %w", err)
	}

	return nil
}

// Sanitize cleans and validates a path, returning the sanitized version.
func (v *Validator) Sanitize(path string) (string, error) {
	// Use the extracted sanitizePath function
	cleaned, err := v.sanitizePath(path)
	if err != nil {
		return "", err
	}

	// Convert to absolute path to avoid ambiguity
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return "", fmt.Errorf("security: cannot resolve absolute path: %w", err)
	}

	// Final validation of the absolute path
	if err := v.Validate(abs); err != nil {
		return "", fmt.Errorf("security: sanitized path failed validation: %w", err)
	}

	return abs, nil
}

// SecureJoin joins path elements in a secure way, preventing directory traversal.
// This implementation is based on the original secureJoin from bundle.go.
func (v *Validator) SecureJoin(base string, elements ...string) (string, error) {
	// Validate base path using sanitize
	cleanBase, err := v.Sanitize(base)
	if err != nil {
		return "", fmt.Errorf("security: invalid base path: %w", err)
	}

	// Validate and clean each element
	cleanElems := make([]string, len(elements))
	for i, elem := range elements {
		// Don't allow absolute paths in elements
		if filepath.IsAbs(elem) {
			return "", fmt.Errorf("security: absolute path not allowed in element: %s", elem)
		}

		clean, err := v.sanitizePath(elem)
		if err != nil {
			return "", fmt.Errorf("security: invalid path element %s: %w", elem, err)
		}
		cleanElems[i] = clean
	}

	// Join all components
	result := filepath.Join(append([]string{cleanBase}, cleanElems...)...)

	// Final validation of the result
	final, err := v.Sanitize(result)
	if err != nil {
		return "", fmt.Errorf("security: final path validation failed: %w", err)
	}

	return final, nil
}

// isWithinDirectory checks if a path is within a given directory.
func (v *Validator) isWithinDirectory(dir, path string) (bool, error) {
	// Make both paths absolute
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false, fmt.Errorf("security: cannot resolve directory path: %w", err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return false, fmt.Errorf("security: cannot resolve file path: %w", err)
	}

	// Check if path starts with directory
	rel, err := filepath.Rel(absDir, absPath)
	if err != nil {
		return false, fmt.Errorf("security: cannot compute relative path: %w", err)
	}

	// If the relative path starts with "..", it's outside the directory
	return !strings.HasPrefix(rel, ".."), nil
}

// sanitizePath validates and cleans a file path.
// This function is extracted from the original bundle.go for better organization.
func (v *Validator) sanitizePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("security: path cannot be empty")
	}

	// Clean the path to resolve any .. or . elements
	cleaned := filepath.Clean(path)

	// Check for path traversal attempts
	if strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("security: path traversal detected in: %s", path)
	}

	// Check for absolute paths that could escape intended boundaries
	if filepath.IsAbs(cleaned) && !v.isAllowedAbsolutePath(cleaned) {
		return "", fmt.Errorf("security: absolute path not allowed: %s", path)
	}

	// Prevent null bytes and other dangerous characters
	if strings.ContainsAny(cleaned, "\x00\r\n") {
		return "", fmt.Errorf("security: invalid characters in path: %s", path)
	}

	return cleaned, nil
}

// isAllowedAbsolutePath checks if an absolute path is within allowed directories.
func (v *Validator) isAllowedAbsolutePath(path string) bool {
	allowedPrefixes := []string{
		"/tmp/",
		"/var/folders/", // macOS temp directories
		os.TempDir(),
	}

	// Allow GOPATH and its subdirectories
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		allowedPrefixes = append(allowedPrefixes, gopath)
	}

	// Allow user home directory and its subdirectories
	if home, err := os.UserHomeDir(); err == nil {
		allowedPrefixes = append(allowedPrefixes, home)
	}

	// Allow standard development directories and system binaries
	allowedPrefixes = append(allowedPrefixes, "/usr/local/", "/opt/", "/usr/bin/", "/bin/", "/System/")

	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

// validateExecutablePath validates that an executable path is safe to use.
// This is the original validateExecutablePath function from bundle.go.
func (v *Validator) validateExecutablePath(execPath string) error {
	if execPath == "" {
		return fmt.Errorf("security: executable path cannot be empty")
	}

	// Clean and validate the path
	_, err := v.Sanitize(execPath)
	if err != nil {
		return fmt.Errorf("security: invalid executable path: %w", err)
	}

	// Check if the file exists and is executable
	info, err := os.Stat(execPath)
	if err != nil {
		return fmt.Errorf("security: executable not accessible: %w", err)
	}

	// Ensure it's a regular file
	if !info.Mode().IsRegular() {
		return fmt.Errorf("security: executable path is not a regular file: %s", execPath)
	}

	return nil
}

// ValidateExecutablePath provides public access to executable validation.
func (v *Validator) ValidateExecutablePath(execPath string) error {
	return v.validateExecutablePath(execPath)
}

// Compile-time check that Validator implements PathValidator
var _ macgo.PathValidator = (*Validator)(nil)
