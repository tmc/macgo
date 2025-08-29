// Entitlements Package Demo
// This example demonstrates how to use the entitlements package for setting TCC permissions
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tmc/misc/macgo"
	"github.com/tmc/misc/macgo/entitlements"
)

func init() {
	// Configure app details
	macgo.SetAppName("EntitlementsDemo")
	macgo.SetBundleID("com.example.entitlements-demo")

	// Set custom icon
	macgo.SetIconFile("/System/Library/CoreServices/CoreTypes.bundle/Contents/Resources/ToolbarCustomizeIcon.icns")

	// Enable improved signal handling
	macgo.EnableImprovedSignalHandling()

	// Enable debug logging
	macgo.EnableDebug()

	// Use entitlements package for semantic permission setting
	// This is more readable than using macgo.RequestEntitlements()
	entitlements.SetAppSandbox()      // Enable app sandbox
	entitlements.SetCamera()          // Request camera access
	entitlements.SetMic()             // Request microphone access
	entitlements.SetLocation()        // Request location access
	entitlements.SetContacts()        // Request contacts access
	entitlements.SetPhotos()          // Request photos access
	entitlements.SetCalendar()        // Request calendar access
	entitlements.SetReminders()       // Request reminders access

	// Network entitlements (note: these don't affect Go's net/http)
	entitlements.SetNetworkClient()   // Allow outgoing network connections
	entitlements.SetNetworkServer()   // Allow incoming network connections

	// Device access
	entitlements.SetBluetooth()       // Allow Bluetooth access
	entitlements.SetUSB()             // Allow USB device access
	entitlements.SetAudioInput()      // Allow audio input access
	entitlements.SetPrinting()        // Allow printing

	// You can also use convenience functions
	// entitlements.SetAllTCCPermissions()  // Enable all TCC permissions
	// entitlements.SetAllDeviceAccess()    // Enable all device access
	// entitlements.SetAllNetworking()      // Enable all networking

	// Start macgo
	macgo.Start()
}

func main() {
	fmt.Println("Entitlements Package Demo")
	fmt.Println("=========================")
	fmt.Println()

	// Check if we're running in an app bundle
	if macgo.IsInAppBundle() {
		fmt.Println("✓ Running inside app bundle")
	} else {
		fmt.Println("ℹ Running outside app bundle - will relaunch")
	}

	fmt.Println("\nThis example demonstrates:")
	fmt.Println("  - Using entitlements package for semantic permission setting")
	fmt.Println("  - App sandbox enabled")
	fmt.Println("  - All TCC permissions requested")
	fmt.Println("  - Device access permissions")
	fmt.Println("  - Network permissions")
	fmt.Println("  - Custom app icon")
	fmt.Println("  - Improved signal handling")
	fmt.Println()

	// Set up signal handling
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Start a countdown
	fmt.Println("Press Ctrl+C to test signal handling...")
	go func() {
		for i := 10; i > 0; i-- {
			fmt.Printf("\rCountdown: %d ", i)
			time.Sleep(1 * time.Second)
		}
		fmt.Println("\rTimeout reached")
		os.Exit(0)
	}()

	// Wait for signal
	sig := <-c
	fmt.Printf("\n\nReceived signal: %v\n", sig)
	fmt.Println("Signal handling is working correctly!")
	fmt.Println("Shutting down...")
}