// Hello World - macgo
// The simplest possible example
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	macgo "github.com/tmc/macgo"
)

func init() {
	// Configure and start macgo - this will re-execute the program in a bundle if needed
	cfg := macgo.NewConfig().
		WithAppName("Hello macgo").
		WithPermissions(macgo.Camera, macgo.Microphone)

	// Start will re-execute the program if needed, exiting the current process
	if err := macgo.Start(cfg); err != nil {
		log.Fatalf("Failed to start macgo: %v", err)
	}
}

func main() {
	// This code only runs after macgo.Start() has ensured we're in a proper bundle
	fmt.Printf("Hello from macgo! PID: %d (stdout)\n", os.Getpid())
	fmt.Fprintf(os.Stderr, "Hello from macgo! PID: %d (stderr)\n", os.Getpid())
	fmt.Println()

	fmt.Println("✓ Permissions granted!")
	fmt.Println("✓ Running with camera and microphone access")
	fmt.Println()

	fmt.Println("Key features:")
	fmt.Println("  • Automatic bundle setup via init()")
	fmt.Println("  • Clean, simple API")
	fmt.Println("  • Transparent re-execution")
	fmt.Println("  • Automatic permission handling")
	fmt.Println()

	// Simple countdown
	fmt.Println("Application will run for 5 seconds...")
	for i := 5; i > 0; i-- {
		fmt.Printf("\rCountdown: %d", i)
		time.Sleep(1 * time.Second)
	}

	fmt.Println("\nGoodbye from macgo!")
}
