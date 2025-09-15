// Auto-Initialization Example
// This example demonstrates macgo's auto-initialization packages
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Auto-initialization packages provide simplified imports for common configurations
	// Choose ONE of these approaches:

	// Basic auto-initialization (no sandbox)
	// _ "github.com/tmc/misc/macgo/auto"

	// With app sandbox
	// _ "github.com/tmc/misc/macgo/auto/sandbox"

	// With app sandbox and user read access
	// _ "github.com/tmc/misc/macgo/auto/sandbox/readonly"

	// With improved signal handling (better Ctrl+C handling)
	_ "github.com/tmc/misc/macgo/auto/sandbox/signalhandler"

	// You can also use the main macgo package for additional configuration
	"github.com/tmc/misc/macgo"
)

func init() {
	// When using auto-initialization packages, basic setup is done automatically
	// You can still add additional configuration here

	// Configure app details
	macgo.SetAppName("AutoInitExample")
	macgo.SetBundleID("com.example.autoinit")

	// Set custom icon
	macgo.SetIconFile("/System/Library/CoreServices/CoreTypes.bundle/Contents/Resources/ToolbarCustomizeIcon.icns")

	// Add additional entitlements beyond what auto-initialization provides
	macgo.RequestEntitlement(macgo.EntCamera)
	macgo.RequestEntitlement(macgo.EntMicrophone)
	macgo.RequestEntitlement(macgo.EntLocation)

	// Enable debug logging
	macgo.EnableDebug()

	// Note: No need to call macgo.Start() when using auto-initialization packages
	// The auto package handles this automatically
}

func main() {
	fmt.Println("Auto-Initialization Example")
	fmt.Println("===========================")
	fmt.Println()

	// Check if we're running in an app bundle
	if macgo.IsInAppBundle() {
		fmt.Println("✓ Running inside app bundle")
	} else {
		fmt.Println("ℹ Running outside app bundle - will relaunch")
	}

	fmt.Println()
	fmt.Println("This example demonstrates:")
	fmt.Println("  - Using auto-initialization packages")
	fmt.Println("  - Automatic sandbox setup")
	fmt.Println("  - Improved signal handling (from auto package)")
	fmt.Println("  - Additional configuration on top of auto setup")
	fmt.Println()

	fmt.Println("Available auto-initialization packages:")
	fmt.Println("  - github.com/tmc/misc/macgo/auto")
	fmt.Println("  - github.com/tmc/misc/macgo/auto/sandbox")
	fmt.Println("  - github.com/tmc/misc/macgo/auto/sandbox/readonly")
	fmt.Println("  - github.com/tmc/misc/macgo/auto/sandbox/signalhandler")
	fmt.Println()

	// Set up signal handling
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	fmt.Println("Press Ctrl+C to test signal handling...")

	// Start a countdown
	go func() {
		for i := 15; i > 0; i-- {
			fmt.Printf("\rCountdown: %d ", i)
			time.Sleep(1 * time.Second)
		}
		fmt.Println("\rTimeout reached")
		os.Exit(0)
	}()

	// Wait for signal
	sig := <-c
	fmt.Printf("\n\nReceived signal: %v\n", sig)
	fmt.Println("✓ Auto-initialization signal handling is working correctly!")
	fmt.Println("Shutting down...")
}
