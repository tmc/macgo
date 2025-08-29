// Using go:embed with the LoadEntitlementsFromJSON API
// This example shows two ways to configure entitlements:
// 1. Using the entitlements package with semantic functions
// 2. Using JSON configuration with go:embed
// Run with: MACGO_DEBUG=1 go run main.go
package main

import (
	"embed"
	"fmt"
	"log"
	"os"

	// Import both macgo and entitlements packages
	"github.com/tmc/misc/macgo"
	"github.com/tmc/misc/macgo/entitlements"
)

// Define custom entitlements in a JSON file and embed it
//
//go:embed entitlements.json
var entitlementsData []byte

// Embed using the embed.FS type for multiple files
//
//go:embed *.json
var entitlementsFS embed.FS

func init() {
	// This example demonstrates two approaches to setting entitlements:
	
	// Approach 1: Using entitlements package (recommended for most use cases)
	fmt.Println("Setting up entitlements using the entitlements package...")
	entitlements.SetAppSandbox()
	entitlements.SetCamera()
	entitlements.SetMic()
	entitlements.SetLocation()
	entitlements.SetPhotos()
	entitlements.SetNetworkClient()
	entitlements.SetNetworkServer()
	entitlements.SetVirtualization()
	
	// Approach 2: Using go:embed with JSON configuration
	// This approach is useful when you want to maintain entitlements in a separate file
	fmt.Println("Loading additional entitlements from embedded JSON data...")
	if err := macgo.LoadEntitlementsFromJSON(entitlementsData); err != nil {
		log.Fatalf("Failed to load entitlements: %v", err)
	}

	// Additional configuration
	macgo.SetAppName("MacGoEmbed")
	macgo.SetBundleID("com.example.macgo.embed")
	macgo.EnableDebug() // Enable debug output
	
	// Start macgo
	macgo.Start()
}

func main() {
	fmt.Println("\nMacGo Entitlements Configuration Example")
	fmt.Println("========================================")
	fmt.Println("This example demonstrates two methods for configuring entitlements:")
	fmt.Println()
	fmt.Println("Method 1: entitlements package (recommended)")
	fmt.Println("- More readable and type-safe")
	fmt.Println("- Semantic function names like SetCamera(), SetMic()")
	fmt.Println("- Compile-time validation")
	fmt.Println()
	fmt.Println("Method 2: JSON configuration with go:embed")
	fmt.Println("- Keep entitlements configuration in a separate JSON file")
	fmt.Println("- Embed the configuration directly in the binary")
	fmt.Println("- No code changes needed to update entitlements")
	fmt.Println("- Configuration can be selected at build time")
	fmt.Println()

	// Check if we're running in an app bundle
	if macgo.IsInAppBundle() {
		fmt.Println("✓ Running inside app bundle")
	} else {
		fmt.Println("ℹ Running outside app bundle - will relaunch")
	}

	// Show what permissions are configured
	fmt.Println("\nConfigured entitlements:")
	fmt.Println("- App Sandbox")
	fmt.Println("- Network client/server")
	fmt.Println("- Camera and microphone access")
	fmt.Println("- Location and photos access")
	fmt.Println("- Virtualization support")
	fmt.Println()

	// Try to read some directories
	home, _ := os.UserHomeDir()
	dirs := []string{
		home + "/Desktop",
		home + "/Pictures",
		home + "/Documents",
	}

	for _, dir := range dirs {
		fmt.Printf("Reading %s: ", dir)

		files, err := os.ReadDir(dir)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			continue
		}

		fmt.Printf("%d files\n", len(files))
		// Show first few files
		for i, f := range files {
			if i >= 3 {
				fmt.Println("...")
				break
			}
			fmt.Printf("- %s\n", f.Name())
		}
		fmt.Println()
	}
}
