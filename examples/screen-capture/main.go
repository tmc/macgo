// Package main wraps macOS screencapture with window support.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/tmc/macgo"
)

// fatal logs the error and exits, ensuring macgo.Cleanup() is called first.
// This is necessary because log.Fatalf calls os.Exit which bypasses defer statements.
func fatal(format string, args ...interface{}) {
	macgo.Cleanup()
	log.Fatalf(format, args...)
}

var (
	output            = flag.String("output", "", "output file path (default: /tmp/screenshot-TIMESTAMP.png)")
	delay             = flag.Int("delay", 0, "delay in seconds before capture")
	windowID          = flag.String("window", "", "capture window by ID (non-interactive)")
	windowIDShort     = flag.String("l", "", "capture window by ID (alias for -window)")
	windowInteractive = flag.Bool("window-interactive", false, "capture a window interactively")
	interactive       = flag.Bool("interactive", false, "show numbered list of windows for selection")
	selection         = flag.Bool("selection", false, "capture a selection interactively")
	display           = flag.Int("display", 0, "display to capture (0 for main display)")
	region            = flag.String("region", "", "capture region as x,y,width,height")
	app               = flag.String("app", "", "capture window of specified application")
	listWindows       = flag.Bool("list-windows", false, "list available windows")
	includeOffscreen  = flag.Bool("include-offscreen", false, "include minimized, hidden, or windows on different Spaces")
	verbose           = flag.Bool("verbose", false, "enable verbose output")
	help              = flag.Bool("help", false, "show help")
	examples          = flag.Bool("examples", false, "show comprehensive examples and use cases")

	// Retry configuration
	retryAttempts  = flag.Int("retry-attempts", 3, "number of retry attempts for transient failures")
	retryDelay     = flag.Int("retry-delay", 500, "initial retry delay in milliseconds")
	retryMaxDelay  = flag.Int("retry-max-delay", 5000, "maximum retry delay in milliseconds")
	retryBackoff   = flag.Float64("retry-backoff", 2.0, "retry backoff multiplier (exponential backoff)")
)

