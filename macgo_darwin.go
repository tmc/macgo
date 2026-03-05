package macgo

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

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

	// Detect relaunched child via env vars set by parent's open --env.
	// Any pipe or control env var means we are the child process.
	isChild := os.Getenv("MACGO_STDOUT_PIPE") != "" ||
		os.Getenv("MACGO_STDERR_PIPE") != "" ||
		os.Getenv("MACGO_STDIN_PIPE") != "" ||
		os.Getenv("MACGO_CONTROL_PIPE") != ""

	if isChild {
		if cfg.Debug {
			fmt.Fprintf(os.Stderr, "macgo: detected relaunch via environment (PID: %d)\n", os.Getpid())
		}

		// Write our PID to the control FIFO so parent can forward signals
		if err := writeChildPID(cfg.Debug); err != nil {
			if cfg.Debug {
				fmt.Fprintf(os.Stderr, "macgo: failed to write child PID: %v\n", err)
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

		// Redirect stdout/stderr/stdin to the named pipes
		if err := setupPipeRedirection(cfg.Debug); err != nil {
			if cfg.Debug {
				fmt.Fprintf(os.Stderr, "macgo: failed to setup pipe redirection: %v\n", err)
			}
		}

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

	// Single-process mode: codesign in-place, re-exec, setActivationPolicy.
	// Bypasses bundle creation entirely.
	if cfg.SingleProcess || os.Getenv("MACGO_SINGLE_PROCESS") == "1" {
		if cfg.Debug {
			fmt.Fprintf(os.Stderr, "macgo: using single-process mode\n")
		}
		return launchSingleProcess(ctx, cfg)
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

	// Store the original executable path so the inner process can find it.
	// os.Executable() inside the bundle returns the bundle binary path,
	// so this is the only way for the child to know the real source binary.
	os.Setenv("MACGO_ORIGINAL_EXECUTABLE", execPath)

	// In DevMode, store the source path in env for mismatch detection
	if cfg.DevMode {
		if cfg.Debug {
			fmt.Fprintf(os.Stderr, "macgo: dev mode enabled - source path: %s\n", execPath)
		}
		os.Setenv("MACGO_DEV_SOURCE", execPath)
	}

	// Relaunch in bundle
	if err := relaunchInBundle(ctx, bundlePath, execPath, cfg); err != nil {
		return err
	}
	// The parent process (launcher) also returns nil as cleanup is redundant for default FIFO IO
	return nil
}

// launchSingleProcess runs the single-process launcher.
func launchSingleProcess(ctx context.Context, cfg *Config) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("macgo: get executable: %w", err)
	}

	permissions := convertPermissions(cfg.Permissions)
	permissions = append(permissions, cfg.Custom...)

	launchCfg := &launch.Config{
		AppName:       cfg.AppName,
		BundleID:      cfg.BundleID,
		Permissions:   permissions,
		Debug:         cfg.Debug,
		SingleProcess: true,
		Entitlements:  cfg.Custom,
		UIMode:        string(cfg.UIMode),
		IconPath:      cfg.IconPath,
	}

	manager := launch.New()
	return manager.Launch(ctx, "", execPath, launchCfg)
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

	bundleCfg := &bundle.Config{
		AppName:               cfg.AppName,
		BundleID:              cfg.BundleID,
		Version:               cfg.Version,
		Permissions:           permissions,
		Custom:                cfg.Custom,
		CustomStrings:         cfg.CustomStrings,
		CustomArrays:          cfg.CustomArrays,
		AppGroups:             cfg.AppGroups,
		Debug:                 cfg.Debug,
		CleanupBundle:         cfg.CleanupBundle,
		CodeSignIdentity:      cfg.CodeSignIdentity,
		CodeSigningIdentifier: cfg.CodeSigningIdentifier,
		AutoSign:              cfg.AutoSign,
		AdHocSign:             cfg.AdHocSign,
		Info:                  cfg.Info,
		UIMode:                bundle.UIMode(cfg.UIMode),
		DevMode:               cfg.DevMode,
		ProvisioningProfile:   cfg.ProvisioningProfile,
		IconPath:              cfg.IconPath,
	}

	b, err := bundle.New(execPath, bundleCfg)
	if err != nil {
		return nil, err
	}

	if err := b.Create(); err != nil {
		return nil, err
	}

	// Run user hook between Create and Sign.
	if cfg.PostCreateHook != nil {
		if err := cfg.PostCreateHook(b.Path, cfg); err != nil {
			return nil, fmt.Errorf("post-create hook: %w", err)
		}
		// Hook modified bundle contents, so force re-signing even if
		// Create() determined the bundle was up-to-date.
		b.ForceResign()
	}

	if err := b.Sign(); err != nil {
		return nil, err
	}

	return b, nil
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


// setupPipeRedirection transparently redirects stdout/stderr/stdin to named pipes if present.
// Pipe paths are received via open --env from the parent process.
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
//   - SIGINT, SIGTERM, SIGHUP: Exit with 128+signal
//   - SIGQUIT: Dump all goroutine stacks to stderr, then exit with 128+signal
//   - SIGWINCH: Log receipt (debug mode only), do not exit
//
// The parent process forwards signals to us since we have PPID=1 (adopted by launchd)
// and are not in the terminal's foreground process group.
func registerExitHandler(debug bool) {
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

// writeChildPID writes this process's PID to the control FIFO.
// The parent reads this to enable signal forwarding.
func writeChildPID(debug bool) error {
	// Avoid re-writing the PID if Start is called more than once in the
	// relaunched child process.
	if os.Getenv("MACGO_CHILD_PID_WRITTEN") == "1" {
		return nil
	}

	controlPipe := os.Getenv("MACGO_CONTROL_PIPE")
	if controlPipe == "" {
		return nil // no control pipe configured
	}

	// Open non-blocking so a missing reader cannot wedge the child forever.
	f, err := os.OpenFile(controlPipe, os.O_WRONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		// No reader is expected once the parent has already consumed the first PID write.
		if pathErr, ok := err.(*os.PathError); ok && pathErr.Err == syscall.ENXIO {
			return nil
		}
		return fmt.Errorf("open control pipe: %w", err)
	}
	defer f.Close()

	pid := os.Getpid()
	if _, err := fmt.Fprintf(f, "%d\n", pid); err != nil {
		if pathErr, ok := err.(*os.PathError); ok && pathErr.Err == syscall.EPIPE {
			return nil
		}
		return fmt.Errorf("write PID to control pipe: %w", err)
	}
	_ = os.Setenv("MACGO_CHILD_PID_WRITTEN", "1")

	if debug {
		fmt.Fprintf(os.Stderr, "macgo: wrote child PID %d to control pipe %s\n", pid, controlPipe)
	}
	return nil
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
