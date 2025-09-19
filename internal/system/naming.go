package system

import (
	"github.com/tmc/misc/macgo/helpers"
)

// CleanAppName removes problematic characters from app names.
// It replaces filesystem-problematic characters with hyphens and
// removes non-printable ASCII characters.
func CleanAppName(name string) string {
	return helpers.CleanAppName(name)
}

// InferBundleID creates a reasonable bundle ID from the Go module path and app name.
// It uses the module path from build info to create meaningful, unique bundle IDs
// that reflect the actual Go module and program name.
func InferBundleID(appName string) string {
	return helpers.InferBundleID(appName)
}

// ExtractAppNameFromPath extracts a reasonable app name from an executable path.
func ExtractAppNameFromPath(execPath string) string {
	return helpers.ExtractAppNameFromPath(execPath)
}

// ValidateBundleID checks if a bundle ID follows Apple's naming conventions.
func ValidateBundleID(bundleID string) error {
	return helpers.ValidateBundleID(bundleID)
}

// ValidateAppName checks if an app name is reasonable for macOS.
func ValidateAppName(name string) error {
	return helpers.ValidateAppName(name)
}

// LimitAppNameLength truncates an app name to a reasonable length,
// reserving space for the .app extension.
func LimitAppNameLength(name string, maxLength int) string {
	return helpers.LimitAppNameLength(name, maxLength)
}