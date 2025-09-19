package plist

import (
	"fmt"
	"os"
	"strings"
)

// Permission represents a macOS system permission that can be requested.
// These correspond to TCC (Transparency, Consent, Control) permission types.
type Permission string

// Core permissions covering 95% of use cases.
const (
	Camera     Permission = "camera"     // Camera access (com.apple.security.device.camera)
	Microphone Permission = "microphone" // Microphone access (com.apple.security.device.audio-input)
	Location   Permission = "location"   // Location services (com.apple.security.personal-information.location)
	Files      Permission = "files"      // File system access with user selection
	Network    Permission = "network"    // Network client/server access
	Sandbox    Permission = "sandbox"    // App sandbox with restricted file access
)

// EntitlementsConfig holds configuration for generating entitlements.plist files.
type EntitlementsConfig struct {
	Permissions []Permission
	Custom      []string
	AppGroups   []string
}

// WriteEntitlements creates an entitlements.plist file at the specified path.
// It generates entitlements based on the provided permissions, custom entitlements, and app groups.
func WriteEntitlements(path string, cfg EntitlementsConfig) error {
	content := generateEntitlementsContent(cfg)

	// If no entitlements are specified, don't create an empty file
	if content == "" {
		return nil
	}

	return os.WriteFile(path, []byte(content), 0644)
}

// generateEntitlementsContent generates the XML content for an entitlements.plist file.
func generateEntitlementsContent(cfg EntitlementsConfig) string {
	var entries []string

	// Add standard permissions
	for _, perm := range cfg.Permissions {
		if entitlement := permissionToEntitlement(perm); entitlement != "" {
			entries = append(entries, xmlKeyBool(entitlement, true))
		}
	}

	// Add custom entitlements
	for _, custom := range cfg.Custom {
		entries = append(entries, xmlKeyBool(custom, true))
	}

	// Add app groups entitlements
	if len(cfg.AppGroups) > 0 {
		entries = append(entries, xmlKeyArray("com.apple.security.application-groups", cfg.AppGroups))
	}

	if len(entries) == 0 {
		return ""
	}

	dictContent := strings.Join(entries, "\n")
	return wrapPlist(wrapDict(dictContent))
}

// permissionToEntitlement maps a Permission to its corresponding entitlement key.
func permissionToEntitlement(perm Permission) string {
	switch perm {
	case Camera:
		return "com.apple.security.device.camera"
	case Microphone:
		return "com.apple.security.device.microphone"
	case Location:
		return "com.apple.security.personal-information.location"
	case Sandbox:
		return "com.apple.security.app-sandbox"
	case Files:
		return "com.apple.security.files.user-selected.read-only"
	case Network:
		return "com.apple.security.network.client"
	default:
		return ""
	}
}

// GetAvailablePermissions returns all available standard permissions.
func GetAvailablePermissions() []Permission {
	return []Permission{
		Camera,
		Microphone,
		Location,
		Files,
		Network,
		Sandbox,
	}
}

// PermissionDescription returns a human-readable description of a permission.
func PermissionDescription(perm Permission) string {
	switch perm {
	case Camera:
		return "Camera access for video capture and photo apps"
	case Microphone:
		return "Microphone access for audio recording and voice apps"
	case Location:
		return "Location services for GPS and location-aware features"
	case Files:
		return "User-selected file access with sandbox restrictions"
	case Network:
		return "Network client access for internet connectivity"
	case Sandbox:
		return "App sandbox isolation for enhanced security"
	default:
		return string(perm)
	}
}

// ValidatePermissions checks if all provided permissions are recognized.
func ValidatePermissions(permissions []Permission) error {
	available := make(map[Permission]bool)
	for _, p := range GetAvailablePermissions() {
		available[p] = true
	}

	for _, perm := range permissions {
		if !available[perm] {
			return fmt.Errorf("unknown permission: %s", perm)
		}
	}

	return nil
}

// ValidateAppGroups validates app group identifiers.
func ValidateAppGroups(appGroups []string) error {
	for _, group := range appGroups {
		if !strings.HasPrefix(group, "group.") {
			return fmt.Errorf("app group identifier must start with 'group.': %s", group)
		}
		if len(group) <= 6 { // "group." is 6 characters
			return fmt.Errorf("app group identifier too short: %s", group)
		}
	}
	return nil
}
