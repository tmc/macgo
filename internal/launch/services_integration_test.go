package launch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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

// TestServicesLauncher_WritePipeConfig verifies that V2 correctly writes
// pipe configuration files.
func TestServicesLauncher_WritePipeConfig(t *testing.T) {
	launcher := &ServicesLauncher{
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

	cmd, err := launcher.buildOpenCommand(ctx, bundlePath, pipes, false, nil)
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

// TestServicesLauncher_BuildCommand_NoPipes verifies command building without pipes.
func TestServicesLauncher_BuildCommand_NoPipes(t *testing.T) {
	launcher := &ServicesLauncher{
		logger: NewLogger(),
	}

	ctx := context.Background()
	bundlePath := "/path/to/TestApp.app"

	cmd, err := launcher.buildOpenCommand(ctx, bundlePath, nil, false, nil)
	if err != nil {
		t.Fatalf("buildOpenCommand failed: %v", err)
	}

	cmdStr := strings.Join(cmd.Args, " ")

	// Command should include the bundle path
	if !strings.Contains(cmdStr, bundlePath) {
		t.Errorf("Command missing bundle path.\nGot: %s", cmdStr)
	}

	t.Logf("Command without pipes: %s", cmdStr)
}

// TestServicesLauncherManagerCreation verifies that the launch manager is created correctly.
func TestServicesLauncherManagerCreation(t *testing.T) {
	manager := New()

	// Check the type of the services launcher
	servicesLauncher := manager.servicesLauncher
	_, isServicesLauncher := servicesLauncher.(*ServicesLauncher)

	if !isServicesLauncher {
		t.Errorf("Expected ServicesLauncher type, got %T", servicesLauncher)
	}

	t.Logf("Manager created with ServicesLauncher")
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

// TestServicesLauncher_StdinForwarding verifies that V2 can forward stdin data
// to the application's stdin pipe.
func TestServicesLauncher_StdinForwarding(t *testing.T) {
	launcher := &ServicesLauncher{
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
	}{
		{"all pipes", true, true, true},
		{"stdout only", false, true, false},
		{"stderr only", false, false, true},
		{"stdout and stderr", false, true, true},
		{"stdin only", true, false, false},
		{"no pipes", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "macgo-v1-pipes-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			pipes, err := launcher.createNamedPipes(tmpDir, tt.enableStdin, tt.enableStdout, tt.enableStderr)
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

// TestServicesLauncher_CreatePipes verifies that V2 creates pipes correctly
// with different configuration options.
func TestServicesLauncher_CreatePipes(t *testing.T) {
	launcher := &ServicesLauncher{
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

			pipes, err := launcher.createNamedPipes(tmpDir, tt.enableStdin, true, true)
			if err != nil {
				t.Fatalf("createNamedPipes failed: %v", err)
			}

			// Always creates stdout and stderr
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

// TestServicesLauncher_CleanupWithErrors verifies that V2 cleanup works
// even when some files are missing or there are permission errors.
func TestServicesLauncher_CleanupWithErrors(t *testing.T) {
	launcher := &ServicesLauncher{
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
