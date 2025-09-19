package macgo

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"syscall"
)

// startDarwin implements the macOS-specific logic.
func startDarwin(ctx context.Context, cfg *Config) error {
	// Check for permission reset flag
	if os.Getenv("MACGO_RESET_PERMISSIONS") == "1" {
		if err := resetTCCPermissions(cfg); err != nil {
			if cfg.Debug {
				fmt.Fprintf(os.Stderr, "macgo: failed to reset permissions: %v\n", err)
			}
		}
	}

	// Skip if already in app bundle
	if isInAppBundle() {
		if cfg.Debug {
			fmt.Fprintf(os.Stderr, "macgo: already in app bundle\n")
		}
		return nil
	}

	// Skip if disabled
	if os.Getenv("MACGO_NO_RELAUNCH") == "1" {
		if cfg.Debug {
			fmt.Fprintf(os.Stderr, "macgo: relaunch disabled\n")
		}
		return nil
	}

	// Get current executable
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("macgo: get executable: %w", err)
	}

	// Create or reuse bundle
	bundlePath, err := createSimpleBundle(execPath, cfg)
	if err != nil {
		return fmt.Errorf("macgo: bundle operation: %w", err)
	}

	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "macgo: using bundle at %s\n", bundlePath)
		fmt.Fprintf(os.Stderr, "macgo: permissions requested: %v\n", cfg.Permissions)
	}

	// Relaunch in bundle
	return relaunchInBundle(ctx, bundlePath, execPath, cfg)
}

// createSimpleBundle creates a minimal app bundle with the given configuration.
func createSimpleBundle(execPath string, cfg *Config) (string, error) {
	// Determine app name
	appName := cfg.AppName
	if appName == "" {
		appName = filepath.Base(execPath)
		appName = strings.TrimSuffix(appName, filepath.Ext(appName))
	}

	// Clean and limit app name length
	appName = cleanAppName(appName)
	if len(appName) > 251 { // Reserve 4 chars for ".app"
		appName = appName[:251]
	}

	// Determine bundle ID
	bundleID := cfg.BundleID
	if bundleID == "" {
		bundleID = inferBundleID(appName)
	}

	// Determine version
	version := cfg.Version
	if version == "" {
		version = "1.0.0"
	}

	// Determine bundle location - prefer ~/go/bin/ if it exists
	bundleBaseDir := os.TempDir()
	if goPath := os.Getenv("GOPATH"); goPath != "" {
		bundleBaseDir = filepath.Join(goPath, "bin")
	} else if homeDir, err := os.UserHomeDir(); err == nil {
		goBinDir := filepath.Join(homeDir, "go", "bin")
		if _, err := os.Stat(goBinDir); err == nil {
			bundleBaseDir = goBinDir
		}
	}

	// Create bundle directory
	bundleDir := filepath.Join(bundleBaseDir, appName+".app")

	// Check if bundle already exists and should be kept
	if cfg.shouldKeepBundle() {
		if _, err := os.Stat(bundleDir); err == nil {
			if cfg.Debug {
				fmt.Fprintf(os.Stderr, "macgo: reusing existing bundle at %s\n", bundleDir)
			}
			return bundleDir, nil
		}
	} else {
		// Remove old bundle if not keeping it
		if err := os.RemoveAll(bundleDir); err != nil && !os.IsNotExist(err) {
			return "", err
		}
	}

	// Create directory structure
	contentsDir := filepath.Join(bundleDir, "Contents")
	macosDir := filepath.Join(contentsDir, "MacOS")
	if err := os.MkdirAll(macosDir, 0755); err != nil {
		return "", err
	}

	// Copy the executable directly
	execName := filepath.Base(appName)
	destExec := filepath.Join(macosDir, execName)
	if err := copyFile(execPath, destExec); err != nil {
		return "", err
	}
	if err := os.Chmod(destExec, 0755); err != nil {
		return "", err
	}

	// Create Info.plist
	plistPath := filepath.Join(contentsDir, "Info.plist")
	if err := writeInfoPlist(plistPath, appName, bundleID, execName, version); err != nil {
		return "", err
	}

	// Create entitlements if needed (not for ad-hoc signing)
	if (len(cfg.Permissions) > 0 || len(cfg.Custom) > 0) && cfg.CodeSignIdentity != "-" && !cfg.AdHocSign {
		entPath := filepath.Join(contentsDir, "entitlements.plist")
		if err := writeEntitlements(entPath, cfg); err != nil {
			return "", err
		}
	}

	// Code sign the bundle if identity is provided, auto-detect, or ad-hoc
	if cfg.CodeSignIdentity != "" {
		if err := codeSignBundle(bundleDir, cfg); err != nil {
			return "", fmt.Errorf("code signing failed: %w", err)
		}
		if cfg.Debug {
			fmt.Fprintf(os.Stderr, "macgo: code signed with identity: %s\n", cfg.CodeSignIdentity)
		}
	} else if cfg.AdHocSign {
		cfg.CodeSignIdentity = "-"
		if err := codeSignBundle(bundleDir, cfg); err != nil {
			return "", fmt.Errorf("ad-hoc signing failed: %w", err)
		}
		if cfg.Debug {
			fmt.Fprintf(os.Stderr, "macgo: ad-hoc signed\n")
		}
	} else if cfg.AutoSign {
		if identity := findDeveloperID(cfg.Debug); identity != "" {
			cfg.CodeSignIdentity = identity
			if err := codeSignBundle(bundleDir, cfg); err != nil {
				if cfg.Debug {
					fmt.Fprintf(os.Stderr, "macgo: auto-signing failed, continuing unsigned: %v\n", err)
				}
			} else {
				if cfg.Debug {
					fmt.Fprintf(os.Stderr, "macgo: auto-signed with identity: %s\n", identity)
				}
			}
		} else if cfg.Debug {
			fmt.Fprintf(os.Stderr, "macgo: no Developer ID found, creating unsigned bundle\n")
		}
	}

	return bundleDir, nil
}

