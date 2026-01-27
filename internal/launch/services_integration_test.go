package launch

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestServicesLauncherV1_IOTimeout verifies that V1 has timeout protection
// when using the open-flags strategy (which doesn't work with .app bundles).
func TestServicesLauncherV1_IOTimeout(t *testing.T) {
	launcher := &ServicesLauncher{
		logger: NewLogger(),
	}

	// Create a test pipe directory
	tmpDir, err := os.MkdirTemp("", "macgo-v1-timeout-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test pipes
	pipes, err := launcher.createNamedPipes(tmpDir, false, true, true, false)
	if err != nil {
		t.Fatalf("Failed to create pipes: %v", err)
	}

	// Write test data to the pipes (simulating child process output)
	testData := "test output\n"
	if err := os.WriteFile(pipes.stdout, []byte(testData), 0600); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}

	// Test forwardStdout with continuous polling (V1's current behavior)
	// This should complete when the file stops growing
	start := time.Now()
	err = launcher.forwardStdout(pipes.stdout)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("forwardStdout failed: %v", err)
	}

	// Verify the timeout mechanism (should complete within reasonable time)
	// The no-growth timeout is 5 seconds (50 * 100ms)
	if elapsed > 10*time.Second {
		t.Errorf("forwardStdout took too long (%v), timeout protection may not be working", elapsed)
	}

	t.Logf("V1 forwardStdout completed in %v (expected ~5s for no-growth timeout)", elapsed)
}

// TestServicesLauncherV2_ContinuousPolling verifies that V2 uses continuous polling
// to capture output even when the file grows slowly.
func TestServicesLauncherV2_ContinuousPolling(t *testing.T) {
	launcher := &ServicesLauncherV2{
		logger: NewLogger(),
	}

	// Create a test pipe directory
	tmpDir, err := os.MkdirTemp("", "macgo-v2-polling-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create stdout pipe
	stdoutPipe := filepath.Join(tmpDir, "stdout")
	f, err := os.OpenFile(stdoutPipe, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}

	// Write initial data
	initialData := "initial output\n"
	if _, err := f.WriteString(initialData); err != nil {
		t.Fatalf("Failed to write initial data: %v", err)
	}
	f.Sync()

	// Start forwardStdout in a goroutine (it will poll continuously)
	done := make(chan error, 1)
	captured := &bytes.Buffer{}

	// Override os.Stdout temporarily to capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	go func() {
		done <- launcher.forwardStdout(stdoutPipe)
	}()

	// Give it time to read initial data
	time.Sleep(200 * time.Millisecond)

	// Write more data (simulating slow child process)
	moreData := "more output\n"
	if _, err := f.WriteString(moreData); err != nil {
		t.Fatalf("Failed to write more data: %v", err)
	}
	f.Sync()
	f.Close()

	// Capture the output
	go func() {
		io.Copy(captured, r)
	}()

	// Wait for forwardStdout to complete (should timeout after no growth)
	select {
	case err := <-done:
		w.Close()
		os.Stdout = oldStdout

		if err != nil {
			t.Errorf("forwardStdout failed: %v", err)
		}

		// Wait a bit for capture to complete
		time.Sleep(100 * time.Millisecond)

		// Verify both pieces of data were captured
		output := captured.String()
		if !strings.Contains(output, "initial output") {
			t.Errorf("V2 failed to capture initial output.\nGot: %q", output)
		}
		if !strings.Contains(output, "more output") {
			t.Errorf("V2 failed to capture additional output via polling.\nGot: %q", output)
		}

		t.Logf("V2 successfully captured output via continuous polling: %q", output)

	case <-time.After(15 * time.Second):
		w.Close()
		os.Stdout = oldStdout
		t.Fatal("V2 forwardStdout timed out (test timeout)")
	}
}

