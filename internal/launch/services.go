package launch

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// ServicesLauncher implements launching via LaunchServices using the 'open' command.
//
// # Signal Handling Architecture
//
// When launching apps via LaunchServices, the child process is adopted by launchd
// (PPID=1) and is NOT in the parent's process group. This means terminal signals
// like Ctrl+C (SIGINT) only reach the parent wrapper, not the child.
//
// To ensure proper signal handling, we implement the following:
//
//   - PID Tracking: Child writes its PID to a file in the pipe directory after
//     startup. Parent polls for this file and stores the PID for signal forwarding.
//
//   - Signal Forwarding: When parent receives SIGINT/SIGTERM (via context
//     cancellation), it forwards these signals to the child before exiting.
//     This prevents orphaned child processes.
//
//   - SIGWINCH Forwarding: Terminal resize events are forwarded to the child
//     so apps can adjust their output to the new terminal size.
//
//   - Exit Codes: Parent exits with 130 (128+SIGINT) when interrupted by Ctrl+C.
//     Child exits with 128+signal when terminated by a signal.
//
// Note: SIGTSTP/SIGCONT (job control) is not currently supported. Pressing Ctrl+Z
// will suspend the parent but the child will continue running.
type ServicesLauncher struct {
	logger        *Logger
	mu            sync.Mutex    // protects process access during signal forwarding
	doneFile      string        // path to sentinel file that child writes when exiting
	firstOutputCh chan struct{} // closed when first output is received (signals successful launch)
	useFifo       bool          // true if using FIFOs (EOF signals completion, no done file needed)
	childPID      int           // PID of the actual app process (for signal forwarding)
	pidFile       string        // path to file where child writes its PID
	lastSignal    atomic.Int32  // most recent signal received by the parent
}

// pipeSet holds the paths to the named pipes used for I/O forwarding.
type pipeSet struct {
	stdin  string
	stdout string
	stderr string
	done   string // sentinel file written by child when it exits
	pid    string // file where child writes its PID
}

