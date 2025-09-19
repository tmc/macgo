// Package main demonstrates screen capture using the v2 macgo API.
// This example wraps the macOS screencapture tool with proper permissions.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	macgo "github.com/tmc/misc/macgo/v2"
)

var (
	output    = flag.String("output", "", "output file path (default: screenshot-TIMESTAMP.png)")
	delay     = flag.Int("delay", 0, "delay in seconds before capture")
	window    = flag.Bool("window", false, "capture a window interactively")
	selection = flag.Bool("selection", false, "capture a selection interactively")
	display   = flag.Int("display", 0, "display to capture (0 for main display)")
	help      = flag.Bool("help", false, "show help")
)

func main() {
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	// Request screen capture permissions with macgo
	cfg := &macgo.Config{
		AppName:     "ScreenCapture",
		Permissions: []macgo.Permission{macgo.Files},
		Custom: []string{
			"com.apple.security.device.capture",
			"com.apple.security.device.screen-capture",
		},
		Debug: os.Getenv("MACGO_DEBUG") == "1",
	}

	err := macgo.Start(cfg)
	if err != nil {
		log.Fatalf("Failed to start macgo: %v", err)
	}

	fmt.Println("Screen Capture - macgo v2")
	fmt.Println("=========================")
	fmt.Println()

	// Generate output filename if not specified
	outputFile := *output
	if outputFile == "" {
		timestamp := time.Now().Format("2006-01-02-150405")
		outputFile = fmt.Sprintf("screenshot-%s.png", timestamp)
	}

	// Make output path absolute
	if !filepath.IsAbs(outputFile) {
		wd, _ := os.Getwd()
		outputFile = filepath.Join(wd, outputFile)
	}

	// Add delay if specified
	if *delay > 0 {
		fmt.Printf("⏱️  Waiting %d seconds before capture...\n", *delay)
		time.Sleep(time.Duration(*delay) * time.Second)
	}

	// Build screencapture command
	args := []string{}

	if *window {
		args = append(args, "-w")
		fmt.Println("🖱️  Click on a window to capture...")
	} else if *selection {
		args = append(args, "-s")
		fmt.Println("🖱️  Select an area to capture...")
	} else {
		if *display > 0 {
			args = append(args, fmt.Sprintf("-D%d", *display))
		}
		fmt.Printf("📺 Capturing display %d...\n", *display)
	}

	// Add output file
	args = append(args, outputFile)

	// Execute screencapture
	cmd := exec.Command("screencapture", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("🎯 Running: screencapture %v\n", args)

	err = cmd.Run()
	if err != nil {
		log.Fatalf("Screen capture failed: %v", err)
	}

	// Check if file was created
	if _, err := os.Stat(outputFile); err == nil {
		fmt.Printf("✅ Screenshot saved to: %s\n", outputFile)

		// Show file info
		if info, err := os.Stat(outputFile); err == nil {
			fmt.Printf("📄 File size: %d bytes\n", info.Size())
		}
	} else {
		fmt.Printf("❌ Failed to create screenshot: %v\n", err)
	}
}

func showHelp() {
	fmt.Println("Screen Capture - macgo v2")
	fmt.Println("Wraps macOS screencapture tool with proper permissions")
	fmt.Println()
	fmt.Println("Usage:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ./screen-capture                           # Capture main display")
	fmt.Println("  ./screen-capture -window                   # Capture a window interactively")
	fmt.Println("  ./screen-capture -selection                # Capture a selection")
	fmt.Println("  ./screen-capture -delay 3                  # Wait 3 seconds before capture")
	fmt.Println("  ./screen-capture -output ~/Desktop/shot.png # Save to specific file")
	fmt.Println("  ./screen-capture -display 2                # Capture display 2")
	fmt.Println()
	fmt.Println("Permissions:")
	fmt.Println("  This tool requests Screen Recording permission from macOS")
	fmt.Println("  You may be prompted to grant permission in System Settings")
	fmt.Println()
}