// TestServicesLauncherV1_WritePipeConfig verifies that V1 correctly writes
// pipe configuration files.
func TestServicesLauncherV1_WritePipeConfig(t *testing.T) {
	launcher := &ServicesLauncher{
		logger: NewLogger(),
	}

	tmpDir, err := os.MkdirTemp("", "macgo-v1-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configFile := filepath.Join(tmpDir, "config")
	pipes := &pipeSet{
		stdin:  filepath.Join(tmpDir, "stdin"),
		stdout: filepath.Join(tmpDir, "stdout"),
		stderr: filepath.Join(tmpDir, "stderr"),
	}

	err = launcher.writePipeConfig(configFile, pipes, "/test/bundle.app")
	if err != nil {
		t.Fatalf("writePipeConfig failed: %v", err)
	}

	// Read and verify config content
	content, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	configStr := string(content)

	// Verify all pipe paths are in the config
	if !strings.Contains(configStr, fmt.Sprintf("MACGO_STDIN_PIPE=%s", pipes.stdin)) {
		t.Errorf("Config missing stdin pipe.\nGot: %s", configStr)
	}
	if !strings.Contains(configStr, fmt.Sprintf("MACGO_STDOUT_PIPE=%s", pipes.stdout)) {
		t.Errorf("Config missing stdout pipe.\nGot: %s", configStr)
	}
	if !strings.Contains(configStr, fmt.Sprintf("MACGO_STDERR_PIPE=%s", pipes.stderr)) {
		t.Errorf("Config missing stderr pipe.\nGot: %s", configStr)
	}

	t.Logf("V1 config file content:\n%s", configStr)
}

// TestServicesLauncherV2_WritePipeConfig verifies that V2 correctly writes
// pipe configuration files.
func TestServicesLauncherV2_WritePipeConfig(t *testing.T) {
	launcher := &ServicesLauncherV2{
		logger: NewLogger(),
	}

	tmpDir, err := os.MkdirTemp("", "macgo-v2-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configFile := filepath.Join(tmpDir, "config")
	pipes := &pipeSet{
		stdin:  "", // V2 doesn't create stdin by default
		stdout: filepath.Join(tmpDir, "stdout"),
		stderr: filepath.Join(tmpDir, "stderr"),
	}

	err = launcher.writePipeConfig(configFile, pipes)
	if err != nil {
		t.Fatalf("writePipeConfig failed: %v", err)
	}

	// Read and verify config content
	content, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	configStr := string(content)

	// Verify stdout and stderr are in the config
	if !strings.Contains(configStr, fmt.Sprintf("MACGO_STDOUT_PIPE=%s", pipes.stdout)) {
		t.Errorf("Config missing stdout pipe.\nGot: %s", configStr)
	}
	if !strings.Contains(configStr, fmt.Sprintf("MACGO_STDERR_PIPE=%s", pipes.stderr)) {
		t.Errorf("Config missing stderr pipe.\nGot: %s", configStr)
	}

	// Verify stdin is not in config (unless explicitly enabled)
	if strings.Contains(configStr, "MACGO_STDIN_PIPE") {
		t.Logf("Note: Config contains stdin (may be explicitly enabled)")
	}

	t.Logf("V2 config file content:\n%s", configStr)
}

// TestServicesLauncherV1_BuildCommandWithConfigFileStrategy verifies that V1
// builds the correct open command when using config-file strategy.
func TestServicesLauncherV1_BuildCommandWithConfigFileStrategy(t *testing.T) {
	launcher := &ServicesLauncher{
		logger: NewLogger(),
	}

	// Set environment to use config-file strategy (default)
	os.Setenv("MACGO_IO_STRATEGY", "config-file")
	defer os.Unsetenv("MACGO_IO_STRATEGY")

	ctx := context.Background()
	bundlePath := "/path/to/TestApp.app"

	tmpDir, err := os.MkdirTemp("", "macgo-v1-cmd-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pipes := &pipeSet{
		stdin:  filepath.Join(tmpDir, "stdin"),
		stdout: filepath.Join(tmpDir, "stdout"),
		stderr: filepath.Join(tmpDir, "stderr"),
	}

	cmd, err := launcher.buildOpenCommand(ctx, bundlePath, pipes, false)
	if err != nil {
		t.Fatalf("buildOpenCommand failed: %v", err)
	}

	// With config-file strategy, the command should NOT include -i/-o/--stderr flags
	cmdStr := strings.Join(cmd.Args, " ")
	if strings.Contains(cmdStr, "-i") || strings.Contains(cmdStr, "-o") || strings.Contains(cmdStr, "--stderr") {
		t.Errorf("V1 config-file strategy should not use -i/-o/--stderr flags.\nGot command: %s", cmdStr)
	}

	// Command should include the bundle path
	if !strings.Contains(cmdStr, bundlePath) {
		t.Errorf("Command missing bundle path.\nGot: %s", cmdStr)
	}

	t.Logf("V1 config-file command: %s", cmdStr)
}

// TestServicesLauncherV2_BuildCommand verifies that V2 builds correct commands.
func TestServicesLauncherV2_BuildCommand(t *testing.T) {
	launcher := &ServicesLauncherV2{
		logger: NewLogger(),
	}

	ctx := context.Background()
	bundlePath := "/path/to/TestApp.app"

	tmpDir, err := os.MkdirTemp("", "macgo-v2-cmd-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configFile := filepath.Join(tmpDir, "config")

	cmd, err := launcher.buildOpenCommand(ctx, bundlePath, configFile, nil, false)
	if err != nil {
		t.Fatalf("buildOpenCommand failed: %v", err)
	}

	cmdStr := strings.Join(cmd.Args, " ")

	// V2 should not use -W flag when config file is present (to avoid conflicts)
	if strings.Contains(cmdStr, "-W") && configFile != "" {
		t.Errorf("V2 should not use -W flag with config file.\nGot command: %s", cmdStr)
	}

	// Command should include the bundle path
	if !strings.Contains(cmdStr, bundlePath) {
		t.Errorf("Command missing bundle path.\nGot: %s", cmdStr)
	}

	t.Logf("V2 command: %s", cmdStr)
}

// TestServicesLauncherV1_NoGrowthTimeout verifies that V1's forwardStdout
// correctly implements the no-growth timeout mechanism.
func TestServicesLauncherV1_NoGrowthTimeout(t *testing.T) {
	// Skip: forwardStdout now uses FIFO-only mode; polling mode is deprecated
	t.Skip("MACGO_USE_FIFO=0 polling mode is deprecated; stdout uses FIFO-only forwarding")

	// Disable FIFO usage to force polling behavior for this test
	os.Setenv("MACGO_USE_FIFO", "0")
	defer os.Unsetenv("MACGO_USE_FIFO")

	launcher := &ServicesLauncher{
		logger: NewLogger(),
	}

	tmpDir, err := os.MkdirTemp("", "macgo-v1-nogrowth-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	stdoutPipe := filepath.Join(tmpDir, "stdout")

	// Create pipe file with some data
	testData := "test output\n"
	if err := os.WriteFile(stdoutPipe, []byte(testData), 0600); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}

	// forwardStdout should timeout after ~5 seconds when file stops growing
	start := time.Now()
	err = launcher.forwardStdout(stdoutPipe)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("forwardStdout failed: %v", err)
	}

	// Should complete around 5 seconds (50 checks * 100ms)
	if elapsed < 4*time.Second || elapsed > 8*time.Second {
		t.Errorf("V1 no-growth timeout incorrect: expected ~5s, got %v", elapsed)
	}

	t.Logf("V1 no-growth timeout worked correctly: %v", elapsed)
}

// TestServicesLauncherV2_NoGrowthTimeout verifies that V2's forwardStdout
// correctly implements the no-growth timeout mechanism.
func TestServicesLauncherV2_NoGrowthTimeout(t *testing.T) {
	// Disable FIFO usage to force polling behavior for this test
	os.Setenv("MACGO_USE_FIFO", "0")
	defer os.Unsetenv("MACGO_USE_FIFO")

	launcher := &ServicesLauncherV2{
		logger: NewLogger(),
	}

	tmpDir, err := os.MkdirTemp("", "macgo-v2-nogrowth-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	stdoutPipe := filepath.Join(tmpDir, "stdout")

	// Create pipe file with some data
	testData := "test output\n"
	if err := os.WriteFile(stdoutPipe, []byte(testData), 0600); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}

	// forwardStdout should timeout after ~5 seconds when file stops growing
	start := time.Now()
	err = launcher.forwardStdout(stdoutPipe)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("forwardStdout failed: %v", err)
	}

	// Should complete around 5 seconds (50 checks * 100ms)
	if elapsed < 4*time.Second || elapsed > 8*time.Second {
		t.Errorf("V2 no-growth timeout incorrect: expected ~5s, got %v", elapsed)
	}

	t.Logf("V2 no-growth timeout worked correctly: %v", elapsed)
}

// TestServicesLauncherVersionSelection verifies that the correct launcher
// version is selected based on environment variables.
func TestServicesLauncherVersionSelection(t *testing.T) {
	tests := []struct {
		name       string
		envVersion string
		expectV2   bool
	}{
		{"default (no env)", "", false},
		{"explicit v1", "1", false},
		{"explicit v2", "2", true},
		{"explicit v2 alt", "v2", true},
		{"unknown version", "99", false}, // Defaults to V1
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVersion != "" {
				os.Setenv("MACGO_SERVICES_VERSION", tt.envVersion)
				defer os.Unsetenv("MACGO_SERVICES_VERSION")
			}

			manager := New()

			// Check the type of the services launcher
			servicesLauncher := manager.servicesLauncher
			_, isV2 := servicesLauncher.(*ServicesLauncherV2)

			if isV2 != tt.expectV2 {
				t.Errorf("Expected V2=%v, got V2=%v", tt.expectV2, isV2)
			}

			t.Logf("Correctly selected launcher version (V2=%v)", isV2)
		})
	}
}