// waitForChildPID waits for the child to write its PID file and returns the PID.
// Returns 0 if the PID file doesn't appear within the timeout.
func (s *ServicesLauncher) waitForChildPID(pidFile string, timeout time.Duration) int {
	if pidFile == "" {
		return 0
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(pidFile)
		if err == nil {
			pidStr := strings.TrimSpace(string(data))
			if pid, err := strconv.Atoi(pidStr); err == nil && pid > 0 {
				s.mu.Lock()
				s.childPID = pid
				s.mu.Unlock()
				s.logger.Info("read child PID for signal forwarding", "pid", pid)
				return pid
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	s.logger.Warn("timeout waiting for child PID file", "file", pidFile, "timeout", timeout)
	return 0
}

// forwardSignalToChild sends a signal to the child process if we know its PID.
func (s *ServicesLauncher) forwardSignalToChild(sig syscall.Signal) {
	s.mu.Lock()
	pid := s.childPID
	s.mu.Unlock()

	if pid <= 0 {
		s.logger.Warn("cannot forward signal - child PID unknown", "signal", sig)
		return
	}

	if err := syscall.Kill(pid, sig); err != nil {
		// Process may have already exited
		if err != syscall.ESRCH {
			s.logger.Warn("failed to forward signal to child", "signal", sig, "pid", pid, "error", err)
		}
	} else {
		s.logger.Info("forwarded signal to child", "signal", sig, "pid", pid)
	}
}

func (s *ServicesLauncher) recordedSignal() syscall.Signal {
	sig := syscall.Signal(s.lastSignal.Load())
	if sig == 0 {
		return syscall.SIGINT
	}
	return sig
}

func (s *ServicesLauncher) forwardSignalToChildWithGrace(sig syscall.Signal, grace time.Duration) {
	s.forwardSignalToChild(sig)
	time.Sleep(grace)

	s.mu.Lock()
	pid := s.childPID
	s.mu.Unlock()

	if pid <= 0 {
		return
	}
	if err := syscall.Kill(pid, 0); err != nil {
		return
	}
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && err != syscall.ESRCH {
		s.logger.Warn("failed to force-kill child after signal grace period", "pid", pid, "error", err)
	}
}

func (s *ServicesLauncher) cancellationExitCode(defaultCode int) int {
	sig := s.recordedSignal()
	if sig > 0 {
		return 128 + int(sig)
	}
	return defaultCode
}

func (s *ServicesLauncher) exitWithSignalForwarding(pipeDir string, defaultCode int) {
	sig := s.recordedSignal()
	s.forwardSignalToChildWithGrace(sig, 150*time.Millisecond)
	if pipeDir != "" {
		s.cleanupPipeDirectory(pipeDir)
	}
	os.Exit(s.cancellationExitCode(defaultCode))
}

// Launch executes the application using LaunchServices with I/O forwarding.
func (s *ServicesLauncher) Launch(ctx context.Context, bundlePath, execPath string, cfg *Config) error {
	// Initialize logger if not already set
	if s.logger == nil {
		s.logger = NewLogger()
	}

	// Set up signal handling context
	// Note: SIGQUIT is handled separately for stack dumps (forwarded to child)
	sigCtx, stop := signal.NotifyContext(ctx,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGHUP,
		syscall.SIGPIPE, // Handle broken pipe when output is piped
	)
	defer stop()
	ctx = sigCtx
	s.lastSignal.Store(0)

	sigRecordCh := make(chan os.Signal, 1)
	signal.Notify(sigRecordCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGPIPE)
	defer signal.Stop(sigRecordCh)
	go func() {
		for {
			select {
			case rawSig := <-sigRecordCh:
				if sig, ok := rawSig.(syscall.Signal); ok {
					s.lastSignal.Store(int32(sig))
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Determine if we should use no-wait mode
	noWait := os.Getenv("MACGO_NO_WAIT") == "1" || os.Getenv("MACGO_SERVICES_VERSION") == "3"

	if noWait {
		s.logger.Debug("launching with LaunchServices (no-wait mode)")
	} else {
		s.logger.Debug("launching with LaunchServices (wait mode)")
	}

	var pipes *pipeSet
	var pipeDir string

	// Create pipes for I/O forwarding (using FIFOs by default, or regular files)
	//
	// IMPORTANT: I/O Forwarding with LaunchServices
	// ==============================================
	//
	// CONFIG-FILE STRATEGY (DEFAULT, WORKS):
	// 1. Parent creates FIFOs in temp directory
	// 2. Parent writes FIFO paths to config file
	// 3. Parent runs: open -n bundle.app (NO -i/-o flags)
	// 4. xpcproxy spawns app with /dev/null for stdio (no FIFO involvement at spawn)
	// 5. Child reads config file, opens FIFOs for writing
	// 6. Parent's blocking open(O_RDONLY) completes when child opens
	// 7. io.Copy works, EOF when child closes - clean termination
	//
	// WHY FIFOs ARE SAFE WITH CONFIG-FILE:
	// - xpcproxy never touches FIFOs (no -i/-o flags passed to open)
	// - Child opens FIFOs AFTER being spawned, not during posix_spawn
	// - Parent opens with O_RDONLY in goroutine (blocks until child opens)
	// - Clean EOF semantics when child closes write end
	//
	// BROKEN APPROACHES (DO NOT USE):
	// 1. open -i fifo -o fifo --stderr fifo bundle.app
	//    - xpcproxy tries to open() FIFOs during posix_spawn setup
	//    - FIFO open(O_WRONLY) blocks waiting for reader
	//    - Deadlock: xpcproxy waits for parent, parent waits for app
	//
	// 2. MACGO_IO_STRATEGY=env-vars
	//    - macOS's `open --env` does NOT pass env vars to bundled apps
	//    - Child never receives MACGO_STDOUT_PIPE/MACGO_STDERR_PIPE
	//
	// 3. Direct execution bypasses TCC permission prompts
	//    - Works for I/O but user never sees permission dialogs
	//
	// I/O Forwarding Control:
	// - Default: Pipe-based I/O forwarding (works reliably with LaunchServices)
	// - MACGO_TTY_PASSTHROUGH=1           Pass TTY device directly to child (experimental)
	// - MACGO_DISABLE_IO_FORWARDING=1     Disable all I/O forwarding
	// - MACGO_USE_FIFO=0                  Use regular files instead of FIFOs (when using pipes)
	// - MACGO_IO_LOG_DIR=<path>           Directory to write I/O debug logs
	//
	// TTY Passthrough (experimental):
	// When MACGO_TTY_PASSTHROUGH=1 is set and running on a terminal, the parent's
	// terminal device is passed directly to the child. This preserves isatty()=true
	// but may have issues with some terminal configurations.
	useTTYPassthrough := os.Getenv("MACGO_TTY_PASSTHROUGH") == "1" && getTTYPath() != ""
	if useTTYPassthrough {
		s.logger.Debug("TTY passthrough enabled via MACGO_TTY_PASSTHROUGH=1", "tty", getTTYPath())
	}

	disableIO := os.Getenv("MACGO_DISABLE_IO_FORWARDING") == "1" || useTTYPassthrough

	// When TTY passthrough is active, no pipes are needed
	// When not on a TTY (CI, background, etc.), use pipe-based I/O forwarding
	enableStdin := !disableIO && shouldAutoEnableStdin()
	enableStdout := !disableIO
	enableStderr := !disableIO

	// FIFOs are default (clean EOF semantics)
	// Disable with MACGO_USE_FIFO=0 to use regular files with polling instead
	useFifo := os.Getenv("MACGO_USE_FIFO") != "0"
	s.useFifo = useFifo

	var configFile string
	// Always create pipe directory for PID tracking (even in TTY passthrough mode)
	// This allows signal forwarding to work regardless of I/O mode
	var err error
	pipeDir, err = s.createPipeDirectory()
	if err != nil {
		return fmt.Errorf("create pipe directory: %w", err)
	}
	defer s.cleanupPipeDirectory(pipeDir)

	if enableStdin || enableStdout || enableStderr {
		// Create named pipes for I/O forwarding
		pipes, err = s.createNamedPipes(pipeDir, enableStdin, enableStdout, enableStderr, useFifo)
		if err != nil {
			return fmt.Errorf("create named pipes: %w", err)
		}
	} else {
		// No I/O forwarding, but still need PID file path
		pipes = &pipeSet{
			pid: filepath.Join(pipeDir, "child.pid"),
		}
	}

	// Write config file for child discovery (includes bundle path for matching)
	configFile = filepath.Join(pipeDir, "config")
	if err := s.writePipeConfig(configFile, pipes, bundlePath); err != nil {
		return fmt.Errorf("write pipe config: %w", err)
	}
	s.logger.Debug("wrote pipe config", "config", configFile)

	// Build the launch command
	cmd, err := s.buildOpenCommand(sigCtx, bundlePath, pipes, noWait, cfg)
	if err != nil {
		return fmt.Errorf("build open command: %w", err)
	}

	// Set process group to ensure child processes are cleaned up
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Capture stderr of open command for debugging failures
	var openStderr bytes.Buffer
	cmd.Stderr = &openStderr

	s.logger.Debug("launching command",
		"path", cmd.Path,
		"args", cmd.Args[1:],
		"full_command", cmd.String())

	// Start the open command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start open command: %w", err)
	}

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
			s.logger.Warn("invalid launch timeout", "value", timeoutEnv, "error", err)
		}
	}

	// Create channel for signaling first output received (successful launch)
	// Must be created BEFORE the timer watcher goroutine starts
	s.firstOutputCh = make(chan struct{})

	// Create a timer to kill hung processes - but cancel it when first output is received
	launchTimerCh := make(chan struct{})
	var launchTimer *time.Timer
	launchTimer = time.AfterFunc(launchTimeout, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		select {
		case <-launchTimerCh:
			// Timer was cancelled, don't kill
			return
		default:
		}
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
	stopTimer := func() {
		launchTimer.Stop()
		// Also ensure channel closed signal logic knows it's stopped
		select {
		case <-launchTimerCh:
		default:
			close(launchTimerCh)
		}
	}

	// Watch for first output and cancel the launch timer when received
	if s.firstOutputCh != nil {
		go func() {
			select {
			case <-s.firstOutputCh:
				stopTimer()
			case <-launchTimerCh:
				// Timer already cancelled or fired
			}
		}()
	}

	if !noWait {
		// Forward SIGWINCH (terminal resize) to child process
		// This runs in a separate goroutine because SIGWINCH doesn't terminate
		// the process - it's an informational signal that apps use to adjust
		// their output to the new terminal size
		winchChan := make(chan os.Signal, 1)
		signal.Notify(winchChan, syscall.SIGWINCH)
		go func() {
			for range winchChan {
				s.logger.Debug("received SIGWINCH, forwarding to child", "parent_pid", os.Getpid())
				s.forwardSignalToChild(syscall.SIGWINCH)
			}
		}()

		// Forward SIGQUIT to child for stack dumps
		// The child's macgo signal handler will dump goroutine stacks
		// Set MACGO_NO_SIGQUIT_FORWARD=1 to disable (for custom handling)
		if os.Getenv("MACGO_NO_SIGQUIT_FORWARD") != "1" {
			quitChan := make(chan os.Signal, 1)
			signal.Notify(quitChan, syscall.SIGQUIT)
			go func() {
				for range quitChan {
					s.logger.Debug("received SIGQUIT, forwarding to child for stack dump", "parent_pid", os.Getpid())
					s.forwardSignalToChild(syscall.SIGQUIT)
				}
			}()
		}
	}

	// Always set up PID tracking for signal forwarding
	s.doneFile = pipes.done
	s.pidFile = pipes.pid

	// Start waiting for child PID in background (for signal forwarding)
	go func() {
		s.waitForChildPID(s.pidFile, 30*time.Second)
	}()

	// Set up I/O forwarding only if pipes are available
	var ioErrChan chan error
	var expectedIOCount int

	// Count how many output pipes we're waiting for (stdout and/or stderr)
	if pipes.stdout != "" {
		expectedIOCount++
	}
	if pipes.stderr != "" {
		expectedIOCount++
	}

	if expectedIOCount > 0 {
		ioErrChan = make(chan error, expectedIOCount)

		// Create cancellable context for stdin forwarding
		// NOTE: We use the main context, not a separate cancellable one, because
		// stdin forwarding should continue even after stdout/stderr complete.
		// For interactive apps, the child may be waiting for stdin input while
		// stdout/stderr are idle. Stdin is cancelled when:
		// 1. The main context is cancelled (signal received)
		// 2. The done file appears (child exited)
		// 3. The child closes its stdin (io.Copy returns)
		s.startIOForwarding(ctx, pipes, ioErrChan)
	}

	// Handle wait vs no-wait modes
	if noWait {
		return s.handleNoWaitMode(sigCtx, ioErrChan, expectedIOCount, pipeDir, launchTimer, stopTimer)
	} else {
		return s.handleWaitMode(sigCtx, cmd, ioErrChan, expectedIOCount, pipeDir, launchTimer, stopTimer, &openStderr)
	}
}

// handleWaitMode waits for command completion in wait mode
func (s *ServicesLauncher) handleWaitMode(ctx context.Context, cmd *exec.Cmd, ioErrChan chan error, expectedIOCount int, pipeDir string, launchTimer *time.Timer, stopTimer func(), openStderr *bytes.Buffer) error {
	// Wait for either command completion or IO forwarding completion
	cmdDone := make(chan error, 1)
	go func() {
		err := cmd.Wait()
		if err != nil && openStderr != nil && openStderr.Len() > 0 {
			s.logger.Debug("open command failed", "error", err, "stderr", openStderr.String())
		}
		stopTimer()
		cmdDone <- err
	}()

	// If we have pipes, wait for stdout/stderr or command completion
	if ioErrChan != nil && expectedIOCount > 0 {

		// I/O timeout is disabled by default since config-file strategy has proper EOF handling.
		// Can be explicitly enabled via MACGO_IO_TIMEOUT for debugging.
		var ioTimerChan <-chan time.Time
		if timeoutEnv := os.Getenv("MACGO_IO_TIMEOUT"); timeoutEnv != "" {
			if d, err := time.ParseDuration(timeoutEnv); err == nil && d > 0 {
				ioTimer := time.NewTimer(d)
				defer ioTimer.Stop()
				ioTimerChan = ioTimer.C
			}
		}

		// Collect IO errors
		var ioErrors []error
		ioCompleted := 0
		launchTimerStopped := false

		// Wait for all expected I/O streams to complete OR command to exit
		for ioCompleted < expectedIOCount {
			select {
			case err := <-ioErrChan:
				ioCompleted++
				// Stop launch timer on first I/O completion - app has successfully started
				if !launchTimerStopped && launchTimer != nil {
					launchTimer.Stop()
					launchTimerStopped = true
				}
				if err != nil && err != context.Canceled && !isBrokenPipeError(err) {
					ioErrors = append(ioErrors, err)
				}

			case cmdErr := <-cmdDone:
				// "open" command finished
				if cmdErr != nil {
					// If open command failed with -1712 (AppleEvent timeout), treat it as success.
					// This happens with pure Go binaries that don't process the AppleEvent loop.
					if exitErr, ok := cmdErr.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
						stderr := openStderr.String()
						s.logger.Debug("open command failed", "error", cmdErr, "stderr", stderr)
						if strings.Contains(stderr, "-1712") {
							s.logger.Warn("open command timed out (-1712) waiting for app check-in; ignoring as app seems compatible", "stderr", stderr)
							cmdErr = nil // Treat as success
						} else {
							s.logger.Error("open command failed", "error", cmdErr, "stderr", stderr)
							if pipeDir != "" {
								s.cleanupPipeDirectory(pipeDir)
							}
							os.Exit(exitErr.ExitCode())
						}
					} else {
						s.logger.Error("open command failed", "error", cmdErr)
						if pipeDir != "" {
							s.cleanupPipeDirectory(pipeDir)
						}
						return fmt.Errorf("open command failed: %w", cmdErr)
					}
				}

				// If successful (or ignored error), continue waiting for I/O
				// We don't exit here because we need I/O forwarding to finish (ioDone)
				continue

			case <-ctx.Done():
				// Kill the process and exit
				s.mu.Lock()
				if cmd.Process != nil {
					_ = cmd.Process.Kill()
				}
				s.mu.Unlock()
				s.exitWithSignalForwarding(pipeDir, 130)

			case <-ioTimerChan:
				s.logger.Warn("I/O forwarding timeout exceeded",
					"completed", ioCompleted,
					"expected", expectedIOCount)
				// Exit gracefully - the app likely completed but I/O never connected
				s.mu.Lock()
				if cmd.Process != nil {
					_ = cmd.Process.Kill()
				}
				s.mu.Unlock()
				// Cleanup before exit (defer won't run with os.Exit)
				if pipeDir != "" {
					s.cleanupPipeDirectory(pipeDir)
				}
				os.Exit(0)
			}
		}

		// All IO forwarding completed

		// With FIFOs, EOF on stdout/stderr means the child closed them (typically by exiting).
		// This is a reliable signal - no need to wait for the done file.
		// Only wait for done file when using regular files (polling mode) where we can't
		// detect child exit from EOF.
		if s.useFifo {
			goto exitCleanly
		}

		// If we have a doneFile sentinel, wait for it before exiting
		// This allows long-running servers to keep running even after initial output
		if s.doneFile != "" {
			for {
				select {
				case <-ctx.Done():
					s.mu.Lock()
					if cmd.Process != nil {
						_ = cmd.Process.Kill()
					}
					s.mu.Unlock()
					s.exitWithSignalForwarding(pipeDir, 130)
				default:
					if _, err := os.Stat(s.doneFile); err == nil {
						goto exitCleanly
					}
					time.Sleep(500 * time.Millisecond)
				}
			}
		}

	exitCleanly:
		// Kill the open process if it's still running
		s.mu.Lock()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		s.mu.Unlock()
		// Cleanup before exit (defer won't run with os.Exit)
		if pipeDir != "" {
			s.cleanupPipeDirectory(pipeDir)
		}
		os.Exit(0)

	} else {
		// No pipes (TTY passthrough mode), wait for command or signal
		select {
		case cmdErr := <-cmdDone:
			if cmdErr != nil {
				// Check for -1712 (AppleEvent timeout) - treat as success
				// This happens with pure Go binaries that don't process the AppleEvent loop.
				if exitErr, ok := cmdErr.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
					stderr := openStderr.String()
					if strings.Contains(stderr, "-1712") {
						s.logger.Warn("open command timed out (-1712) waiting for app check-in; app ran successfully", "stderr", stderr)
						if pipeDir != "" {
							s.cleanupPipeDirectory(pipeDir)
						}
						os.Exit(0)
					}
				}
				if exitErr, ok := cmdErr.(*exec.ExitError); ok {
					s.logger.Error("open command failed", "error", cmdErr, "stderr", openStderr.String())
					if pipeDir != "" {
						s.cleanupPipeDirectory(pipeDir)
					}
					os.Exit(exitErr.ExitCode())
				}
				return fmt.Errorf("open command failed: %w", cmdErr)
			}
		case <-ctx.Done():
			// Signal received - kill open command and exit with signal code
			s.mu.Lock()
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			s.mu.Unlock()
			s.exitWithSignalForwarding(pipeDir, 130)
		}
		if pipeDir != "" {
			s.cleanupPipeDirectory(pipeDir)
		}
		os.Exit(0)
	}
	return nil
}

// handleNoWaitMode handles launching without waiting for command completion
func (s *ServicesLauncher) handleNoWaitMode(ctx context.Context, ioErrChan chan error, expectedIOCount int, pipeDir string, launchTimer *time.Timer, stopTimer func()) error {
	// In no-wait mode, we stop the timer immediately if we got here (Launch presumed successful)
	stopTimer()

	// Since we're not waiting for the app, monitor for IO completion or signals
	if ioErrChan != nil && expectedIOCount > 0 {
		ioCompleted := 0
		launchTimerStopped := false
		for ioCompleted < expectedIOCount {
			select {
			case err := <-ioErrChan:
				ioCompleted++
				if !launchTimerStopped && launchTimer != nil {
					launchTimer.Stop()
					launchTimerStopped = true
				}
				_ = err // Error already logged in startIOForwarding

			case <-ctx.Done():
				s.exitWithSignalForwarding(pipeDir, 130)
			}
		}

		// With FIFOs, EOF on stdout/stderr means the child closed them (typically by exiting).
		if s.useFifo {
			if pipeDir != "" {
				s.cleanupPipeDirectory(pipeDir)
			}
			os.Exit(0)
		}

		// If we have a doneFile sentinel, wait for it before exiting
		if s.doneFile != "" {
			for {
				select {
				case <-ctx.Done():
					s.exitWithSignalForwarding(pipeDir, 130)
				default:
					if _, err := os.Stat(s.doneFile); err == nil {
						if pipeDir != "" {
							s.cleanupPipeDirectory(pipeDir)
						}
						os.Exit(0)
					}
					time.Sleep(500 * time.Millisecond)
				}
			}
		}

		// No done file configured, exit immediately
		// Cleanup before exit (defer won't run with os.Exit)
		if pipeDir != "" {
			s.cleanupPipeDirectory(pipeDir)
		}
		os.Exit(0)

	} else {
		// No pipes - behavior depends on MACGO_PARENT_WAIT flag
		if os.Getenv("MACGO_PARENT_WAIT") == "1" {
			<-ctx.Done()
			s.exitWithSignalForwarding(pipeDir, 130)
		}
		if pipeDir != "" {
			s.cleanupPipeDirectory(pipeDir)
		}
		os.Exit(0)
	}

	return nil
}

// createPipeDirectory creates a temporary directory for the named pipes.
// Uses PID + timestamp to ensure uniqueness across rapid sequential calls.
// Pipes are stored in ~/Library/Application Support/macgo/ for security -
// this location is user-specific and protected by macOS sandbox rules.
func (s *ServicesLauncher) createPipeDirectory() (string, error) {
	// Clean up stale directories from previous runs (non-blocking)
	go cleanupStalePipeDirectories()

	// Use ~/Library/Application Support/macgo/ for security
	// Falls back to /tmp/macgo/ if home dir unavailable
	var baseDir string
	if home, err := os.UserHomeDir(); err == nil {
		baseDir = filepath.Join(home, "Library", "Application Support", "macgo", "pipes")
	} else {
		baseDir = filepath.Join(os.TempDir(), "macgo")
	}
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return "", fmt.Errorf("create macgo base directory %s: %w", baseDir, err)
	}

	// Include nanosecond timestamp to ensure each invocation gets unique pipes
	// even when called rapidly from the same parent process
	pipeDir := filepath.Join(baseDir, fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano()))
	if err := os.MkdirAll(pipeDir, 0700); err != nil {
		return "", fmt.Errorf("create directory %s: %w", pipeDir, err)
	}
	return pipeDir, nil
}

// cleanupPipeDirectory removes the temporary pipe directory.
func (s *ServicesLauncher) cleanupPipeDirectory(pipeDir string) {
	if err := os.RemoveAll(pipeDir); err != nil {
		s.logger.Warn("failed to cleanup pipe directory", "path", pipeDir, "error", err)
	}
}

// cleanupStalePipeDirectories removes pipe directories older than 24 hours.
// Called asynchronously on startup to prevent accumulation of stale directories.
func cleanupStalePipeDirectories() {
	var baseDir string
	if home, err := os.UserHomeDir(); err == nil {
		baseDir = filepath.Join(home, "Library", "Application Support", "macgo", "pipes")
	} else {
		baseDir = filepath.Join(os.TempDir(), "macgo")
	}

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return // Directory may not exist yet
	}

	maxAge := 24 * time.Hour
	now := time.Now()

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if now.Sub(info.ModTime()) > maxAge {
			_ = os.RemoveAll(filepath.Join(baseDir, entry.Name()))
		}
	}
}