type WindowInfo struct {
	WindowID  int32  `json:"window_id"`
	OwnerPID  int32  `json:"owner_pid"`
	OwnerName string `json:"owner_name"`
	DisplayID uint32 `json:"display_id"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Width     float64 `json:"width"`
	Height    float64 `json:"height"`
}

// getWindowInfo retrieves window information using list-app-windows
func getWindowInfo(windowID, appName string) (windowIDOut, owner string, err error) {
	if _, err := exec.LookPath("list-app-windows"); err != nil {
		return "", "", fmt.Errorf("list-app-windows not found in PATH")
	}

	var cmd *exec.Cmd
	args := []string{"-json"}
	if *includeOffscreen {
		args = append(args, "-include-offscreen")
	}
	if appName != "" {
		args = append(args, "-app", appName)
		cmd = exec.Command("list-app-windows", args...)
	} else if windowID != "" {
		cmd = exec.Command("list-app-windows", args...)
	} else {
		return "", "", fmt.Errorf("either windowID or appName must be specified")
	}

	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to run list-app-windows: %v", err)
	}

	var windows []WindowInfo
	if err := json.Unmarshal(output, &windows); err != nil {
		return "", "", fmt.Errorf("failed to parse JSON: %v", err)
	}

	if len(windows) == 0 {
		if appName != "" {
			return "", "", fmt.Errorf("application '%s' not found or has no windows\n\nTroubleshooting:\n  - Verify the application name is correct\n  - Check if the app is running: ps aux | grep '%s'\n  - List available apps: screen-capture -list-windows\n  - Note: App names are case-sensitive (e.g., 'Safari', not 'safari')", appName, appName)
		}
		return "", "", fmt.Errorf("no windows found")
	}

	// If searching by window ID, find matching window
	if windowID != "" {
		targetID, _ := strconv.ParseInt(windowID, 10, 32)
		for _, w := range windows {
			if w.WindowID == int32(targetID) {
				return fmt.Sprintf("%d", w.WindowID), w.OwnerName, nil
			}
		}
		return "", "", fmt.Errorf("window not found")
	}

	// Return first window
	w := windows[0]
	return fmt.Sprintf("%d", w.WindowID), w.OwnerName, nil
}

// getWindowInfoByIndex retrieves window information for the Nth window of an app (1-indexed)
func getWindowInfoByIndex(appName string, index int) (windowIDOut, owner string, err error) {
	if _, err := exec.LookPath("list-app-windows"); err != nil {
		return "", "", fmt.Errorf("list-app-windows not found in PATH")
	}

	args := []string{"-json", "-app", appName}
	if *includeOffscreen {
		args = append(args, "-include-offscreen")
	}
	cmd := exec.Command("list-app-windows", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to run list-app-windows: %v", err)
	}

	var windows []WindowInfo
	if err := json.Unmarshal(output, &windows); err != nil {
		return "", "", fmt.Errorf("failed to parse JSON: %v", err)
	}

	if len(windows) == 0 {
		return "", "", fmt.Errorf("application '%s' not found or has no windows\n\nTroubleshooting:\n  - Verify the application name is correct\n  - Check if the app is running: ps aux | grep '%s'\n  - List available apps: screen-capture -list-windows\n  - Note: App names are case-sensitive (e.g., 'Safari', not 'safari')", appName, appName)
	}

	if index < 1 || index > len(windows) {
		return "", "", fmt.Errorf("window index %d out of range (app has %d windows)", index, len(windows))
	}

	w := windows[index-1]
	return fmt.Sprintf("%d", w.WindowID), w.OwnerName, nil
}

// tryListAppWindows attempts to use list-app-windows tool, offering to install it if not found
func tryListAppWindows() error {
	// Check if list-app-windows is already available
	if _, err := exec.LookPath("list-app-windows"); err == nil {
		cmd := exec.Command("list-app-windows")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	// Check if go is available
	goPath, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("list-app-windows not found and go not available")
	}

	// Prompt user to install
	fmt.Fprintf(os.Stderr, "\nlist-app-windows tool not found in PATH.\n")
	fmt.Fprintf(os.Stderr, "This tool provides accurate CGWindowIDs for use with screencapture -l.\n")
	fmt.Fprintf(os.Stderr, "\nInstall list-app-windows? [Y/n]: ")

	var response string
	fmt.Scanln(&response)
	response = strings.ToLower(strings.TrimSpace(response))

	// Default to yes if empty or starts with 'y'
	if response == "" || response == "y" || response == "yes" {
		fmt.Fprintf(os.Stderr, "Installing list-app-windows...\n")

		// Get the path to list-app-windows source
		// Assuming we're in examples/screen-capture, go up to find list-app-windows
		listAppWindowsPath := "../list-app-windows"

		installCmd := exec.Command(goPath, "install")
		installCmd.Dir = listAppWindowsPath
		installCmd.Env = append(os.Environ(), "CGO_ENABLED=1")
		installCmd.Stdout = os.Stderr
		installCmd.Stderr = os.Stderr

		if err := installCmd.Run(); err != nil {
			return fmt.Errorf("failed to install list-app-windows: %v", err)
		}

		fmt.Fprintf(os.Stderr, "list-app-windows installed successfully!\n\n")

		// Now try to run it
		cmd := exec.Command("list-app-windows")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	return fmt.Errorf("user declined installation")
}

// selectWindowInteractively shows a numbered list of windows and lets the user select one
func selectWindowInteractively() (windowID, owner string, err error) {
	// Check if list-app-windows is available
	if _, err := exec.LookPath("list-app-windows"); err != nil {
		return "", "", fmt.Errorf("list-app-windows not found in PATH (required for interactive mode)")
	}

	// Get all windows
	args := []string{"-json"}
	if *includeOffscreen {
		args = append(args, "-include-offscreen")
	}
	cmd := exec.Command("list-app-windows", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to get window list: %v", err)
	}

	var windows []WindowInfo
	if err := json.Unmarshal(output, &windows); err != nil {
		return "", "", fmt.Errorf("failed to parse window list: %v", err)
	}

	if len(windows) == 0 {
		return "", "", fmt.Errorf("no windows found")
	}

	// Display numbered list of windows
	fmt.Fprintf(os.Stderr, "\nAvailable windows:\n\n")
	for i, w := range windows {
		dims := fmt.Sprintf("%.0fx%.0f", w.Width, w.Height)
		fmt.Fprintf(os.Stderr, "  %2d. %-20s  %s  (Window ID: %d, Display: %d)\n",
			i+1, w.OwnerName, dims, w.WindowID, w.DisplayID)
	}

	// Prompt for selection
	fmt.Fprintf(os.Stderr, "\nSelect a window (1-%d) or 'q' to quit: ", len(windows))

	var input string
	if _, err := fmt.Scanln(&input); err != nil {
		return "", "", fmt.Errorf("failed to read input: %v", err)
	}

	input = strings.TrimSpace(input)
	if input == "q" || input == "quit" {
		return "", "", fmt.Errorf("user cancelled selection")
	}

	// Parse selection
	selection, err := strconv.Atoi(input)
	if err != nil || selection < 1 || selection > len(windows) {
		return "", "", fmt.Errorf("invalid selection: must be between 1 and %d", len(windows))
	}

	w := windows[selection-1]
	return fmt.Sprintf("%d", w.WindowID), w.OwnerName, nil
}

// permissionConfig holds configuration for permission waiting
type permissionConfig struct {
	maxAttempts int
	baseDelay   time.Duration
	maxDelay    time.Duration
	timeout     time.Duration
}

// getPermissionConfig returns permission configuration from environment variables
func getPermissionConfig() permissionConfig {
	config := permissionConfig{
		maxAttempts: 10,
		baseDelay:   500 * time.Millisecond,
		maxDelay:    5 * time.Second,
		timeout:     60 * time.Second, // Default 60 second total timeout
	}

	// Allow environment variable override
	if envAttempts := os.Getenv("SCREENCAPTURE_PERMISSION_ATTEMPTS"); envAttempts != "" {
		if attempts, err := strconv.Atoi(envAttempts); err == nil && attempts > 0 {
			config.maxAttempts = attempts
		}
	}
	if envDelay := os.Getenv("SCREENCAPTURE_PERMISSION_DELAY"); envDelay != "" {
		if delay, err := strconv.Atoi(envDelay); err == nil && delay > 0 {
			config.baseDelay = time.Duration(delay) * time.Millisecond
		}
	}
	if envMaxDelay := os.Getenv("SCREENCAPTURE_PERMISSION_MAX_DELAY"); envMaxDelay != "" {
		if maxDelay, err := strconv.Atoi(envMaxDelay); err == nil && maxDelay > 0 {
			config.maxDelay = time.Duration(maxDelay) * time.Millisecond
		}
	}
	if envTimeout := os.Getenv("SCREENCAPTURE_PERMISSION_TIMEOUT"); envTimeout != "" {
		if timeout, err := strconv.Atoi(envTimeout); err == nil && timeout > 0 {
			config.timeout = time.Duration(timeout) * time.Second
		}
	}

	return config
}

// waitForScreenCapturePermission waits for TCC screen capture permission to be granted
// It performs test captures and provides clear feedback to the user
// Returns an error if permission is not granted within the configured timeout
func waitForScreenCapturePermission(pid int) error {
	config := getPermissionConfig()

	startTime := time.Now()
	attempt := 0

	for {
		attempt++
		elapsed := time.Since(startTime)

		// Check timeout first
		if elapsed >= config.timeout {
			fmt.Fprintf(os.Stderr, "\n\n❌ Timeout: Screen capture permission not granted after %.1f seconds\n", elapsed.Seconds())
			fmt.Fprintf(os.Stderr, "   Please check System Settings → Privacy & Security → Screen Recording\n")
			fmt.Fprintf(os.Stderr, "   Ensure your terminal or application is listed and enabled\n")
			fmt.Fprintf(os.Stderr, "\n   You can configure the timeout with SCREENCAPTURE_PERMISSION_TIMEOUT (default: 60 seconds)\n")
			return fmt.Errorf("screen capture permission timeout after %.1f seconds", elapsed.Seconds())
		}

		// Check attempt limit (safety net in case timeout is very large)
		if attempt > config.maxAttempts {
			fmt.Fprintf(os.Stderr, "\n\n❌ Screen capture permission not granted after %d attempts (%.1f seconds)\n", config.maxAttempts, elapsed.Seconds())
			fmt.Fprintf(os.Stderr, "   Please check System Settings → Privacy & Security → Screen Recording\n")
			fmt.Fprintf(os.Stderr, "   Ensure your terminal or application is listed and enabled\n")
			return fmt.Errorf("screen capture permission not available after %d attempts", config.maxAttempts)
		}

		// Try a simple test capture to clipboard (avoids multi-monitor coordinate issues)
		// Use -x flag to suppress the snapshot sound during permission check
		// Use -c flag to capture to clipboard (no file, no coordinates needed)
		cmd := exec.Command("screencapture", "-x", "-c")
		cmd.Stdout = nil
		cmd.Stderr = nil

		err := cmd.Run()

		if err == nil {
			// Clipboard capture succeeded - permission is granted
			if os.Getenv("MACGO_DEBUG") == "1" {
				fmt.Fprintf(os.Stderr, "[screen-capture:%d] Screen capture permission verified (%.1fs elapsed)\n", pid, elapsed.Seconds())
			}
			return nil
		}

		// Permission not granted yet, provide feedback
		if attempt == 1 {
			fmt.Fprintf(os.Stderr, "\n⚠️  Waiting for Screen Recording permission...\n")
			fmt.Fprintf(os.Stderr, "   Please grant permission in System Settings → Privacy & Security → Screen Recording\n")
			inBundle := os.Getenv("MACGO_IN_BUNDLE") == "1"
			if !inBundle {
				fmt.Fprintf(os.Stderr, "   The permission dialog should appear automatically.\n")
			}
			fmt.Fprintf(os.Stderr, "   Timeout: %.0f seconds\n", config.timeout.Seconds())
			fmt.Fprintf(os.Stderr, "\n")
		}

		// Print waiting indicator with elapsed time
		dots := strings.Repeat(".", (attempt % 10))
		if dots == "" {
			dots = "."
		}
		timeRemaining := config.timeout - elapsed
		fmt.Fprintf(os.Stderr, "\r   Waiting%s (%.1fs elapsed, %.1fs remaining)   ",
			dots, elapsed.Seconds(), timeRemaining.Seconds())

		// Exponential backoff with jitter
		delay := time.Duration(float64(config.baseDelay) * (1 + float64(attempt)*0.5))
		if delay > config.maxDelay {
			delay = config.maxDelay
		}

		// Don't sleep longer than remaining timeout
		if elapsed+delay > config.timeout {
			delay = config.timeout - elapsed
			if delay <= 0 {
				continue // Will exit on next iteration
			}
		}

		time.Sleep(delay)
	}
}

// retryConfig holds retry configuration
type retryConfig struct {
	maxAttempts int
	baseDelay   time.Duration
	maxDelay    time.Duration
	backoff     float64
}

// getRetryConfig returns retry configuration from flags and environment variables
func getRetryConfig() retryConfig {
	config := retryConfig{
		maxAttempts: *retryAttempts,
		baseDelay:   time.Duration(*retryDelay) * time.Millisecond,
		maxDelay:    time.Duration(*retryMaxDelay) * time.Millisecond,
		backoff:     *retryBackoff,
	}

	// Allow environment variable override
	if envAttempts := os.Getenv("SCREENCAPTURE_RETRY_ATTEMPTS"); envAttempts != "" {
		if attempts, err := strconv.Atoi(envAttempts); err == nil && attempts >= 0 {
			config.maxAttempts = attempts
		}
	}
	if envDelay := os.Getenv("SCREENCAPTURE_RETRY_DELAY"); envDelay != "" {
		if delay, err := strconv.Atoi(envDelay); err == nil && delay >= 0 {
			config.baseDelay = time.Duration(delay) * time.Millisecond
		}
	}
	if envMaxDelay := os.Getenv("SCREENCAPTURE_RETRY_MAX_DELAY"); envMaxDelay != "" {
		if maxDelay, err := strconv.Atoi(envMaxDelay); err == nil && maxDelay >= 0 {
			config.maxDelay = time.Duration(maxDelay) * time.Millisecond
		}
	}
	if envBackoff := os.Getenv("SCREENCAPTURE_RETRY_BACKOFF"); envBackoff != "" {
		if backoff, err := strconv.ParseFloat(envBackoff, 64); err == nil && backoff >= 1.0 {
			config.backoff = backoff
		}
	}

	return config
}

// isTransientError checks if an error is likely transient and worth retrying
func isTransientError(err error, outputFile string) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	// Check if output file exists despite error (sometimes screencapture returns error but succeeds)
	if info, statErr := os.Stat(outputFile); statErr == nil && info.Size() > 0 {
		return false // File created successfully, not a transient error
	}

	// Common transient error patterns
	transientPatterns := []string{
		"window",           // Window-related errors (closed, minimized, invalid ID)
		"communication",    // WindowServer communication issues
		"timeout",          // Timeout errors
		"busy",             // System busy
		"temporarily",      // Temporary unavailable
		"locked",           // Screen locked
		"sleep",            // System in sleep mode
		"invalid window",   // Invalid window ID (window closed/changed)
		"CGError",          // Core Graphics errors (often transient)
		"kCGError",         // Core Graphics error codes
	}

	errLower := strings.ToLower(errMsg)
	for _, pattern := range transientPatterns {
		if strings.Contains(errLower, pattern) {
			return true
		}
	}

	// Exit code checking for common transient failures
	if exitErr, ok := err.(*exec.ExitError); ok {
		// Exit codes that might be transient:
		// - Non-zero exit without specific message often indicates temporary failure
		exitCode := exitErr.ExitCode()
		// Codes like 1-3 are often transient, codes like 127 (command not found) are not
		if exitCode > 0 && exitCode < 10 {
			return true
		}
	}

	return false
}

// executeWithRetry executes screencapture command with retry logic
func executeWithRetry(args []string, outputFile string, pid int) error {
	config := getRetryConfig()

	// If retries disabled, execute once
	if config.maxAttempts <= 1 {
		return executeSingleCapture(args, outputFile, pid)
	}

	var lastErr error
	for attempt := 1; attempt <= config.maxAttempts; attempt++ {
		if attempt > 1 {
			verboseLog("retry attempt %d/%d", attempt, config.maxAttempts)
			if os.Getenv("MACGO_DEBUG") == "1" {
				fmt.Fprintf(os.Stderr, "[screen-capture:%d] Retry attempt %d/%d\n", pid, attempt, config.maxAttempts)
			}
		}

		err := executeSingleCapture(args, outputFile, pid)

		if err == nil {
			// Success!
			if attempt > 1 {
				verboseLog("succeeded on retry attempt %d", attempt)
				if os.Getenv("MACGO_DEBUG") == "1" {
					fmt.Fprintf(os.Stderr, "[screen-capture:%d] ✓ Succeeded on retry attempt %d\n", pid, attempt)
				}
			}
			return nil
		}

		lastErr = err

		// Check if this is a transient error worth retrying
		if !isTransientError(err, outputFile) {
			verboseLog("non-transient error, not retrying: %v", err)
			if os.Getenv("MACGO_DEBUG") == "1" {
				fmt.Fprintf(os.Stderr, "[screen-capture:%d] Non-transient error (not retrying): %v\n", pid, err)
			}
			return err
		}

		// Don't sleep after last attempt
		if attempt < config.maxAttempts {
			// Calculate delay with exponential backoff
			delay := time.Duration(float64(config.baseDelay) * (1.0 + float64(attempt-1)*(*retryBackoff-1.0)))
			if delay > config.maxDelay {
				delay = config.maxDelay
			}

			verboseLog("transient error detected, retrying in %v: %v", delay, err)
			if os.Getenv("MACGO_DEBUG") == "1" {
				fmt.Fprintf(os.Stderr, "[screen-capture:%d] ⚠ Transient error (attempt %d/%d), retrying in %v: %v\n",
					pid, attempt, config.maxAttempts, delay, err)
			}

			time.Sleep(delay)
		}
	}

	// All retries exhausted
	verboseLog("all retry attempts exhausted")
	if os.Getenv("MACGO_DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "[screen-capture:%d] ✗ All %d retry attempts exhausted\n", pid, config.maxAttempts)
	}

	return fmt.Errorf("screen capture failed after %d attempts: %w", config.maxAttempts, lastErr)
}

// executeSingleCapture performs a single screen capture attempt
func executeSingleCapture(args []string, outputFile string, pid int) error {
	verboseLog("executing: screencapture %v", args)
	if os.Getenv("MACGO_DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "[screen-capture:%d] Executing: screencapture %v\n", pid, args)
	}

	cmd := exec.Command("screencapture", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	verboseLog("running screencapture command")
	if os.Getenv("MACGO_DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "[screen-capture:%d] Running screencapture command\n", pid)
	}

	return cmd.Run()
}

func init() {
	// Check if -list-windows flag is present
	// If so, skip macgo initialization and run list-app-windows directly
	for _, arg := range os.Args[1:] {
		if arg == "-list-windows" || arg == "--list-windows" {
			// Run list-app-windows directly without macgo
			if err := tryListAppWindows(); err != nil {
				// Fallback to lsappinfo
				fmt.Fprintln(os.Stderr, "Note: Using lsappinfo (window IDs not available)")
				fmt.Fprintln(os.Stderr, "Install list-app-windows for accurate window IDs")
				fmt.Println("\nAvailable applications with windows:")
				cmd := exec.Command("lsappinfo", "list")
				output, err := cmd.Output()
				if err != nil {
					fatal("Failed to list applications: %v", err)
				}

				lines := strings.Split(string(output), "\n")
				var currentApp string
				for _, line := range lines {
					if strings.Contains(line, `"`) && strings.Contains(line, "ASN:") {
						start := strings.Index(line, `"`)
						if start != -1 {
							end := strings.Index(line[start+1:], `"`)
							if end != -1 {
								currentApp = line[start+1 : start+1+end]
							}
						}
					}

					if currentApp != "" && strings.Contains(line, "pid = ") {
						if strings.Contains(line, "type=\"Foreground\"") {
							pidStart := strings.Index(line, "pid = ")
							if pidStart != -1 {
								remaining := line[pidStart+6:]
								pidEnd := strings.Index(remaining, " ")
								if pidEnd != -1 {
									pid := remaining[:pidEnd]
									fmt.Printf("  %s (PID: %s)\n", currentApp, pid)
								}
							}
						}
						if strings.Contains(line, "pid = ") {
							currentApp = ""
						}
					}
				}
				fmt.Println("\nUse: screen-capture -app \"<Application Name>\" /path/to/output.png")
			}
			macgo.Cleanup()
			os.Exit(0)
		}
	}

	// ServicesLauncher (V1) is now the stable default with continuous polling and config-file strategy
	// V2 is experimental - set MACGO_SERVICES_VERSION=2 to use it
	// Both versions now support config-file I/O forwarding with continuous polling

	// Initialize macgo early (before main) to ensure I/O redirection happens first
	// This allows child process output to be properly forwarded through pipes
	pid := os.Getpid()
	if os.Getenv("MACGO_DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "[screen-capture:%d] Starting macgo initialization\n", pid)
	}

	cfg := &macgo.Config{
		Permissions: []macgo.Permission{macgo.Files},
		Custom: []string{
			"com.apple.security.device.screen-capture",
		},
		Debug: os.Getenv("MACGO_DEBUG") == "1",
	}

	if err := macgo.Start(cfg); err != nil {
		fatal("Failed to start macgo: %v", err)
	}

	if os.Getenv("MACGO_DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "[screen-capture:%d] Macgo initialized successfully\n", pid)
	}
}

