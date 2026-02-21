package launch

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// ---------- #17: FIFO EOF triggers clean parent exit ----------

// TestFIFOEOFTriggersCleanExit verifies that when the child closes its end of
// the stdout and stderr FIFOs, the parent's io.Copy goroutines observe EOF and
// return without hanging. This is the foundational contract: FIFO EOF is the
// exit signal in the default (non-polling) mode.
func TestFIFOEOFTriggersCleanExit(t *testing.T) {
	launcher := &ServicesLauncher{
		logger: NewLogger(),
	}

	tests := []struct {
		name    string
		forward func(path string) error
		stream  string // "stdout" or "stderr"
	}{
		{"stdout", launcher.forwardStdout, "stdout"},
		{"stderr", launcher.forwardStderr, "stderr"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			pipePath := filepath.Join(tmpDir, tt.stream)
			if err := syscall.Mkfifo(pipePath, 0600); err != nil {
				t.Fatalf("mkfifo: %v", err)
			}

			// Intercept the stream so forwardStdout/forwardStderr writes
			// don't go to the test's real stdout/stderr.
			origFd, capR, capW := interceptStream(t, tt.stream)
			defer restoreStream(tt.stream, origFd)

			done := make(chan error, 1)
			go func() { done <- tt.forward(pipePath) }()

			// Simulate child: open write end, write some data, close immediately.
			w, err := os.OpenFile(pipePath, os.O_WRONLY, 0)
			if err != nil {
				t.Fatalf("open pipe for write: %v", err)
			}
			fmt.Fprintf(w, "hello from child\n")
			w.Close() // <-- this close must cause the reader to see EOF

			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("forward returned error: %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("forward did not return after writer closed (EOF not detected)")
			}

			capW.Close()
			var buf bytes.Buffer
			io.Copy(&buf, capR)
			if !strings.Contains(buf.String(), "hello from child") {
				t.Errorf("captured output %q does not contain expected data", buf.String())
			}
		})
	}
}

