package macgo

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/tmc/macgo/internal/bundle"
	"github.com/tmc/macgo/internal/launch"
	"github.com/tmc/macgo/internal/system"
	"github.com/tmc/macgo/internal/tcc"
	"github.com/tmc/macgo/teamid"
)

// startDarwin implements the macOS-specific logic.
func startDarwin(ctx context.Context, cfg *Config) error {
	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "macgo: starting on darwin (PID: %d)\n", os.Getpid())
	}

	// Auto-detect and substitute team ID in app groups if needed
	if err := substituteTeamID(cfg); err != nil && cfg.Debug {
		fmt.Fprintf(os.Stderr, "macgo: failed to substitute team ID: %v\n", err)
	}
	// Check for permission reset flag
	if system.IsResetPermissionsEnabled() {
		resolutionCfg := tcc.ResolutionConfig{
			BundleID: cfg.BundleID,
			AppName:  cfg.AppName,
			Debug:    cfg.Debug,
		}
		if err := tcc.ResetWithConfig(resolutionCfg); err != nil {
			if cfg.Debug {
				fmt.Fprintf(os.Stderr, "macgo: failed to reset permissions: %v\n", err)
			}
		}
	}

	// Check for MACGO_*_PIPE env vars (restored for V2 env-flags support)
	if stdin := os.Getenv("MACGO_STDIN_PIPE"); stdin != "" {
		if cfg.Debug {
			fmt.Fprintf(os.Stderr, "macgo: detected pipes via environment\n")
		}
		if err := setupPipeRedirection(cfg.Debug); err != nil {
			if cfg.Debug {
				fmt.Fprintf(os.Stderr, "macgo: failed to setup pipe redirection: %v\n", err)
			}
		} else {
			// Successfully set up pipes via env, register exit handler and return
			registerExitHandler(cfg.Debug)
			return nil
		}
	}

	// Check for V2 launcher config file
	// Since `open --args` doesn't reliably pass arguments to .app bundles,
	// we look for a config file in a well-known location based on parent PID
	configFile := findPipeConfig(cfg.Debug)

	// Note: We check env vars first now to support open --env

	// If we found a matching config file, we're in the relaunched app
	if configFile != "" {
		if cfg.Debug {
			execPath, _ := os.Executable()
			fmt.Fprintf(os.Stderr, "macgo: detected relaunch with I/O pipes (PID: %d, exec: %s)\n", os.Getpid(), execPath)
			if configFile != "" {
				fmt.Fprintf(os.Stderr, "macgo: loading pipe config from: %s\n", configFile)
			}
		}

		// Load config file if provided (V2 launcher)
		if configFile != "" {
			if err := loadPipeConfig(configFile); err != nil {
				if cfg.Debug {
					fmt.Fprintf(os.Stderr, "macgo: failed to load pipe config: %v\n", err)
				}
			}
			// Write our PID so parent can forward signals to us
			if err := writeChildPID(configFile, cfg.Debug); err != nil {
				if cfg.Debug {
					fmt.Fprintf(os.Stderr, "macgo: failed to write child PID: %v\n", err)
				}
			}
		}

		// Restore original working directory (open command changes it)
		if cwd := os.Getenv("MACGO_CWD"); cwd != "" {
			if err := os.Chdir(cwd); err != nil {
				if cfg.Debug {
					fmt.Fprintf(os.Stderr, "macgo: failed to restore CWD to %s: %v\n", cwd, err)
				}
			} else if cfg.Debug {
				fmt.Fprintf(os.Stderr, "macgo: restored CWD to %s\n", cwd)
			}
		}

		// Transparently redirect stdout/stderr to the named pipes
		if err := setupPipeRedirection(cfg.Debug); err != nil {
			if cfg.Debug {
				fmt.Fprintf(os.Stderr, "macgo: failed to setup pipe redirection: %v\n", err)
			}
		}

		// Register signal handler to write done file on termination
		registerExitHandler(cfg.Debug)

		return nil
	}

	// Skip if already in app bundle
	if system.IsInAppBundle() {
		if cfg.Debug {
			fmt.Fprintf(os.Stderr, "macgo: already in app bundle\n")
		}
		// In DevMode, exec the development binary instead of running bundled code
		if cfg.DevMode {
			if cfg.Debug {
				fmt.Fprintf(os.Stderr, "macgo: dev mode active in bundle - will exec source binary\n")
			}
			if err := execDevTarget(cfg); err != nil {
				if cfg.Debug {
					fmt.Fprintf(os.Stderr, "macgo: dev mode exec failed: %v\n", err)
				}
				// Fall through to normal execution if exec fails
			}
			// If execDevTarget returns without error, it means no target was found
			// Continue with normal bundled execution
			if cfg.Debug {
				fmt.Fprintf(os.Stderr, "macgo: dev mode - no target found, running bundled binary\n")
			}
		}
		return nil
	}

	// Skip if disabled
	if system.IsRelaunchDisabled() {
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
	bundleObj, err := createSimpleBundle(execPath, cfg)
	if err != nil {
		return fmt.Errorf("macgo: bundle operation: %w", err)
	}
	bundlePath := bundleObj.Path

	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "macgo: using bundle at %s\n", bundlePath)
		fmt.Fprintf(os.Stderr, "macgo: permissions requested: %v\n", cfg.Permissions)
	}

	// In DevMode, store the source path in env for mismatch detection
	if cfg.DevMode {
		if cfg.Debug {
			fmt.Fprintf(os.Stderr, "macgo: dev mode enabled - source path: %s\n", execPath)
		}
		os.Setenv("MACGO_DEV_SOURCE", execPath)
	}

	// Relaunch in bundle
	return relaunchInBundle(ctx, bundlePath, execPath, cfg)
}

