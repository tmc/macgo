package launch

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

// DirectLauncher implements direct execution of the binary within the bundle.
type DirectLauncher struct{}

// Launch executes the binary directly within the app bundle.
func (d *DirectLauncher) Launch(ctx context.Context, bundlePath, execPath string, cfg *Config) error {
	// Set up signal handling context
	// We handle: SIGINT, SIGTERM, SIGQUIT, SIGHUP
	sigCtx, stop := signal.NotifyContext(ctx,
		syscall.SIGINT,  // Interrupt (Ctrl+C)
		syscall.SIGTERM, // Termination request
		syscall.SIGQUIT, // Quit with core dump (Ctrl+\)
		syscall.SIGHUP,  // Hangup
		syscall.SIGPIPE, // Broken pipe when downstream closes early
	)
	defer stop()
	ctx = sigCtx
	bundleExec, err := d.getBundleExecutablePath(bundlePath, execPath, cfg)
	if err != nil {
		return fmt.Errorf("get bundle executable path: %w", err)
	}

	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "macgo: executing bundle binary: %s\n", bundleExec)
	}

	// Verify the executable exists
	if _, err := os.Stat(bundleExec); err != nil {
		return fmt.Errorf("bundle executable not found at %s: %w", bundleExec, err)
	}

	cmd := exec.CommandContext(sigCtx, bundleExec, os.Args[1:]...)

	// Set up I/O redirection
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Configure process attributes for proper signal handling
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	// Override CommandContext's default SIGKILL behavior: send SIGINT instead
	// so the child process can perform graceful cleanup (e.g., VM suspend).
	// WaitDelay gives the child up to 3 minutes to finish cleanup before SIGKILL.
	cmd.Cancel = func() error {
		if cfg.Debug {
			fmt.Fprintf(os.Stderr, "macgo: forwarding interrupt to child process %d\n", cmd.Process.Pid)
		}
		return cmd.Process.Signal(syscall.SIGINT)
	}
	cmd.WaitDelay = 3 * time.Minute

	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "macgo: starting process with args: %v\n", cmd.Args)
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start bundle process: %w", err)
	}

	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "macgo: process started with PID: %d\n", cmd.Process.Pid)
	}

	// Wait for the process to complete
	err = cmd.Wait()
	if err != nil {
		// Handle exit errors by forwarding the exit code
		if exitErr, ok := err.(*exec.ExitError); ok {
			if cfg.Debug {
				fmt.Fprintf(os.Stderr, "macgo: process exited with code: %d\n", exitErr.ExitCode())
			}
			os.Exit(exitErr.ExitCode())
		}
		// When the context is cancelled (e.g., SIGINT to process group),
		// cmd.Wait returns context.Canceled instead of an ExitError.
		// The child received SIGINT via cmd.Cancel and may have exited
		// cleanly. Treat this as a signal exit, not a launch failure.
		if sigCtx.Err() != nil {
			if cfg.Debug {
				fmt.Fprintf(os.Stderr, "macgo: context cancelled, exiting with signal code\n")
			}
			os.Exit(130) // 128 + SIGINT
		}
		return fmt.Errorf("bundle execution failed: %w", err)
	}

	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "macgo: process completed successfully\n")
	}

	// Exit successfully
	os.Exit(0)
	return nil
}

// getBundleExecutablePath determines the path to the executable within the bundle.
func (d *DirectLauncher) getBundleExecutablePath(bundlePath, execPath string, cfg *Config) (string, error) {
	// Determine the executable name
	execName := ""
	if cfg.AppName != "" {
		execName = filepath.Base(cfg.AppName)
	}
	if execName == "" {
		execName = filepath.Base(execPath)
		// Remove extension if present
		if ext := filepath.Ext(execName); ext != "" {
			execName = execName[:len(execName)-len(ext)]
		}
	}

	if execName == "" {
		return "", fmt.Errorf("could not determine executable name")
	}

	bundleExec := filepath.Join(bundlePath, "Contents", "MacOS", execName)
	return bundleExec, nil
}
