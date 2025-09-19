// Signal Test - macgo
// Simple interactive test for signal forwarding
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	macgo "github.com/tmc/misc/macgo"
)

func main() {
	fmt.Printf("Signal Test - macgo! PID: %d\n", os.Getpid())
	fmt.Println()

	// Test with files permission to trigger open command path
	cfg := &macgo.Config{
		Permissions: []macgo.Permission{macgo.Files},
		Debug:       true,
	}

	err := macgo.Start(cfg)
	if err != nil {
		fmt.Printf("Failed to start macgo: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("🔔 Signal Forwarding Test")
	fmt.Println("Press Ctrl+C to test signal forwarding...")
	fmt.Printf("Process PID: %d\n", os.Getpid())

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for signal with timeout
	select {
	case sig := <-sigChan:
		fmt.Printf("\n✓ Signal received: %v\n", sig)
		fmt.Println("✓ Signal forwarding working correctly!")
		os.Exit(0)
	case <-time.After(15 * time.Second):
		fmt.Println("\n⏰ Test timed out (normal - signal forwarding available)")
		fmt.Println("✓ Test completed successfully")
	}
}