// execDevTarget execs the development source binary if DevMode is enabled.
// This allows the signed bundle to exec the rebuilt source binary,
// preserving TCC permissions while allowing rapid iteration.
//
// The target path is read from .dev_target in the bundle's Contents directory.
// This file is written at bundle creation time and contains the path to the
// binary that created the bundle. This prevents arbitrary exec attacks.
//
// If exec succeeds, this function never returns.
// If no target found, returns nil.
// If exec fails, returns an error.
func execDevTarget(cfg *Config) error {
	execPath, err := os.Executable()
	if err != nil {
		return nil // Can't determine bundle path, skip
	}
	// execPath is like /path/to/App.app/Contents/MacOS/binary
	// We want /path/to/App.app/Contents/.dev_target
	contentsDir := filepath.Dir(filepath.Dir(execPath))
	targetFile := filepath.Join(contentsDir, ".dev_target")
	data, err := os.ReadFile(targetFile)
	if err != nil {
		return nil // No stored target, skip
	}
	target := strings.TrimSpace(string(data))

	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "macgo: .dev_target contains: %q\n", target)
	}

	if target == "" {
		return nil // No target found
	}

	// Check for mismatch between stored target and current source
	// MACGO_DEV_SOURCE is set by the source binary before relaunch
	if source := os.Getenv("MACGO_DEV_SOURCE"); source != "" && source != target {
		fmt.Fprintf(os.Stderr, "macgo: warning: dev mode source mismatch\n")
		fmt.Fprintf(os.Stderr, "  current source: %s\n", source)
		fmt.Fprintf(os.Stderr, "  bundle target:  %s\n", target)
		fmt.Fprintf(os.Stderr, "  (bundle was created from a different path)\n")
	}

	// Verify target exists
	if _, err := os.Stat(target); err != nil {
		if cfg.Debug {
			fmt.Fprintf(os.Stderr, "macgo: dev target not found: %s\n", target)
		}
		return nil // Target doesn't exist, skip
	}

	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "macgo: dev mode - exec'ing target: %s\n", target)
	}

	// Set MACGO_NO_RELAUNCH to prevent the target from creating another bundle
	os.Setenv("MACGO_NO_RELAUNCH", "1")

	// Exec the target - this replaces the current process
	// The TCC permissions from the signed bundle apply to the exec'd process
	return syscall.Exec(target, os.Args, os.Environ())
}

// createSimpleBundle creates a minimal app bundle with the given configuration.
func createSimpleBundle(execPath string, cfg *Config) (*bundle.Bundle, error) {
	// Convert permissions to strings
	var permissions []string
	for _, perm := range cfg.Permissions {
		permissions = append(permissions, string(perm))
	}

	// Use the bundle package to create the bundle
	return bundle.Create(
		execPath,
		cfg.AppName,
		cfg.BundleID,
		cfg.Version,
		permissions,
		cfg.Custom,
		cfg.AppGroups,
		cfg.Debug,
		cfg.CleanupBundle,
		cfg.CodeSignIdentity,
		cfg.CodeSigningIdentifier,
		cfg.AutoSign,
		cfg.AdHocSign,
		cfg.Info,
		bundle.UIMode(cfg.UIMode),
		cfg.DevMode,
	)
}

