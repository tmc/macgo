package bundle

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"
)

// CleanAppName removes problematic characters from app names.
// It replaces filesystem-problematic characters with hyphens and
// removes non-printable ASCII characters.
//
// Environment variable MACGO_APP_NAME_PREFIX can be used to force
// a prefix on all app names. This is useful for development or
// organizational requirements.
//
// This function is useful for sanitizing user-provided app names
// before creating app bundles.
func CleanAppName(name string) string {
	if name == "" {
		return ""
	}

	// Apply prefix from environment variable if set
	if prefix := os.Getenv("MACGO_APP_NAME_PREFIX"); prefix != "" {
		name = prefix + name
	}

	// Replace problematic filesystem characters with hyphens
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	name = strings.ReplaceAll(name, ":", "-")
	name = strings.ReplaceAll(name, "*", "-")
	name = strings.ReplaceAll(name, "?", "-")
	name = strings.ReplaceAll(name, "\"", "-")
	name = strings.ReplaceAll(name, "<", "-")
	name = strings.ReplaceAll(name, ">", "-")
	name = strings.ReplaceAll(name, "|", "-")

	// Keep only printable ASCII characters
	var result strings.Builder
	for _, r := range name {
		if r >= 32 && r < 127 {
			result.WriteRune(r)
		}
	}

	cleaned := result.String()

	// Remove leading/trailing hyphens and spaces
	cleaned = strings.Trim(cleaned, "- ")

	// Collapse multiple consecutive hyphens into single hyphens
	re := regexp.MustCompile("-+")
	cleaned = re.ReplaceAllString(cleaned, "-")

	return cleaned
}

// InferBundleID creates a reasonable bundle ID from the Go module path and app name.
// It uses the module path from build info to create meaningful, unique bundle IDs
// that reflect the actual Go module and program name.
//
// Environment variable MACGO_BUNDLE_ID_PREFIX can be used to force a prefix
// on all bundle IDs. This is useful for development or organizational requirements.
//
// Examples:
//   - github.com/user/repo + "myapp" -> com.github.user.repo.myapp
//   - example.com/project + "tool" -> com.example.project.tool
//   - local/project + "app" -> local.project.app
//   - With MACGO_BUNDLE_ID_PREFIX="dev": dev.com.github.user.repo.myapp
//
// If no module information is available, it creates a fallback bundle ID
// based on the user's environment.
func InferBundleID(appName string) string {
	if appName == "" {
		appName = "app"
	}

	var bundleID string

	// Try to get module path from build info
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Path != "" {
		modulePath := info.Main.Path

		// Convert module path to reverse DNS format
		bundleID = modulePathToBundleID(modulePath, appName)
	} else {
		// If no module info, use a more generic but still meaningful format
		// based on the current working directory or executable location
		bundleID = inferFallbackBundleID(appName)
	}

	// Apply prefix from environment variable if set
	if prefix := os.Getenv("MACGO_BUNDLE_ID_PREFIX"); prefix != "" {
		// If the prefix doesn't end with a dot, add one
		if !strings.HasSuffix(prefix, ".") {
			prefix = prefix + "."
		}
		bundleID = prefix + bundleID
	}

	return SanitizeBundleID(bundleID)
}

// modulePathToBundleID converts a Go module path to a bundle ID format.
// Examples:
//   github.com/user/repo -> com.github.user.repo.appname
//   example.com/project -> com.example.project.appname
//   local/project -> local.project.appname
func modulePathToBundleID(modulePath, appName string) string {
	// Handle common patterns
	parts := strings.Split(modulePath, "/")

	// Reverse the domain parts to follow reverse DNS convention
	var bundleParts []string

	if len(parts) >= 1 {
		// Handle domain-like first part (github.com, gitlab.com, etc.)
		domain := parts[0]
		if strings.Contains(domain, ".") {
			// Reverse domain parts: github.com -> com.github
			domainParts := strings.Split(domain, ".")
			for i := len(domainParts) - 1; i >= 0; i-- {
				bundleParts = append(bundleParts, domainParts[i])
			}
		} else {
			// Simple domain without dots (e.g., "local")
			bundleParts = append(bundleParts, domain)
		}

		// Add the rest of the path components
		bundleParts = append(bundleParts, parts[1:]...)
	}

	// Add the app name as the final component
	bundleParts = append(bundleParts, appName)

	return strings.Join(bundleParts, ".")
}

// inferFallbackBundleID creates a bundle ID when no module info is available.
// It creates a reasonable default based on the user's system.
func inferFallbackBundleID(appName string) string {
	// Ensure we have a valid app name
	if appName == "" {
		appName = "app"
	}

	// Try to get a reasonable domain from the environment or system
	// This provides a better default than "com.macgo"

	// Check for common environment variables that might indicate the user/organization
	if user := getEnvironmentIdentifier(); user != "" {
		return fmt.Sprintf("dev.%s.%s", sanitizeComponent(user), appName)
	}

	// Final fallback - use a generic but descriptive format
	return fmt.Sprintf("local.app.%s", appName)
}

// getEnvironmentIdentifier tries to get a reasonable identifier from the environment.
func getEnvironmentIdentifier() string {
	// Try various environment variables that might give us a good identifier
	identifiers := []string{
		"LOGNAME",  // Unix username
		"USER",     // Unix username
		"USERNAME", // Windows username
	}

	for _, env := range identifiers {
		if envValue := os.Getenv(env); envValue != "" {
			// Clean up the environment value
			value := strings.TrimSpace(strings.ToLower(envValue))
			value = strings.ReplaceAll(value, " ", "")
			value = filepath.Base(value) // Remove any path components

			if value != "" && value != "root" && value != "admin" {
				return value
			}
		}
	}

	return ""
}

