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
	// Configure app details
	macgo.SetAppName("ComprehensiveExample")
	macgo.SetBundleID("com.example.comprehensive")

	// Set custom icon (using a system icon for demonstration)
	macgo.SetIconFile("/System/Library/CoreServices/CoreTypes.bundle/Contents/Resources/ToolbarCustomizeIcon.icns")

	// Enable improved signal handling for better Ctrl+C support
	macgo.EnableImprovedSignalHandling()

	// Request various entitlements using the entitlements package
	// This is more readable and semantic than using macgo.RequestEntitlements()
	macgo.RequestEntitlement(macgo.EntAppSandbox)
	macgo.RequestEntitlement(macgo.EntCamera)
	macgo.RequestEntitlement(macgo.EntMicrophone)
	macgo.RequestEntitlement(macgo.EntLocation)
	macgo.RequestEntitlement(macgo.EntPhotos)
	macgo.RequestEntitlement(macgo.EntAddressBook)
	macgo.RequestEntitlement(macgo.EntNetworkClient)
	macgo.RequestEntitlement(macgo.EntNetworkServer)

	// Also demonstrate the direct API approach
	macgo.RequestEntitlements(
		macgo.EntUserSelectedReadOnly,
		macgo.EntDownloadsReadOnly,
	)

	// Enable debugging to see what's happening
	macgo.EnableDebug()
}

func main() {
	fmt.Println("Starting comprehensive macgo example...")

	// Check if we're in an app bundle
	if macgo.IsInAppBundle() {
		fmt.Println("✓ Running inside app bundle")
	} else {
		fmt.Println("Running outside app bundle - will relaunch")
	}

	// Use context for better control
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start macgo with context
	macgo.StartWithContext(ctx)

	// Set up signal handling to demonstrate improved signal handling
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	fmt.Println("Running with:")
	fmt.Println("  - Custom app icon (SetIconFile)")
	fmt.Println("  - Improved signal handling (EnableImprovedSignalHandling)")
	fmt.Println("  - App sandbox enabled")
	fmt.Println("  - Camera and microphone access")
	fmt.Println("  - Location and photos access")
	fmt.Println("  - Contacts access")
	fmt.Println("  - Network client/server access")
	fmt.Println("  - User-selected file read access")
	fmt.Println("  - Downloads folder read access")
	fmt.Println()
	fmt.Println("Press Ctrl+C to test signal handling...")

	// Wait for signal
	select {
	case sig := <-c:
		fmt.Printf("\nReceived signal: %v\n", sig)
		fmt.Println("Signal handling is working correctly!")
	case <-ctx.Done():
		fmt.Println("\nTimeout reached")
	}

	fmt.Println("Shutting down...")
}