// writePipeConfig writes pipe paths to a config file (for config-file strategy).
func (s *ServicesLauncher) writePipeConfig(configFile string, pipes *pipeSet, bundlePath string) error {
	var config string

	// Preserve current working directory
	cwd, _ := os.Getwd()

	// Write stdout, stderr, bundle path (for matching), and CWD
	config = fmt.Sprintf("MACGO_STDOUT_PIPE=%s\nMACGO_STDERR_PIPE=%s\nMACGO_BUNDLE_PATH=%s\nMACGO_CWD=%s\n",
		pipes.stdout, pipes.stderr, bundlePath, cwd)

	// Only write done file path for non-FIFO mode
	if pipes.done != "" {
		config = fmt.Sprintf("MACGO_DONE_FILE=%s\n%s", pipes.done, config)
	}

	// Only write stdin if it was created
	if pipes.stdin != "" {
		config = fmt.Sprintf("MACGO_STDIN_PIPE=%s\n%s", pipes.stdin, config)
	}

	// Pass original executable path so the child can find the real binary
	if origExec := os.Getenv("MACGO_ORIGINAL_EXECUTABLE"); origExec != "" {
		config = fmt.Sprintf("MACGO_ORIGINAL_EXECUTABLE=%s\n%s", origExec, config)
	}

	if err := os.WriteFile(configFile, []byte(config), 0600); err != nil {
		s.logger.Error("writePipeConfig: failed to write config file", "path", configFile, "error", err)
		return fmt.Errorf("write config file: %w", err)
	}
	s.logger.Debug("writePipeConfig: wrote config file", "path", configFile, "bundlePath", bundlePath)

	return nil
}

