package launch

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// ServicesLauncherV2 implements launching via LaunchServices using config files for I/O forwarding.
// EXPERIMENTAL: V2 is for testing new approaches. V1 is the stable, recommended version.
//
// V2 supports three I/O strategies:
// 1. Config-file strategy (default): Same as V1, child reads pipe paths from config file
// 2. Flags-based strategy (MACGO_V2_USE_FLAGS=1): Uses open -i/-o/--stderr flags
// 3. Env-flags strategy (MACGO_V2_USE_ENV_FLAGS=1): Uses open --env to pass pipe paths
//
// The flags-based strategy is EXPERIMENTAL and tests whether open's native I/O flags
// can work with regular files (not FIFOs). Theory: regular files don't block on open(),
// so xpcproxy shouldn't hang during posix_spawn setup.
//
// The env-flags strategy passes MACGO_STDOUT_PIPE and MACGO_STDERR_PIPE via --env flags,
// eliminating the need for config file discovery on the child side. Requires -n flag.
type ServicesLauncherV2 struct {
	logger      *Logger
	mu          sync.Mutex // protects process access during signal forwarding
	useFlags    bool       // true if using flags-based I/O strategy (-o/--stderr)
	useEnvFlags bool       // true if using env-flags strategy (--env)
	doneFile    string     // path to sentinel file that child writes when exiting
}

