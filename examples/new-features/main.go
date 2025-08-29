// Example demonstrating the newly documented macgo features
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/tmc/misc/macgo"
)

func init() {
	// Set a custom icon for the app bundle
	macgo.SetIconFile("/System/Library/CoreServices/CoreTypes.bundle/Contents/Resources/DocumentIcon.icns")
	
	// Enable improved signal handling for better Ctrl+C support
	macgo.EnableImprovedSignalHandling()
	
	// Request camera access
	macgo.RequestEntitlements(macgo.EntCamera)
	
	// Enable debugging to see what's happening
	macgo.EnableDebug()
}

func main() {
	fmt.Println("New Features Example")
	fmt.Println("===================")
	
	// Check if we're already in an app bundle
	if macgo.IsInAppBundle() {
		fmt.Println("✓ Running inside app bundle")
	} else {
		fmt.Println("ℹ Running outside app bundle - will relaunch")
	}
	
	// Use StartWithContext for better control
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Start macgo with context
	macgo.StartWithContext(ctx)
	
	// At this point, if we were relaunched, we should be in an app bundle
	if macgo.IsInAppBundle() {
		fmt.Println("✓ Successfully relaunched inside app bundle")
		fmt.Println("✓ Camera access should now be available")
		fmt.Println("✓ Custom icon should be visible")
		fmt.Println("✓ Improved signal handling is active")
	} else {
		log.Println("Warning: Not in app bundle")
	}
	
	fmt.Println("\nPress Ctrl+C to test signal handling...")
	
	// Simulate some work
	select {
	case <-ctx.Done():
		fmt.Println("\nTimeout reached or context cancelled")
	}
}