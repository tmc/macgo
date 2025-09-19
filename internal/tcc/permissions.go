// Package tcc provides TCC (Transparency, Consent, Control) permission utilities.
package tcc

import (
	"github.com/tmc/misc/macgo/helpers"
)

// Permission represents a macOS system permission that can be requested.
// These correspond to TCC (Transparency, Consent, Control) permission types.
type Permission = helpers.Permission

// Core permissions covering 95% of use cases.
const (
	Camera     = helpers.Camera     // Camera access (com.apple.security.device.camera)
	Microphone = helpers.Microphone // Microphone access (com.apple.security.device.audio-input)
	Location   = helpers.Location   // Location services (com.apple.security.personal-information.location)
	Files      = helpers.Files      // File system access with user selection
	Network    = helpers.Network    // Network client/server access
	Sandbox    = helpers.Sandbox    // App sandbox with restricted file access
)

// ValidatePermissions checks if the provided permissions are valid and compatible.
func ValidatePermissions(perms []Permission) error {
	return helpers.ValidatePermissions(perms)
}

// ValidateAppGroups checks if app groups configuration is valid.
// App groups require sandbox permission to be enabled.
func ValidateAppGroups(groups []string, perms []Permission) error {
	return helpers.ValidateAppGroups(groups, perms)
}

// GetEntitlements returns the entitlement strings for the given permissions.
func GetEntitlements(perms []Permission) []string {
	return helpers.GetEntitlements(perms)
}

// RequiresTCC returns true if any of the permissions require TCC prompts.
func RequiresTCC(perms []Permission) bool {
	return helpers.RequiresTCC(perms)
}

// GetTCCServices returns the TCC service names for permissions that support tccutil reset.
func GetTCCServices(perms []Permission) []string {
	return helpers.GetTCCServices(perms)
}