// inferBundleID creates a reasonable bundle ID from the app name.
func inferBundleID(appName string) string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Path != "" {
		bundleID := strings.ReplaceAll(info.Main.Path, "/", ".")
		// Append the app name to make bundle IDs unique per binary
		bundleID = fmt.Sprintf("%s.%s", bundleID, appName)
		return bundleID
	}
	return fmt.Sprintf("com.macgo.%s", appName)
}

// cleanAppName removes problematic characters from app names.
func cleanAppName(name string) string {
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	name = strings.ReplaceAll(name, ":", "-")
	name = strings.ReplaceAll(name, "*", "-")
	name = strings.ReplaceAll(name, "?", "-")
	name = strings.ReplaceAll(name, "\"", "-")
	name = strings.ReplaceAll(name, "<", "-")
	name = strings.ReplaceAll(name, ">", "-")
	name = strings.ReplaceAll(name, "|", "-")

	var result strings.Builder
	for _, r := range name {
		if r >= 32 && r < 127 {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// relaunchInBundle launches the app bundle.
func relaunchInBundle(ctx context.Context, bundlePath, execPath string, cfg *Config) error {
	// Determine if we should use LaunchServices or direct execution
	needsLaunchServices := false

	// Check if any permissions require TCC (and thus LaunchServices)
	for _, perm := range cfg.Permissions {
		switch perm {
		case Files, Camera, Microphone, Location:
			needsLaunchServices = true
		}
	}

	if cfg.ForceLaunchServices {
		needsLaunchServices = true
		if cfg.Debug {
			fmt.Fprintf(os.Stderr, "macgo: forced LaunchServices via config\n")
		}
	} else if cfg.ForceDirectExecution {
		needsLaunchServices = false
		if cfg.Debug {
			fmt.Fprintf(os.Stderr, "macgo: forced direct execution via config\n")
		}
	} else {
		if os.Getenv("MACGO_FORCE_LAUNCH_SERVICES") == "1" {
			needsLaunchServices = true
		} else if os.Getenv("MACGO_FORCE_DIRECT") == "1" {
			needsLaunchServices = false
		}
	}

	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "macgo: needsLaunchServices: %v\n", needsLaunchServices)
	}

	if !needsLaunchServices {
		if cfg.Debug {
			fmt.Fprintf(os.Stderr, "macgo: using direct execution\n")
		}
		return relaunchDirect(ctx, bundlePath, execPath, cfg)
	}

	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "macgo: using LaunchServices\n")
	}
	return relaunchWithOpen(ctx, bundlePath, cfg)
}

