// Package main demonstrates screen recording using the v2 macgo API.
// This example shows the simplified permission handling in v2.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	macgo "github.com/tmc/misc/macgo/v2"
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

	// Build permissions list based on what's needed
	permissions := []macgo.Permission{
		macgo.Screen, // Screen recording permission
		macgo.Files,  // Save recordings
	}

	if *withAudio {
		permissions = append(permissions, macgo.Microphone)
	}

	if *withCamera {
		permissions = append(permissions, macgo.Camera)
	}

	// Configure macgo v2
	cfg := &macgo.Config{
		AppName:     "ScreenRecorder",
		BundleID:    "com.example.screenrecorder",
		Permissions: permissions,
		LSUIElement: !*showInDock, // Hide from dock unless requested
		Debug:       os.Getenv("MACGO_DEBUG") == "1",

		// Custom entitlements for privacy descriptions
		Custom: []string{
			"NSMicrophoneUsageDescription:This app needs microphone access to record audio with your screen recording.",
			"NSCameraUsageDescription:This app needs camera access to include video overlay in recordings.",
			"NSScreenCaptureUsageDescription:This app needs screen recording permission to capture your screen.",
		},
	}

	// Start macgo
	if err := macgo.Start(cfg); err != nil {
		log.Fatalf("Failed to start macgo: %v", err)
	}

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
	// Build ffmpeg command for better quality recording
	args := []string{
		"-f", "avfoundation",
		"-capture_cursor", "1",
		"-capture_mouse_clicks", "1",
	}

	// Video input (screen)
	videoInput := "1" // Capture screen
	audioInput := ""

	if withAudio {
		audioInput = ":0" // Default audio device
	}

	args = append(args, "-i", videoInput+audioInput)

	// Camera overlay if requested
	if withCamera {
		args = append(args,
			"-f", "avfoundation",
			"-i", "0", // Camera device
		)
	}

	// Video codec and quality settings
	args = append(args,
		"-c:v", "libx264",
		"-preset", "fast",
		"-crf", "18", // High quality
		"-pix_fmt", "yuv420p",
	)

	// Audio settings
	if withAudio {
		args = append(args,
			"-c:a", "aac",
			"-b:a", "128k",
		)
	}

	// Camera overlay filter
	if withCamera {
		args = append(args,
			"-filter_complex", "[0:v][1:v]overlay=W-w-10:10[out]",
			"-map", "[out]",
		)
		if withAudio {
			args = append(args, "-map", "0:a")
		}
	}

	// Frame rate
	args = append(args, "-r", "30")

	// Output file
	args = append(args, "-y", output)

	cmd := exec.Command("ffmpeg", args...)
	return cmd
}

func stopRecording(cmd *exec.Cmd) error {
	// Send interrupt signal to ffmpeg
	if cmd.Process != nil {
		// ffmpeg responds to SIGINT gracefully
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

	// Use ffmpeg to list AVFoundation devices
	cmd := exec.Command("ffmpeg", "-f", "avfoundation", "-list_devices", "true", "-i", "")
	output, _ := cmd.CombinedOutput()

	// Parse and display devices
	lines := strings.Split(string(output), "\n")
	videoDevices := false
	audioDevices := false

	fmt.Println("\n🎥 Video Devices:")
	for _, line := range lines {
		if strings.Contains(line, "AVFoundation video devices") {
			videoDevices = true
			audioDevices = false
			continue
		}
		if strings.Contains(line, "AVFoundation audio devices") {
			videoDevices = false
			audioDevices = true
			fmt.Println("\n🎤 Audio Devices:")
			continue
		}

		if (videoDevices || audioDevices) && strings.Contains(line, "]") {
			parts := strings.Split(line, "] ")
			if len(parts) > 1 {
				fmt.Printf("  %s\n", parts[1])
			}
		}
	}

	// Show screen capture devices
	fmt.Println("\n📺 Screen Capture:")
	fmt.Println("  Screen 1 (Main Display)")
	fmt.Println("  All Screens")

	fmt.Println("\n💡 Usage Examples:")
	fmt.Println("  # Record screen only")
	fmt.Println("  screen-recorder -output recording.mp4")
	fmt.Println("")
	fmt.Println("  # Record with audio")
	fmt.Println("  screen-recorder -output recording.mp4 -audio")
	fmt.Println("")
	fmt.Println("  # Record with camera overlay")
	fmt.Println("  screen-recorder -output recording.mp4 -camera -audio")
	fmt.Println("")
	fmt.Println("  # Record for 30 seconds")
	fmt.Println("  screen-recorder -output recording.mp4 -duration 30s")
}