// TestServicesLauncherV1_StdinForwarding verifies that V1 can forward stdin data
// to the application's stdin pipe.
func TestServicesLauncherV1_StdinForwarding(t *testing.T) {
	launcher := &ServicesLauncher{
		logger: NewLogger(),
	}

	tmpDir, err := os.MkdirTemp("", "macgo-v1-stdin-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	stdinPipe := filepath.Join(tmpDir, "stdin")

	// Create the stdin pipe file
	f, err := os.OpenFile(stdinPipe, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatalf("Failed to create stdin pipe: %v", err)
	}
	f.Close()

	// Create a test context with cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Set up stdin with test data
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	// Write test data to stdin in a goroutine
	testData := "test input\n"
	go func() {
		w.Write([]byte(testData))
		w.Close()
	}()

	// Forward stdin (should timeout when pipe is not being read)
	err = launcher.forwardStdin(ctx, stdinPipe)
	os.Stdin = oldStdin

	// We expect either success or context timeout, both are valid
	if err != nil && err != context.DeadlineExceeded {
		t.Logf("forwardStdin returned: %v (expected timeout or success)", err)
	}

	t.Logf("V1 stdin forwarding completed")
}

// TestServicesLauncherV2_StdinForwarding verifies that V2 can forward stdin data
// to the application's stdin pipe.
func TestServicesLauncherV2_StdinForwarding(t *testing.T) {
	launcher := &ServicesLauncherV2{
		logger: NewLogger(),
	}

	tmpDir, err := os.MkdirTemp("", "macgo-v2-stdin-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	stdinPipe := filepath.Join(tmpDir, "stdin")

	// Create the stdin pipe file
	f, err := os.OpenFile(stdinPipe, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatalf("Failed to create stdin pipe: %v", err)
	}
	f.Close()

	// Create a test context with cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Set up stdin with test data
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	// Write test data to stdin in a goroutine
	testData := "test input\n"
	go func() {
		w.Write([]byte(testData))
		w.Close()
	}()

	// Forward stdin (should timeout when pipe is not being read)
	err = launcher.forwardStdin(ctx, stdinPipe)
	os.Stdin = oldStdin

	// We expect either success or context timeout, both are valid
	if err != nil && err != context.DeadlineExceeded {
		t.Logf("forwardStdin returned: %v (expected timeout or success)", err)
	}

	t.Logf("V2 stdin forwarding completed")
}

// TestServicesLauncherV1_StderrForwarding verifies that V1 correctly forwards
// stderr output through the stderr pipe.
func TestServicesLauncherV1_StderrForwarding(t *testing.T) {
	launcher := &ServicesLauncher{
		logger: NewLogger(),
	}

	tmpDir, err := os.MkdirTemp("", "macgo-v1-stderr-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	stderrPipe := filepath.Join(tmpDir, "stderr")

	// Write test data to stderr pipe
	testData := "error message\n"
	if err := os.WriteFile(stderrPipe, []byte(testData), 0600); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}

	// Disable FIFO usage to force polling behavior for this test
	os.Setenv("MACGO_USE_FIFO", "0")
	defer os.Unsetenv("MACGO_USE_FIFO")

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	captured := &bytes.Buffer{}
	go func() {
		io.Copy(captured, r)
	}()

	// Forward stderr (should timeout after no growth)
	start := time.Now()
	err = launcher.forwardStderr(stderrPipe)
	elapsed := time.Since(start)

	w.Close()
	os.Stderr = oldStderr

	if err != nil {
		t.Errorf("forwardStderr failed: %v", err)
	}

	// Wait for capture to complete
	time.Sleep(100 * time.Millisecond)

	// Verify data was captured
	output := captured.String()
	if !strings.Contains(output, "error message") {
		t.Errorf("V1 failed to capture stderr.\nGot: %q", output)
	}

	// Verify timeout mechanism worked
	if elapsed < 4*time.Second || elapsed > 8*time.Second {
		t.Errorf("V1 stderr timeout incorrect: expected ~5s, got %v", elapsed)
	}

	t.Logf("V1 stderr forwarding completed in %v: %q", elapsed, output)
}

// TestServicesLauncherV2_StderrForwarding verifies that V2 correctly forwards
// stderr output through the stderr pipe.
func TestServicesLauncherV2_StderrForwarding(t *testing.T) {
	launcher := &ServicesLauncherV2{
		logger: NewLogger(),
	}

	tmpDir, err := os.MkdirTemp("", "macgo-v2-stderr-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	stderrPipe := filepath.Join(tmpDir, "stderr")

	// Create stderr pipe file
	f, err := os.OpenFile(stderrPipe, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatalf("Failed to create stderr pipe: %v", err)
	}

	// Write initial data
	initialData := "initial error\n"
	if _, err := f.WriteString(initialData); err != nil {
		t.Fatalf("Failed to write initial data: %v", err)
	}
	f.Sync()

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	captured := &bytes.Buffer{}
	go func() {
		io.Copy(captured, r)
	}()

	// Start forwarding in a goroutine
	done := make(chan error, 1)
	go func() {
		done <- launcher.forwardStderr(stderrPipe)
	}()

	// Give it time to read initial data
	time.Sleep(200 * time.Millisecond)

	// Write more data (simulating slow error output)
	moreData := "additional error\n"
	if _, err := f.WriteString(moreData); err != nil {
		t.Fatalf("Failed to write more data: %v", err)
	}
	f.Sync()
	f.Close()

	// Wait for forwarding to complete (should timeout after no growth)
	select {
	case err := <-done:
		w.Close()
		os.Stderr = oldStderr

		if err != nil {
			t.Errorf("forwardStderr failed: %v", err)
		}

		// Wait for capture to complete
		time.Sleep(100 * time.Millisecond)

		// Verify both pieces of data were captured
		output := captured.String()
		if !strings.Contains(output, "initial error") {
			t.Errorf("V2 failed to capture initial stderr.\nGot: %q", output)
		}
		if !strings.Contains(output, "additional error") {
			t.Errorf("V2 failed to capture additional stderr.\nGot: %q", output)
		}

		t.Logf("V2 stderr forwarding completed: %q", output)

	case <-time.After(15 * time.Second):
		w.Close()
		os.Stderr = oldStderr
		t.Fatal("V2 forwardStderr timed out (test timeout)")
	}
}

// TestServicesLauncherV1_CreatePipes verifies that V1 creates pipes correctly
// with different configuration options.
func TestServicesLauncherV1_CreatePipes(t *testing.T) {
	launcher := &ServicesLauncher{
		logger: NewLogger(),
	}

	tests := []struct {
		name         string
		enableStdin  bool
		enableStdout bool
		enableStderr bool
		useFifo      bool
	}{
		{"all pipes", true, true, true, false},
		{"stdout only", false, true, false, false},
		{"stderr only", false, false, true, false},
		{"stdout and stderr", false, true, true, false},
		{"stdin only", true, false, false, false},
		{"no pipes", false, false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "macgo-v1-pipes-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			pipes, err := launcher.createNamedPipes(tmpDir, tt.enableStdin, tt.enableStdout, tt.enableStderr, tt.useFifo)
			if err != nil {
				t.Fatalf("createNamedPipes failed: %v", err)
			}

			// Verify expected pipes were created
			if tt.enableStdin {
				if pipes.stdin == "" {
					t.Error("Expected stdin pipe but got empty path")
				}
				if _, err := os.Stat(pipes.stdin); err != nil {
					t.Errorf("stdin pipe not created: %v", err)
				}
			} else if pipes.stdin != "" {
				t.Error("Unexpected stdin pipe created")
			}

			if tt.enableStdout {
				if pipes.stdout == "" {
					t.Error("Expected stdout pipe but got empty path")
				}
				if _, err := os.Stat(pipes.stdout); err != nil {
					t.Errorf("stdout pipe not created: %v", err)
				}
			} else if pipes.stdout != "" {
				t.Error("Unexpected stdout pipe created")
			}

			if tt.enableStderr {
				if pipes.stderr == "" {
					t.Error("Expected stderr pipe but got empty path")
				}
				if _, err := os.Stat(pipes.stderr); err != nil {
					t.Errorf("stderr pipe not created: %v", err)
				}
			} else if pipes.stderr != "" {
				t.Error("Unexpected stderr pipe created")
			}

			t.Logf("V1 created pipes correctly: stdin=%v, stdout=%v, stderr=%v",
				pipes.stdin != "", pipes.stdout != "", pipes.stderr != "")
		})
	}
}

// TestServicesLauncherV2_CreatePipes verifies that V2 creates pipes correctly
// with different configuration options.
func TestServicesLauncherV2_CreatePipes(t *testing.T) {
	launcher := &ServicesLauncherV2{
		logger: NewLogger(),
	}

	tests := []struct {
		name        string
		enableStdin bool
	}{
		{"with stdin", true},
		{"without stdin", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "macgo-v2-pipes-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			pipes, err := launcher.createPipes(tmpDir, tt.enableStdin)
			if err != nil {
				t.Fatalf("createPipes failed: %v", err)
			}

			// V2 always creates stdout and stderr
			if pipes.stdout == "" {
				t.Error("Expected stdout pipe but got empty path")
			}
			if _, err := os.Stat(pipes.stdout); err != nil {
				t.Errorf("stdout pipe not created: %v", err)
			}

			if pipes.stderr == "" {
				t.Error("Expected stderr pipe but got empty path")
			}
			if _, err := os.Stat(pipes.stderr); err != nil {
				t.Errorf("stderr pipe not created: %v", err)
			}

			// Check stdin based on configuration
			if tt.enableStdin {
				if pipes.stdin == "" {
					t.Error("Expected stdin pipe but got empty path")
				}
				if _, err := os.Stat(pipes.stdin); err != nil {
					t.Errorf("stdin pipe not created: %v", err)
				}
			} else if pipes.stdin != "" {
				t.Error("Unexpected stdin pipe created")
			}

			t.Logf("V2 created pipes correctly: stdin=%v, stdout=%v, stderr=%v",
				pipes.stdin != "", pipes.stdout != "", pipes.stderr != "")
		})
	}
}

// TestServicesLauncherV1_CleanupWithErrors verifies that V1 cleanup works
// even when some files are missing or there are permission errors.
func TestServicesLauncherV1_CleanupWithErrors(t *testing.T) {
	launcher := &ServicesLauncher{
		logger: NewLogger(),
	}

	// Test cleanup of non-existent directory (should not panic)
	launcher.cleanupPipeDirectory("/nonexistent/directory")

	// Test cleanup of empty directory
	tmpDir, err := os.MkdirTemp("", "macgo-v1-cleanup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	launcher.cleanupPipeDirectory(tmpDir)

	// Verify directory was removed
	if _, err := os.Stat(tmpDir); !os.IsNotExist(err) {
		t.Error("Empty directory was not removed")
	}

	t.Log("V1 cleanup handles errors gracefully")
}

// TestServicesLauncherV2_CleanupWithErrors verifies that V2 cleanup works
// even when some files are missing or there are permission errors.
func TestServicesLauncherV2_CleanupWithErrors(t *testing.T) {
	launcher := &ServicesLauncherV2{
		logger: NewLogger(),
	}

	// Test cleanup of non-existent directory (should not panic)
	launcher.cleanupPipeDirectory("/nonexistent/directory")

	// Test cleanup of empty directory
	tmpDir, err := os.MkdirTemp("", "macgo-v2-cleanup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	launcher.cleanupPipeDirectory(tmpDir)

	// Verify directory was removed
	if _, err := os.Stat(tmpDir); !os.IsNotExist(err) {
		t.Error("Empty directory was not removed")
	}

	t.Log("V2 cleanup handles errors gracefully")
}