// relaunchDirect directly executes the binary in the bundle.
func relaunchDirect(ctx context.Context, bundlePath, execPath string, cfg *Config) error {
	execName := ""
	if cfg.AppName != "" {
		execName = filepath.Base(cfg.AppName)
	}
	if execName == "" {
		execName = filepath.Base(execPath)
	}

	bundleExec := filepath.Join(bundlePath, "Contents", "MacOS", execName)

	cmd := exec.CommandContext(ctx, bundleExec, os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("macgo: start bundle: %w", err)
	}

	err := cmd.Wait()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("macgo: bundle execution: %w", err)
	}

	os.Exit(0)
	return nil
}

// relaunchWithOpen uses the 'open' command with I/O forwarding.
func relaunchWithOpen(ctx context.Context, bundlePath string, cfg *Config) error {
	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "macgo: launching with open command\n")
	}

	// Create named pipes for I/O forwarding
	pipeDir := filepath.Join(os.TempDir(), fmt.Sprintf("macgo-%d", os.Getpid()))
	if err := os.MkdirAll(pipeDir, 0700); err != nil {
		return fmt.Errorf("macgo: create pipe dir: %w", err)
	}
	defer os.RemoveAll(pipeDir)

	stdinPipe := filepath.Join(pipeDir, "stdin")
	stdoutPipe := filepath.Join(pipeDir, "stdout")
	stderrPipe := filepath.Join(pipeDir, "stderr")

	// Create FIFOs
	for _, pipe := range []string{stdinPipe, stdoutPipe, stderrPipe} {
		if err := syscall.Mkfifo(pipe, 0600); err != nil {
			return fmt.Errorf("macgo: create pipe %s: %w", pipe, err)
		}
	}

	// Build open command with I/O redirection
	args := []string{
		"-a", bundlePath,
		"--wait-apps",
		"--stdin", stdinPipe,
		"--stdout", stdoutPipe,
		"--stderr", stderrPipe,
	}

	// Add command line arguments
	if len(os.Args) > 1 {
		args = append(args, "--args")
		args = append(args, os.Args[1:]...)
	}

	cmd := exec.CommandContext(ctx, "open", args...)

	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "macgo: launching: open %v\n", args)
	}

	// Start the open command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("macgo: start open: %w", err)
	}

	// Set up I/O forwarding goroutines
	errChan := make(chan error, 3)

	// Forward stdin
	go func() {
		w, err := os.OpenFile(stdinPipe, os.O_WRONLY, 0)
		if err != nil {
			errChan <- err
			return
		}
		defer w.Close()
		_, err = io.Copy(w, os.Stdin)
		errChan <- err
	}()

	// Forward stdout
	go func() {
		r, err := os.OpenFile(stdoutPipe, os.O_RDONLY, 0)
		if err != nil {
			errChan <- err
			return
		}
		defer r.Close()
		_, err = io.Copy(os.Stdout, r)
		errChan <- err
	}()

	// Forward stderr
	go func() {
		r, err := os.OpenFile(stderrPipe, os.O_RDONLY, 0)
		if err != nil {
			errChan <- err
			return
		}
		defer r.Close()
		_, err = io.Copy(os.Stderr, r)
		errChan <- err
	}()

	// Wait for open command
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("macgo: open failed: %w", err)
	}

	// The app should handle its own exit
	os.Exit(0)
	return nil
}

// isInAppBundle checks if we're already running inside an app bundle.
func isInAppBundle() bool {
	execPath, err := os.Executable()
	if err != nil {
		return false
	}
	return strings.Contains(execPath, ".app/Contents/MacOS/")
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = dstFile.ReadFrom(srcFile)
	return err
}

