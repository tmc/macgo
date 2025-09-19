package launch

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// DirectLauncher implements direct execution of the binary within the bundle.
type DirectLauncher struct{}

// Launch executes the binary directly within the app bundle.
func (d *DirectLauncher) Launch(ctx context.Context, bundlePath, execPath string, cfg *Config) error {
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

	cmd := exec.CommandContext(ctx, bundleExec, os.Args[1:]...)

	// Set up I/O redirection
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Configure process attributes for proper signal handling
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

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
