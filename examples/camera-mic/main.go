// Camera & Microphone Access - macgo
// Shows how to request and use camera/microphone permissions
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	macgo "github.com/tmc/misc/macgo"
)

func main() {
	fmt.Println("macgo - Camera & Microphone Example")
	fmt.Println("======================================")
	fmt.Println()

	// Simple approach for camera and mic
	err := macgo.Request(macgo.Camera, macgo.Microphone)
	if err != nil {
		log.Fatalf("Failed to request permissions: %v", err)
	}

	fmt.Println("✓ Camera permission granted")
	fmt.Println("✓ Microphone permission granted")
	fmt.Println()

	// Now we can access camera and microphone
	testCameraAccess()
	testMicrophoneAccess()

	fmt.Println("\nPress Ctrl+C to exit...")

	// Wait for interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("\nExiting...")
}

func testCameraAccess() {
	fmt.Println("Testing camera access...")

	// Check if we can list video devices (macOS specific)
	cmd := exec.Command("system_profiler", "SPCameraDataType")
	output, err := cmd.Output()

	if err != nil {
		fmt.Printf("  ❌ Could not list cameras: %v\n", err)
		return
	}

	fmt.Println("  ✓ Successfully accessed camera information")

	// Show first few lines of output
	lines := string(output)
	if len(lines) > 200 {
		fmt.Printf("  Camera info: %s...\n", lines[:200])
	} else {
		fmt.Printf("  Camera info: %s\n", lines)
	}
}

func testMicrophoneAccess() {
	fmt.Println("\nTesting microphone access...")

	// Check if we can list audio devices
	cmd := exec.Command("system_profiler", "SPAudioDataType")
	output, err := cmd.Output()

	if err != nil {
		fmt.Printf("  ❌ Could not list audio devices: %v\n", err)
		return
	}

	fmt.Println("  ✓ Successfully accessed audio information")

	// Show first few lines of output
	lines := string(output)
	if len(lines) > 200 {
		fmt.Printf("  Audio info: %s...\n", lines[:200])
	} else {
		fmt.Printf("  Audio info: %s\n", lines)
	}
}

