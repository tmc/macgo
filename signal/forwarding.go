package signal

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
)

// RelaunchWithRobustSignalHandling relaunches the app with robust signal handling.
// This approach is inspired by the Go tools implementation and works better
// in many scenarios, especially with Ctrl+C handling.
func RelaunchWithRobustSignalHandling(appPath, execPath string, args []string) {
	RelaunchWithRobustSignalHandlingContext(context.Background(), appPath, execPath, args)
}

// RelaunchWithRobustSignalHandlingContext relaunches the app with robust signal handling and context support.
func RelaunchWithRobustSignalHandlingContext(ctx context.Context, appPath, execPath string, args []string) {
	debugf("=== relaunchWithRobustSignalHandling START ===")
	debugf("appPath: %s", appPath)
	debugf("execPath: %s", execPath)
	debugf("args: %v", args)

	// Set environment to prevent relaunching again
	os.Setenv("MACGO_NO_RELAUNCH", "1")

	if DisableSignalHandling {
		debugf("Signal handling is disabled, using basic relaunch")
		basicRelaunch(appPath, execPath, args)
		return
	}

	// Create execution context
	execCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Launch app bundle
	executablePath := filepath.Join(appPath, "Contents", "MacOS", filepath.Base(execPath))
	cmd := exec.CommandContext(execCtx, executablePath, args...)

	// Set up process group for signal handling
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	// Set up I/O
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the process
	if err := cmd.Start(); err != nil {
		debugf("Failed to start app bundle: %v", err)
		os.Exit(1)
	}

	// Set up signal forwarding
	handler := NewHandler()
	if err := handler.Forward(execCtx, cmd.Process); err != nil {
		debugf("Failed to set up signal forwarding: %v", err)
	}

	// Wait for completion
	err := cmd.Wait()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		debugf("App bundle execution error: %v", err)
		os.Exit(1)
	}

	os.Exit(0)
}

// ImprovedRelaunch provides an improved relaunch function that uses the Go tool's signal handling pattern.
// This is extracted from the signalhandler package.
func ImprovedRelaunch(appPath, execPath string, args []string) {
	// Set environment to prevent relaunching again
	os.Setenv("MACGO_NO_RELAUNCH", "1")

	// Launch app bundle with robust approach
	toolPath, err := exec.LookPath("open")
	if err != nil {
		debugf("error finding open command: %v", err)
		return
	}

	toolCmd := &exec.Cmd{
		Path:   toolPath,
		Args:   append([]string{toolPath}, args...),
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		SysProcAttr: &syscall.SysProcAttr{
			Setpgid: true,
			Pgid:    0, // Use the parent's process group
		},
	}

	err = toolCmd.Start()
	if err == nil {
		c := make(chan os.Signal, 100)
		signal.Notify(c)
		go func() {
			for sig := range c {
				debugf("Forwarding signal %v to app bundle process group", sig)
				// Forward to entire process group using negative PID
				sigNum := sig.(syscall.Signal)

				// Skip SIGCHLD as we don't need to forward it
				if sigNum == syscall.SIGCHLD {
					continue
				}

				// Using negative PID sends to the entire process group
				if err := syscall.Kill(-toolCmd.Process.Pid, sigNum); err != nil {
					debugf("Error forwarding signal %v: %v", sigNum, err)
				}

				// Special handling for terminal signals
				if sigNum == syscall.SIGTSTP || sigNum == syscall.SIGTTIN || sigNum == syscall.SIGTTOU {
					// Use SIGSTOP for terminal signals
					syscall.Kill(syscall.Getpid(), syscall.SIGSTOP)
				}
			}
		}()
		err = toolCmd.Wait()
		signal.Stop(c)
		close(c)
	}

	if err != nil {
		// Only print about the exit status if the command
		// didn't even run or it didn't exit cleanly
		if e, ok := err.(*exec.ExitError); !ok || !e.Exited() {
			debugf("error waiting for app bundle: %v", err)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}

	os.Exit(0)
}

// basicRelaunch provides a simple relaunch without signal handling for fallback.
func basicRelaunch(appPath, execPath string, args []string) {
	executablePath := filepath.Join(appPath, "Contents", "MacOS", filepath.Base(execPath))
	cmd := exec.Command(executablePath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
	os.Exit(0)
}

// Legacy compatibility functions

// DisableSignals disables signal handling - legacy compatibility function.
func DisableSignals() {
	DisableSignalHandling = true
}

// DisableRobustSignals is for backward compatibility.
func DisableRobustSignals() {
	DisableSignalHandling = true
}

// EnableLegacySignalHandling is for backward compatibility.
func EnableLegacySignalHandling() {
	DisableSignalHandling = true
}

// FallbackDirectExecutionContext performs direct execution without signal handling as a fallback.
// This is used when the app bundle approach fails.
func FallbackDirectExecutionContext(ctx context.Context, appPath, execPath string) {
	debugf("=== fallbackDirectExecutionContext START ===")
	debugf("appPath: %s", appPath)
	debugf("execPath: %s", execPath)

	// Set environment to prevent relaunching again
	os.Setenv("MACGO_NO_RELAUNCH", "1")

	// Try to execute the binary directly
	executablePath := filepath.Join(appPath, "Contents", "MacOS", filepath.Base(execPath))

	// Check if the executable exists
	if _, err := os.Stat(executablePath); os.IsNotExist(err) {
		debugf("Executable not found at %s, trying original path", executablePath)
		executablePath = execPath
	}

	if _, err := os.Stat(executablePath); os.IsNotExist(err) {
		debugf("Executable not found at %s, exiting", executablePath)
		os.Exit(1)
	}

	// Create execution context
	cmd := exec.CommandContext(ctx, executablePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		debugf("Direct execution error: %v", err)
		os.Exit(1)
	}

	os.Exit(0)
}
