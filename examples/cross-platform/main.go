// Package main demonstrates cross-platform usage of macgo.
// This example works on all platforms - on macOS it creates an app bundle
// with camera permissions, on other platforms it's a no-op.
package main

import (
	"fmt"
	"runtime"

	"github.com/tmc/misc/macgo"
)

func main() {
	fmt.Printf("Running on %s/%s\n", runtime.GOOS, runtime.GOARCH)

	// These macgo calls work on all platforms
	// On macOS: Will create app bundle with camera/microphone permissions
	// On other platforms: Will be no-ops (with debug messages if MACGO_DEBUG=1)
	macgo.RequestEntitlements(
		macgo.EntCamera,
		macgo.EntMicrophone,
		macgo.EntAppSandbox,
	)

	macgo.SetAppName("CrossPlatformExample")
	macgo.SetBundleID("com.example.crossplatform")

	// Start macgo - this handles the app bundle creation and relaunching on macOS
	macgo.Start()

	// Your application logic goes here
	fmt.Println("Application starting...")

	// Check if we're running in an app bundle (macOS only)
	if macgo.IsInAppBundle() {
		fmt.Println("✓ Running inside macOS app bundle with TCC permissions")
	} else {
		if runtime.GOOS == "darwin" {
			fmt.Println("⚠ Running as command-line binary on macOS (limited permissions)")
		} else {
			fmt.Println("✓ Running as regular binary on", runtime.GOOS)
		}
	}

	// Simulate application work
	fmt.Println("Application completed successfully!")

	// Clean shutdown
	macgo.Stop()
}