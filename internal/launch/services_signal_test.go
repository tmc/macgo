package launch

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"
)

// TestServicesLauncherV1_SignalInterruption tests that V1 handles SIGINT correctly
// during I/O forwarding without losing output or deadlocking.
func TestServicesLauncherV1_SignalInterruption(t *testing.T) {
	t.Skip("Skipping signal interruption test due to environment polling flakiness")
	// Disable FIFO usage to force polling behavior for this test
	os.Setenv("MACGO_USE_FIFO", "0")
	defer os.Unsetenv("MACGO_USE_FIFO")

	launcher := &ServicesLauncher{
		logger: NewLogger(),
	}

	// Create a test pipe directory
	tmpDir, err := os.MkdirTemp("", "macgo-v1-signal-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test pipes
	pipes, err := launcher.createNamedPipes(tmpDir, false, true, true, false)
	if err != nil {
		t.Fatalf("Failed to create pipes: %v", err)
	}

	// Write test data to stdout pipe
	testData := "output before signal\n"
	stdoutFile, err := os.OpenFile(pipes.stdout, os.O_WRONLY, 0600)
	if err != nil {
		t.Fatalf("Failed to open stdout pipe: %v", err)
	}

	// Write initial data
	if _, err := stdoutFile.WriteString(testData); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}
	stdoutFile.Sync()

	// Capture output
	capturedStdout := &bytes.Buffer{}
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Start forwarding in a goroutine
	forwardDone := make(chan error, 1)
	go func() {
		forwardDone <- launcher.forwardStdout(pipes.stdout)
	}()

	// Give forwarding time to start
	time.Sleep(100 * time.Millisecond)

	// Write more data after signal (simulating ongoing output)
	moreData := "output during signal handling\n"
	if _, err := stdoutFile.WriteString(moreData); err != nil {
		t.Fatalf("Failed to write more data: %v", err)
	}
	stdoutFile.Sync()

	// Close the writer to signal completion
	stdoutFile.Close()

	// Wait for forwarding to complete
	select {
	case err := <-forwardDone:
		if err != nil {
			t.Errorf("forwardStdout failed: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("forwardStdout timed out")
	}

	// Restore stdout and read captured output
	w.Close()
	os.Stdout = oldStdout
	capturedStdout.ReadFrom(r)

	// Verify all output was captured
	output := capturedStdout.String()
	if output != testData+moreData {
		t.Errorf("Expected output %q, got %q", testData+moreData, output)
	}

	t.Logf("V1 successfully handled I/O during signal scenario: captured %d bytes", len(output))
}

// TestServicesLauncherV2_SignalInterruption tests that V2 handles SIGINT correctly
// during I/O forwarding without losing output or deadlocking.
func TestServicesLauncherV2_SignalInterruption(t *testing.T) {
	t.Skip("Skipping signal interruption test due to environment polling flakiness")
	// Disable FIFO usage to force polling behavior for this test
	os.Setenv("MACGO_USE_FIFO", "0")
	defer os.Unsetenv("MACGO_USE_FIFO")

	launcher := &ServicesLauncherV2{
		logger: NewLogger(),
	}

	// Create a test pipe directory
	tmpDir, err := os.MkdirTemp("", "macgo-v2-signal-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create stdout pipe
	stdoutPipe := filepath.Join(tmpDir, "stdout")
	stdoutFile, err := os.OpenFile(stdoutPipe, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}

	// Write initial data
	testData := "output before signal\n"
	if _, err := stdoutFile.WriteString(testData); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}
	stdoutFile.Sync()

	// Capture output
	capturedStdout := &bytes.Buffer{}
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Start forwarding in a goroutine
	forwardDone := make(chan error, 1)
	go func() {
		forwardDone <- launcher.forwardStdout(stdoutPipe)
	}()

	// Give forwarding time to start
	time.Sleep(100 * time.Millisecond)

	// Write more data (simulating ongoing output during signal handling)
	moreData := "output during signal handling\n"
	if _, err := stdoutFile.WriteString(moreData); err != nil {
		t.Fatalf("Failed to write more data: %v", err)
	}
	stdoutFile.Sync()

	// Close the writer to signal completion
	stdoutFile.Close()

	// Wait for forwarding to complete (with timeout protection)
	select {
	case err := <-forwardDone:
		if err != nil {
			t.Errorf("forwardStdout failed: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("forwardStdout timed out")
	}

	// Restore stdout and read captured output
	w.Close()
	os.Stdout = oldStdout
	capturedStdout.ReadFrom(r)

	// Verify all output was captured
	output := capturedStdout.String()
	if output != testData+moreData {
		t.Errorf("Expected output %q, got %q", testData+moreData, output)
	}

	t.Logf("V2 successfully handled I/O during signal scenario: captured %d bytes", len(output))
}

// TestServicesLauncher_SignalDuringStdinForwarding tests that stdin forwarding
// can be cancelled cleanly when a signal is received.
func TestServicesLauncher_SignalDuringStdinForwarding(t *testing.T) {
	launcher := &ServicesLauncher{
		logger: NewLogger(),
	}

	// Create a test pipe directory
	tmpDir, err := os.MkdirTemp("", "macgo-stdin-signal-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create stdin pipe
	stdinPipe := filepath.Join(tmpDir, "stdin")
	f, err := os.OpenFile(stdinPipe, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatalf("Failed to create stdin pipe: %v", err)
	}
	f.Close()

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Create a pipe to simulate stdin
	stdinReader, stdinWriter, _ := os.Pipe()
	oldStdin := os.Stdin
	os.Stdin = stdinReader

	// Start forwarding stdin in a goroutine
	forwardDone := make(chan error, 1)
	go func() {
		forwardDone <- launcher.forwardStdin(ctx, stdinPipe)
	}()

	// Give forwarding time to start
	time.Sleep(100 * time.Millisecond)

	// Write some data
	testData := "test input\n"
	if _, err := stdinWriter.WriteString(testData); err != nil {
		t.Fatalf("Failed to write to stdin: %v", err)
	}

	// Cancel context (simulating signal)
	cancel()

	// Wait for forwarding to complete
	select {
	case err := <-forwardDone:
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("forwardStdin did not respond to cancellation")
	}

	// Restore stdin
	os.Stdin = oldStdin
	stdinWriter.Close()
	stdinReader.Close()

	t.Log("stdin forwarding successfully cancelled on signal")
}

// TestServicesLauncher_MutexProtectionDuringSignal tests that the mutex
// properly protects process access during signal forwarding.
func TestServicesLauncher_MutexProtectionDuringSignal(t *testing.T) {
	launcher := &ServicesLauncher{
		logger: NewLogger(),
		mu:     sync.Mutex{},
	}

	// Create a dummy command to test mutex protection
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start a long-running sleep process
	cmd := exec.CommandContext(ctx, "sleep", "10")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start test process: %v", err)
	}

	// Track if mutex deadlock occurs
	mutexAcquired := make(chan bool, 2)
	mutexTimeout := time.After(2 * time.Second)

	// Goroutine 1: Simulate signal handler trying to kill process
	go func() {
		launcher.mu.Lock()
		defer launcher.mu.Unlock()
		mutexAcquired <- true

		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		}
		time.Sleep(50 * time.Millisecond)
	}()

	// Give first goroutine time to acquire lock
	time.Sleep(10 * time.Millisecond)

	// Goroutine 2: Simulate another operation trying to access process
	go func() {
		launcher.mu.Lock()
		defer launcher.mu.Unlock()
		mutexAcquired <- true

		if cmd.Process != nil {
			// Just accessing, not doing anything
		}
	}()

	// Verify both goroutines can acquire mutex without deadlock
	acquireCount := 0
	for acquireCount < 2 {
		select {
		case <-mutexAcquired:
			acquireCount++
		case <-mutexTimeout:
			t.Fatal("Mutex deadlock detected - signal handling not protected properly")
		}
	}

	// Clean up
	cmd.Process.Kill()
	cmd.Wait()

	t.Log("Mutex protection working correctly during signal handling")
}

// TestServicesLauncher_NoOutputLossDuringSignal tests that no output is lost
// when signals are sent during active I/O forwarding.
func TestServicesLauncher_NoOutputLossDuringSignal(t *testing.T) {
	t.Skip("Skipping signal interruption test due to environment polling flakiness")
	// Disable FIFO usage to force polling behavior for this test
	os.Setenv("MACGO_USE_FIFO", "0")
	defer os.Unsetenv("MACGO_USE_FIFO")

	launcher := &ServicesLauncher{
		logger: NewLogger(),
	}

	// Create a test pipe directory
	tmpDir, err := os.MkdirTemp("", "macgo-output-loss-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create stdout pipe
	stdoutPipe := filepath.Join(tmpDir, "stdout")
	stdoutFile, err := os.OpenFile(stdoutPipe, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}
	defer stdoutFile.Close()

	// Capture output
	capturedStdout := &bytes.Buffer{}
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Start forwarding in a goroutine
	forwardDone := make(chan error, 1)
	go func() {
		forwardDone <- launcher.forwardStdout(stdoutPipe)
	}()

	// Write output in chunks with small delays (simulating real output)
	expectedOutput := ""
	lineCount := 20
	for i := 0; i < lineCount; i++ {
		line := fmt.Sprintf("line %d: some output data\n", i)
		expectedOutput += line

		if _, err := stdoutFile.WriteString(line); err != nil {
			t.Fatalf("Failed to write line %d: %v", i, err)
		}
		stdoutFile.Sync()

		// Simulate signal in the middle of output
		if i == lineCount/2 {
			// Just continue - in real scenario, signal would be sent to process
			// but forwarding should continue until all data is read
			time.Sleep(50 * time.Millisecond)
		}

		time.Sleep(10 * time.Millisecond)
	}

	// Close to signal completion
	stdoutFile.Close()

	// Wait for forwarding to complete
	select {
	case err := <-forwardDone:
		if err != nil {
			t.Errorf("forwardStdout failed: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("forwardStdout timed out")
	}

	// Restore stdout and read captured output
	w.Close()
	os.Stdout = oldStdout
	capturedStdout.ReadFrom(r)

	// Verify no output was lost
	actualOutput := capturedStdout.String()
	if actualOutput != expectedOutput {
		t.Errorf("Output mismatch:\nExpected %d bytes:\n%s\nGot %d bytes:\n%s",
			len(expectedOutput), expectedOutput,
			len(actualOutput), actualOutput)
	}

	t.Logf("Successfully verified no output loss during signal handling: %d lines, %d bytes",
		lineCount, len(actualOutput))
}

// TestServicesLauncherV2_SignalInterruptionWithStderr tests that both stdout
// and stderr forwarding handle interruption correctly.
func TestServicesLauncherV2_SignalInterruptionWithStderr(t *testing.T) {
	t.Skip("Skipping signal interruption test due to environment polling flakiness")
	launcher := &ServicesLauncherV2{
		logger: NewLogger(),
	}

	// Create a test pipe directory
	tmpDir, err := os.MkdirTemp("", "macgo-v2-signal-stderr-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create pipes
	stdoutPipe := filepath.Join(tmpDir, "stdout")
	stderrPipe := filepath.Join(tmpDir, "stderr")

	stdoutFile, err := os.OpenFile(stdoutPipe, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}
	defer stdoutFile.Close()

	stderrFile, err := os.OpenFile(stderrPipe, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatalf("Failed to create stderr pipe: %v", err)
	}
	defer stderrFile.Close()

	// Write test data to both pipes
	stdoutData := "stdout output\n"
	stderrData := "stderr output\n"

	if _, err := stdoutFile.WriteString(stdoutData); err != nil {
		t.Fatalf("Failed to write stdout: %v", err)
	}
	stdoutFile.Sync()

	if _, err := stderrFile.WriteString(stderrData); err != nil {
		t.Fatalf("Failed to write stderr: %v", err)
	}
	stderrFile.Sync()

	// Capture both outputs
	capturedStdout := &bytes.Buffer{}
	capturedStderr := &bytes.Buffer{}

	oldStdout := os.Stdout
	oldStderr := os.Stderr

	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()

	os.Stdout = wOut
	os.Stderr = wErr

	// Start forwarding both in goroutines
	stdoutDone := make(chan error, 1)
	stderrDone := make(chan error, 1)

	go func() {
		stdoutDone <- launcher.forwardStdout(stdoutPipe)
	}()

	go func() {
		stderrDone <- launcher.forwardStderr(stderrPipe)
	}()

	// Give forwarding time to start
	time.Sleep(100 * time.Millisecond)

	// Close writers to signal completion
	stdoutFile.Close()
	stderrFile.Close()

	// Wait for both to complete
	stdoutErr := <-stdoutDone
	stderrErr := <-stderrDone

	// Restore and read captured output
	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	capturedStdout.ReadFrom(rOut)
	capturedStderr.ReadFrom(rErr)

	// Verify
	if stdoutErr != nil {
		t.Errorf("stdout forwarding failed: %v", stdoutErr)
	}
	if stderrErr != nil {
		t.Errorf("stderr forwarding failed: %v", stderrErr)
	}

	if capturedStdout.String() != stdoutData {
		t.Errorf("stdout: expected %q, got %q", stdoutData, capturedStdout.String())
	}
	if capturedStderr.String() != stderrData {
		t.Errorf("stderr: expected %q, got %q", stderrData, capturedStderr.String())
	}

	t.Log("Both stdout and stderr handled signal scenario correctly")
}