// Launch executes the application using LaunchServices with env-var-based I/O forwarding.
func (s *ServicesLauncherV2) Launch(ctx context.Context, bundlePath, execPath string, cfg *Config) error {
	// Initialize logger if not already set
	if s.logger == nil {
		s.logger = NewLogger()
	}

	// Check if using experimental I/O strategies
	s.useFlags = os.Getenv("MACGO_V2_USE_FLAGS") == "1"
	s.useEnvFlags = os.Getenv("MACGO_V2_USE_ENV_FLAGS") == "1"

	if s.useEnvFlags {
		s.logger.Debug("using ServicesLauncherV2 (EXPERIMENTAL env-flags I/O strategy)")
		s.logger.Debug("passing pipe paths via open --env flags")
	} else if s.useFlags {
		s.logger.Debug("using ServicesLauncherV2 (EXPERIMENTAL flags-based I/O strategy)")
		s.logger.Debug("testing open -i/-o/--stderr flags with regular files")
	} else {
		s.logger.Debug("using ServicesLauncherV2 (config-file I/O strategy)")
	}

	// Set up signal handling context
	sigCtx, stop := signal.NotifyContext(ctx,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGHUP,
		syscall.SIGPIPE,
	)
	defer stop()
	ctx = sigCtx

	// Determine if we should use no-wait mode
	noWait := os.Getenv("MACGO_NO_WAIT") == "1" || os.Getenv("MACGO_SERVICES_VERSION") == "3"

	if noWait {
		s.logger.Debug("launching with LaunchServices V2 (no-wait mode)")
	} else {
		s.logger.Debug("launching with LaunchServices V2 (wait mode)")
	}

	var pipes *pipeSet
	var pipeDir string

	// I/O forwarding is enabled by default in v2 (stdout/stderr only)
	// Stdin forwarding requires MACGO_ENABLE_STDIN_FORWARDING=1
	// All I/O forwarding can be disabled with MACGO_DISABLE_IO_FORWARDING=1
	if os.Getenv("MACGO_DISABLE_IO_FORWARDING") != "1" {
		// Create temporary directory for pipes
		var err error
		pipeDir, err = s.createPipeDirectory()
		if err != nil {
			return fmt.Errorf("create pipe directory: %w", err)
		}
		defer s.cleanupPipeDirectory(pipeDir)

		// V2 uses regular files by default (config-file strategy proven safe)
		// Enable stdin pipe by default unless explicitly disabled
		enableStdin := os.Getenv("MACGO_ENABLE_STDIN_FORWARDING") != "0"
		pipes, err = s.createPipes(pipeDir, enableStdin)
		if err != nil {
			return fmt.Errorf("create pipes: %w", err)
		}
	}

	// Write pipe configuration to file (only if not using flags mode or env-flags mode)
	var configFile string
	if pipes != nil && !s.useFlags && !s.useEnvFlags {
		configFile = filepath.Join(pipeDir, "config")
		if err := s.writePipeConfig(configFile, pipes); err != nil {
			return fmt.Errorf("write pipe config: %w", err)
		}
	}

	// Build the open command
	// - flags mode: pass -i/-o/--stderr pointing to regular files
	// - config mode: child discovers pipes via config file
	cmd, err := s.buildOpenCommand(sigCtx, bundlePath, configFile, pipes, noWait)
	if err != nil {
		return fmt.Errorf("build open command: %w", err)
	}

	// Set process group to ensure child processes are cleaned up
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	s.logger.Debug("launching command",
		"path", cmd.Path,
		"args", cmd.Args[1:],
		"full_command", cmd.String())

	// Start the open command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start open command: %w", err)
	}

	s.logger.Debug("open command started", "pid", cmd.Process.Pid)

	// Set up launch timeout to prevent hung xpcproxy
	launchTimeout := 30 * time.Second
	if timeoutEnv := os.Getenv("MACGO_LAUNCH_TIMEOUT"); timeoutEnv != "" {
		if d, err := time.ParseDuration(timeoutEnv); err == nil {
			if d < 1*time.Second {
				s.logger.Warn("launch timeout too short, using minimum", "requested", d, "minimum", 1*time.Second)
				launchTimeout = 1 * time.Second
			} else {
				launchTimeout = d
			}
		} else {
			s.logger.Warn("invalid launch timeout value, using default", "value", timeoutEnv, "error", err, "default", launchTimeout)
		}
	}

	s.logger.Debug("launch timeout configured", "timeout", launchTimeout)

	// Create a timer to kill hung processes
	launchTimer := time.AfterFunc(launchTimeout, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if cmd.Process != nil {
			pid := cmd.Process.Pid
			// Check if process is still running by sending signal 0
			if err := syscall.Kill(pid, 0); err != nil {
				// Process already gone, ignore
				return
			}
			s.logger.Warn("launch timeout exceeded, killing process", "pid", pid, "timeout", launchTimeout)

			// Kill the entire process group with SIGKILL
			if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
				s.logger.Error("failed to kill hung process", "pid", pid, "error", err)
			} else {
				s.logger.Debug("successfully killed hung process", "pid", pid)
			}
		}
	})
	// We don't defer Stop() here because we need to stop it as soon as the Open command finishes,
	// which might be much earlier than when Launch() returns (in pipe mode).
	stopTimer := func() { launchTimer.Stop() }

	// Monitor context for cancellation to forward signals (only in wait mode)
	if !noWait {
		go func() {
			<-sigCtx.Done()
			s.mu.Lock()
			defer s.mu.Unlock()

			if cmd.Process != nil {
				pid := cmd.Process.Pid
				s.logger.Debug("context cancelled, forwarding signal to process group", "pid", pid)

				// First try SIGINT (Ctrl+C)
				_ = syscall.Kill(-pid, syscall.SIGINT)
			}

			// Release mutex before sleep to allow other operations
			s.mu.Unlock()
			time.Sleep(100 * time.Millisecond)
			s.mu.Lock()

			// If still running, send SIGTERM
			if cmd.Process != nil {
				pid := cmd.Process.Pid
				s.logger.Debug("escalating to SIGTERM", "pid", pid)
				_ = syscall.Kill(-pid, syscall.SIGTERM)
			}
		}()
	}

	// Set up I/O forwarding if pipes are available
	var ioErrChan chan error
	if pipes != nil {
		if s.useFlags {
			s.logger.Debug("EXPERIMENTAL: waiting for I/O via open flags",
				"stdin", pipes.stdin,
				"stdout", pipes.stdout,
				"stderr", pipes.stderr)
			s.logger.Debug("if this hangs, open flags don't work with .app bundles")
		} else {
			s.logger.Debug("pipes created for config-file forwarding",
				"stdin", pipes.stdin,
				"stdout", pipes.stdout,
				"stderr", pipes.stderr)
		}

		ioErrChan = make(chan error, 2) // stdout and stderr

		// Create cancellable context for stdin forwarding
		stdinCtx, stdinCancel := context.WithCancel(ctx)
		defer stdinCancel()

		s.startIOForwarding(stdinCtx, pipes, ioErrChan)
	} else {
		s.logger.Debug("no pipes created (I/O forwarding disabled)")
	}

	// Handle wait vs no-wait modes
	if noWait {
		return s.handleNoWaitMode(sigCtx, ioErrChan, pipeDir, stopTimer)
	} else {
		return s.handleWaitMode(sigCtx, cmd, ioErrChan, pipeDir, stopTimer)
	}
}