// sanitizeComponent cleans a single bundle ID component.
func sanitizeComponent(component string) string {
	// Convert to lowercase and replace invalid characters
	component = strings.ToLower(component)

	var result strings.Builder
	for _, r := range component {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result.WriteRune(r)
		} else {
			result.WriteRune('-')
		}
	}

	cleaned := result.String()
	cleaned = strings.Trim(cleaned, "-")

	// Ensure it doesn't start with a number
	if len(cleaned) > 0 && cleaned[0] >= '0' && cleaned[0] <= '9' {
		cleaned = "user" + cleaned
	}

	if cleaned == "" {
		return "user"
	}

	return cleaned
}

// SanitizeBundleID ensures the bundle ID follows proper conventions.
// Bundle IDs should only contain alphanumeric characters, hyphens, and periods.
func SanitizeBundleID(bundleID string) string {
	if bundleID == "" {
		return "app"
	}

	// Convert to lowercase
	bundleID = strings.ToLower(bundleID)

	// Replace invalid characters with hyphens
	var result strings.Builder
	for _, r := range bundleID {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '-' {
			result.WriteRune(r)
		} else {
			result.WriteRune('-')
		}
	}

	cleaned := result.String()

	// Remove leading/trailing dots and hyphens
	cleaned = strings.Trim(cleaned, ".-")

	// Ensure it doesn't start with a number
	if len(cleaned) > 0 && cleaned[0] >= '0' && cleaned[0] <= '9' {
		cleaned = "app" + cleaned
	}

	// Collapse multiple consecutive dots or hyphens
	re := regexp.MustCompile(`[.-]+`)
	cleaned = re.ReplaceAllStringFunc(cleaned, func(match string) string {
		if strings.Contains(match, ".") {
			return "."
		}
		return "-"
	})

	if cleaned == "" {
		return "app"
	}

	return cleaned
}

// ExtractAppNameFromPath extracts a reasonable app name from an executable path.
// This function removes file extensions and cleans up the filename to create
// a suitable app name.
func ExtractAppNameFromPath(execPath string) string {
	if execPath == "" {
		return ""
	}

	// Get the base filename
	base := filepath.Base(execPath)

	// Remove common executable extensions, but be conservative
	// Only remove extensions that are clearly file extensions
	ext := filepath.Ext(base)
	if len(ext) > 0 {
		// Only remove extensions that look like real file extensions
		// (start with a dot and contain only alphanumeric characters)
		validExt := true
		if len(ext) < 2 || ext[0] != '.' {
			validExt = false
		} else {
			for _, r := range ext[1:] {
				if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
					validExt = false
					break
				}
			}
		}

		if validExt {
			base = strings.TrimSuffix(base, ext)
		}
	}

	return base
}

// ValidateBundleID checks if a bundle ID follows Apple's naming conventions.
// Bundle IDs must use reverse DNS notation and contain only valid characters.
//
// Requirements:
//   - Must contain at least one dot (reverse DNS format)
//   - Can only contain alphanumeric characters, dots, and hyphens
//   - Cannot start or end with dots or hyphens
//   - Cannot contain consecutive dots or hyphens
//   - Components cannot start with numbers
func ValidateBundleID(bundleID string) error {
	if bundleID == "" {
		return fmt.Errorf("bundle ID cannot be empty")
	}

	// Must contain at least one dot
	if !strings.Contains(bundleID, ".") {
		return fmt.Errorf("bundle ID must contain at least one dot (reverse DNS format)")
	}

	// Check for valid characters
	re := regexp.MustCompile(`^[a-zA-Z0-9.-]+$`)
	if !re.MatchString(bundleID) {
		return fmt.Errorf("bundle ID can only contain alphanumeric characters, dots, and hyphens")
	}

	// Check that it doesn't start or end with dot or hyphen
	if strings.HasPrefix(bundleID, ".") || strings.HasPrefix(bundleID, "-") ||
		strings.HasSuffix(bundleID, ".") || strings.HasSuffix(bundleID, "-") {
		return fmt.Errorf("bundle ID cannot start or end with dots or hyphens")
	}

	// Check for consecutive dots or hyphens
	if strings.Contains(bundleID, "..") || strings.Contains(bundleID, "--") {
		return fmt.Errorf("bundle ID cannot contain consecutive dots or hyphens")
	}

	// Split by dots and validate each component
	components := strings.Split(bundleID, ".")
	for _, component := range components {
		if component == "" {
			return fmt.Errorf("bundle ID cannot have empty components")
		}

		// Component cannot start with a number
		if len(component) > 0 && component[0] >= '0' && component[0] <= '9' {
			return fmt.Errorf("bundle ID component cannot start with a number: %s", component)
		}
	}

	return nil
}

// ValidateAppName checks if an app name is reasonable for macOS.
// App names should not contain filesystem-problematic characters
// and should be of reasonable length.
func ValidateAppName(name string) error {
	if name == "" {
		return fmt.Errorf("app name cannot be empty")
	}

	if len(name) > 255 {
		return fmt.Errorf("app name too long (max 255 characters)")
	}

	// Check for problematic characters
	problematic := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range problematic {
		if strings.Contains(name, char) {
			return fmt.Errorf("app name cannot contain character: %s", char)
		}
	}

	return nil
}

// LimitAppNameLength truncates an app name to a reasonable length,
// reserving space for the .app extension.
func LimitAppNameLength(name string, maxLength int) string {
	if len(name) <= maxLength {
		return name
	}
	return name[:maxLength]
}