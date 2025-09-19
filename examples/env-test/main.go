// Environment Variable Test - macgo
// Tests loading configuration from environment variables
package main

import (
	"fmt"
	"log"
	"os"

	macgo "github.com/tmc/misc/macgo"
)

func main() {
	fmt.Printf("Environment Variable Test - macgo! PID: %d\n", os.Getpid())
	fmt.Println()

	// Use Auto() to load configuration from environment
	err := macgo.Auto()
	if err != nil {
		log.Fatalf("Failed to initialize macgo: %v", err)
	}

	fmt.Println("✓ macgo initialized from environment variables!")
	fmt.Println()
	fmt.Println("Environment variables recognized:")
	fmt.Println("  MACGO_DEBUG=1             # Enable debug output")
	fmt.Println("  MACGO_CODE_SIGN_IDENTITY  # Specific signing identity")
	fmt.Println("  MACGO_AUTO_SIGN=1         # Auto-detect certificates")
	fmt.Println("  MACGO_AD_HOC_SIGN=1       # Use ad-hoc signing")
	fmt.Println("  MACGO_FILES=1             # Request file permissions")
	fmt.Println("  MACGO_CAMERA=1            # Request camera permissions")
	fmt.Println("  MACGO_MICROPHONE=1        # Request microphone permissions")
}
