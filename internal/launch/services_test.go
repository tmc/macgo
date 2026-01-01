package launch

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServicesLauncher_createPipeDirectory(t *testing.T) {
	launcher := &ServicesLauncher{}

	pipeDir, err := launcher.createPipeDirectory()
	if err != nil {
		t.Fatalf("createPipeDirectory() failed: %v", err)
	}
	defer func() { _ = os.RemoveAll(pipeDir) }()

	// Check that the directory was created
	if _, err := os.Stat(pipeDir); os.IsNotExist(err) {
		t.Errorf("Pipe directory was not created: %s", pipeDir)
	}

	// Check that it's actually a directory
	info, err := os.Stat(pipeDir)
	if err != nil {
		t.Fatalf("Failed to stat pipe directory: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("Pipe directory is not a directory: %s", pipeDir)
	}

	// Check that the path contains "macgo" (either in basename or parent directory)
	// Implementation uses ~/Library/Application Support/macgo/pipes/PID-TIMESTAMP
	if !strings.Contains(pipeDir, "macgo") {
		t.Errorf("Pipe directory path doesn't contain 'macgo': %s", pipeDir)
	}
}

func TestServicesLauncher_createNamedPipes(t *testing.T) {
	launcher := &ServicesLauncher{
		logger: NewLogger(),
	}

	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "macgo-pipe-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	pipes, err := launcher.createNamedPipes(tmpDir, true, true, true, false)
	if err != nil {
		t.Fatalf("createNamedPipes() failed: %v", err)
	}

	// Check that all pipe paths are set
	if pipes.stdin == "" {
		t.Error("stdin pipe path is empty")
	}
	if pipes.stdout == "" {
		t.Error("stdout pipe path is empty")
	}
	if pipes.stderr == "" {
		t.Error("stderr pipe path is empty")
	}

	// Check that pipe paths are in the expected directory
	expectedDir := tmpDir
	if !strings.HasPrefix(pipes.stdin, expectedDir) {
		t.Errorf("stdin pipe not in expected directory: %s", pipes.stdin)
	}
	if !strings.HasPrefix(pipes.stdout, expectedDir) {
		t.Errorf("stdout pipe not in expected directory: %s", pipes.stdout)
	}
	if !strings.HasPrefix(pipes.stderr, expectedDir) {
		t.Errorf("stderr pipe not in expected directory: %s", pipes.stderr)
	}

	// Check that pipe files were created
	for _, pipe := range []string{pipes.stdin, pipes.stdout, pipes.stderr} {
		if _, err := os.Stat(pipe); os.IsNotExist(err) {
			t.Errorf("Pipe was not created: %s", pipe)
		}
	}
}

func TestServicesLauncher_buildOpenCommand(t *testing.T) {
	launcher := &ServicesLauncher{
		logger: NewLogger(),
	}
	ctx := context.Background()
	bundlePath := "/path/to/TestApp.app"

	pipes := &pipeSet{
		stdin:  "/tmp/stdin",
		stdout: "/tmp/stdout",
		stderr: "/tmp/stderr",
	}

	// Save original os.Args and restore after test
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// NOTE: open-flags and env-vars strategies were removed as they don't work with LaunchServices.
	// Only config-file strategy is supported, which doesn't add I/O flags to the open command.
	tests := []struct {
		name     string
		args     []string
		wantArgs []string
	}{
		{
			name: "no args",
			args: []string{"program"},
			wantArgs: []string{
				"open",
				bundlePath,
			},
		},
		{
			name: "with args",
			args: []string{"program", "arg1", "arg2", "--flag"},
			wantArgs: []string{
				"open",
				bundlePath,
				"--args",
				"arg1", "arg2", "--flag",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args

			cmd, err := launcher.buildOpenCommand(ctx, bundlePath, pipes, false, nil)
			if err != nil {
				t.Fatalf("buildOpenCommand() failed: %v", err)
			}

			// The command path might be resolved to full path, check that it ends with "open"
			if !strings.HasSuffix(cmd.Path, "open") {
				t.Errorf("Command path = %s, want path ending with 'open'", cmd.Path)
			}

			gotArgs := cmd.Args
			if len(gotArgs) != len(tt.wantArgs) {
				t.Errorf("Command args length = %d, want %d", len(gotArgs), len(tt.wantArgs))
				t.Errorf("Got args: %v", gotArgs)
				t.Errorf("Want args: %v", tt.wantArgs)
				return
			}

			for i, want := range tt.wantArgs {
				if gotArgs[i] != want {
					t.Errorf("Command args[%d] = %s, want %s", i, gotArgs[i], want)
				}
			}
		})
	}
}

func TestServicesLauncher_cleanupPipeDirectory(t *testing.T) {
	launcher := &ServicesLauncher{}

	// Create a temporary directory to test cleanup
	tmpDir, err := os.MkdirTemp("", "macgo-cleanup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Create a test file in the directory
	testFile := filepath.Join(tmpDir, "testfile")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Verify directory and file exist
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Fatal("Test directory doesn't exist before cleanup")
	}
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatal("Test file doesn't exist before cleanup")
	}

	// Test cleanup
	launcher.cleanupPipeDirectory(tmpDir)

	// Verify directory was removed
	if _, err := os.Stat(tmpDir); !os.IsNotExist(err) {
		t.Error("Directory was not cleaned up")
	}
}

func TestServicesLauncher_Launch_ErrorHandling(t *testing.T) {
	t.Skip("Skipping integration test that might hang in CI")
	// This test would verify error handling but may hang in some environments
}

// TestPipeSet verifies the pipeSet structure
func TestPipeSet(t *testing.T) {
	pipes := &pipeSet{
		stdin:  "/tmp/stdin",
		stdout: "/tmp/stdout",
		stderr: "/tmp/stderr",
	}

	if pipes.stdin != "/tmp/stdin" {
		t.Errorf("stdin = %s, want /tmp/stdin", pipes.stdin)
	}
	if pipes.stdout != "/tmp/stdout" {
		t.Errorf("stdout = %s, want /tmp/stdout", pipes.stdout)
	}
	if pipes.stderr != "/tmp/stderr" {
		t.Errorf("stderr = %s, want /tmp/stderr", pipes.stderr)
	}
}
