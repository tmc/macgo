// Hello World - macgo
// The simplest possible example
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	macgo "github.com/tmc/misc/macgo"
)

func main() {
	fmt.Printf("Hello from macgo! PID: %d\n", os.Getpid())
	fmt.Println()

	// Simple one-line setup for camera and microphone
	err := macgo.Request(macgo.Camera, macgo.Microphone)
	if err != nil {
		log.Fatalf("Failed to request permissions: %v", err)
	}

	fmt.Println("✓ Permissions granted!")
	fmt.Println("✓ Running with camera and microphone access")
	fmt.Println()

	fmt.Println("Key features:")
	fmt.Println("  • No init() function magic")
	fmt.Println("  • Clean, simple API")
	fmt.Println("  • One line to request permissions")
	fmt.Println("  • Explicit configuration")
	fmt.Println()

	// Simple countdown
	fmt.Println("Application will run for 5 seconds...")
	for i := 5; i > 0; i-- {
		fmt.Printf("\rCountdown: %d", i)
		time.Sleep(1 * time.Second)
	}

	fmt.Println("\nGoodbye from macgo!")
}