func setupSignalHandlers() {
	// Create buffered channel for signals
	sigChan := make(chan os.Signal, 1)

	// Register for signals
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	processType := "[parent]"
	if os.Getenv("MACGO_NOBUNDLE") != "1" && os.Getenv("MACGO_IN_BUNDLE") == "1" {
		processType = "[child]"
	}

	go func() {
		for sig := range sigChan {
			switch sig {
			case syscall.SIGQUIT:
				// Print stack traces for all goroutines
				fmt.Fprintf(os.Stderr, "\n%s Received SIGQUIT (PID: %d)\n", processType, os.Getpid())
				fmt.Fprintf(os.Stderr, "%s Printing stack traces:\n", processType)
				buf := make([]byte, 1<<20) // 1MB buffer
				stackLen := runtime.Stack(buf, true)
				fmt.Fprintf(os.Stderr, "%s\n", buf[:stackLen])
				fmt.Fprintf(os.Stderr, "%s Stack trace complete (PID: %d)\n", processType, os.Getpid())
			case syscall.SIGINT:
				fmt.Fprintf(os.Stderr, "\n%s Received SIGINT (PID: %d), exiting gracefully\n", processType, os.Getpid())
				// Start force-kill timer in case we're blocked on I/O
				go func() {
					time.Sleep(100 * time.Millisecond)
					fmt.Fprintf(os.Stderr, "%s Forcing exit after grace period (PID: %d)\n", processType, os.Getpid())
					syscall.Kill(os.Getpid(), syscall.SIGKILL)
				}()
				// Try graceful exit - cleanup first to signal parent
				macgo.Cleanup()
				os.Exit(0)
			case syscall.SIGTERM:
				fmt.Fprintf(os.Stderr, "\n%s Received SIGTERM (PID: %d), exiting gracefully\n", processType, os.Getpid())
				// Start force-kill timer in case we're blocked on I/O
				go func() {
					time.Sleep(100 * time.Millisecond)
					fmt.Fprintf(os.Stderr, "%s Forcing exit after grace period (PID: %d)\n", processType, os.Getpid())
					syscall.Kill(os.Getpid(), syscall.SIGKILL)
				}()
				// Try graceful exit - cleanup first to signal parent
				macgo.Cleanup()
				os.Exit(0)
			}
		}
	}()
}