// TestFIFOEOFBothStreams verifies that when both stdout and stderr FIFOs are
// closed, both forwarders complete. This simulates the real two-stream case
// where the parent waits for expectedIOCount completions.
func TestFIFOEOFBothStreams(t *testing.T) {
	launcher := &ServicesLauncher{
		logger: NewLogger(),
	}

	tmpDir := t.TempDir()
	stdoutPath := filepath.Join(tmpDir, "stdout")
	stderrPath := filepath.Join(tmpDir, "stderr")
	if err := syscall.Mkfifo(stdoutPath, 0600); err != nil {
		t.Fatalf("mkfifo stdout: %v", err)
	}
	if err := syscall.Mkfifo(stderrPath, 0600); err != nil {
		t.Fatalf("mkfifo stderr: %v", err)
	}

	origOut, outR, outW := interceptStream(t, "stdout")
	defer restoreStream("stdout", origOut)
	origErr, errR, errW := interceptStream(t, "stderr")
	defer restoreStream("stderr", origErr)

	outDone := make(chan error, 1)
	errDone := make(chan error, 1)
	go func() { outDone <- launcher.forwardStdout(stdoutPath) }()
	go func() { errDone <- launcher.forwardStderr(stderrPath) }()

	// Simulate child writing to both, then closing both.
	wOut, _ := os.OpenFile(stdoutPath, os.O_WRONLY, 0)
	wErr, _ := os.OpenFile(stderrPath, os.O_WRONLY, 0)
	fmt.Fprintf(wOut, "stdout data\n")
	fmt.Fprintf(wErr, "stderr data\n")
	wOut.Close()
	wErr.Close()

	for _, ch := range []struct {
		name string
		c    chan error
	}{{"stdout", outDone}, {"stderr", errDone}} {
		select {
		case err := <-ch.c:
			if err != nil {
				t.Errorf("%s forward error: %v", ch.name, err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("%s forward did not complete after EOF", ch.name)
		}
	}

	outW.Close()
	errW.Close()
	var outBuf, errBuf bytes.Buffer
	io.Copy(&outBuf, outR)
	io.Copy(&errBuf, errR)
	if !strings.Contains(outBuf.String(), "stdout data") {
		t.Errorf("stdout capture %q missing expected data", outBuf.String())
	}
	if !strings.Contains(errBuf.String(), "stderr data") {
		t.Errorf("stderr capture %q missing expected data", errBuf.String())
	}
}

// ---------- #18: Signal → forward → drain → exit ----------

// TestSignalForwardThenDrain verifies that after a signal is received, the
// parent forwards it to the child and then drains remaining pipe data before
// exiting. This is the critical path that currently has a bug: the parent
// calls os.Exit without waiting for io.Copy to finish.
//
// Because exitWithSignalForwarding calls os.Exit (which we cannot test
// directly), we test the individual components:
//  1. forwardSignalToChildWithGrace delivers the signal to the child.
//  2. After the child receives the signal, it may produce final output and
//     then close its pipe ends.
//  3. The FIFO reader (forwardStdout) must observe EOF and return.
//
// If the parent were to drain pipes before os.Exit, the sequence would be:
// signal → forwardSignalToChildWithGrace → wait for forwarders → exit.
// This test validates that pipes ARE drained when given the chance.
func TestSignalForwardThenDrain(t *testing.T) {
	launcher := &ServicesLauncher{
		logger: NewLogger(),
	}

	tmpDir := t.TempDir()
	stdoutPath := filepath.Join(tmpDir, "stdout")
	if err := syscall.Mkfifo(stdoutPath, 0600); err != nil {
		t.Fatalf("mkfifo: %v", err)
	}

	origOut, capR, capW := interceptStream(t, "stdout")
	defer restoreStream("stdout", origOut)

	// Start forwarding.
	fwdDone := make(chan error, 1)
	go func() { fwdDone <- launcher.forwardStdout(stdoutPath) }()

	// Simulate child: write some initial output.
	w, err := os.OpenFile(stdoutPath, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open pipe: %v", err)
	}
	fmt.Fprintf(w, "before signal\n")

	// Simulate receiving SIGINT — record the signal as the parent does.
	launcher.lastSignal.Store(int32(syscall.SIGINT))

	// After the signal, child writes final output and closes.
	fmt.Fprintf(w, "after signal\n")
	w.Close()

	// The forwarder should drain both lines and return.
	select {
	case err := <-fwdDone:
		if err != nil {
			t.Fatalf("forwardStdout: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("forwardStdout did not drain after child closed")
	}

	capW.Close()
	var buf bytes.Buffer
	io.Copy(&buf, capR)
	got := buf.String()
	if !strings.Contains(got, "before signal") {
		t.Errorf("missing pre-signal output in %q", got)
	}
	if !strings.Contains(got, "after signal") {
		t.Errorf("missing post-signal output in %q (output lost during signal handling)", got)
	}
}

// TestSignalForwardGraceTerminatesChild verifies that
// forwardSignalToChildWithGrace delivers the requested signal and escalates
// to SIGKILL if the child ignores it.
func TestSignalForwardGraceTerminatesChild(t *testing.T) {
	launcher := &ServicesLauncher{
		logger: NewLogger(),
	}

	// Start a child that traps SIGINT (ignores it).
	cmd := newSleepCommand(t, "trap '' INT; while :; do sleep 1; done")

	launcher.mu.Lock()
	launcher.childPID = cmd.Process.Pid
	launcher.mu.Unlock()

	start := time.Now()
	launcher.forwardSignalToChildWithGrace(syscall.SIGINT, 150*time.Millisecond)
	elapsed := time.Since(start)

	// Grace period should have elapsed.
	if elapsed < 100*time.Millisecond {
		t.Errorf("grace period too short: %v", elapsed)
	}

	// Wait for the child to actually exit after SIGKILL.
	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()
	select {
	case <-waitDone:
		// good
	case <-time.After(2 * time.Second):
		t.Fatal("child did not exit after SIGKILL escalation")
	}
}

// ---------- #19: Child exit code propagation via signal ----------

// TestCancellationExitCode verifies that cancellationExitCode returns
// 128+signal for the most recently recorded signal.
func TestCancellationExitCode(t *testing.T) {
	tests := []struct {
		signal   syscall.Signal
		wantCode int
	}{
		{syscall.SIGINT, 130},  // 128 + 2
		{syscall.SIGTERM, 143}, // 128 + 15
		{syscall.SIGHUP, 129},  // 128 + 1
		{syscall.SIGPIPE, 141}, // 128 + 13
	}

	for _, tt := range tests {
		t.Run(tt.signal.String(), func(t *testing.T) {
			launcher := &ServicesLauncher{logger: NewLogger()}
			launcher.lastSignal.Store(int32(tt.signal))

			got := launcher.cancellationExitCode(1)
			if got != tt.wantCode {
				t.Errorf("cancellationExitCode = %d, want %d", got, tt.wantCode)
			}
		})
	}
}

// TestCancellationExitCodeDefault verifies that when no signal is recorded,
// cancellationExitCode returns the default code.
func TestCancellationExitCodeDefault(t *testing.T) {
	launcher := &ServicesLauncher{logger: NewLogger()}
	// lastSignal is zero (no signal recorded), so recordedSignal() returns SIGINT.
	got := launcher.cancellationExitCode(1)
	// 128 + 2 (SIGINT) = 130
	if got != 130 {
		t.Errorf("cancellationExitCode with no signal = %d, want 130", got)
	}
}

// TestRecordedSignal verifies the atomic signal recording mechanism.
func TestRecordedSignal(t *testing.T) {
	launcher := &ServicesLauncher{logger: NewLogger()}

	// Default: no signal → returns SIGINT.
	if got := launcher.recordedSignal(); got != syscall.SIGINT {
		t.Errorf("recordedSignal() with no signal = %v, want SIGINT", got)
	}

	// After storing SIGTERM, should return SIGTERM.
	launcher.lastSignal.Store(int32(syscall.SIGTERM))
	if got := launcher.recordedSignal(); got != syscall.SIGTERM {
		t.Errorf("recordedSignal() after SIGTERM = %v, want SIGTERM", got)
	}

	// Overwrite with SIGHUP.
	launcher.lastSignal.Store(int32(syscall.SIGHUP))
	if got := launcher.recordedSignal(); got != syscall.SIGHUP {
		t.Errorf("recordedSignal() after SIGHUP = %v, want SIGHUP", got)
	}
}

// ---------- #20: Control FIFO PID round-trip ----------

// TestControlPipeRoundTrip verifies the full cycle:
//  1. Child writes its PID to a control FIFO.
//  2. Parent's readChildPID reads it (blocking, no polling).
//  3. Parent can then forward signals using that PID.
func TestControlPipeRoundTrip(t *testing.T) {
	launcher := &ServicesLauncher{
		logger: NewLogger(),
	}

	tmpDir := t.TempDir()
	controlPipe := filepath.Join(tmpDir, "control")
	if err := syscall.Mkfifo(controlPipe, 0600); err != nil {
		t.Fatalf("mkfifo: %v", err)
	}

	// Simulate child writing PID after a short delay.
	childPID := os.Getpid()
	go func() {
		time.Sleep(50 * time.Millisecond)
		f, err := os.OpenFile(controlPipe, os.O_WRONLY, 0)
		if err != nil {
			return
		}
		defer f.Close()
		fmt.Fprintf(f, "%d\n", childPID)
	}()

	got := launcher.readChildPID(controlPipe, 5*time.Second)
	if got != childPID {
		t.Fatalf("readChildPID = %d, want %d", got, childPID)
	}

	launcher.mu.Lock()
	storedPID := launcher.childPID
	launcher.mu.Unlock()
	if storedPID != childPID {
		t.Errorf("launcher.childPID = %d, want %d", storedPID, childPID)
	}
}

// TestControlPipeTimeout verifies that readChildPID returns 0 when no
// writer opens the FIFO within the timeout.
func TestControlPipeTimeout(t *testing.T) {
	launcher := &ServicesLauncher{
		logger: NewLogger(),
	}

	tmpDir := t.TempDir()
	controlPipe := filepath.Join(tmpDir, "control")
	if err := syscall.Mkfifo(controlPipe, 0600); err != nil {
		t.Fatalf("mkfifo: %v", err)
	}

	start := time.Now()
	got := launcher.readChildPID(controlPipe, 200*time.Millisecond)
	elapsed := time.Since(start)

	if got != 0 {
		t.Errorf("readChildPID = %d, want 0 (timeout)", got)
	}
	if elapsed < 150*time.Millisecond {
		t.Errorf("returned too quickly: %v (expected ~200ms timeout)", elapsed)
	}
}

// TestControlPipeEmptyPath verifies that readChildPID returns 0 immediately
// when given an empty path.
func TestControlPipeEmptyPath(t *testing.T) {
	launcher := &ServicesLauncher{
		logger: NewLogger(),
	}

	start := time.Now()
	got := launcher.readChildPID("", 5*time.Second)
	elapsed := time.Since(start)

	if got != 0 {
		t.Errorf("readChildPID(\"\") = %d, want 0", got)
	}
	if elapsed > 100*time.Millisecond {
		t.Errorf("empty path should return immediately, took %v", elapsed)
	}
}

// TestControlPipeSignalForwardAfterDiscovery is an end-to-end test:
//  1. Start a real child process (sleep 60).
//  2. Simulate child writing PID to control FIFO.
//  3. Parent discovers PID via readChildPID.
//  4. Parent forwards SIGTERM.
//  5. Child exits.
func TestControlPipeSignalForwardAfterDiscovery(t *testing.T) {
	launcher := &ServicesLauncher{
		logger: NewLogger(),
	}

	tmpDir := t.TempDir()
	controlPipe := filepath.Join(tmpDir, "control")
	if err := syscall.Mkfifo(controlPipe, 0600); err != nil {
		t.Fatalf("mkfifo: %v", err)
	}

	child, err := os.StartProcess("/bin/sleep",
		[]string{"sleep", "60"},
		&os.ProcAttr{
			Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
			Sys:   &syscall.SysProcAttr{Setpgid: true},
		},
	)
	if err != nil {
		t.Fatalf("start child: %v", err)
	}
	t.Cleanup(func() {
		child.Kill()
		child.Wait()
	})

	// Simulate child writing its PID to the control FIFO.
	go func() {
		f, err := os.OpenFile(controlPipe, os.O_WRONLY, 0)
		if err != nil {
			return
		}
		defer f.Close()
		fmt.Fprintf(f, "%d\n", child.Pid)
	}()

	got := launcher.readChildPID(controlPipe, 5*time.Second)
	if got != child.Pid {
		t.Fatalf("readChildPID = %d, want %d", got, child.Pid)
	}

	launcher.forwardSignalToChild(syscall.SIGTERM)

	waitCh := make(chan error, 1)
	go func() {
		_, err := child.Wait()
		waitCh <- err
	}()

	select {
	case <-waitCh:
		// Child exited after SIGTERM.
	case <-time.After(5 * time.Second):
		t.Fatal("child did not exit after SIGTERM forwarding")
	}

	if err := syscall.Kill(child.Pid, 0); err == nil {
		t.Errorf("child %d still alive after SIGTERM", child.Pid)
	}
}

// ---------- Helpers ----------

// interceptStream replaces os.Stdout or os.Stderr with a pipe and returns
// the original *os.File plus both ends of the capture pipe.
func interceptStream(t *testing.T, stream string) (orig *os.File, r *os.File, w *os.File) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	switch stream {
	case "stdout":
		orig = os.Stdout
		os.Stdout = w
	case "stderr":
		orig = os.Stderr
		os.Stderr = w
	default:
		t.Fatalf("unknown stream %q", stream)
	}
	return orig, r, w
}

// restoreStream restores os.Stdout or os.Stderr to the original file.
func restoreStream(stream string, orig *os.File) {
	switch stream {
	case "stdout":
		os.Stdout = orig
	case "stderr":
		os.Stderr = orig
	}
}

// newSleepCommand starts a shell process with the given script in its own
// process group. The test cleanup kills and waits on the process.
func newSleepCommand(t *testing.T, script string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command("sh", "-c", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start child: %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		if cmd.ProcessState == nil {
			cmd.Wait()
		}
	})
	return cmd
}