// createNamedPipes creates the named pipes (FIFOs or regular files) for I/O forwarding.
func (s *ServicesLauncher) createNamedPipes(pipeDir string, enableStdin, enableStdout, enableStderr, useFifo bool) (*pipeSet, error) {
	pipes := &pipeSet{}

	// Only create pipes that are enabled
	pipesToCreate := make(map[string]*string)
	if enableStdin {
		pipes.stdin = filepath.Join(pipeDir, "stdin")
		pipesToCreate["stdin"] = &pipes.stdin
	}
	if enableStdout {
		pipes.stdout = filepath.Join(pipeDir, "stdout")
		pipesToCreate["stdout"] = &pipes.stdout
	}
	if enableStderr {
		pipes.stderr = filepath.Join(pipeDir, "stderr")
		pipesToCreate["stderr"] = &pipes.stderr
	}

	for name, path := range pipesToCreate {
		if useFifo {
			if err := syscall.Mkfifo(*path, 0600); err != nil {
				return nil, fmt.Errorf("create FIFO %s: %w", *path, err)
			}
		} else {
			f, err := os.OpenFile(*path, os.O_CREATE|os.O_RDWR, 0600)
			if err != nil {
				return nil, fmt.Errorf("create file %s: %w", *path, err)
			}
			f.Close()
		}
		_ = name // unused but useful for debugging
	}

	// Only create done file path for non-FIFO mode
	// With FIFOs, EOF on stdout/stderr reliably signals child exit
	if !useFifo {
		pipes.done = filepath.Join(pipeDir, "done")
	}

	// Always set up PID file path - child will write its PID here
	pipes.pid = filepath.Join(pipeDir, "child.pid")

	s.logger.Debug("pipes ready", "dir", pipeDir, "fifo", useFifo)

	return pipes, nil
}