// verboseLog prints a message if verbose mode is enabled
func verboseLog(format string, args ...interface{}) {
	if *verbose {
		fmt.Fprintf(os.Stderr, "[verbose] "+format+"\n", args...)
	}
}

func main() {
	// Ensure macgo cleanup happens on normal exit (writes done file for parent)
	defer macgo.Cleanup()

	// Set up signal handlers early
	setupSignalHandlers()

	pid := os.Getpid()
	inBundle := os.Getenv("MACGO_IN_BUNDLE") == "1"

	// Debug output - write to both stderr AND a file to ensure we can see if bundle runs
	if os.Getenv("MACGO_DEBUG") == "1" {
		logMsg := ""
		if inBundle {
			logMsg = fmt.Sprintf("[screen-capture:%d] Running in bundle\n", pid)
			logMsg += fmt.Sprintf("[screen-capture:%d] Args: %v\n", pid, os.Args)
			logMsg += fmt.Sprintf("[screen-capture:%d] MACGO_STDOUT_PIPE=%s\n", pid, os.Getenv("MACGO_STDOUT_PIPE"))
			logMsg += fmt.Sprintf("[screen-capture:%d] MACGO_STDERR_PIPE=%s\n", pid, os.Getenv("MACGO_STDERR_PIPE"))
			fmt.Fprintf(os.Stderr, "%s", logMsg)
			// Write to file since stderr might not work with pipes
			os.WriteFile("/tmp/screen-capture-debug.log", []byte(logMsg), 0644)
		} else {
			logMsg = fmt.Sprintf("[screen-capture:%d] Running as parent process\n", pid)
			fmt.Fprintf(os.Stderr, "%s", logMsg)
			os.WriteFile("/tmp/screen-capture-debug.log", []byte(logMsg), 0644)
		}
	}

	flag.Parse()

	verboseLog("screen-capture starting (PID: %d, bundle: %v)", pid, inBundle)
	verboseLog("flags: output=%q, delay=%d, window=%q, app=%q", *output, *delay, *windowID, *app)

	// Support -l as alias for -window (like screencapture -l)
	if *windowIDShort != "" && *windowID == "" {
		*windowID = *windowIDShort
		verboseLog("using -l alias: window=%q", *windowID)
	}

	if *help {
		showHelp()
		return
	}

	if *examples {
		showExamples()
		return
	}

	if *listWindows {
		// Try to use list-app-windows for accurate CGWindowIDs
		if err := tryListAppWindows(); err != nil {
			// Fallback to lsappinfo
			fmt.Fprintln(os.Stderr, "Note: Using lsappinfo (window IDs not available)")
			fmt.Fprintln(os.Stderr, "Install list-app-windows for accurate window IDs")
			fmt.Println("\nAvailable applications with windows:")
			cmd := exec.Command("lsappinfo", "list")
			output, err := cmd.Output()
			if err != nil {
				fatal("Failed to list applications: %v", err)
			}

			lines := strings.Split(string(output), "\n")
			var currentApp string
			for _, line := range lines {
				// Check for application name line (starts with number and has quotes)
				if strings.Contains(line, `"`) && strings.Contains(line, "ASN:") {
					start := strings.Index(line, `"`)
					if start != -1 {
						end := strings.Index(line[start+1:], `"`)
						if end != -1 {
							currentApp = line[start+1 : start+1+end]
						}
					}
				}

				// Check for PID line and app type
				if currentApp != "" && strings.Contains(line, "pid = ") {
					// Only show applications that have windows (Foreground or some UIElements)
					if strings.Contains(line, "type=\"Foreground\"") {
						pidStart := strings.Index(line, "pid = ")
						if pidStart != -1 {
							remaining := line[pidStart+6:]
							pidEnd := strings.Index(remaining, " ")
							if pidEnd != -1 {
								pid := remaining[:pidEnd]
								fmt.Printf("  %s (PID: %s)\n", currentApp, pid)
							}
						}
					}
					// Reset for next app only after processing PID line
					if strings.Contains(line, "pid = ") {
						currentApp = ""
					}
				}
			}
			fmt.Println("\nUse: screen-capture -app \"<Application Name>\" /path/to/output.png")
		}
		return
	}

	// Wait for TCC permissions to be granted with user feedback
	// Note: macgo.Start() is called in init() to ensure early I/O redirection
	verboseLog("checking screen capture permission")
	if err := waitForScreenCapturePermission(pid); err != nil {
		fatal("Screen capture permission not available: %v", err)
	}
	verboseLog("screen capture permission verified")

	// Generate output filename if not specified
	outputFile := *output
	var err error

	// Check for positional argument as output file
	if outputFile == "" && len(flag.Args()) > 0 {
		outputFile = flag.Args()[0]
		verboseLog("using positional argument for output: %q", outputFile)
	}

	if outputFile == "" {
		timestamp := time.Now().Format("2006-01-02-150405")
		outputFile = fmt.Sprintf("/tmp/screenshot-%s.png", timestamp)
		verboseLog("generated output filename: %q", outputFile)
	}

	// Make output path absolute
	if !filepath.IsAbs(outputFile) {
		outputFile = filepath.Join("/tmp", outputFile)
		verboseLog("converted to absolute path: %q", outputFile)
	}

	// Add delay if specified
	if *delay > 0 {
		fmt.Printf("Waiting %d seconds before capture...\n", *delay)
		verboseLog("delaying %d seconds", *delay)
		time.Sleep(time.Duration(*delay) * time.Second)
	}

	// Build screencapture command
	verboseLog("building screencapture command")
	args := []string{}

	if *region != "" {
		coords, err := validateRegion(*region)
		if err != nil {
			fatal("Invalid region: %v", err)
		}
		args = append(args, "-R", coords)
		verboseLog("capture mode: region (%s)", coords)
	} else if *app != "" {
		// Check for multiple windows
		verboseLog("checking application windows for: %q", *app)
		if err := checkApplicationWindows(*app); err != nil {
			fatal("Window validation failed: %v", err)
		}

		// Determine which window to capture
		var winID, owner string
		var winErr error

		// If -window is specified with -app, treat it as an index
		if *windowID != "" {
			// Try to parse as integer index
			if windowIndex, err := strconv.Atoi(*windowID); err == nil {
				verboseLog("looking up window by index: %d", windowIndex)
				winID, owner, winErr = getWindowInfoByIndex(*app, windowIndex)
			} else {
				// Not a number, treat as window ID
				verboseLog("looking up window by ID: %q", *windowID)
				winID, owner, winErr = getWindowInfo(*windowID, *app)
			}
		} else {
			// No -window specified, get first window
			verboseLog("getting first window for app: %q", *app)
			winID, owner, winErr = getWindowInfo("", *app)
		}

		if winErr == nil {
			args = append(args, "-l", winID)
			fmt.Printf("Capturing %s window %s\n", owner, winID)
			verboseLog("window ID: %s, owner: %s", winID, owner)
		} else {
			fatal("Could not get window ID: %v", winErr)
		}
	} else if *interactive {
		// Interactive mode - show numbered list of windows
		verboseLog("starting interactive window selection")
		winID, owner, winErr := selectWindowInteractively()
		if winErr != nil {
			fatal("Interactive selection failed: %v", winErr)
		}
		args = append(args, "-l", winID)
		fmt.Printf("Capturing %s window %s\n", owner, winID)
		verboseLog("selected window ID: %s, owner: %s", winID, owner)
	} else if *windowID != "" {
		args = append(args, "-l", *windowID)
		verboseLog("capture mode: window ID %q", *windowID)

		// Try to get window info
		if _, owner, err := getWindowInfo(*windowID, ""); err == nil {
			fmt.Printf("Capturing %s window %s\n", owner, *windowID)
			verboseLog("window owner: %s", owner)
		}
	} else if *windowInteractive {
		args = append(args, "-w")
		verboseLog("capture mode: interactive window selection")
	} else if *selection {
		args = append(args, "-s")
		verboseLog("capture mode: interactive selection")
	} else {
		if *display > 0 {
			args = append(args, fmt.Sprintf("-D%d", *display))
			verboseLog("capture mode: display %d", *display)
		} else {
			verboseLog("capture mode: full screen (main display)")
		}
	}

	// Add output file
	args = append(args, outputFile)

	// Execute screencapture with retry logic
	err = executeWithRetry(args, outputFile, pid)
	if err != nil {
		if os.Getenv("MACGO_DEBUG") == "1" {
			fmt.Fprintf(os.Stderr, "[screen-capture:%d] Screen capture failed: %v\n", pid, err)
		}
		fatal("Screen capture failed: %v", err)
	}
	verboseLog("screencapture command completed successfully")

	if os.Getenv("MACGO_DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "[screen-capture:%d] Screen capture command completed\n", pid)
	}

	// Check if file was created
	verboseLog("verifying output file: %s", outputFile)
	if info, err := os.Stat(outputFile); err == nil {
		fmt.Printf("Screenshot saved: %s (%d bytes)\n", outputFile, info.Size())
		verboseLog("file size: %d bytes", info.Size())
		if os.Getenv("MACGO_DEBUG") == "1" {
			fmt.Fprintf(os.Stderr, "[screen-capture:%d] File verified: %s (%d bytes)\n", pid, outputFile, info.Size())
		}

		// Write success marker for testing
		if testMarker := os.Getenv("MACGO_TEST_MARKER"); testMarker != "" {
			successMsg := fmt.Sprintf("SUCCESS: %s created (%d bytes)\n", outputFile, info.Size())
			os.WriteFile(testMarker, []byte(successMsg), 0644)
		}
	} else {
		if os.Getenv("MACGO_DEBUG") == "1" {
			fmt.Fprintf(os.Stderr, "[screen-capture:%d] Failed to verify file: %v\n", pid, err)
		}

		// Write failure marker for testing
		if testMarker := os.Getenv("MACGO_TEST_MARKER"); testMarker != "" {
			failMsg := fmt.Sprintf("FAILED: %s not created: %v\n", outputFile, err)
			os.WriteFile(testMarker, []byte(failMsg), 0644)
		}

		fatal("Failed to create screenshot: %v", err)
	}

	if os.Getenv("MACGO_DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "[screen-capture:%d] Exiting successfully\n", pid)
	}

}