// codeSignBundle signs the app bundle.
func codeSignBundle(bundlePath string, cfg *Config) error {
	args := []string{
		"--sign", cfg.CodeSignIdentity,
		"--force",
	}

	if cfg.CodeSignIdentity != "-" {
		args = append(args, "--timestamp")
		args = append(args, "--options", "runtime")
	}

	// Add identifier - use custom identifier if specified, otherwise use bundle ID
	identifier := cfg.CodeSigningIdentifier
	if cfg.Debug {
		fmt.Printf("macgo: codesign identifier from config: %q\n", identifier)
	}
	if identifier == "" {
		// Read bundle ID from Info.plist
		plistPath := filepath.Join(bundlePath, "Contents", "Info.plist")
		if bundleID, err := readBundleIDFromPlist(plistPath); err == nil && bundleID != "" {
			identifier = bundleID
			if cfg.Debug {
				fmt.Printf("macgo: using bundle ID as identifier: %q\n", identifier)
			}
		} else if cfg.Debug {
			fmt.Printf("macgo: failed to read bundle ID: %v\n", err)
		}
	}
	if identifier != "" {
		args = append(args, "--identifier", identifier)
		if cfg.Debug {
			fmt.Printf("macgo: codesign will use identifier: %q\n", identifier)
		}
	}

	if cfg.CodeSignIdentity != "-" {
		entitlementsPath := filepath.Join(bundlePath, "Contents", "entitlements.plist")
		if _, err := os.Stat(entitlementsPath); err == nil {
			args = append(args, "--entitlements", entitlementsPath)
		}
	}

	args = append(args, bundlePath)

	cmd := exec.Command("codesign", args...)
	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "macgo: running: codesign %s\n", strings.Join(args, " "))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("codesign failed: %w\nOutput: %s", err, string(output))
	}

	if cfg.Debug && len(output) > 0 {
		fmt.Fprintf(os.Stderr, "macgo: codesign output: %s\n", string(output))
	}

	return nil
}

// findDeveloperID attempts to find a Developer ID Application certificate.
func findDeveloperID(debug bool) string {
	cmd := exec.Command("security", "find-identity", "-v", "-p", "codesigning")
	output, err := cmd.Output()
	if err != nil {
		if debug {
			fmt.Fprintf(os.Stderr, "macgo: failed to query code signing identities: %v\n", err)
		}
		return ""
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Developer ID Application") {
			if start := strings.Index(line, `"`); start != -1 {
				if end := strings.LastIndex(line, `"`); end != -1 && end > start {
					identity := line[start+1 : end]
					if debug {
						fmt.Fprintf(os.Stderr, "macgo: found Developer ID: %s\n", identity)
					}
					return identity
				}
			}
		}
	}

	for _, line := range lines {
		if strings.Contains(line, "valid identities found") {
			continue
		}
		if strings.Contains(line, `"`) && !strings.Contains(line, "invalid") {
			if start := strings.Index(line, `"`); start != -1 {
				if end := strings.LastIndex(line, `"`); end != -1 && end > start {
					identity := line[start+1 : end]
					if debug {
						fmt.Fprintf(os.Stderr, "macgo: found fallback identity: %s\n", identity)
					}
					return identity
				}
			}
		}
	}

	return ""
}

// readBundleIDFromPlist reads the CFBundleIdentifier from an Info.plist file.
func readBundleIDFromPlist(plistPath string) (string, error) {
	cmd := exec.Command("plutil", "-extract", "CFBundleIdentifier", "raw", plistPath)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// resetTCCPermissions resets all TCC permissions for the app bundle.
func resetTCCPermissions(cfg *Config) error {
	// Determine bundle ID
	bundleID := cfg.BundleID
	if bundleID == "" {
		appName := cfg.AppName
		if appName == "" {
			execPath, err := os.Executable()
			if err != nil {
				return fmt.Errorf("failed to get executable path: %w", err)
			}
			appName = strings.TrimSuffix(filepath.Base(execPath), filepath.Ext(execPath))
		}
		bundleID = fmt.Sprintf("com.macgo.%s", strings.ToLower(appName))
	}

	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "macgo: resetting TCC permissions for bundle ID: %s\n", bundleID)
	}

	// Execute tccutil reset All command
	cmd := exec.Command("tccutil", "reset", "All", bundleID)
	output, err := cmd.CombinedOutput()

	if err != nil {
		if cfg.Debug {
			fmt.Fprintf(os.Stderr, "macgo: tccutil output: %s\n", string(output))
		}
		return fmt.Errorf("tccutil reset failed: %w", err)
	}

	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "macgo: successfully reset TCC permissions for %s\n", bundleID)
		if len(output) > 0 {
			fmt.Fprintf(os.Stderr, "macgo: tccutil output: %s\n", string(output))
		}
	}

	return nil
}