// convertPermissions converts Permission values to strings for the launch package.
func convertPermissions(permissions []Permission) []string {
	var result []string
	for _, perm := range permissions {
		result = append(result, string(perm))
	}
	return result
}

// relaunchInBundle launches the app bundle using the launch package.
func relaunchInBundle(ctx context.Context, bundlePath, execPath string, cfg *Config) error {
	// Convert main config to launch config
	// Include both standard permissions and custom entitlements so Launch Services is used for TCC
	permissions := convertPermissions(cfg.Permissions)
	// Add custom entitlements as permissions to trigger Launch Services
	permissions = append(permissions, cfg.Custom...)

	launchCfg := &launch.Config{
		AppName:              cfg.AppName,
		BundleID:             cfg.BundleID,
		Permissions:          permissions,
		Debug:                cfg.Debug,
		ForceDirectExecution: cfg.ForceDirectExecution,
		Background:           cfg.UIMode == "" || cfg.UIMode == UIModeBackground,
	}

	// Create launch manager and execute
	manager := launch.New()
	return manager.Launch(ctx, bundlePath, execPath, launchCfg)
}

// findPipeConfig looks for a V2 launcher config file in /tmp/macgo-*/config.
// Returns the path to the config file if found, empty string otherwise.
// It matches config files by bundle path to handle nested macgo calls correctly.
func findPipeConfig(debug bool) string {
	// Get current executable path to match against config files
	execPath, err := os.Executable()
	if err != nil {
		if debug {
			fmt.Fprintf(os.Stderr, "macgo: error getting executable path: %v\n", err)
		}
		return ""
	}

	// The bundle path is the .app directory containing our executable
	// e.g., /path/to/App.app/Contents/MacOS/App -> /path/to/App.app
	bundlePath := ""
	if idx := strings.Index(execPath, ".app/"); idx != -1 {
		bundlePath = execPath[:idx+4] // Include ".app"
	}

	if debug {
		fmt.Fprintf(os.Stderr, "macgo: findPipeConfig: execPath=%q\n", execPath)
		fmt.Fprintf(os.Stderr, "macgo: findPipeConfig: determined bundlePath=%q\n", bundlePath)
	}

	if bundlePath == "" {
		if debug {
			fmt.Fprintf(os.Stderr, "macgo: findPipeConfig: no .app/ found in path, assuming CLI mode, checking parent PID configs...\n")
		}
		// Logic to handle CLI tool scenario?
	}

	// Look for recent config files in ~/Library/Application Support/macgo/pipes/
	// This location is user-specific and protected by macOS sandbox rules
	var matches []string
	if home, err := os.UserHomeDir(); err == nil {
		pattern := filepath.Join(home, "Library", "Application Support", "macgo", "pipes", "*", "config")
		if m, err := filepath.Glob(pattern); err == nil {
			matches = m
		}
	}
	// Fallback: also check /tmp/macgo/ for backward compatibility
	if fallbackMatches, err := filepath.Glob(filepath.Join(os.TempDir(), "macgo", "*", "config")); err == nil {
		matches = append(matches, fallbackMatches...)
	}

	// Find the most recent config file that matches our bundle path
	var newestConfig string
	var newestTime time.Time
	var skippedOld, skippedMismatch int

	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}

		// Only consider files modified within the last 10 seconds
		age := time.Since(info.ModTime())
		if age > 10*time.Second {
			skippedOld++
			continue
		}

		// Read the config file to check if it's for our bundle
		configBundle := readBundlePathFromConfig(match)

		if debug {
			fmt.Fprintf(os.Stderr, "macgo: checking candidate %s: bundlePath=%q vs configBundle=%q\n", filepath.Base(match), bundlePath, configBundle)
		}

		// Only use configs that match our bundle path
		// If bundlePath is empty (CLI tool?), matches might be ambiguous.
		if configBundle != "" && configBundle != bundlePath {
			skippedMismatch++
			continue
		}

		if debug {
			fmt.Fprintf(os.Stderr, "macgo: config candidate match found: %s\n", match)
		}

		if newestConfig == "" || info.ModTime().After(newestTime) {
			newestConfig = match
			newestTime = info.ModTime()
		}
	}

	if debug && (skippedOld > 0 || skippedMismatch > 0) {
		fmt.Fprintf(os.Stderr, "macgo: skipped %d old and %d mismatched configs\n", skippedOld, skippedMismatch)
	}

	if newestConfig != "" && debug {
		fmt.Fprintf(os.Stderr, "macgo: selected config file: %s (age: %v)\n", newestConfig, time.Since(newestTime))
	}

	return newestConfig
}

