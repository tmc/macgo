// Package main demonstrates a screen recording application with proper macOS permissions.
// This example shows how to request screen recording, microphone, and camera permissions.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/tmc/misc/macgo"
)

func main() {
	// Parse command-line flags
	var (
		outputFile   = flag.String("output", "", "Output file path (default: ~/Desktop/recording_<timestamp>.mov)")
		withAudio    = flag.Bool("audio", false, "Include microphone audio")
		withCamera   = flag.Bool("camera", false, "Include camera overlay")
		duration     = flag.Duration("duration", 0, "Recording duration (0 for manual stop)")
		showInDock   = flag.Bool("dock", false, "Show application in dock")
		listDevices  = flag.Bool("list", false, "List available recording devices")
	)
	flag.Parse()

	// Configure macgo
	macgo.SetAppName("ScreenRecorder")
	macgo.SetBundleID("com.example.screenrecorder")

	// Request necessary permissions
	entitlements := []interface{}{
		macgo.EntAppSandbox,
		macgo.EntScreenCapture,        // Screen recording permission
		macgo.EntUserSelectedReadWrite, // Save recordings
	}

	if *withAudio {
		entitlements = append(entitlements, macgo.EntMicrophone)
	}

	if *withCamera {
		entitlements = append(entitlements, macgo.EntCamera)
	}

	macgo.RequestEntitlements(entitlements...)

	// Configure dock visibility
	if !*showInDock {
		macgo.AddPlistEntry("LSUIElement", true)
	}

	// Add usage descriptions for privacy prompts
	macgo.AddPlistEntry("NSMicrophoneUsageDescription", "This app needs microphone access to record audio with your screen recording.")
	macgo.AddPlistEntry("NSCameraUsageDescription", "This app needs camera access to include video overlay in recordings.")
	macgo.AddPlistEntry("NSScreenCaptureUsageDescription", "This app needs screen recording permission to capture your screen.")

	// Start macgo
	macgo.Start()

	// List devices and exit if requested
	if *listDevices {
		listRecordingDevices()
		return
	}

	// Set up output file
	output := *outputFile
	if output == "" {
		homeDir, _ := os.UserHomeDir()
		timestamp := time.Now().Format("20060102_150405")
		output = filepath.Join(homeDir, "Desktop", fmt.Sprintf("recording_%s.mov", timestamp))
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(output)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Cannot create output directory: %v", err)
	}

	// Start recording
	fmt.Printf("🔴 Starting screen recording...\n")
	fmt.Printf("   Output: %s\n", output)
	fmt.Printf("   Audio: %v\n", *withAudio)
	fmt.Printf("   Camera: %v\n", *withCamera)
	if *duration > 0 {
		fmt.Printf("   Duration: %v\n", *duration)
	} else {
		fmt.Printf("   Press Ctrl+C to stop recording\n")
	}

	// Create recording command
	cmd := buildRecordingCommand(output, *withAudio, *withCamera)

	// Start recording
	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start recording: %v", err)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Set up duration timer if specified
	var timer *time.Timer
	if *duration > 0 {
		timer = time.NewTimer(*duration)
	} else {
		// Create a timer that never fires
		timer = time.NewTimer(24 * 365 * time.Hour)
		timer.Stop()
	}

	// Wait for stop signal
	select {
	case <-sigChan:
		fmt.Println("\n⏹ Stopping recording...")
	case <-timer.C:
		fmt.Printf("\n⏹ Recording duration reached (%v)\n", *duration)
	}

	// Stop recording gracefully
	if err := stopRecording(cmd); err != nil {
		log.Printf("Warning: Error stopping recording: %v", err)
	}

	// Verify output file
	if info, err := os.Stat(output); err == nil {
		fmt.Printf("✅ Recording saved: %s (%.2f MB)\n", output, float64(info.Size())/1024/1024)

		// Open in Finder if on macOS
		exec.Command("open", "-R", output).Run()
	} else {
		log.Printf("Warning: Could not verify output file: %v", err)
	}
}

func buildRecordingCommand(output string, withAudio, withCamera bool) *exec.Cmd {
	// Using screencapture for basic recording
	// In a real app, you'd use AVFoundation or similar
	args := []string{}

	if withAudio {
		args = append(args, "-a")
	}

	// Video recording mode
	args = append(args, "-v")

	// Output file
	args = append(args, output)

	cmd := exec.Command("screencapture", args...)
	return cmd
}

func stopRecording(cmd *exec.Cmd) error {
	// Send interrupt signal to screencapture
	if cmd.Process != nil {
		// screencapture responds to SIGINT
		if err := cmd.Process.Signal(syscall.SIGINT); err != nil {
			// If SIGINT fails, try SIGTERM
			if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
				// Last resort: SIGKILL
				return cmd.Process.Kill()
			}
		}

		// Wait for process to finish
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-done:
			return nil
		case <-time.After(5 * time.Second):
			// Force kill if it doesn't stop gracefully
			return cmd.Process.Kill()
		}
	}
	return nil
}

func listRecordingDevices() {
	fmt.Println("Available Recording Devices:")
	fmt.Println("============================")

	// List displays
	fmt.Println("\n📺 Displays:")
	// In a real implementation, you'd query the system for displays
	fmt.Println("  • Main Display")
	fmt.Println("  • All Displays")

	// List audio devices (simplified)
	fmt.Println("\n🎤 Audio Input Devices:")
	cmd := exec.Command("system_profiler", "SPAudioDataType")
	if output, err := cmd.Output(); err == nil {
		// Parse output for audio devices
		fmt.Println("  • Built-in Microphone")
		fmt.Println("  • System Audio (requires additional setup)")
	}

	// List cameras
	fmt.Println("\n📹 Cameras:")
	cmd = exec.Command("system_profiler", "SPCameraDataType")
	if output, err := cmd.Output(); err == nil {
		// Parse output for cameras
		_ = output // Would parse this in real implementation
		fmt.Println("  • FaceTime HD Camera")
	}

	fmt.Println("\n💡 Tip: Use 'ffmpeg -f avfoundation -list_devices true -i \"\"' for detailed device list")
}