// buildOpenCommand constructs the open command with appropriate arguments.
func (s *ServicesLauncher) buildOpenCommand(ctx context.Context, bundlePath string, pipes *pipeSet, noWait bool, cfg *Config) (*exec.Cmd, error) {
	args := []string{}

	// Add -g flag to not bring app to foreground (for background/CLI apps)
	// This prevents focus stealing from the terminal
	// Disable with MACGO_OPEN_BACKGROUND=0 if it causes issues
	if os.Getenv("MACGO_OPEN_BACKGROUND") != "0" {
		if (cfg != nil && cfg.Background) || os.Getenv("MACGO_OPEN_BACKGROUND") == "1" {
			args = append(args, "-g")
		}
	}

	// Add -n flag for new instance behavior (always starts fresh process)
	// Default is -n (new instance) to prevent reusing stale processes
	// Disable with MACGO_OPEN_NEW_INSTANCE=0 to reuse existing instances
	if os.Getenv("MACGO_OPEN_NEW_INSTANCE") != "0" {
		args = append(args, "-n")
	}

	// TTY passthrough (experimental): pass the parent's terminal device directly to the child
	// This preserves isatty()=true and full terminal capabilities
	// Opt-in via MACGO_TTY_PASSTHROUGH=1 (default is pipe-based I/O)
	useTTYPassthrough := os.Getenv("MACGO_TTY_PASSTHROUGH") == "1" && getTTYPath() != ""
	if useTTYPassthrough {
		ttyPath := getTTYPath()
		s.logger.Debug("TTY passthrough: passing terminal device to child", "tty", ttyPath)
		args = append(args, "--stdin", ttyPath, "--stdout", ttyPath, "--stderr", ttyPath)
		// With TTY passthrough, we can use -W since we're not using pipe-based I/O
		if !noWait && !isPipeOutput() {
			args = append(args, "-W")
		}
	}

	// Add wait flag when not using TTY passthrough
	if !noWait && !useTTYPassthrough {
		// Only add -W flag if output is not being piped AND no I/O forwarding pipes
		// When output is piped (e.g., to head/tail), we detect broken pipes
		// and exit gracefully instead of waiting indefinitely
		// IMPORTANT: The -W flag conflicts with -i/-o/--stderr flags and prevents
		// the app from launching, so we must NOT use -W when pipes are present
		if !isPipeOutput() && pipes == nil {
			args = append(args, "-W")
		}
	}

	// When using pipes, the config-file strategy is used:
	// Pipe paths are written to a config file that the child discovers and reads.
	// NOTE: Other strategies (open-flags, env-vars) do NOT work with LaunchServices
	// because xpcproxy causes deadlocks with FIFO flags or doesn't pass env vars.
	// The config-file approach is the only working strategy.

	// Add the bundle path
	if noWait {
		// In no-wait mode, use -a flag
		args = append(args, "-a", bundlePath)
	} else {
		args = append(args, bundlePath)
	}

	// Add command line arguments if present
	if len(os.Args) > 1 {
		args = append(args, "--args")
		args = append(args, os.Args[1:]...)
	}

	return exec.CommandContext(ctx, "open", args...), nil
}

