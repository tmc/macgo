// Auto-Signed Example - macgo v2
// Demonstrates automatic code signing using the signed subpackage
package main

import (
	"fmt"
	"log"
	"os"

	// Import the signed package for automatic code signing
	macgo "github.com/tmc/misc/macgo"
	signed "github.com/tmc/misc/macgo/auto/signed"
)

func main() {
	fmt.Printf("Auto-Signed Demo - macgo v2! PID: %d\n", os.Getpid())
	fmt.Println()

	fmt.Println("🔍 Using auto-signed package - will detect Developer ID automatically")
	fmt.Println()

	// The signed package automatically enables auto-signing
	err := signed.Request(macgo.Files)
	if err != nil {
		log.Fatalf("Failed to initialize with auto-signing: %v", err)
	}

	fmt.Println("✓ Auto-signing setup complete!")
	fmt.Println()

	fmt.Println("📋 How this works:")
	fmt.Println("  1. Import: import signed \"github.com/tmc/misc/macgo/auto/signed\"")
	fmt.Println("  2. Use:    signed.Request(macgo.Files) instead of macgo.Request(macgo.Files)")
	fmt.Println("  3. Result: macgo automatically detects and uses Developer ID certificates")
	fmt.Println()

	fmt.Println("💡 Alternative approaches:")
	fmt.Println()
	fmt.Println("Using environment variable:")
	fmt.Println("  export MACGO_AUTO_SIGN=1")
	fmt.Println("  go run main.go")
	fmt.Println()

	fmt.Println("Using configuration:")
	fmt.Println("  cfg := &macgo.Config{")
	fmt.Println("      Permissions: []macgo.Permission{macgo.Files},")
	fmt.Println("      AutoSign: true,")
	fmt.Println("  }")
	fmt.Println("  macgo.Start(cfg)")
	fmt.Println()

	fmt.Println("Using fluent API:")
	fmt.Println("  cfg := new(macgo.Config).WithPermissions(macgo.Files).WithAutoSign()")
	fmt.Println("  macgo.Start(cfg)")
}
