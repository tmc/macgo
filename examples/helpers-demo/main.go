package main

import (
	"fmt"

	"github.com/tmc/misc/macgo/helpers/bundle"
	"github.com/tmc/misc/macgo/helpers/codesign"
	"github.com/tmc/misc/macgo/helpers/permissions"
	"github.com/tmc/misc/macgo/helpers/teamid"
)

func main() {
	fmt.Println("macgo Helpers Package Demo")
	fmt.Println("==========================")

	// 1. Bundle ID and App Name Utilities
	fmt.Println("\n1. Bundle ID and App Name Utilities:")

	// Clean app name
	dirtyName := "My App/Name:With*Bad?Characters"
	cleanName := bundle.CleanAppName(dirtyName)
	fmt.Printf("   Clean app name: %q -> %q\n", dirtyName, cleanName)

	// Infer bundle ID
	bundleID := bundle.InferBundleID("MyApp")
	fmt.Printf("   Inferred bundle ID: %s\n", bundleID)

	// Validate bundle ID
	testBundleID := "com.example.myapp"
	if err := bundle.ValidateBundleID(testBundleID); err != nil {
		fmt.Printf("   Bundle ID validation failed: %v\n", err)
	} else {
		fmt.Printf("   Bundle ID %q is valid\n", testBundleID)
	}

	// Extract app name from path
	execPath := "/path/to/my-executable"
	appName := bundle.ExtractAppNameFromPath(execPath)
	fmt.Printf("   App name from path %q: %q\n", execPath, appName)

	// 2. Team ID Detection and Substitution
	fmt.Println("\n2. Team ID Detection and Substitution:")

	teamID, err := teamid.DetectTeamID()
	if err != nil {
		fmt.Printf("   Team ID detection failed: %v\n", err)
		fmt.Printf("   (This is normal if you don't have Developer ID certificates)\n")
	} else {
		fmt.Printf("   Detected Team ID: %s\n", teamID)

		// Validate the team ID
		if teamid.IsValidTeamID(teamID) {
			fmt.Printf("   Team ID format is valid\n")
		}
	}

	// Demonstrate team ID substitution
	appGroups := []string{
		"group.TEAMID.shared-data",
		"group.TEAMID.cache",
		"group.example.noreplace", // This one won't be changed
	}
	fmt.Printf("   App groups before substitution: %v\n", appGroups)

	detectedTeamID, substitutions, err := teamid.AutoSubstituteTeamIDInGroups(appGroups)
	if err != nil {
		fmt.Printf("   Auto substitution failed: %v\n", err)
	} else {
		fmt.Printf("   App groups after substitution: %v\n", appGroups)
		fmt.Printf("   Team ID used: %s\n", detectedTeamID)
		fmt.Printf("   Substitutions made: %d\n", substitutions)
	}

	// 3. Code Signing Utilities
	fmt.Println("\n3. Code Signing Utilities:")

	// Find Developer ID
	identity := codesign.FindDeveloperID()
	if identity == "" {
		fmt.Printf("   No Developer ID certificate found\n")
	} else {
		fmt.Printf("   Found Developer ID: %s\n", identity)

		// Extract team ID from certificate
		certTeamID := codesign.ExtractTeamIDFromCertificate(identity)
		if certTeamID != "" {
			fmt.Printf("   Team ID from certificate: %s\n", certTeamID)
		}
	}

	// List all available identities
	identities, err := codesign.ListAvailableIdentities()
	if err != nil {
		fmt.Printf("   Failed to list identities: %v\n", err)
	} else {
		fmt.Printf("   Available code signing identities (%d):\n", len(identities))
		for i, id := range identities {
			if i < 3 { // Show first 3 to avoid clutter
				fmt.Printf("     - %s\n", id)
			} else if i == 3 {
				fmt.Printf("     ... and %d more\n", len(identities)-3)
				break
			}
		}
	}

	// Check if Developer ID is available
	if codesign.HasDeveloperIDCertificate() {
		fmt.Printf("   Developer ID certificate is available\n")
	} else {
		fmt.Printf("   No Developer ID certificate available\n")
	}

	// 4. Permission Utilities
	fmt.Println("\n4. Permission Utilities:")

	// List all permissions
	allPerms := permissions.AllPermissions()
	fmt.Printf("   Available permissions (%d):\n", len(allPerms))
	for _, perm := range allPerms {
		fmt.Printf("     - %s: %s\n", permissions.PermissionToString(perm), permissions.PermissionDescription(perm))
	}

	// Test permission validation
	testPerms := []permissions.Permission{permissions.Camera, permissions.Microphone, permissions.Files}
	if err := permissions.ValidatePermissions(testPerms); err != nil {
		fmt.Printf("   Permission validation failed: %v\n", err)
	} else {
		fmt.Printf("   Permissions %v are valid\n", testPerms)
	}

	// Get entitlements for permissions
	entitlements := permissions.GetEntitlements(testPerms)
	fmt.Printf("   Entitlements for permissions: %v\n", entitlements)

	// Check if permissions require TCC
	if permissions.RequiresTCC(testPerms) {
		fmt.Printf("   These permissions require TCC dialogs\n")
	}

	// Get TCC services
	tccServices := permissions.GetTCCServices(testPerms)
	if len(tccServices) > 0 {
		fmt.Printf("   TCC services for reset: %v\n", tccServices)
	}

	// 5. App Groups Validation
	fmt.Println("\n5. App Groups Validation:")

	testAppGroups := []string{
		"group.com.example.shared",
		"group.com.example.cache",
	}
	sandboxPerms := []permissions.Permission{permissions.Sandbox} // App groups require sandbox

	if err := permissions.ValidateAppGroups(testAppGroups, sandboxPerms); err != nil {
		fmt.Printf("   App groups validation failed: %v\n", err)
	} else {
		fmt.Printf("   App groups %v are valid with sandbox permission\n", testAppGroups)
	}

	// Test without sandbox permission
	noSandboxPerms := []permissions.Permission{permissions.Camera}
	if err := permissions.ValidateAppGroups(testAppGroups, noSandboxPerms); err != nil {
		fmt.Printf("   App groups validation (without sandbox): %v\n", err)
	}

	fmt.Println("\nDemo completed! The helpers package provides direct access to macgo utilities.")
}
