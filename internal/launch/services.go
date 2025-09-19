package launch

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// ServicesLauncher implements launching via LaunchServices using the 'open' command.
type ServicesLauncher struct{}

// Launch executes the application using LaunchServices with I/O forwarding.
func (s *ServicesLauncher) Launch(ctx context.Context, bundlePath, execPath string, cfg *Config) error {
	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "macgo: launching with LaunchServices (open command)\n")
	}

	// Create temporary directory for named pipes
	pipeDir, err := s.createPipeDirectory()
	if err != nil {
		return fmt.Errorf("create pipe directory: %w", err)
	}
	defer s.cleanupPipeDirectory(pipeDir, cfg.Debug)

	// Create named pipes for I/O forwarding
	pipes, err := s.createNamedPipes(pipeDir)
	if err != nil {
		return fmt.Errorf("create named pipes: %w", err)
	}

	// Build the open command
	cmd, err := s.buildOpenCommand(ctx, bundlePath, pipes)
	if err != nil {
		return fmt.Errorf("build open command: %w", err)
	}

	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "macgo: launching: %s %v\n", cmd.Path, cmd.Args[1:])
	}

	// Start the open command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start open command: %w", err)
	}

	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "macgo: open command started with PID: %d\n", cmd.Process.Pid)
	}

	// Set up I/O forwarding
	ioErrChan := make(chan error, 3)
	s.startIOForwarding(pipes, ioErrChan, cfg.Debug)

	// Wait for the open command to complete
	if err := cmd.Wait(); err != nil {
		// Handle exit errors by forwarding the exit code
		if exitErr, ok := err.(*exec.ExitError); ok {
			if cfg.Debug {
				fmt.Fprintf(os.Stderr, "macgo: open command exited with code: %d\n", exitErr.ExitCode())
			}
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("open command failed: %w", err)
	}

	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "macgo: open command completed successfully\n")
	}

	// Exit successfully - the launched application should handle its own lifecycle
	os.Exit(0)
	return nil
}

// pipeSet holds the paths to the named pipes for I/O forwarding.
type pipeSet struct {
	stdin  string
	stdout string
	stderr string
}

// createPipeDirectory creates a temporary directory for the named pipes.
func (s *ServicesLauncher) createPipeDirectory() (string, error) {
	pipeDir := filepath.Join(os.TempDir(), fmt.Sprintf("macgo-%d", os.Getpid()))
	if err := os.MkdirAll(pipeDir, 0700); err != nil {
		return "", fmt.Errorf("create directory %s: %w", pipeDir, err)
	}
	return pipeDir, nil
}

// cleanupPipeDirectory removes the temporary pipe directory.
func (s *ServicesLauncher) cleanupPipeDirectory(pipeDir string, debug bool) {
	if err := os.RemoveAll(pipeDir); err != nil && debug {
		fmt.Fprintf(os.Stderr, "macgo: warning: failed to cleanup pipe directory %s: %v\n", pipeDir, err)
	}
}

// createNamedPipes creates the named pipes (FIFOs) for I/O forwarding.
func (s *ServicesLauncher) createNamedPipes(pipeDir string) (*pipeSet, error) {
	pipes := &pipeSet{
		stdin:  filepath.Join(pipeDir, "stdin"),
		stdout: filepath.Join(pipeDir, "stdout"),
		stderr: filepath.Join(pipeDir, "stderr"),
	}

	// Create FIFOs
	for _, pipe := range []string{pipes.stdin, pipes.stdout, pipes.stderr} {
		if err := syscall.Mkfifo(pipe, 0600); err != nil {
			return nil, fmt.Errorf("create FIFO %s: %w", pipe, err)
		}
	}

	return pipes, nil
}

// buildOpenCommand constructs the open command with appropriate arguments.
func (s *ServicesLauncher) buildOpenCommand(ctx context.Context, bundlePath string, pipes *pipeSet) (*exec.Cmd, error) {
	args := []string{
		"-a", bundlePath,
		"--wait-apps",
		"--stdin", pipes.stdin,
		"--stdout", pipes.stdout,
		"--stderr", pipes.stderr,
	}

	// Add command line arguments if present
	if len(os.Args) > 1 {
		args = append(args, "--args")
		args = append(args, os.Args[1:]...)
	}

	return exec.CommandContext(ctx, "open", args...), nil
}

// startIOForwarding sets up goroutines to forward I/O between the parent process and the named pipes.
func (s *ServicesLauncher) startIOForwarding(pipes *pipeSet, errChan chan error, debug bool) {
	// Forward stdin
	go func() {
		if debug {
			fmt.Fprintf(os.Stderr, "macgo: starting stdin forwarding\n")
		}
		err := s.forwardStdin(pipes.stdin)
		if debug && err != nil {
			fmt.Fprintf(os.Stderr, "macgo: stdin forwarding error: %v\n", err)
		}
		errChan <- err
	}()

	// Forward stdout
	go func() {
		if debug {
			fmt.Fprintf(os.Stderr, "macgo: starting stdout forwarding\n")
		}
		err := s.forwardStdout(pipes.stdout)
		if debug && err != nil {
			fmt.Fprintf(os.Stderr, "macgo: stdout forwarding error: %v\n", err)
		}
		errChan <- err
	}()

	// Forward stderr
	go func() {
		if debug {
			fmt.Fprintf(os.Stderr, "macgo: starting stderr forwarding\n")
		}
		err := s.forwardStderr(pipes.stderr)
		if debug && err != nil {
			fmt.Fprintf(os.Stderr, "macgo: stderr forwarding error: %v\n", err)
		}
		errChan <- err
	}()
}

// forwardStdin forwards data from the parent's stdin to the named pipe.
func (s *ServicesLauncher) forwardStdin(stdinPipe string) error {
	w, err := os.OpenFile(stdinPipe, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("open stdin pipe for writing: %w", err)
	}
	defer w.Close()

	_, err = io.Copy(w, os.Stdin)
	if err != nil {
		return fmt.Errorf("copy stdin: %w", err)
	}
	return nil
}

// forwardStdout forwards data from the named pipe to the parent's stdout.
func (s *ServicesLauncher) forwardStdout(stdoutPipe string) error {
	r, err := os.OpenFile(stdoutPipe, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("open stdout pipe for reading: %w", err)
	}
	defer r.Close()

	_, err = io.Copy(os.Stdout, r)
	if err != nil {
		return fmt.Errorf("copy stdout: %w", err)
	}
	return nil
}

// forwardStderr forwards data from the named pipe to the parent's stderr.
func (s *ServicesLauncher) forwardStderr(stderrPipe string) error {
	r, err := os.OpenFile(stderrPipe, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("open stderr pipe for reading: %w", err)
	}
	defer r.Close()

	_, err = io.Copy(os.Stderr, r)
	if err != nil {
		return fmt.Errorf("copy stderr: %w", err)
	}
	return nil
}
