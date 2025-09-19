// Package tcc provides TCC (Transparency, Consent, Control) permission utilities.
package tcc

import (
	"github.com/tmc/misc/macgo/helpers/permissions"
)

// Permission represents a macOS system permission that can be requested.
// These correspond to TCC (Transparency, Consent, Control) permission types.
type Permission = permissions.Permission

// Core permissions covering 95% of use cases.
const (
	Camera     = permissions.Camera     // Camera access (com.apple.security.device.camera)
	Microphone = permissions.Microphone // Microphone access (com.apple.security.device.audio-input)
	Location   = permissions.Location   // Location services (com.apple.security.personal-information.location)
	Files      = permissions.Files      // File system access with user selection
	Network    = permissions.Network    // Network client/server access
	Sandbox    = permissions.Sandbox    // App sandbox with restricted file access
)

// ValidatePermissions checks if the provided permissions are valid and compatible.
func ValidatePermissions(perms []Permission) error {
	return permissions.ValidatePermissions(perms)
}

// ValidateAppGroups checks if app groups configuration is valid.
// App groups require sandbox permission to be enabled.
func ValidateAppGroups(groups []string, perms []Permission) error {
	return permissions.ValidateAppGroups(groups, perms)
}

// GetEntitlements returns the entitlement strings for the given permissions.
func GetEntitlements(perms []Permission) []string {
	return permissions.GetEntitlements(perms)
}

// RequiresTCC returns true if any of the permissions require TCC prompts.
func RequiresTCC(perms []Permission) bool {
	return permissions.RequiresTCC(perms)
}

// GetTCCServices returns the TCC service names for permissions that support tccutil reset.
func GetTCCServices(perms []Permission) []string {
	return permissions.GetTCCServices(perms)
}
