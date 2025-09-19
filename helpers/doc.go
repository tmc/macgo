// Package helpers provides utility functions for macgo that are useful for external users.
//
// The helpers package exposes key functionality from macgo's internal packages,
// making it easy for users to work with macOS app bundle creation, code signing,
// team ID detection, permission management, and validation utilities.
//
// # Team ID Detection and Management
//
// The package provides functions for automatically detecting Apple Developer Team IDs
// from installed code signing certificates and substituting them in app group identifiers:
//
//	teamID, err := helpers.DetectTeamID()
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Team ID: %s\n", teamID)
//
//	// Substitute TEAMID placeholders in app groups
//	appGroups := []string{"group.TEAMID.shared", "group.TEAMID.cache"}
//	substitutions := helpers.SubstituteTeamIDInGroups(appGroups, teamID)
//	fmt.Printf("Made %d substitutions\n", substitutions)
//
// # Bundle ID and App Name Utilities
//
// Functions for creating, validating, and cleaning bundle identifiers and app names:
//
//	// Infer bundle ID from Go module information
//	bundleID := helpers.InferBundleID("MyApp")
//
//	// Validate bundle ID format
//	if err := helpers.ValidateBundleID(bundleID); err != nil {
//		log.Fatal(err)
//	}
//
//	// Clean problematic characters from app names
//	cleanName := helpers.CleanAppName("My App/With:Bad*Characters")
//
// # Code Signing Utilities
//
// Functions for working with code signing certificates and identities:
//
//	// Find Developer ID certificate
//	identity := helpers.FindDeveloperID()
//	if identity != "" {
//		fmt.Printf("Found: %s\n", identity)
//	}
//
//	// List all available identities
//	identities, err := helpers.ListAvailableIdentities()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Verify bundle signature
//	if err := helpers.VerifySignature("/path/to/app.app"); err != nil {
//		log.Fatal(err)
//	}
//
// # Permission Management
//
// Functions for working with macOS permissions and entitlements:
//
//	// Validate permissions
//	perms := []helpers.Permission{helpers.Camera, helpers.Microphone}
//	if err := helpers.ValidatePermissions(perms); err != nil {
//		log.Fatal(err)
//	}
//
//	// Get entitlements for permissions
//	entitlements := helpers.GetEntitlements(perms)
//	fmt.Printf("Entitlements: %v\n", entitlements)
//
//	// Check if permissions require TCC dialogs
//	if helpers.RequiresTCC(perms) {
//		fmt.Println("These permissions require user consent")
//	}
//
// # App Groups Validation
//
// Functions for validating app group configurations:
//
//	appGroups := []string{"group.com.example.shared"}
//	sandboxPerms := []helpers.Permission{helpers.Sandbox}
//
//	if err := helpers.ValidateAppGroups(appGroups, sandboxPerms); err != nil {
//		log.Fatal(err)
//	}
//
// This package is designed to be used alongside the main macgo package
// for applications that need more direct control over bundle creation
// and permission management processes.
package helpers