// validateRegion ensures region format is correct
func validateRegion(region string) (string, error) {
	parts := strings.Split(region, ",")
	if len(parts) != 4 {
		return "", fmt.Errorf("region must be x,y,width,height")
	}

	for i, part := range parts {
		if _, err := strconv.Atoi(strings.TrimSpace(part)); err != nil {
			return "", fmt.Errorf("invalid coordinate at position %d: %s", i+1, part)
		}
	}

	return region, nil
}

// checkApplicationWindows checks for multiple windows and provides information using list-app-windows
func checkApplicationWindows(appName string) error {
	// Check if list-app-windows is available
	if _, err := exec.LookPath("list-app-windows"); err != nil {
		// If list-app-windows not available, just proceed
		return nil
	}

	// Get window count using list-app-windows
	args := []string{"-json", "-app", appName}
	if *includeOffscreen {
		args = append(args, "-include-offscreen")
	}
	cmd := exec.Command("list-app-windows", args...)
	output, err := cmd.Output()
	if err != nil {
		// If we can't get window count, just proceed silently
		return nil
	}

	var windows []WindowInfo
	if err := json.Unmarshal(output, &windows); err != nil {
		// If we can't parse, just proceed silently
		return nil
	}

	if len(windows) == 0 {
		return fmt.Errorf("application '%s' not found or has no windows\n\nTroubleshooting:\n  - Verify the application name is correct\n  - Check if the app is running: ps aux | grep '%s'\n  - List available apps: screen-capture -list-windows\n  - Note: App names are case-sensitive (e.g., 'Safari', not 'safari')", appName, appName)
	}

	return nil
}