// startIOForwarding sets up goroutines to forward I/O between the parent process and the named pipes.
func (s *ServicesLauncher) startIOForwarding(stdinCtx context.Context, pipes *pipeSet, errChan chan error) {
	// Forward stdin with cancellation context (don't report to errChan)
	if pipes.stdin != "" {
		go func() {
			err := s.forwardStdin(stdinCtx, pipes.stdin)
			if err != nil && err != context.Canceled {
				s.logger.Error("stdin forwarding failed", "error", err)
			}
		}()
	}

	// Forward stdout
	if pipes.stdout != "" {
		go func() {
			err := s.forwardStdout(pipes.stdout)
			if err != nil {
				s.logger.Error("stdout forwarding failed", "error", err)
			}
			errChan <- err
		}()
	}

	// Forward stderr
	if pipes.stderr != "" {
		go func() {
			err := s.forwardStderr(pipes.stderr)
			if err != nil {
				s.logger.Error("stderr forwarding failed", "error", err)
			}
			errChan <- err
		}()
	}
}

// forwardStdin forwards data from parent's stdin to the pipe.
func (s *ServicesLauncher) forwardStdin(ctx context.Context, stdinPipe string) error {
	if stdinPipe == "" {
		return fmt.Errorf("stdin pipe path is empty")
	}

	// Open with O_NONBLOCK to avoid blocking if reader not ready yet.
	// Retry on ENXIO (no reader) until child process opens the pipe.
	var w *os.File
	var err error
	maxRetries := 300 // 300 * 100ms = 30 seconds
	for i := 0; i < maxRetries; i++ {
		w, err = os.OpenFile(stdinPipe, os.O_WRONLY|syscall.O_NONBLOCK, 0)
		if err == nil {
			break
		}
		if pathErr, ok := err.(*os.PathError); ok {
			if errno, ok := pathErr.Err.(syscall.Errno); ok && errno == syscall.ENXIO {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(100 * time.Millisecond):
					continue
				}
			}
		}
		return fmt.Errorf("open stdin pipe: %w", err)
	}
	if w == nil {
		return fmt.Errorf("open stdin pipe: timed out waiting for reader")
	}
	defer w.Close()

	// Switch to blocking mode for the copy
	if err := syscall.SetNonblock(int(w.Fd()), false); err != nil {
		return fmt.Errorf("set blocking mode: %w", err)
	}

	// Add I/O logging if configured
	var input io.Reader = os.Stdin
	if stdinLog := createIOLogFile("stdin"); stdinLog != nil {
		defer stdinLog.Close()
		input = &teeReader{primary: os.Stdin, log: stdinLog}
	}

	// Copy with context cancellation support
	done := make(chan error, 1)
	go func() {
		_, err := io.Copy(w, input)
		done <- err
	}()

	select {
	case <-ctx.Done():
		w.Close() // unblock io.Copy
		return ctx.Err()
	case err := <-done:
		if err != nil {
			// Broken pipe is expected when child closes stdin early
			if pathErr, ok := err.(*os.PathError); ok {
				if errno, ok := pathErr.Err.(syscall.Errno); ok && errno == syscall.EPIPE {
					return nil
				}
			}
			return fmt.Errorf("copy stdin: %w", err)
		}
		return nil
	}
}

