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

	// Create bundle
	bundlePath, err := createSimpleBundle(execPath, cfg)
	if err != nil {
		return fmt.Errorf("macgo: create bundle: %w", err)
	}

	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "macgo: created bundle at %s\n", bundlePath)
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
	if err := os.RemoveAll(bundleDir); err != nil && !os.IsNotExist(err) {
		return "", err
	}

	// Create directory structure
	contentsDir := filepath.Join(bundleDir, "Contents")
	macosDir := filepath.Join(contentsDir, "MacOS")
	if err := os.MkdirAll(macosDir, 0755); err != nil {
		return "", err
	}

	// Copy executable
	execName := filepath.Base(appName)
	destExec := filepath.Join(macosDir, execName)
	if err := copyFile(execPath, destExec); err != nil {
		return "", err
	}

	// Make executable
	if err := os.Chmod(destExec, 0755); err != nil {
		return "", err
	}

	// Create Info.plist
	plistPath := filepath.Join(contentsDir, "Info.plist")
	if err := writeInfoPlist(plistPath, appName, bundleID, execName); err != nil {
		return "", err
	}

	// Create entitlements if needed
	if len(cfg.Permissions) > 0 || len(cfg.Custom) > 0 {
		entPath := filepath.Join(contentsDir, "entitlements.plist")
		if err := writeEntitlements(entPath, cfg); err != nil {
			return "", err
		}
	}

	return bundleDir, nil
}

// inferBundleID creates a reasonable bundle ID from the app name.
func inferBundleID(appName string) string {
	// Try to get module path from build info
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Path != "" {
		// Convert module path to bundle ID
		bundleID := strings.ReplaceAll(info.Main.Path, "/", ".")
		return bundleID
	}

	// Fallback to simple format
	return fmt.Sprintf("com.macgo.%s", appName)
}

// cleanAppName removes problematic characters from app names.
func cleanAppName(name string) string {
	// Remove path separators and other problematic characters
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	name = strings.ReplaceAll(name, ":", "-")
	name = strings.ReplaceAll(name, "*", "-")
	name = strings.ReplaceAll(name, "?", "-")
	name = strings.ReplaceAll(name, "\"", "-")
	name = strings.ReplaceAll(name, "<", "-")
	name = strings.ReplaceAll(name, ">", "-")
	name = strings.ReplaceAll(name, "|", "-")

	// Remove control characters
	var result strings.Builder
	for _, r := range name {
		if r >= 32 && r < 127 {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// relaunchInBundle launches the app bundle using macOS 'open' command.
func relaunchInBundle(ctx context.Context, bundlePath, execPath string, cfg *Config) error {
	// For simple CLI tools, we can use direct execution for better I/O handling
	// But for apps that need proper TCC dialogs, we should use 'open'

	// Check if we need LaunchServices (for TCC permissions)
	needsLaunchServices := false
	for _, perm := range cfg.Permissions {
		switch perm {
		case Camera, Microphone, Location:
			needsLaunchServices = true
		}
	}

	if !needsLaunchServices {
		// For simple CLI tools without TCC requirements, use direct execution
		// This preserves stdin/stdout/stderr properly
		return relaunchDirect(ctx, bundlePath, execPath, cfg)
	}

	// For apps needing TCC permissions, use 'open' with I/O redirection
	return relaunchWithOpen(ctx, bundlePath, cfg)
}

// relaunchDirect directly executes the binary in the bundle.
func relaunchDirect(ctx context.Context, bundlePath, execPath string, cfg *Config) error {
	// Determine executable name
	execName := ""
	if cfg.AppName != "" {
		execName = filepath.Base(cfg.AppName)
	}
	if execName == "" {
		execName = filepath.Base(execPath)
	}

	bundleExec := filepath.Join(bundlePath, "Contents", "MacOS", execName)

	// Create command
	cmd := exec.CommandContext(ctx, bundleExec, os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set process group for signal handling
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("macgo: start bundle: %w", err)
	}

	// Wait for completion
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

// relaunchWithOpen uses the 'open' command for proper LaunchServices integration.
func relaunchWithOpen(ctx context.Context, bundlePath string, cfg *Config) error {
	// Create named pipes for I/O redirection
	pipeDir := filepath.Join(os.TempDir(), fmt.Sprintf("macgo-%d", os.Getpid()))
	if err := os.MkdirAll(pipeDir, 0700); err != nil {
		return fmt.Errorf("macgo: create pipe dir: %w", err)
	}
	defer os.RemoveAll(pipeDir)

	stdinPipe := filepath.Join(pipeDir, "stdin")
	stdoutPipe := filepath.Join(pipeDir, "stdout")
	stderrPipe := filepath.Join(pipeDir, "stderr")

	// Create named pipes
	for _, pipe := range []string{stdinPipe, stdoutPipe, stderrPipe} {
		if err := syscall.Mkfifo(pipe, 0600); err != nil {
			return fmt.Errorf("macgo: create pipe %s: %w", pipe, err)
		}
	}

	// Build arguments for open command
	args := []string{
		"-a", bundlePath,
		"--wait-apps",           // Wait for the app to finish
		"--stdin", stdinPipe,    // Connect stdin
		"--stdout", stdoutPipe,  // Connect stdout
		"--stderr", stderrPipe,  // Connect stderr
	}

	// Add command line arguments if any
	if len(os.Args) > 1 {
		args = append(args, "--args")
		args = append(args, os.Args[1:]...)
	}

	// Use 'open' to launch through LaunchServices
	cmd := exec.CommandContext(ctx, "open", args...)

	// Start the app bundle
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("macgo: launch bundle with open: %w", err)
	}

	// Set up I/O forwarding
	go forwardIO(stdinPipe, os.Stdin, nil)
	go forwardIO(stdoutPipe, nil, os.Stdout)
	go forwardIO(stderrPipe, nil, os.Stderr)

	// Wait for the open command to finish
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("macgo: open command failed: %w", err)
	}

	os.Exit(0)
	return nil
}

// forwardIO copies data between a named pipe and stdin/stdout/stderr.
func forwardIO(pipePath string, in *os.File, out *os.File) {
	var err error
	var pipe *os.File

	if in != nil {
		// Writing to the pipe (for stdin)
		pipe, err = os.OpenFile(pipePath, os.O_WRONLY, 0)
		if err != nil {
			return
		}
		defer pipe.Close()
		io.Copy(pipe, in)
	} else if out != nil {
		// Reading from the pipe (for stdout/stderr)
		pipe, err = os.OpenFile(pipePath, os.O_RDONLY, 0)
		if err != nil {
			return
		}
		defer pipe.Close()
		io.Copy(out, pipe)
	}
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