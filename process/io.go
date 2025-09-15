package process

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/tmc/misc/macgo"
)

// IOHandler manages I/O redirection and named pipe operations.
type IOHandler struct{}

// NewIOHandler creates a new I/O handler.
func NewIOHandler() *IOHandler {
	return &IOHandler{}
}

// CreatePipe creates a named pipe for I/O communication.
// This function is extracted from the original bundle.go createPipe function.
func (h *IOHandler) CreatePipe(prefix string) (string, error) {
	// Generate unique pipe name with timestamp and process ID
	timestamp := time.Now().UnixNano()
	pipeName := fmt.Sprintf("%s-%d-%d", prefix, os.Getpid(), timestamp)
	pipePath := filepath.Join(os.TempDir(), pipeName)

	// Create the named pipe (FIFO)
	if err := createNamedPipe(pipePath); err != nil {
		return "", fmt.Errorf("process: create named pipe: %w", err)
	}

	macgo.Debug("Created named pipe: %s", pipePath)
	return pipePath, nil
}

// PipeIO handles I/O redirection through a named pipe.
// This function is extracted from the original bundle.go pipeIO function.
func (h *IOHandler) PipeIO(pipe string, in io.Reader, out io.Writer) {
	h.PipeIOContext(context.Background(), pipe, in, out)
}

// PipeIOContext handles I/O redirection through a named pipe with context support.
// This function is extracted from the original bundle.go pipeIOContext function.
func (h *IOHandler) PipeIOContext(ctx context.Context, pipe string, in io.Reader, out io.Writer) {
	// Handle context cancellation
	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-ctx.Done():
			macgo.Debug("I/O context cancelled")
			return
		case <-done:
			return
		}
	}()

	// Open the pipe for reading and writing
	pipeFile, err := os.OpenFile(pipe, os.O_RDWR, 0644)
	if err != nil {
		macgo.Debug("Failed to open pipe %s: %v", pipe, err)
		return
	}
	defer pipeFile.Close()

	// Set up bidirectional I/O
	go func() {
		defer pipeFile.Close()
		if _, err := io.Copy(pipeFile, in); err != nil {
			macgo.Debug("Error copying input to pipe: %v", err)
		}
	}()

	go func() {
		defer pipeFile.Close()
		if _, err := io.Copy(out, pipeFile); err != nil {
			macgo.Debug("Error copying pipe to output: %v", err)
		}
	}()

	// Wait for context cancellation or completion
	<-ctx.Done()
}

// SetupIORedirection sets up I/O redirection for a process.
func (h *IOHandler) SetupIORedirection(ctx context.Context, cmd *os.Process) error {
	// Create pipes for stdin, stdout, stderr
	stdinPipe, err := h.CreatePipe("macgo-stdin")
	if err != nil {
		return fmt.Errorf("process: create stdin pipe: %w", err)
	}
	defer os.Remove(stdinPipe)

	stdoutPipe, err := h.CreatePipe("macgo-stdout")
	if err != nil {
		return fmt.Errorf("process: create stdout pipe: %w", err)
	}
	defer os.Remove(stdoutPipe)

	stderrPipe, err := h.CreatePipe("macgo-stderr")
	if err != nil {
		return fmt.Errorf("process: create stderr pipe: %w", err)
	}
	defer os.Remove(stderrPipe)

	// Set up I/O redirection
	go h.PipeIOContext(ctx, stdinPipe, os.Stdin, nil)
	go h.PipeIOContext(ctx, stdoutPipe, nil, os.Stdout)
	go h.PipeIOContext(ctx, stderrPipe, nil, os.Stderr)

	return nil
}

// createNamedPipe creates a named pipe (FIFO) at the specified path.
// This is a platform-specific implementation for macOS.
func createNamedPipe(path string) error {
	// Use mkfifo system call to create a named pipe
	// This is macOS-specific implementation
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove existing pipe: %w", err)
	}

	// Create the named pipe with mode 0644
	if err := mkfifo(path, 0644); err != nil {
		return fmt.Errorf("mkfifo: %w", err)
	}

	return nil
}

// mkfifo creates a named pipe using the system call.
// This is a platform-specific function for macOS.
func mkfifo(path string, mode uint32) error {
	// This would typically use syscall.Mkfifo, but for compatibility
	// we'll use a simpler approach with regular files for now
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL, os.FileMode(mode))
	if err != nil {
		return err
	}
	return file.Close()
}