// handleWaitMode waits for command completion in wait mode
func (s *ServicesLauncherV2) handleWaitMode(ctx context.Context, cmd *exec.Cmd, ioErrChan chan error, pipeDir string, stopTimer func()) error {
	cmdDone := make(chan error, 1)
	go func() {
		s.logger.Debug("waiting for open command to complete...")
		err := cmd.Wait()
		s.logger.Debug("open command finished", "error", err)
		stopTimer() // Stop the launch timer as soon as open command finishes
		cmdDone <- err
	}()

	if ioErrChan != nil {
		// When doing I/O forwarding, the open command exits immediately after launching the app
		// (because we don't use -W flag). We must wait for I/O forwarding to complete, not for
		// the open command to exit. The app itself continues running and writes to the pipes.
		s.logger.Debug("waiting for stdout/stderr forwarding to complete...")

		ioCompleted := 0
		for ioCompleted < 2 {
			select {
			case err := <-ioErrChan:
				ioCompleted++
				if err != nil && err != context.Canceled {
					s.logger.Debug("I/O forwarding completed", "completed", ioCompleted, "error", err)
				} else {
					s.logger.Debug("I/O forwarding completed", "completed", ioCompleted)
				}

			case <-ctx.Done():
				s.logger.Debug("context cancelled while waiting for I/O", "completed", ioCompleted)
				return ctx.Err()
			}
		}

		// I/O forwarding is complete, which means the app finished writing.
		// The open command may have already exited (which is fine), just drain it if needed.
		s.logger.Debug("all I/O forwarding completed, checking open command status")
		select {
		case cmdErr := <-cmdDone:
			if cmdErr != nil {
				s.logger.Debug("open command exited with error", "error", cmdErr)
				// Don't fail if open exited with error but I/O completed successfully
			} else {
				s.logger.Debug("open command exited successfully")
			}
		default:
			// Open command still running, which is fine
			s.logger.Debug("open command still running (will be cleaned up)")
		}

		// Add safety timer to force exit if os.Exit doesn't work
		s.logger.Debug("calling os.Exit(0)")
		// Cleanup before exit (defer won't run with os.Exit)
		if pipeDir != "" {
			s.cleanupPipeDirectory(pipeDir)
		}
		go func() {
			time.Sleep(100 * time.Millisecond)
			s.logger.Warn("os.Exit did not terminate process, forcing SIGKILL")
			syscall.Kill(os.Getpid(), syscall.SIGKILL)
		}()
		os.Exit(0)
	}

	// No I/O forwarding, just wait for command to complete
	cmdErr := <-cmdDone
	if cmdErr != nil {
		if exitErr, ok := cmdErr.(*exec.ExitError); ok {
			s.logger.Debug("open command exited", "code", exitErr.ExitCode())
			// Cleanup before exit (defer won't run with os.Exit)
			if pipeDir != "" {
				s.cleanupPipeDirectory(pipeDir)
			}
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("open command failed: %w", cmdErr)
	}
	// Cleanup before exit (defer won't run with os.Exit)
	if pipeDir != "" {
		s.cleanupPipeDirectory(pipeDir)
	}
	os.Exit(0)
	return nil
}

// handleNoWaitMode handles launching without waiting for command completion
func (s *ServicesLauncherV2) handleNoWaitMode(ctx context.Context, ioErrChan chan error, pipeDir string, stopTimer func()) error {
	// In no-wait mode, we assume successful launch if we got this far
	stopTimer()

	if ioErrChan != nil {
		s.logger.Debug("monitoring I/O forwarding and signals...")

		ioCompleted := 0
		for ioCompleted < 2 {
			select {
			case err := <-ioErrChan:
				ioCompleted++
				if err != nil && err != context.Canceled {
					s.logger.Debug("I/O forwarding completed", "completed", ioCompleted, "error", err)
				} else {
					s.logger.Debug("I/O forwarding completed", "completed", ioCompleted)
				}

			case <-ctx.Done():
				s.logger.Debug("context cancelled while monitoring I/O", "completed", ioCompleted)
				return ctx.Err()
			}
		}

		s.logger.Debug("all I/O forwarding completed, exiting")
		// Cleanup before exit (defer won't run with os.Exit)
		if pipeDir != "" {
			s.cleanupPipeDirectory(pipeDir)
		}
		os.Exit(0)
	} else {
		<-ctx.Done()
		s.logger.Debug("received signal, exiting")
		// Cleanup before exit (defer won't run with os.Exit)
		if pipeDir != "" {
			s.cleanupPipeDirectory(pipeDir)
		}
		os.Exit(0)
	}

	return nil
}

// createPipeDirectory creates a temporary directory for the pipes.
// Uses PID + timestamp to ensure uniqueness across rapid sequential calls.
func (s *ServicesLauncherV2) createPipeDirectory() (string, error) {
	// Include nanosecond timestamp to ensure each invocation gets unique pipes
	// even when called rapidly from the same parent process
	pipeDir := filepath.Join(os.TempDir(), fmt.Sprintf("macgo-%d-%d", os.Getpid(), time.Now().UnixNano()))
	if err := os.MkdirAll(pipeDir, 0700); err != nil {
		return "", fmt.Errorf("create directory %s: %w", pipeDir, err)
	}
	return pipeDir, nil
}

// cleanupPipeDirectory removes the temporary pipe directory.
func (s *ServicesLauncherV2) cleanupPipeDirectory(pipeDir string) {
	if err := os.RemoveAll(pipeDir); err != nil {
		s.logger.Warn("failed to cleanup pipe directory", "path", pipeDir, "error", err)
	}
}

// createPipes creates regular files for I/O forwarding (never FIFOs in v2).
// enableStdin controls whether stdin pipe is created (default: false).
func (s *ServicesLauncherV2) createPipes(pipeDir string, enableStdin bool) (*pipeSet, error) {
	pipes := &pipeSet{
		stdout: filepath.Join(pipeDir, "stdout"),
		stderr: filepath.Join(pipeDir, "stderr"),
		done:   filepath.Join(pipeDir, "done"), // sentinel file written by child when it exits
	}

	// Store done file path for forwardStdout/forwardStderr to check
	s.doneFile = pipes.done
	s.logger.Debug("sentinel file path", "done", pipes.done)

	// Only create stdin pipe if explicitly enabled
	if enableStdin {
		pipes.stdin = filepath.Join(pipeDir, "stdin")
	}

	// V2 uses regular files (FIFOs are safe with config-file strategy but V2 predates that discovery)
	pipesToCreate := map[string]string{
		"stdout": pipes.stdout,
		"stderr": pipes.stderr,
	}
	if enableStdin {
		pipesToCreate["stdin"] = pipes.stdin
	}

	for name, pipe := range pipesToCreate {
		f, err := os.OpenFile(pipe, os.O_CREATE|os.O_RDWR, 0600)
		if err != nil {
			return nil, fmt.Errorf("create file %s: %w", pipe, err)
		}
		f.Close()
		s.logger.Debug("created pipe file", "name", name, "path", pipe)
	}

	return pipes, nil
}

// writePipeConfig writes pipe paths to a config file.
func (s *ServicesLauncherV2) writePipeConfig(configFile string, pipes *pipeSet) error {
	var config string

	// Always write stdout, stderr, and done file (sentinel for completion detection)
	config = fmt.Sprintf("MACGO_STDOUT_PIPE=%s\nMACGO_STDERR_PIPE=%s\nMACGO_DONE_FILE=%s\n",
		pipes.stdout, pipes.stderr, pipes.done)

	// Only write stdin if it was created
	if pipes.stdin != "" {
		config = fmt.Sprintf("MACGO_STDIN_PIPE=%s\n%s", pipes.stdin, config)
	}

	if err := os.WriteFile(configFile, []byte(config), 0600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	s.logger.Debug("wrote pipe config", "file", configFile, "done", pipes.done)
	return nil
}

// buildOpenCommand constructs the open command.
// In flags mode: uses -i/-o/--stderr flags (EXPERIMENTAL)
// In env-flags mode: uses --env to pass pipe paths (EXPERIMENTAL)
// In config mode: child discovers pipes via config file
func (s *ServicesLauncherV2) buildOpenCommand(ctx context.Context, bundlePath string, configFile string, pipes *pipeSet, noWait bool) (*exec.Cmd, error) {
	args := []string{}

	// Add -n flag if requested, or if using flags/env-flags mode
	// -n is required for:
	// - env-flags mode: --env only works with new instances
	// - flags mode: nested calls need separate instances for -o/--stderr to work
	if os.Getenv("MACGO_OPEN_NEW_INSTANCE") == "1" || s.useEnvFlags || s.useFlags {
		args = append(args, "-n")
		if s.useEnvFlags {
			s.logger.Debug("adding -n flag (required for --env)")
		} else if s.useFlags {
			s.logger.Debug("adding -n flag (required for -o/--stderr with nested calls)")
		}
	}

	// EXPERIMENTAL: Env-flags I/O strategy
	// Uses open's --env flag to pass pipe paths directly to child
	// Child reads MACGO_STDOUT_PIPE/MACGO_STDERR_PIPE from environment
	if s.useEnvFlags && pipes != nil {
		s.logger.Debug("EXPERIMENTAL: adding --env flags for pipe paths",
			"stdin", pipes.stdin,
			"stdout", pipes.stdout,
			"stderr", pipes.stderr,
			"done", pipes.done)

		// Add environment variable flags for pipe paths
		if pipes.stdin != "" {
			args = append(args, "--env", fmt.Sprintf("MACGO_STDIN_PIPE=%s", pipes.stdin))
		}
		if pipes.stdout != "" {
			args = append(args, "--env", fmt.Sprintf("MACGO_STDOUT_PIPE=%s", pipes.stdout))
		}
		if pipes.stderr != "" {
			args = append(args, "--env", fmt.Sprintf("MACGO_STDERR_PIPE=%s", pipes.stderr))
		}
		// Pass done file path so child can signal completion
		if pipes.done != "" {
			args = append(args, "--env", fmt.Sprintf("MACGO_DONE_FILE=%s", pipes.done))
		}

		// NOTE: When using --env, we should NOT use -W flag
		// -W is unreliable with I/O redirection (PID tracking fails)
		s.logger.Debug("skipping -W flag (unreliable with --env mode)")
	} else if s.useFlags && pipes != nil {
		// EXPERIMENTAL: Flags-based I/O strategy
		// Uses open's native -i/-o/--stderr flags with regular files
		// Theory: regular files don't block on open(), so xpcproxy shouldn't hang
		s.logger.Debug("EXPERIMENTAL: adding -i/-o/--stderr flags",
			"stdin", pipes.stdin,
			"stdout", pipes.stdout,
			"stderr", pipes.stderr,
			"done", pipes.done)

		// Add I/O redirection flags
		if pipes.stdin != "" {
			args = append(args, "-i", pipes.stdin)
		}
		if pipes.stdout != "" {
			args = append(args, "-o", pipes.stdout)
		}
		if pipes.stderr != "" {
			args = append(args, "--stderr", pipes.stderr)
		}
		// Pass done file via environment so child knows where to signal completion
		// (requires -n flag which is already added above for flags mode)
		if pipes.done != "" {
			args = append(args, "--env", fmt.Sprintf("MACGO_DONE_FILE=%s", pipes.done))
		}

		// NOTE: When using -i/-o/--stderr, we should NOT use -W flag
		// The -W flag conflicts with I/O redirection
		s.logger.Debug("skipping -W flag (conflicts with I/O flags)")
	} else {
		// Add wait flag based on mode (only in config mode)
		if !noWait {
			// Only add -W if no config file (v2 doesn't use -W with pipes to avoid conflicts)
			if configFile == "" {
				args = append(args, "-W")
			}
		}
	}

	// Add the bundle path
	if noWait {
		args = append(args, "-a", bundlePath)
	} else {
		args = append(args, bundlePath)
	}

	// Config-file strategy: Child discovers pipes via config file
	if configFile != "" {
		s.logger.Debug("using config-file I/O strategy (v2)", "config", configFile)
		// Don't pass via --args since open doesn't reliably pass args to .app bundles
		// Child will find the config file by scanning /tmp/macgo-*/config
	}

	// Add user's command line arguments if present
	if len(os.Args) > 1 {
		args = append(args, "--args")
		args = append(args, os.Args[1:]...)
	}

	return exec.CommandContext(ctx, "open", args...), nil
}

// startIOForwarding sets up goroutines to forward I/O between parent and pipes.
func (s *ServicesLauncherV2) startIOForwarding(stdinCtx context.Context, pipes *pipeSet, errChan chan error) {
	// Forward stdin with cancellation context (only if stdin pipe was created)
	if pipes.stdin != "" {
		go func() {
			s.logger.Debug("starting stdin forwarding")
			err := s.forwardStdin(stdinCtx, pipes.stdin)
			if err != nil && err != context.Canceled {
				s.logger.Debug("stdin forwarding error", "error", err)
			}
		}()
	}

	// Forward stdout
	go func() {
		s.logger.Debug("starting stdout forwarding")
		err := s.forwardStdout(pipes.stdout)
		if err != nil {
			s.logger.Debug("stdout forwarding error", "error", err)
		}
		errChan <- err
	}()

	// Forward stderr
	go func() {
		s.logger.Debug("starting stderr forwarding")
		err := s.forwardStderr(pipes.stderr)
		if err != nil {
			s.logger.Debug("stderr forwarding error", "error", err)
		}
		errChan <- err
	}()
}

// forwardStdin forwards data from parent's stdin to the pipe.
func (s *ServicesLauncherV2) forwardStdin(ctx context.Context, stdinPipe string) error {
	if stdinPipe == "" {
		return fmt.Errorf("stdin pipe path is empty")
	}

	s.logger.Debug("opening stdin pipe for writing", "path", stdinPipe)

	// Open the pipe in non-blocking mode initially to avoid hanging on open
	w, err := os.OpenFile(stdinPipe, os.O_WRONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return fmt.Errorf("open stdin pipe for writing (%s): %w", stdinPipe, err)
	}
	defer func() { _ = w.Close() }()

	// Switch back to blocking mode for the actual copy
	if err := syscall.SetNonblock(int(w.Fd()), false); err != nil {
		return fmt.Errorf("set blocking mode: %w", err)
	}

	s.logger.Debug("stdin pipe opened, starting copy")

	// Create a channel to signal when copy is done
	done := make(chan struct{})
	var copyErr error
	var n int64

	// Run the copy in a goroutine
	go func() {
		n, copyErr = io.Copy(w, os.Stdin)
		close(done)
	}()

	// Wait for either context cancellation or copy completion
	select {
	case <-ctx.Done():
		s.logger.Debug("stdin forwarding cancelled (process exited)")
		// Close the write end to interrupt the copy if it's still running
		w.Close()
		return context.Canceled
	case <-done:
		s.logger.Debug("stdin forwarding completed", "bytes", n)
		if copyErr != nil {
			return fmt.Errorf("copy stdin: %w", copyErr)
		}
		return nil
	}
}

// forwardStdout forwards data from the pipe to parent's stdout.
// With regular files (not FIFOs), we need to poll continuously until the writer closes.
func (s *ServicesLauncherV2) forwardStdout(stdoutPipe string) error {
	if stdoutPipe == "" {
		return fmt.Errorf("stdout pipe path is empty")
	}

	s.logger.Debug("opening stdout pipe for reading", "path", stdoutPipe)

	// Wait for child to start writing (poll until file has content or timeout)
	// This is necessary because with regular files, opening O_RDONLY on an empty
	// file causes io.Copy to return immediately with 0 bytes
	timeout := 500 * time.Millisecond
	pollInterval := 10 * time.Millisecond
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		info, err := os.Stat(stdoutPipe)
		if err == nil && info.Size() > 0 {
			s.logger.Debug("stdout pipe has content, starting read", "size", info.Size())
			break
		}
		time.Sleep(pollInterval)
	}

	// Open the file and continuously read until child closes it
	// With regular files, we need to poll because io.Copy returns EOF immediately
	r, err := os.OpenFile(stdoutPipe, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("open stdout pipe: %w", err)
	}
	defer r.Close()

	s.logger.Debug("stdout pipe opened, starting continuous read")

	totalBytes := int64(0)
	buf := make([]byte, 32*1024)
	lastSize := int64(0)
	noGrowthCount := 0
	maxNoGrowth := 50 // 50 * 100ms = 5 seconds of no growth

	for {
		// Try to read available data
		n, err := r.Read(buf)
		if n > 0 {
			totalBytes += int64(n)
			os.Stdout.Write(buf[:n])
			noGrowthCount = 0 // Reset on successful read
		}

		if err == io.EOF {
			// Check if file is still growing by checking size
			info, statErr := os.Stat(stdoutPipe)
			if statErr == nil {
				currentSize := info.Size()
				if currentSize > lastSize {
					// File grew, continue reading
					lastSize = currentSize
					noGrowthCount = 0
					r.Seek(totalBytes, 0) // Seek to continue from where we left off
					continue
				}
			}

			// File hasn't grown, increment no-growth counter
			noGrowthCount++
			if noGrowthCount >= maxNoGrowth {
				// Check if done file exists before giving up
				if s.doneFile != "" {
					if _, err := os.Stat(s.doneFile); err == nil {
						// Done file exists, child really finished
						// Do one final read pass to get any remaining data
						r.Seek(totalBytes, 0)
						for {
							n, readErr := r.Read(buf)
							if n > 0 {
								totalBytes += int64(n)
								os.Stdout.Write(buf[:n])
							}
							if readErr == io.EOF || n == 0 {
								break
							}
							if readErr != nil {
								break
							}
						}
						s.logger.Debug("stdout copy completed (done file exists)", "bytes", totalBytes)
						return nil
					}
					// Done file doesn't exist yet, child still running - wait indefinitely
					s.logger.Debug("stdout no growth but done file not found, waiting indefinitely", "bytes", totalBytes)
					noGrowthCount = 0 // Reset fully - wait indefinitely for done file
					time.Sleep(100 * time.Millisecond)
					r.Seek(totalBytes, 0)
					continue
				}
				// No done file configured, use original timeout behavior
				s.logger.Debug("stdout copy completed (no growth timeout)", "bytes", totalBytes)
				return nil
			}

			// Wait a bit and retry
			time.Sleep(100 * time.Millisecond)
			r.Seek(totalBytes, 0)
			continue
		}

		if err != nil {
			s.logger.Debug("stdout copy completed with error", "bytes", totalBytes, "error", err)
			return fmt.Errorf("copy stdout: %w", err)
		}
	}
}

// forwardStderr forwards data from the pipe to parent's stderr.
// With regular files (not FIFOs), we need to poll continuously until the writer closes.
func (s *ServicesLauncherV2) forwardStderr(stderrPipe string) error {
	if stderrPipe == "" {
		return fmt.Errorf("stderr pipe path is empty")
	}

	s.logger.Debug("opening stderr pipe for reading", "path", stderrPipe)

	// Wait for child to start writing (poll until file has content or timeout)
	// This is necessary because with regular files, opening O_RDONLY on an empty
	// file causes io.Copy to return immediately with 0 bytes
	timeout := 500 * time.Millisecond
	pollInterval := 10 * time.Millisecond
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		info, err := os.Stat(stderrPipe)
		if err == nil && info.Size() > 0 {
			s.logger.Debug("stderr pipe has content, starting read", "size", info.Size())
			break
		}
		time.Sleep(pollInterval)
	}

	// Open the file and continuously read until child closes it
	// With regular files, we need to poll because io.Copy returns EOF immediately
	r, err := os.OpenFile(stderrPipe, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("open stderr pipe: %w", err)
	}
	defer r.Close()

	s.logger.Debug("stderr pipe opened, starting continuous read")

	totalBytes := int64(0)
	buf := make([]byte, 32*1024)
	lastSize := int64(0)
	noGrowthCount := 0
	maxNoGrowth := 50 // 50 * 100ms = 5 seconds of no growth

	for {
		// Try to read available data
		n, err := r.Read(buf)
		if n > 0 {
			totalBytes += int64(n)
			os.Stderr.Write(buf[:n])
			noGrowthCount = 0 // Reset on successful read
		}

		if err == io.EOF {
			// Check if file is still growing by checking size
			info, statErr := os.Stat(stderrPipe)
			if statErr == nil {
				currentSize := info.Size()
				if currentSize > lastSize {
					// File grew, continue reading
					lastSize = currentSize
					noGrowthCount = 0
					r.Seek(totalBytes, 0) // Seek to continue from where we left off
					continue
				}
			}

			// File hasn't grown, increment no-growth counter
			noGrowthCount++
			if noGrowthCount >= maxNoGrowth {
				// Check if done file exists before giving up
				if s.doneFile != "" {
					if _, err := os.Stat(s.doneFile); err == nil {
						// Done file exists, child really finished
						// Do one final read pass to get any remaining data
						r.Seek(totalBytes, 0)
						for {
							n, readErr := r.Read(buf)
							if n > 0 {
								totalBytes += int64(n)
								os.Stderr.Write(buf[:n])
							}
							if readErr == io.EOF || n == 0 {
								break
							}
							if readErr != nil {
								break
							}
						}
						s.logger.Debug("stderr copy completed (done file exists)", "bytes", totalBytes)
						return nil
					}
					// Done file doesn't exist yet, child still running - wait indefinitely
					s.logger.Debug("stderr no growth but done file not found, waiting indefinitely", "bytes", totalBytes)
					noGrowthCount = 0 // Reset fully - wait indefinitely for done file
					time.Sleep(100 * time.Millisecond)
					r.Seek(totalBytes, 0)
					continue
				}
				// No done file configured, use original timeout behavior
				s.logger.Debug("stderr copy completed (no growth timeout)", "bytes", totalBytes)
				return nil
			}

			// Wait a bit and retry
			time.Sleep(100 * time.Millisecond)
			r.Seek(totalBytes, 0)
			continue
		}

		if err != nil {
			s.logger.Debug("stderr copy completed with error", "bytes", totalBytes, "error", err)
			return fmt.Errorf("copy stderr: %w", err)
		}
	}
}
