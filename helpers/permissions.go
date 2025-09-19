package helpers

import (
	"fmt"
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

// EntitlementMapping maps permissions to their corresponding entitlements.
// These entitlements are added to the app bundle's entitlements.plist file
// to declare the app's permission requirements.
var EntitlementMapping = map[Permission][]string{
	Camera:     {"com.apple.security.device.camera"},
	Microphone: {"com.apple.security.device.microphone"},
	Location:   {"com.apple.security.personal-information.location"},
	Files:      {"com.apple.security.files.user-selected.read-only"},
	Network:    {"com.apple.security.network.client"},
	Sandbox:    {"com.apple.security.app-sandbox"},
}

// TCCServiceMapping maps permissions to their TCC service names for tccutil.
// These are used when resetting TCC permissions via command line tools.
var TCCServiceMapping = map[Permission]string{
	Camera:     "Camera",
	Microphone: "Microphone",
	Location:   "Location",
}

// PermissionDependencies defines which permissions require other permissions.
// Currently used for validating app groups which require sandbox permission.
var PermissionDependencies = map[Permission][]Permission{
	// App groups require sandbox to be enabled
	// This will be checked when app groups are specified in config
}

// ValidatePermissions checks if the provided permissions are valid and compatible.
// It verifies that all permissions are recognized and that any dependency
// requirements are satisfied.
//
// For example, certain features may require specific permissions to be enabled
// together for proper functionality.
func ValidatePermissions(perms []Permission) error {
	seen := make(map[Permission]bool)

	for _, perm := range perms {
		if seen[perm] {
			continue // Skip duplicates
		}
		seen[perm] = true

		// Check if permission is known
		if _, exists := EntitlementMapping[perm]; !exists {
			return fmt.Errorf("unknown permission: %s", perm)
		}

		// Check dependencies
		if deps, hasDeps := PermissionDependencies[perm]; hasDeps {
			for _, dep := range deps {
				if !seen[dep] {
					return fmt.Errorf("permission %s requires %s to be enabled", perm, dep)
				}
			}
		}
	}

	return nil
}

// ValidateAppGroups checks if app groups configuration is valid.
// App groups require sandbox permission to be enabled and must follow
// proper naming conventions.
//
// App group identifiers must:
//   - Start with "group."
//   - Be at least 7 characters long ("group." + identifier)
//   - Have sandbox permission enabled
func ValidateAppGroups(groups []string, perms []Permission) error {
	if len(groups) == 0 {
		return nil
	}

	// Check if sandbox is enabled
	hasSandbox := false
	for _, perm := range perms {
		if perm == Sandbox {
			hasSandbox = true
			break
		}
	}

	if !hasSandbox {
		return fmt.Errorf("app groups require sandbox permission to be enabled")
	}

	// Validate group ID format
	for _, group := range groups {
		if !strings.HasPrefix(group, "group.") {
			return fmt.Errorf("app group ID must start with 'group.': %s", group)
		}
		if len(group) < 7 { // "group." + at least one char
			return fmt.Errorf("app group ID too short: %s", group)
		}
	}

	return nil
}

// GetEntitlements returns the entitlement strings for the given permissions.
// These entitlements should be included in the app bundle's entitlements.plist
// file to declare the app's permission requirements to macOS.
//
// Duplicate entitlements are automatically removed from the result.
func GetEntitlements(perms []Permission) []string {
	var entitlements []string
	seen := make(map[string]bool)

	for _, perm := range perms {
		if ents, exists := EntitlementMapping[perm]; exists {
			for _, ent := range ents {
				if !seen[ent] {
					entitlements = append(entitlements, ent)
					seen[ent] = true
				}
			}
		}
	}

	return entitlements
}

// RequiresTCC returns true if any of the permissions require TCC prompts.
// TCC (Transparency, Consent, Control) prompts are the system dialogs that
// ask users to grant permission for camera, microphone, location, etc.
//
// This is useful for determining whether the app needs to be launched in
// a way that triggers proper TCC dialog presentation.
func RequiresTCC(perms []Permission) bool {
	for _, perm := range perms {
		switch perm {
		case Camera, Microphone, Location, Files:
			return true
		}
	}
	return false
}

// GetTCCServices returns the TCC service names for permissions that support tccutil reset.
// These service names can be used with the `tccutil reset` command to clear
// previously granted permissions for testing purposes.
//
// Note that not all permissions have corresponding TCC services that can be reset.
func GetTCCServices(perms []Permission) []string {
	var services []string
	seen := make(map[string]bool)

	for _, perm := range perms {
		if service, exists := TCCServiceMapping[perm]; exists {
			if !seen[service] {
				services = append(services, service)
				seen[service] = true
			}
		}
	}

	return services
}

// PermissionFromString converts a string to a Permission type.
// This is useful when parsing permission names from configuration files
// or command line arguments.
//
// Returns the Permission and a boolean indicating whether the conversion
// was successful (i.e., whether the string represents a valid permission).
func PermissionFromString(s string) (Permission, bool) {
	perm := Permission(s)
	_, exists := EntitlementMapping[perm]
	return perm, exists
}

// PermissionToString converts a Permission to its string representation.
// This is useful for serialization and debugging.
func PermissionToString(perm Permission) string {
	return string(perm)
}

// AllPermissions returns a slice of all available permissions.
// This is useful for documentation, testing, or building UI that allows
// users to select from available permissions.
func AllPermissions() []Permission {
	var perms []Permission
	for perm := range EntitlementMapping {
		perms = append(perms, perm)
	}
	return perms
}

// PermissionDescription returns a human-readable description of the permission.
// These descriptions explain what each permission grants access to.
func PermissionDescription(perm Permission) string {
	descriptions := map[Permission]string{
		Camera:     "Access to camera for photo and video capture",
		Microphone: "Access to microphone for audio recording",
		Location:   "Access to device location services",
		Files:      "Access to user-selected files and folders",
		Network:    "Network access for client connections",
		Sandbox:    "App sandbox with restricted file system access",
	}
	if desc, exists := descriptions[perm]; exists {
		return desc
	}
	return "Unknown permission"
}