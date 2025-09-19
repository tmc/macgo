// Camera & Microphone Access - macgo v2
// Shows how to request and use camera/microphone permissions
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	macgo "github.com/tmc/misc/macgo/v2"
)

func main() {
	fmt.Println("macgo v2 - Camera & Microphone Example")
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

// Example with context and timeout
func withContext() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg := &macgo.Config{
		AppName:     "CameraMicApp",
		BundleID:    "com.example.camera-mic",
		Permissions: []macgo.Permission{macgo.Camera, macgo.Microphone},
		Debug:       true,
	}

	if err := macgo.StartContext(ctx, cfg); err != nil {
		log.Fatal(err)
	}

	// Your app code here...
	select {
	case <-ctx.Done():
		fmt.Println("Context cancelled")
	}
}

// Example for recording scenario
func recordingExample() {
	// Request all media permissions
	cfg := &macgo.Config{
		AppName:  "RecordingApp",
		BundleID: "com.example.recording",
		Permissions: []macgo.Permission{
			macgo.Camera,
			macgo.Microphone,
			macgo.Files, // For saving recordings
		},
		Debug: true,
	}

	if err := macgo.Start(cfg); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Ready to record with camera, microphone, and file access!")
	// Recording logic here...
}
