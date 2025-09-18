package macgo

import (
	"context"
	"fmt"
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

	// Create bundle directory
	bundleDir := filepath.Join(os.TempDir(), appName+".app")
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

// relaunchInBundle launches the app bundle and forwards signals/IO.
func relaunchInBundle(ctx context.Context, bundlePath, execPath string, cfg *Config) error {
	// Determine executable name
	execName := ""
	if cfg.AppName != "" {
		execName = filepath.Base(cfg.AppName)
	}
	if execName == "" {
		execName = filepath.Base(execPath)
		// Don't strip extension - Go binaries typically don't have one anyway
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
	return nil // unreachable
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