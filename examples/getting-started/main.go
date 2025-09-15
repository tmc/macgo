// Getting Started with macgo
// This example shows the basic patterns for using macgo with proper API usage
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tmc/misc/macgo"
)

func init() {
	// Step 1: Configure app details
	macgo.SetAppName("GettingStarted")
	macgo.SetBundleID("com.example.macgo.getting-started")

	// Step 2: Set custom icon (optional)
	// Using a system icon for demonstration
	macgo.SetIconFile("/System/Library/CoreServices/CoreTypes.bundle/Contents/Resources/ToolbarCustomizeIcon.icns")

	// Step 3: Enable improved signal handling (recommended)
	// This provides better Ctrl+C handling and preserves stdin/stdout/stderr
	macgo.EnableImprovedSignalHandling()

	// Step 4: Request permissions using entitlements package (recommended approach)
	// This is more readable than using macgo.RequestEntitlements() directly
	macgo.RequestEntitlement(macgo.EntAppSandbox) // Enable app sandbox
	macgo.RequestEntitlement(macgo.EntCamera)     // Request camera access
	macgo.RequestEntitlement(macgo.EntMicrophone) // Request microphone access
	macgo.RequestEntitlement(macgo.EntLocation)   // Request location access

	// Alternative: You can also use the direct API
	// macgo.RequestEntitlements(
	//     macgo.EntAppSandbox,
	//     macgo.EntCamera,
	//     macgo.EntMicrophone,
	//     macgo.EntLocation,
	// )

	// Step 5: Enable debug logging (optional, useful for development)
	macgo.EnableDebug()

	// Step 6: Start macgo - this creates the app bundle and relaunches if needed
	// Must be called after all configuration is done
	macgo.Start()
}

func main() {
	fmt.Println("Getting Started with macgo")
	fmt.Println("==========================")
	fmt.Println()

	// Check if we're running in an app bundle
	if macgo.IsInAppBundle() {
		fmt.Println("✓ Running inside app bundle")
		fmt.Println("✓ TCC permissions should now be available")
		fmt.Println("✓ Custom icon is visible")
		fmt.Println("✓ Improved signal handling is active")
	} else {
		fmt.Println("ℹ Running outside app bundle - will relaunch")
	}

	fmt.Println()
	fmt.Println("This example demonstrates:")
	fmt.Println("  - Basic macgo configuration")
	fmt.Println("  - Using the entitlements package for permissions")
	fmt.Println("  - Setting a custom app icon")
	fmt.Println("  - Enabling improved signal handling")
	fmt.Println("  - Using StartWithContext for better control")
	fmt.Println()

	// Using context for better control (optional but recommended)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// You can also use macgo.StartWithContext() instead of macgo.Start()
	// This allows for better lifecycle management and cancellation support
	// macgo.StartWithContext(ctx)

	// Set up signal handling to demonstrate improved signal handling
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	fmt.Println("Press Ctrl+C to test signal handling...")

	// Wait for signal or timeout
	select {
	case sig := <-c:
		fmt.Printf("\nReceived signal: %v\n", sig)
		fmt.Println("✓ Signal handling is working correctly!")
	case <-ctx.Done():
		fmt.Println("\nTimeout reached")
	}

	fmt.Println("Shutting down...")
}