// readBundlePathFromConfig reads the MACGO_BUNDLE_PATH from a config file.
func readBundlePathFromConfig(configFile string) string {
	f, err := os.Open(configFile)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MACGO_BUNDLE_PATH=") {
			return strings.TrimPrefix(line, "MACGO_BUNDLE_PATH=")
		}
	}
	return ""
}

// loadPipeConfig reads pipe paths from a config file and sets them as environment variables.
func loadPipeConfig(configFile string) error {
	f, err := os.Open(configFile)
	if err != nil {
		return fmt.Errorf("open config file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Set MACGO_*_PIPE, MACGO_DONE_FILE, and MACGO_CWD variables
		if strings.HasPrefix(key, "MACGO_") && (strings.HasSuffix(key, "_PIPE") || key == "MACGO_DONE_FILE" || key == "MACGO_CWD") {
			os.Setenv(key, value)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	return nil
}

// setupPipeRedirection transparently redirects stdout/stderr/stdin to named pipes if present.
// This is used by ServicesLauncherV2 which passes pipe paths via config file.
func setupPipeRedirection(debug bool) error {
	// Log before any redirection in case stderr gets redirected
	if debug {
		stdoutPipe := os.Getenv("MACGO_STDOUT_PIPE")
		stderrPipe := os.Getenv("MACGO_STDERR_PIPE")
		stdinPipe := os.Getenv("MACGO_STDIN_PIPE")
		fmt.Fprintf(os.Stderr, "macgo: setting up pipe redirection (stdout=%s, stderr=%s, stdin=%s)\n",
			stdoutPipe, stderrPipe, stdinPipe)
	}

	// Handle stdin pipe first (before stdout/stderr in case of errors)
	if stdinPipe := os.Getenv("MACGO_STDIN_PIPE"); stdinPipe != "" {
		pipe, err := os.OpenFile(stdinPipe, os.O_RDONLY, 0)
		if err != nil {
			return fmt.Errorf("failed to open stdin pipe %s: %w", stdinPipe, err)
		}
		os.Stdin = pipe
		if debug {
			fmt.Fprintf(os.Stderr, "macgo: redirected stdin to %s\n", stdinPipe)
		}
	}

	// Handle stdout pipe
	if stdoutPipe := os.Getenv("MACGO_STDOUT_PIPE"); stdoutPipe != "" {
		pipe, err := os.OpenFile(stdoutPipe, os.O_WRONLY, 0)
		if err != nil {
			return fmt.Errorf("failed to open stdout pipe %s: %w", stdoutPipe, err)
		}
		// Replace os.Stdout with the pipe
		os.Stdout = pipe
		if debug {
			fmt.Fprintf(os.Stderr, "macgo: redirected stdout to %s\n", stdoutPipe)
		}
	}

	// Handle stderr pipe last (so debug messages work as long as possible)
	if stderrPipe := os.Getenv("MACGO_STDERR_PIPE"); stderrPipe != "" {
		pipe, err := os.OpenFile(stderrPipe, os.O_WRONLY, 0)
		if err != nil {
			// Can't use stderr for error message if it's being redirected
			return fmt.Errorf("failed to open stderr pipe %s: %w", stderrPipe, err)
		}
		// Replace os.Stderr with the pipe
		os.Stderr = pipe
		// Note: can't log to stderr after this point since it's redirected
	}

	return nil
}

// registerExitHandler sets up signal handlers for cleanup and SIGQUIT stack dumps.
//
// This function is called in the child process (the app running inside the bundle).
// It handles the following signals:
//
//   - SIGINT, SIGTERM, SIGHUP: Write done file (if configured), exit with 128+signal
//   - SIGQUIT: Dump all goroutine stacks to stderr, then exit with 128+signal
//   - SIGWINCH: Log receipt (debug mode only), do not exit
//
// The parent process forwards signals to us since we have PPID=1 (adopted by launchd)
// and are not in the terminal's foreground process group.
func registerExitHandler(debug bool) {
	doneFile := os.Getenv("MACGO_DONE_FILE")

	// Always register signal handlers for stack dumps and cleanup
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
	go func() {
		sig := <-c
		if debug {
			fmt.Fprintf(os.Stderr, "macgo: received signal %v\n", sig)
		}
		// For SIGQUIT, dump all goroutine stacks (Go's default behavior)
		if sig == syscall.SIGQUIT {
			dumpGoroutineStacks()
		}
		// Convert signal to exit code (128 + signal number)
		exitCode := 128 + int(sig.(syscall.Signal))
		// Write done file if configured (non-FIFO mode)
		if doneFile != "" {
			writeDoneFile()
		}
		os.Exit(exitCode)
	}()

	// Handle SIGWINCH (terminal resize) - log but don't exit
	// This helps with debugging signal forwarding from parent to child
	winchChan := make(chan os.Signal, 1)
	signal.Notify(winchChan, syscall.SIGWINCH)
	go func() {
		for range winchChan {
			if debug {
				fmt.Fprintf(os.Stderr, "macgo: child received SIGWINCH (pid=%d)\n", os.Getpid())
			}
		}
	}()
}

// dumpGoroutineStacks prints stack traces for all goroutines to stderr.
// This mimics Go's default SIGQUIT behavior.
func dumpGoroutineStacks() {
	buf := make([]byte, 64*1024*1024) // 64MB buffer
	n := runtime.Stack(buf, true)     // true = all goroutines
	fmt.Fprintf(os.Stderr, "\n*** goroutine dump ***\n%s\n", buf[:n])
}

// writeChildPID writes this process's PID to a file in the pipe directory.
// This allows the parent process to forward signals to us.
func writeChildPID(configFile string, debug bool) error {
	// Derive PID file path from config file path (same directory)
	pipeDir := filepath.Dir(configFile)
	pidFile := filepath.Join(pipeDir, "child.pid")

	content := fmt.Sprintf("%d\n", os.Getpid())
	if err := os.WriteFile(pidFile, []byte(content), 0600); err != nil {
		return fmt.Errorf("write PID file: %w", err)
	}

	if debug {
		fmt.Fprintf(os.Stderr, "macgo: wrote child PID %d to %s\n", os.Getpid(), pidFile)
	}
	return nil
}

// writeDoneFile writes the done/sentinel file to signal that the child has exited.
func writeDoneFile() {
	doneFile := os.Getenv("MACGO_DONE_FILE")
	if doneFile == "" {
		return
	}

	// Flush stdout and stderr before writing done file to ensure all data is written
	// This is critical: the parent will stop reading pipes once it sees the done file
	_ = os.Stdout.Sync()
	_ = os.Stderr.Sync()

	// Longer delay to allow the data to propagate through the pipe chain
	// In nested macgo scenarios, data needs to flow through multiple pipe layers
	time.Sleep(200 * time.Millisecond)

	// Write process exit info to the done file
	content := fmt.Sprintf("done\npid=%d\ntime=%s\n", os.Getpid(), time.Now().Format(time.RFC3339))
	if err := os.WriteFile(doneFile, []byte(content), 0600); err != nil {
		// Can't do much if this fails, just try to continue
		fmt.Fprintf(os.Stderr, "macgo: failed to write done file: %v\n", err)
	}
}

// substituteTeamID automatically detects team ID and substitutes "TEAMID" placeholders in app groups
func substituteTeamID(cfg *Config) error {
	if len(cfg.AppGroups) == 0 {
		return nil
	}

	// Use the helpers package for team ID detection and substitution
	teamID, substitutions, err := teamid.AutoSubstituteTeamIDInGroups(cfg.AppGroups)
	if err != nil {
		return fmt.Errorf("team ID detection failed: %w", err)
	}

	if cfg.Debug && substitutions > 0 {
		fmt.Printf("macgo: detected team ID %s, updated app groups: %v\n", teamID, cfg.AppGroups)
	}

	return nil
}
