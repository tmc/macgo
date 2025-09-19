// Getting Started with macgo v2
// This example shows the simplified API following Russ Cox's principles
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	macgo "github.com/tmc/misc/macgo"
)

func main() {
	fmt.Println("Getting Started with macgo v2")
	fmt.Println("=============================")
	fmt.Println()
	fmt.Println("Key improvements over v1:")
	fmt.Println("  ✓ No global state or init() magic")
	fmt.Println("  ✓ Explicit configuration")
	fmt.Println("  ✓ Simpler API with fewer concepts")
	fmt.Println("  ✓ Better error handling")
	fmt.Println()

	// All configuration in one place - no init() function!
	cfg := &macgo.Config{
		AppName:  "GettingStarted",
		BundleID: "com.example.macgo.getting-started",

		// Simple permission model - just 5 core types
		Permissions: []macgo.Permission{
			macgo.Camera,
			macgo.Microphone,
			macgo.Location,
		},

		Debug: true, // Enable debug logging
	}

	// Create context for lifecycle management
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Single explicit call to start - no magic!
	fmt.Println("Starting macgo with configuration...")
	err := macgo.StartContext(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to start macgo: %v", err)
	}

	fmt.Println("✓ App bundle created and permissions configured")
	fmt.Println()
	fmt.Println("This example demonstrates:")
	fmt.Println("  - Explicit configuration (no init magic)")
	fmt.Println("  - Simple permission model (5 core types)")
	fmt.Println("  - Context-based lifecycle management")
	fmt.Println("  - Clean error handling")
	fmt.Println()

	// Set up signal handling
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	fmt.Println("Press Ctrl+C to test signal handling...")

	// Wait for signal or timeout
	select {
	case sig := <-c:
		fmt.Printf("\nReceived signal: %v\n", sig)
		fmt.Println("✓ Signal handling works correctly!")
	case <-ctx.Done():
		fmt.Println("\nTimeout reached")
	}

	fmt.Println("Shutting down...")
}

// Alternative approaches for different use cases:

func simpleApproach() {
	// For quick scripts - one line!
	if err := macgo.Request(macgo.Camera, macgo.Microphone); err != nil {
		log.Fatal(err)
	}
	// Your app code here...
}

func builderApproach() {
	// Using the builder pattern
	err := macgo.Start(
		new(macgo.Config).
			WithPermissions(macgo.Camera, macgo.Microphone).
			WithDebug(),
	)
	if err != nil {
		log.Fatal(err)
	}
	// Your app code here...
}

func environmentApproach() {
	// For deployment scenarios
	// Set: MACGO_CAMERA=1 MACGO_MICROPHONE=1 MACGO_DEBUG=1
	if err := macgo.Auto(); err != nil {
		log.Fatal(err)
	}
	// Your app code here...
}