// signalOnFirstWrite wraps a writer to signal a channel on the first write
type signalOnFirstWrite struct {
	w        io.Writer
	signalCh chan struct{}
	signaled bool
}

func (s *signalOnFirstWrite) Write(p []byte) (int, error) {
	if !s.signaled && len(p) > 0 && s.signalCh != nil {
		s.signaled = true
		select {
		case <-s.signalCh:
			// Already closed
		default:
			close(s.signalCh)
		}
	}
	return s.w.Write(p)
}

// forwardStdout forwards data from the named pipe to the parent's stdout.
// With FIFOs (default), io.Copy blocks until the writer closes.
func (s *ServicesLauncher) forwardStdout(stdoutPipe string) error {
	if stdoutPipe == "" {
		return fmt.Errorf("stdout pipe path is empty")
	}

	r, err := os.OpenFile(stdoutPipe, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("open stdout pipe: %w", err)
	}
	defer r.Close()

	// Wrap stdout if configured, and wrap with first-write signaler
	output := io.Writer(NewIOWrapper(os.Stdout, "stdout"))
	if s.firstOutputCh != nil {
		output = &signalOnFirstWrite{w: output, signalCh: s.firstOutputCh}
	}

	// Add I/O logging if configured
	if stdoutLog := createIOLogFile("stdout"); stdoutLog != nil {
		defer stdoutLog.Close()
		output = &teeWriter{primary: output, log: stdoutLog}
	}

	// FIFO: io.Copy blocks until writer closes
	_, err = io.Copy(output, r)
	if err != nil {
		// Broken pipe is expected when piping to head, tail, etc.
		if pathErr, ok := err.(*os.PathError); ok {
			if errno, ok := pathErr.Err.(syscall.Errno); ok && errno == syscall.EPIPE {
				return nil
			}
		}
		return fmt.Errorf("copy stdout: %w", err)
	}
	return nil
}

// forwardStderr forwards data from the named pipe to the parent's stderr.
// With regular files (not FIFOs), we need to poll continuously until the writer closes.
func (s *ServicesLauncher) forwardStderr(stderrPipe string) error {
	if stderrPipe == "" {
		return fmt.Errorf("stderr pipe path is empty")
	}

	// Check if using FIFOs (default with config-file) or regular files (polling)
	ioStrategy := os.Getenv("MACGO_IO_STRATEGY")
	useFifo := os.Getenv("MACGO_USE_FIFO") != "0" && (ioStrategy == "" || ioStrategy == "config-file")

	r, err := os.OpenFile(stderrPipe, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("open stderr pipe: %w", err)
	}
	defer func() { _ = r.Close() }()

	// Wrap stderr if configured
	output := io.Writer(NewIOWrapper(os.Stderr, "stderr"))

	// Add I/O logging if configured
	if stderrLog := createIOLogFile("stderr"); stderrLog != nil {
		defer stderrLog.Close()
		output = &teeWriter{primary: output, log: stderrLog}
	}

	if useFifo {
		// FIFO behavior (default): simple io.Copy blocks until writer closes
		_, err := io.Copy(output, r)
		if err != nil {
			// Broken pipe is expected when piping to head, tail, etc.
			if pathErr, ok := err.(*os.PathError); ok {
				if errno, ok := pathErr.Err.(syscall.Errno); ok && errno == syscall.EPIPE {
					return nil
				}
			}
			return fmt.Errorf("copy stderr: %w", err)
		}
		return nil
	}

	// Regular file behavior: continuous polling (MACGO_USE_FIFO=0)
	totalBytes := int64(0)
	buf := make([]byte, 32*1024)
	lastSize := int64(0)
	noGrowthCount := 0
	maxNoGrowth := 50 // 50 * 100ms = 5 seconds of no growth

	for {
		n, err := r.Read(buf)
		if n > 0 {
			totalBytes += int64(n)
			output.Write(buf[:n])
			noGrowthCount = 0
		}

		if err == io.EOF {
			// Check if file is still growing
			info, statErr := os.Stat(stderrPipe)
			if statErr == nil {
				currentSize := info.Size()
				if currentSize > lastSize {
					lastSize = currentSize
					noGrowthCount = 0
					r.Seek(totalBytes, 0)
					continue
				}
			}

			noGrowthCount++
			if noGrowthCount >= maxNoGrowth {
				// Check if done file exists before giving up
				if s.doneFile != "" {
					if _, err := os.Stat(s.doneFile); err == nil {
						// Done file exists, do final read pass
						r.Seek(totalBytes, 0)
						for {
							n, readErr := r.Read(buf)
							if n > 0 {
								totalBytes += int64(n)
								output.Write(buf[:n])
							}
							if readErr == io.EOF || n == 0 || readErr != nil {
								break
							}
						}
						return nil
					}
					// Done file doesn't exist yet, wait indefinitely
					noGrowthCount = 0
					time.Sleep(100 * time.Millisecond)
					r.Seek(totalBytes, 0)
					continue
				}
				return nil
			}

			time.Sleep(100 * time.Millisecond)
			r.Seek(totalBytes, 0)
			continue
		}

		if err != nil {
			return fmt.Errorf("copy stderr: %w", err)
		}
	}
}

