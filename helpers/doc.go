// Package helpers provides utility functions for macgo organized into focused subpackages.
//
// The helpers packages expose key functionality from macgo's internal packages,
// making it easy for users to work with macOS app bundle creation, code signing,
// team ID detection, permission management, and validation utilities.
//
// # Subpackage Organization
//
// The helpers functionality is organized into focused subpackages:
//
//   - bundle: App bundle creation, naming, and validation utilities
//   - codesign: Code signing certificate management and validation
//   - permissions: Permission validation and entitlement management
//   - teamid: Apple Developer Team ID detection and substitution
//
// # Bundle Package (github.com/tmc/misc/macgo/helpers/bundle)
//
// Functions for working with app bundles and bundle identifiers:
//
//	import "github.com/tmc/misc/macgo/helpers/bundle"
//
//	// Infer bundle ID from Go module information
//	bundleID := bundle.InferBundleID("MyApp")
//
//	// Validate bundle ID format
//	if err := bundle.ValidateBundleID(bundleID); err != nil {
//		log.Fatal(err)
//	}
//
//	// Clean problematic characters from app names
//	cleanName := bundle.CleanAppName("My App/With:Bad*Characters")
//
// # TeamID Package (github.com/tmc/misc/macgo/helpers/teamid)
//
// Functions for detecting and managing Apple Developer Team IDs:
//
//	import "github.com/tmc/misc/macgo/helpers/teamid"
//
//	// Detect team ID from certificates
//	teamID, err := teamid.DetectTeamID()
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Team ID: %s\n", teamID)
//
//	// Substitute TEAMID placeholders in app groups
//	appGroups := []string{"group.TEAMID.shared", "group.TEAMID.cache"}
//	substitutions := teamid.SubstituteTeamIDInGroups(appGroups, teamID)
//	fmt.Printf("Made %d substitutions\n", substitutions)
//
// # CodeSign Package (github.com/tmc/misc/macgo/helpers/codesign)
//
// Functions for working with code signing certificates and identities:
//
//	import "github.com/tmc/misc/macgo/helpers/codesign"
//
//	// Find Developer ID certificate
//	identity := codesign.FindDeveloperID()
//	if identity != "" {
//		fmt.Printf("Found: %s\n", identity)
//	}
//
//	// List all available identities
//	identities, err := codesign.ListAvailableIdentities()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Verify bundle signature
//	if err := codesign.VerifySignature("/path/to/app.app"); err != nil {
//		log.Fatal(err)
//	}
//
// # Permissions Package (github.com/tmc/misc/macgo/helpers/permissions)
//
// Functions for working with macOS permissions and entitlements:
//
//	import "github.com/tmc/misc/macgo/helpers/permissions"
//
//	// Validate permissions
//	perms := []permissions.Permission{permissions.Camera, permissions.Microphone}
//	if err := permissions.ValidatePermissions(perms); err != nil {
//		log.Fatal(err)
//	}
//
//	// Get entitlements for permissions
//	entitlements := permissions.GetEntitlements(perms)
//	fmt.Printf("Entitlements: %v\n", entitlements)
//
//	// Check if permissions require TCC dialogs
//	if permissions.RequiresTCC(perms) {
//		fmt.Println("These permissions require user consent")
//	}
//
// # Complete Example
//
//	import (
//		"fmt"
//		"log"
//
//		"github.com/tmc/misc/macgo/helpers/bundle"
//		"github.com/tmc/misc/macgo/helpers/codesign"
//		"github.com/tmc/misc/macgo/helpers/permissions"
//		"github.com/tmc/misc/macgo/helpers/teamid"
//	)
//
//	func main() {
//		// Clean app name and generate bundle ID
//		cleanName := bundle.CleanAppName("My App")
//		bundleID := bundle.InferBundleID(cleanName)
//
//		// Detect team ID
//		teamID, _ := teamid.DetectTeamID()
//		if teamid.IsValidTeamID(teamID) {
//			fmt.Printf("Team ID: %s\n", teamID)
//		}
//
//		// Check for Developer ID certificate
//		if codesign.HasDeveloperIDCertificate() {
//			identity := codesign.FindDeveloperID()
//			fmt.Printf("Signing identity: %s\n", identity)
//		}
//
//		// Validate permissions
//		perms := []permissions.Permission{permissions.Camera}
//		if err := permissions.ValidatePermissions(perms); err != nil {
//			log.Fatal(err)
//		}
//	}
//
// This package structure allows for more focused imports and clearer code organization
// while maintaining backward compatibility through the internal package wrappers.
package helpers
