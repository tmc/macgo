package main

import (
	"fmt"
	"os"
	"time"

	"github.com/tmc/misc/macgo"
	"github.com/tmc/misc/macgo/debug" // Using the dedicated debug package
)

func init() {
	// Initialize macgo's debug package early.
	// This reads env vars like MACGO_SIGNAL_DEBUG, MACGO_PPROF.
	debug.Init()

	// Enable macgo's internal debug logging (MACGO_DEBUG=1)
	macgo.EnableDebug()

	// Configure macgo (optional, defaults will be used if not set)
	macgo.SetAppName("HelloMacgoApp")
	macgo.SetBundleID("com.example.hellomacgo")

	// Set custom icon
	macgo.SetIconFile("/System/Library/CoreServices/CoreTypes.bundle/Contents/Resources/ToolbarCustomizeIcon.icns")

	// Enable improved signal handling
	macgo.EnableImprovedSignalHandling()

	// Example: Request some entitlements using macgo
	macgo.RequestEntitlements(
		macgo.EntAppSandbox,
		macgo.EntCamera,     // Example: if camera needed
		macgo.EntMicrophone, // Example: if microphone needed
	)

	// Start macgo - creates bundle and relaunches if necessary
	macgo.Start()
}

func main() {
	fmt.Printf("Hello from macgo! PID: %d\n", os.Getpid())
	fmt.Printf("Running in app bundle: %t\n", macgo.IsInAppBundle())
	fmt.Println()

	// Show debug status
	if os.Getenv("MACGO_DEBUG") == "1" {
		fmt.Println("✓ macgo internal debug logging is enabled.")
	}
	if debug.IsTraceEnabled() {
		fmt.Println("✓ macgo.debug signal tracing is enabled (check logs).")
	}
	if debug.IsPprofEnabled() {
		fmt.Println("✓ macgo.debug pprof server for this app is enabled (check logs for port).")
	}

	fmt.Println()
	fmt.Println("This example demonstrates:")
	fmt.Println("  - Basic macgo setup with entitlements package")
	fmt.Println("  - Custom app icon")
	fmt.Println("  - Improved signal handling")
	fmt.Println("  - Debug package usage")
	fmt.Println("  - App sandbox and TCC permissions")
	fmt.Println()

	fmt.Println("Application will run for 5 seconds then exit.")
	fmt.Println("Press Ctrl+C to test signal handling...")

	// Simple countdown with signal handling
	for i := 5; i > 0; i-- {
		fmt.Printf("\rCountdown: %d", i)
		time.Sleep(1 * time.Second)
	}

	fmt.Println("\nExiting HelloMacgoApp.")
	// Clean up debug package resources if it was used for logging to file
	debug.Close()
}
