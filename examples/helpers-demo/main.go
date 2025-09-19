package main

import (
	"fmt"
	"os"

	"github.com/tmc/misc/macgo/helpers"
)

func main() {
	fmt.Println("macgo Helpers Package Demo")
	fmt.Println("==========================")

	// 1. Bundle ID and App Name Utilities
	fmt.Println("\n1. Bundle ID and App Name Utilities:")

	// Clean app name
	dirtyName := "My App/Name:With*Bad?Characters"
	cleanName := helpers.CleanAppName(dirtyName)
	fmt.Printf("   Clean app name: %q -> %q\n", dirtyName, cleanName)

	// Infer bundle ID
	bundleID := helpers.InferBundleID("MyApp")
	fmt.Printf("   Inferred bundle ID: %s\n", bundleID)

	// Validate bundle ID
	testBundleID := "com.example.myapp"
	if err := helpers.ValidateBundleID(testBundleID); err != nil {
		fmt.Printf("   Bundle ID validation failed: %v\n", err)
	} else {
		fmt.Printf("   Bundle ID %q is valid\n", testBundleID)
	}

	// Extract app name from path
	execPath := "/path/to/my-executable"
	appName := helpers.ExtractAppNameFromPath(execPath)
	fmt.Printf("   App name from path %q: %q\n", execPath, appName)

	// 2. Team ID Detection and Substitution
	fmt.Println("\n2. Team ID Detection and Substitution:")

	teamID, err := helpers.DetectTeamID()
	if err != nil {
		fmt.Printf("   Team ID detection failed: %v\n", err)
		fmt.Printf("   (This is normal if you don't have Developer ID certificates)\n")
	} else {
		fmt.Printf("   Detected Team ID: %s\n", teamID)

		// Validate the team ID
		if helpers.IsValidTeamID(teamID) {
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

	detectedTeamID, substitutions, err := helpers.AutoSubstituteTeamIDInGroups(appGroups)
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
	identity := helpers.FindDeveloperID()
	if identity == "" {
		fmt.Printf("   No Developer ID certificate found\n")
	} else {
		fmt.Printf("   Found Developer ID: %s\n", identity)

		// Extract team ID from certificate
		certTeamID := helpers.ExtractTeamIDFromCertificate(identity)
		if certTeamID != "" {
			fmt.Printf("   Team ID from certificate: %s\n", certTeamID)
		}
	}

	// List all available identities
	identities, err := helpers.ListAvailableIdentities()
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
	if helpers.HasDeveloperIDCertificate() {
		fmt.Printf("   Developer ID certificate is available\n")
	} else {
		fmt.Printf("   No Developer ID certificate available\n")
	}

	// 4. Permission Utilities
	fmt.Println("\n4. Permission Utilities:")

	// List all permissions
	allPerms := helpers.AllPermissions()
	fmt.Printf("   Available permissions (%d):\n", len(allPerms))
	for _, perm := range allPerms {
		fmt.Printf("     - %s: %s\n", helpers.PermissionToString(perm), helpers.PermissionDescription(perm))
	}

	// Test permission validation
	testPerms := []helpers.Permission{helpers.Camera, helpers.Microphone, helpers.Files}
	if err := helpers.ValidatePermissions(testPerms); err != nil {
		fmt.Printf("   Permission validation failed: %v\n", err)
	} else {
		fmt.Printf("   Permissions %v are valid\n", testPerms)
	}

	// Get entitlements for permissions
	entitlements := helpers.GetEntitlements(testPerms)
	fmt.Printf("   Entitlements for permissions: %v\n", entitlements)

	// Check if permissions require TCC
	if helpers.RequiresTCC(testPerms) {
		fmt.Printf("   These permissions require TCC dialogs\n")
	}

	// Get TCC services
	tccServices := helpers.GetTCCServices(testPerms)
	if len(tccServices) > 0 {
		fmt.Printf("   TCC services for reset: %v\n", tccServices)
	}

	// 5. App Groups Validation
	fmt.Println("\n5. App Groups Validation:")

	testAppGroups := []string{
		"group.com.example.shared",
		"group.com.example.cache",
	}
	sandboxPerms := []helpers.Permission{helpers.Sandbox} // App groups require sandbox

	if err := helpers.ValidateAppGroups(testAppGroups, sandboxPerms); err != nil {
		fmt.Printf("   App groups validation failed: %v\n", err)
	} else {
		fmt.Printf("   App groups %v are valid with sandbox permission\n", testAppGroups)
	}

	// Test without sandbox permission
	noSandboxPerms := []helpers.Permission{helpers.Camera}
	if err := helpers.ValidateAppGroups(testAppGroups, noSandboxPerms); err != nil {
		fmt.Printf("   App groups validation (without sandbox): %v\n", err)
	}

	fmt.Println("\nDemo completed! The helpers package provides direct access to macgo utilities.")
}