// isPipeOutput checks if stdout is piped to another command
func isPipeOutput() bool {
	stat, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

// isBrokenPipeError checks if an error is a broken pipe error
func isBrokenPipeError(err error) bool {
	if pathErr, ok := err.(*os.PathError); ok {
		if errno, ok := pathErr.Err.(syscall.Errno); ok {
			return errno == syscall.EPIPE
		}
	}
	return false
}

// getTTYPath returns the path to the current TTY device (e.g., /dev/ttys001).
// Returns empty string if stdin is not a terminal.
func getTTYPath() string {
	if !isTerminal(int(os.Stdin.Fd())) {
		return ""
	}
	// Run the tty command to get the terminal device path.
	// Must inherit stdin so tty can read the terminal device.
	cmd := exec.Command("tty")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// shouldAutoEnableStdin determines if stdin forwarding should be automatically enabled.
// Returns true for:
//   - TTY (interactive terminal) when we're in the foreground process group
//   - Pipe (piped input like echo 'foo' | app)
//   - Regular file (redirected from file like app < input.txt)
//
// Returns false for:
//   - /dev/null or similar device files (daemon/background mode)
//   - TTY when not in foreground process group (would trigger SIGTTIN)
//   - Unknown/error cases (safe default)
func shouldAutoEnableStdin() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false // Can't stat stdin, don't enable
	}

	mode := stat.Mode()

	// TTY - interactive terminal
	// But only if we're in the foreground process group, otherwise reading
	// from TTY will trigger SIGTTIN and stop the process (e.g., when running in a script)
	if isTerminal(int(os.Stdin.Fd())) {
		// Check if we're in the foreground process group
		if !isInForegroundProcessGroup() {
			return false
		}
		return true
	}

	// Pipe - piped input (echo 'foo' | app)
	if (mode & os.ModeNamedPipe) != 0 {
		return true
	}

	// Regular file - redirected from file (app < input.txt)
	if mode.IsRegular() {
		return true
	}

	// Character device that's NOT a TTY is likely /dev/null
	// Don't enable stdin for background/daemon processes
	if (mode & os.ModeCharDevice) != 0 {
		return false
	}

	// Socket, symlink, or other special file - don't enable
	return false
}

// isTerminal returns whether fd is a terminal.
func isTerminal(fd int) bool {
	_, err := unix.IoctlGetTermios(fd, unix.TIOCGETA)
	return err == nil
}

// isInForegroundProcessGroup checks if we're in the terminal's foreground process group.
// If not, reading from the TTY would trigger SIGTTIN and stop the process.
func isInForegroundProcessGroup() bool {
	fd := int(os.Stdin.Fd())
	fpgrp, err := unix.IoctlGetInt(fd, unix.TIOCGPGRP)
	if err != nil {
		return false
	}
	return unix.Getpgrp() == fpgrp
}

// createIOLogFile creates a log file for I/O debugging in the specified directory.
// Returns nil if MACGO_IO_LOG_DIR is not set or file creation fails.
func createIOLogFile(name string) *os.File {
	logDir := os.Getenv("MACGO_IO_LOG_DIR")
	if logDir == "" {
		return nil
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "macgo: failed to create I/O log dir %s: %v\n", logDir, err)
		return nil
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("20060102-150405")
	logPath := filepath.Join(logDir, fmt.Sprintf("%s-%s-%d.log", name, timestamp, os.Getpid()))

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "macgo: failed to create I/O log file %s: %v\n", logPath, err)
		return nil
	}

	// Write header
	fmt.Fprintf(f, "# macgo %s log - started %s (PID %d)\n", name, time.Now().Format(time.RFC3339), os.Getpid())

	return f
}

// teeWriter wraps a writer and copies all writes to a log file
type teeWriter struct {
	primary io.Writer
	log     *os.File
}

func (t *teeWriter) Write(p []byte) (n int, err error) {
	n, err = t.primary.Write(p)
	if t.log != nil && n > 0 {
		// Write to log with timestamp prefix for each chunk
		fmt.Fprintf(t.log, "[%s] ", time.Now().Format("15:04:05.000"))
		t.log.Write(p[:n])
		if len(p) > 0 && p[len(p)-1] != '\n' {
			t.log.Write([]byte("\n"))
		}
	}
	return n, err
}

// teeReader wraps a reader and copies all reads to a log file
type teeReader struct {
	primary io.Reader
	log     *os.File
}

func (t *teeReader) Read(p []byte) (n int, err error) {
	n, err = t.primary.Read(p)
	if t.log != nil && n > 0 {
		// Write to log with timestamp prefix for each chunk
		fmt.Fprintf(t.log, "[%s] ", time.Now().Format("15:04:05.000"))
		t.log.Write(p[:n])
		if n > 0 && p[n-1] != '\n' {
			t.log.Write([]byte("\n"))
		}
	}
	return n, err
}
