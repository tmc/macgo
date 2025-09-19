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
	"strings"
	"time"

	macgo "github.com/tmc/misc/macgo"
)

const version = "v2.6.0"

var (
	output      = flag.String("output", "", "output file path (default: screenshot-TIMESTAMP.png)")
	delay       = flag.Int("delay", 0, "delay in seconds before capture")
	window      = flag.Bool("window", false, "capture a window interactively")
	selection   = flag.Bool("selection", false, "capture a selection interactively")
	display     = flag.Int("display", 0, "display to capture (0 for main display)")
	list        = flag.String("list", "", "list windows or apps (windows|apps)")
	app         = flag.String("app", "", "capture specific app by name")
	windowID    = flag.Int("windowid", 0, "capture specific window by ID")
	showVersion = flag.Bool("version", false, "show version")
	help        = flag.Bool("help", false, "show help")
)

func main() {
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	if *showVersion {
		fmt.Printf("Screen Capture %s\n", version)
		return
	}

	// Handle listing commands
	if *list != "" {
		switch *list {
		case "windows":
			listWindows()
		case "apps":
			listApps()
		default:
			fmt.Printf("Invalid list option: %s (use 'windows' or 'apps')\n", *list)
		}
		return
	}

	// Request screen capture permissions with macgo
	cfg := &macgo.Config{
		//BundleID:    "screen-capture.screen-capture",
		Permissions: []macgo.Permission{macgo.Files},
		Version:     version,
		Custom: []string{
			"com.apple.security.device.capture",
			"com.apple.security.device.screen-capture",
		},
		Debug:     os.Getenv("MACGO_DEBUG") == "1",
		AdHocSign: true, // Enable ad-hoc signing to ensure correct identifier
	}

	err := macgo.Start(cfg)
	if err != nil {
		log.Fatalf("Failed to start macgo: %v", err)
	}

	fmt.Printf("Screen Capture - macgo %s\n", version)
	fmt.Println("============================")
	fmt.Println()

	// Generate output filename if not specified
	outputFile := *output
	if outputFile == "" {
		timestamp := time.Now().Format("2006-01-02-150405")
		outputFile = fmt.Sprintf("/tmp/screenshot-%s.png", timestamp)
	}

	// Make output path absolute
	if !filepath.IsAbs(outputFile) {
		outputFile = filepath.Join("/tmp", outputFile)
	}

	// Add delay if specified
	if *delay > 0 {
		fmt.Printf("⏱️  Waiting %d seconds before capture...\n", *delay)
		time.Sleep(time.Duration(*delay) * time.Second)
	}

	// Build screencapture command
	args := []string{}

	if *windowID > 0 {
		// Capture specific window by ID
		args = append(args, fmt.Sprintf("-l%d", *windowID))
		fmt.Printf("🪟 Capturing window ID %d...\n", *windowID)
	} else if *app != "" {
		// Capture app window by bringing it to front first
		fmt.Printf("🔍 Finding app: %s\n", *app)
		if err := bringAppToFront(*app); err == nil {
			time.Sleep(1 * time.Second) // Give app time to come to front
			// Try window ID first, but immediately fall back if it fails
			windowID, err := getAppWindowID(*app)
			if err == nil && windowID != "" {
				// Test if window ID capture will work by trying a quick capture first
				testArgs := []string{fmt.Sprintf("-l%s", windowID), "/tmp/test-capture.png"}
				testCmd := exec.Command("screencapture", testArgs...)
				if testCmd.Run() == nil {
					// Window ID works, use it
					args = append(args, fmt.Sprintf("-l%s", windowID))
					fmt.Printf("📸 Capturing %s window (ID: %s)...\n", *app, windowID)
					// Clean up test file
					os.Remove("/tmp/test-capture.png")
				} else {
					// Window ID failed, use non-interactive whole screen capture
					args = append(args, "-x") // No sound
					fmt.Printf("📸 Capturing %s (full screen - window ID failed)...\n", *app)
				}
			} else {
				// Couldn't get window ID, use full screen capture
				args = append(args, "-x") // No sound
				fmt.Printf("📸 Capturing %s (full screen - no window ID)...\n", *app)
			}
		} else {
			fmt.Printf("⚠️  Warning: Could not activate %s: %v\n", *app, err)
			args = append(args, "-w")
			fmt.Println("🖱️  Click on a window to capture...")
		}
	} else if *window {
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

func listWindows() {
	fmt.Println("🪟 Listing Windows:")
	fmt.Println("==================")

	// Use AppleScript to get window information
	script := `
tell application "System Events"
	set windowList to ""
	repeat with proc in (every process whose visible is true)
		try
			repeat with win in windows of proc
				set windowList to windowList & "App: " & name of proc & " | Window: " & name of win & " | ID: " & id of win & "\n"
			end repeat
		end try
	end repeat
	return windowList
end tell`

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error listing windows: %v\n", err)
		fmt.Println("\nTry using: screencapture -l <window_id> to capture a specific window")
		return
	}

	fmt.Println(string(output))
	fmt.Println("\nUse -windowid <ID> to capture a specific window")
}

func listApps() {
	fmt.Println("📱 Listing Running Apps:")
	fmt.Println("=======================")

	// Use AppleScript to get running applications
	script := `
tell application "System Events"
	set appList to ""
	repeat with proc in (every process whose visible is true)
		set appList to appList & name of proc & "\n"
	end repeat
	return appList
end tell`

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error listing apps: %v\n", err)
		return
	}

	fmt.Println(string(output))
	fmt.Println("\nUse -app '<app name>' to capture a specific app's window")
}

func bringAppToFront(appName string) error {
	// Use AppleScript to activate the app
	script := fmt.Sprintf(`tell application "%s" to activate`, appName)
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

func getAppWindowID(appName string) (string, error) {
	// Use AppleScript to get the window ID of the frontmost window for the app
	script := fmt.Sprintf(`
tell application "%s"
	if (count of windows) > 0 then
		return id of window 1
	else
		return ""
	end if
end tell`, appName)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

func showHelp() {
	fmt.Printf("Screen Capture - macgo %s\n", version)
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
	fmt.Println("  ./screen-capture -list windows             # List all windows with IDs")
	fmt.Println("  ./screen-capture -list apps                # List all running apps")
	fmt.Println("  ./screen-capture -app 'System Settings'    # Capture System Settings window")
	fmt.Println("  ./screen-capture -windowid 1234            # Capture window with ID 1234")
	fmt.Println()
	fmt.Println("Permissions:")
	fmt.Println("  This tool requests Screen Recording permission from macOS")
	fmt.Println("  You may be prompted to grant permission in System Settings")
	fmt.Println()
}
