// Package process provides process management and I/O handling functionality for macgo.
// This package consolidates process launching, I/O redirection, and pipe management
// that was previously scattered throughout bundle.go.
package process

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/tmc/misc/macgo"
)

// Launcher implements the ProcessLauncher interface.
type Launcher struct {
	signalHandler macgo.SignalForwarder
}

// NewLauncher creates a new process launcher with the provided signal handler.
func NewLauncher(signalHandler macgo.SignalForwarder) *Launcher {
	return &Launcher{
		signalHandler: signalHandler,
	}
}

// Launch starts a process within an app bundle with the given arguments.
func (l *Launcher) Launch(ctx context.Context, bundlePath string, args []string) error {
	// Extract executable name from bundle path
	execName := filepath.Base(bundlePath)
	if filepath.Ext(execName) == ".app" {
		execName = execName[:len(execName)-4] // Remove .app extension
	}

	// Construct path to executable inside bundle
	execPath := filepath.Join(bundlePath, "Contents", "MacOS", execName)

	// Create command
	cmd := exec.CommandContext(ctx, execPath, args...)

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
		return err
	}

	// Set up signal forwarding if available
	if l.signalHandler != nil {
		if err := l.signalHandler.Forward(ctx, cmd.Process); err != nil {
			macgo.Debug("Failed to set up signal forwarding: %v", err)
		}
	}

	// Wait for completion
	return cmd.Wait()
}

// Relaunch relaunches the current process within an app bundle.
func (l *Launcher) Relaunch(ctx context.Context, bundlePath, execPath string, args []string) error {
	// Set environment to prevent relaunching again
	os.Setenv("MACGO_NO_RELAUNCH", "1")

	// Extract executable name from the original path
	execName := filepath.Base(execPath)

	// Construct path to executable inside bundle
	bundleExecPath := filepath.Join(bundlePath, "Contents", "MacOS", execName)

	// Create command
	cmd := exec.CommandContext(ctx, bundleExecPath, args...)

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
		return err
	}

	// Set up signal forwarding if available
	if l.signalHandler != nil {
		if err := l.signalHandler.Forward(ctx, cmd.Process); err != nil {
			macgo.Debug("Failed to set up signal forwarding: %v", err)
		}
	}

	// Wait for completion
	err := cmd.Wait()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		macgo.Debug("Process execution error: %v", err)
		os.Exit(1)
	}

	os.Exit(0)
	return nil // This line will never be reached, but satisfies the return requirement
}

// Compile-time check that Launcher implements ProcessLauncher
var _ macgo.ProcessLauncher = (*Launcher)(nil)