func showHelp() {
	fmt.Println("screen-capture - macOS screen capture tool")
	fmt.Println()
	fmt.Println("DESCRIPTION:")
	fmt.Println("  Powerful CLI for capturing screenshots on macOS with support for:")
	fmt.Println("  • Window capture (by ID or app name)")
	fmt.Println("  • Interactive selection")
	fmt.Println("  • Region and display capture")
	fmt.Println("  • Offscreen window capture")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  screen-capture [options] [output-file]")
	fmt.Println()
	fmt.Println("COMMON EXAMPLES:")
	fmt.Println("  screen-capture                              # Capture entire screen")
	fmt.Println("  screen-capture -list-windows                # List available windows")
	fmt.Println("  screen-capture -window 12345 output.png     # Capture by window ID")
	fmt.Println("  screen-capture -app Safari safari.png       # Capture by app name")
	fmt.Println("  screen-capture -interactive output.png      # Interactive selection")
	fmt.Println("  screen-capture -region 0,0,800,600 reg.png  # Capture region")
	fmt.Println()
	fmt.Println("For comprehensive examples and use cases:")
	fmt.Println("  screen-capture --examples")
	fmt.Println()
	fmt.Println("OPTIONS:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("PERMISSIONS:")
	fmt.Println("  Requires Screen Recording permission")
	fmt.Println("  System Settings → Privacy & Security → Screen Recording")
	fmt.Println()
	fmt.Println("NOTES:")
	fmt.Println("  • Use list-app-windows for accurate window IDs")
	fmt.Println("  • Default output: /tmp/screenshot-TIMESTAMP.png")
	fmt.Println("  • -include-offscreen captures minimized/hidden windows")
	fmt.Println()
}

func showExamples() {
	fmt.Println("screen-capture - Comprehensive Examples")
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println("BASIC OPERATIONS")
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("1. Capture entire screen")
	fmt.Println("   $ screen-capture")
	fmt.Println()
	fmt.Println("2. Capture to specific file")
	fmt.Println("   $ screen-capture ~/Desktop/screenshot.png")
	fmt.Println()
	fmt.Println("3. Capture with delay")
	fmt.Println("   $ screen-capture -delay 5 delayed.png")
	fmt.Println()
	fmt.Println("4. List available windows")
	fmt.Println("   $ screen-capture -list-windows")
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println("WINDOW CAPTURE")
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("5. Capture by window ID")
	fmt.Println("   $ screen-capture -window 12345 window.png")
	fmt.Println()
	fmt.Println("6. Capture by app name")
	fmt.Println("   $ screen-capture -app Safari safari.png")
	fmt.Println()
	fmt.Println("7. Interactive selection")
	fmt.Println("   $ screen-capture -interactive output.png")
	fmt.Println()
	fmt.Println("8. Capture offscreen window")
	fmt.Println("   $ screen-capture -app Safari -include-offscreen minimized.png")
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println("REGION CAPTURE")
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("9. Capture region")
	fmt.Println("   $ screen-capture -region 100,100,800,600 region.png")
	fmt.Println()
	fmt.Println("10. Interactive selection")
	fmt.Println("    $ screen-capture -selection output.png")
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println("SCRIPTING & AUTOMATION")
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("11. Capture with timestamp")
	fmt.Println("    $ screen-capture screenshot-$(date +%Y%m%d-%H%M%S).png")
	fmt.Println()
	fmt.Println("12. Capture and open")
	fmt.Println("    $ screen-capture capture.png && open capture.png")
	fmt.Println()
	fmt.Println("13. Get window ID and capture")
	fmt.Println("    $ WID=$(list-app-windows -app Safari | awk 'NR==2 {print $1}')")
	fmt.Println("    $ screen-capture -window $WID safari.png")
	fmt.Println()
	fmt.Println("14. Periodic screenshots")
	fmt.Println("    $ while true; do")
	fmt.Println("        screen-capture monitoring-$(date +%H%M%S).png")
	fmt.Println("        sleep 60")
	fmt.Println("      done")
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println("REAL-WORLD USE CASES")
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("Bug reporting:")
	fmt.Println("  $ screen-capture -app \"My App\" -delay 3 error.png")
	fmt.Println()
	fmt.Println("Documentation:")
	fmt.Println("  $ screen-capture -app \"System Settings\" settings.png")
	fmt.Println()
	fmt.Println("Testing:")
	fmt.Println("  $ ./run-tests.sh && screen-capture -app Terminal results.png")
	fmt.Println()
	fmt.Println("CI/CD:")
	fmt.Println("  $ screen-capture -app Xcode -include-offscreen build.png")
	fmt.Println()
	fmt.Println("For full help: screen-capture --help")
	fmt.Println()
}
