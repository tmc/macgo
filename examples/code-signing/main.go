// Code Signing Example - macgo
// Demonstrates optional code signing for enhanced security and distribution
package main

import (
	"fmt"
	"log"
	"os"

	macgo "github.com/tmc/macgo"
)

func main() {
	fmt.Printf("Code Signing Demo - macgo! PID: %d\n", os.Getpid())
	fmt.Println()

	// Example 1: No code signing (default behavior)
	fmt.Println("🔓 Creating unsigned app bundle...")
	err := macgo.Request(macgo.Files)
	if err != nil {
		log.Fatalf("Failed to create unsigned bundle: %v", err)
	}
	fmt.Println("✓ Unsigned bundle created successfully")
	fmt.Println()

	// Note: For actual code signing, you would use one of these approaches:
	showCodeSigningExamples()
}

func showCodeSigningExamples() {
	fmt.Println("📋 Code Signing Options:")
	fmt.Println()

	fmt.Println("1. With environment variable:")
	fmt.Println("   export MACGO_CODE_SIGN_IDENTITY=\"Developer ID Application\"")
	fmt.Println("   go run main.go")
	fmt.Println()

	fmt.Println("2. With explicit configuration:")
	fmt.Println("   cfg := &macgo.Config{")
	fmt.Println("       Permissions: []macgo.Permission{macgo.Files},")
	fmt.Println("       CodeSignIdentity: \"Developer ID Application\",")
	fmt.Println("       Debug: true,")
	fmt.Println("   }")
	fmt.Println("   macgo.Start(cfg)")
	fmt.Println()

	fmt.Println("3. With fluent API:")
	fmt.Println("   cfg := new(macgo.Config).WithPermissions(macgo.Files).WithCodeSigning(\"Developer ID Application\")")
	fmt.Println("   macgo.Start(cfg)")
	fmt.Println()

	fmt.Println("💡 Benefits of code signing:")
	fmt.Println("  • Better TCC dialog presentation (shows developer name)")
	fmt.Println("  • Required for notarization and distribution")
	fmt.Println("  • Enhanced security and user trust")
	fmt.Println("  • Hardened runtime protection")
	fmt.Println()

	fmt.Println("🔍 To check if current bundle is signed:")
	fmt.Println("   codesign -dv ~/go/bin/main.app")
}
