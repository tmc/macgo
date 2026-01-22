// Auto-Signed Example - macgo
// Demonstrates automatic code signing using explicit configuration
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/tmc/macgo"
)

func main() {
	fmt.Printf("Auto-Signed Demo - macgo! PID: %d\n", os.Getpid())
	fmt.Println()

	fmt.Println("🔍 Using explicit config - will detect Developer ID automatically")
	fmt.Println()

	// Simple one-liner equivalent with explicit config
	cfg := &macgo.Config{
		Permissions: []macgo.Permission{macgo.Files},
		AutoSign:    true,
	}

	if err := macgo.Start(cfg); err != nil {
		log.Fatalf("Failed to initialize with auto-signing: %v", err)
	}

	fmt.Println("✓ Auto-signing setup complete!")
	fmt.Println()

	fmt.Println("📋 How this works:")
	fmt.Println("  1. Config: Set AutoSign: true")
	fmt.Println("  2. Result: macgo automatically detects and uses Developer ID certificates")
}
