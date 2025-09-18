// Example demonstrating the simplified v2 API following Russ Cox's principles
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	macgo "github.com/tmc/misc/macgo/v2"
)

func main() {
	fmt.Println("=== macgo v2: Simplified API Demo ===")
	fmt.Println("Following Russ Cox's design principles:")
	fmt.Println("• Simple is better than complex")
	fmt.Println("• Explicit is better than implicit")
	fmt.Println("• APIs should be hard to misuse")
	fmt.Println()

	// Choose your approach based on your needs:

	// Approach 1: Simple permission request (covers 80% of use cases)
	simpleApproach()

	// Approach 2: Full configuration (for complex cases)
	// configuredApproach()

	// Approach 3: Environment-driven (for deployments)
	// environmentApproach()
}

func simpleApproach() {
	fmt.Println("Using SIMPLE approach:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━")

	// Just request what you need - one line!
	err := macgo.Request(macgo.Camera, macgo.Microphone)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("✓ Permissions granted")
	runApp()
}

func configuredApproach() {
	fmt.Println("Using CONFIGURED approach:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━")

	// Explicit configuration with all options
	cfg := &macgo.Config{
		AppName:  "MacGoV2Demo",
		BundleID: "com.example.macgo.v2",

		// Core permissions (95% of use cases)
		Permissions: []macgo.Permission{
			macgo.Camera,
			macgo.Microphone,
			macgo.Location,
			macgo.Files,   // Simplified file access
			macgo.Network, // Simplified network access
		},

		// For the 5% edge cases
		Custom: []string{
			"com.apple.security.personal-information.photos-library",
		},

		Debug:      true,
		KeepBundle: false,
	}

	// Use context for lifecycle management
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := macgo.StartContext(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("✓ Started with full configuration")
	runAppWithContext(ctx)
}

func environmentApproach() {
	fmt.Println("Using ENVIRONMENT approach:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Set environment variables (usually done outside the app)
	os.Setenv("MACGO_APP_NAME", "EnvDemo")
	os.Setenv("MACGO_BUNDLE_ID", "com.example.env")
	os.Setenv("MACGO_CAMERA", "1")
	os.Setenv("MACGO_MICROPHONE", "1")
	os.Setenv("MACGO_DEBUG", "1")

	// Single call loads from environment
	err := macgo.Auto()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("✓ Configured from environment")
	runApp()
}

func runApp() {
	// Set up signal handling
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	fmt.Println("\nApp is running with permissions!")
	fmt.Println("Press Ctrl+C to exit...")

	<-c
	fmt.Println("\n✓ Clean shutdown")
}

func runAppWithContext(ctx context.Context) {
	// Set up signal handling
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	fmt.Println("\nApp is running with permissions and context!")
	fmt.Println("Press Ctrl+C to exit (or wait for timeout)...")

	select {
	case <-c:
		fmt.Println("\n✓ User interrupted")
	case <-ctx.Done():
		fmt.Println("\n✓ Context timeout")
	}
}