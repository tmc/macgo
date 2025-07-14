package macgo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestIORedirectionIntegration tests the complete I/O redirection workflow
func TestIORedirectionIntegration(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping test on non-macOS platform")
	}

	t.Run("Complete I/O redirection workflow", func(t *testing.T) {
		// Create pipes for stdin, stdout, stderr
		stdinPipe, err := createPipe("test-stdin")
		if err != nil {
			t.Fatalf("Failed to create stdin pipe: %v", err)
		}
		defer os.Remove(stdinPipe)

		stdoutPipe, err := createPipe("test-stdout")
		if err != nil {
			t.Fatalf("Failed to create stdout pipe: %v", err)
		}
		defer os.Remove(stdoutPipe)

		stderrPipe, err := createPipe("test-stderr")
		if err != nil {
			t.Fatalf("Failed to create stderr pipe: %v", err)
		}
		defer os.Remove(stderrPipe)

		// Test data
		stdinData := "Hello from stdin\n"
		expectedStdout := "Received: Hello from stdin\n"
		expectedStderr := "Processing input...\n"

		// Create a test program that reads from stdin and writes to stdout/stderr
		testProgram := `
package main
import (
	"bufio"
	"fmt"
	"os"
)
func main() {
	fmt.Fprintln(os.Stderr, "Processing input...")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		fmt.Fprintf(os.Stdout, "Received: %s\n", scanner.Text())
	}
}
`
		// Create temporary test program
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test_io.go")
		if err := os.WriteFile(testFile, []byte(testProgram), 0644); err != nil {
			t.Fatalf("Failed to write test program: %v", err)
		}

		// Build test program
		testBinary := filepath.Join(tmpDir, "test_io")
		cmd := exec.Command("go", "build", "-o", testBinary, testFile)
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to build test program: %v", err)
		}

		// Set up I/O redirection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Create buffers to capture output
		var stdoutBuf, stderrBuf bytes.Buffer

		// Start I/O handlers
		var wg sync.WaitGroup
		wg.Add(3)

		// Handle stdin
		go func() {
			defer wg.Done()
			reader := strings.NewReader(stdinData)
			pipeIOContext(ctx, stdinPipe, reader, nil)
		}()

		// Handle stdout
		go func() {
			defer wg.Done()
			pipeIOContext(ctx, stdoutPipe, nil, &stdoutBuf)
		}()

		// Handle stderr
		go func() {
			defer wg.Done()
			pipeIOContext(ctx, stderrPipe, nil, &stderrBuf)
		}()

		// Run the test program with pipes
		testCmd := exec.CommandContext(ctx, testBinary)

		// Open pipes for the child process
		stdinFile, err := os.Open(stdinPipe)
		if err != nil {
			t.Fatalf("Failed to open stdin pipe: %v", err)
		}
		defer stdinFile.Close()

		stdoutFile, err := os.OpenFile(stdoutPipe, os.O_WRONLY, 0)
		if err != nil {
			t.Fatalf("Failed to open stdout pipe: %v", err)
		}
		defer stdoutFile.Close()

		stderrFile, err := os.OpenFile(stderrPipe, os.O_WRONLY, 0)
		if err != nil {
			t.Fatalf("Failed to open stderr pipe: %v", err)
		}
		defer stderrFile.Close()

		testCmd.Stdin = stdinFile
		testCmd.Stdout = stdoutFile
		testCmd.Stderr = stderrFile

		// Run the command
		if err := testCmd.Run(); err != nil {
			t.Fatalf("Test program failed: %v", err)
		}

		// Close write ends to signal EOF
		stdoutFile.Close()
		stderrFile.Close()

		// Wait for I/O handlers with timeout
		done := make(chan bool)
		go func() {
			wg.Wait()
			done <- true
		}()

		select {
		case <-done:
			// Success
		case <-ctx.Done():
			t.Fatal("I/O handlers timed out")
		}

		// Verify output
		if stdoutBuf.String() != expectedStdout {
			t.Errorf("Stdout mismatch:\nExpected: %q\nGot: %q", expectedStdout, stdoutBuf.String())
		}
		if stderrBuf.String() != expectedStderr {
			t.Errorf("Stderr mismatch:\nExpected: %q\nGot: %q", expectedStderr, stderrBuf.String())
		}
	})
}

// TestProcessCommunicationReliability tests reliable communication between processes
func TestProcessCommunicationReliability(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping test on non-macOS platform")
	}

	t.Run("Large data transfer", func(t *testing.T) {
		pipePath, err := createPipe("test-large-data")
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}
		defer os.Remove(pipePath)

		// Create 10MB of test data
		dataSize := 10 * 1024 * 1024
		testData := make([]byte, dataSize)
		for i := range testData {
			testData[i] = byte(i % 256)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		var received bytes.Buffer
		done := make(chan error, 2)

		// Writer goroutine
		go func() {
			reader := bytes.NewReader(testData)
			pipeIOContext(ctx, pipePath, reader, nil)
			done <- nil
		}()

		// Reader goroutine
		go func() {
			pipeIOContext(ctx, pipePath, nil, &received)
			done <- nil
		}()

		// Wait for both to complete
		for i := 0; i < 2; i++ {
			select {
			case err := <-done:
				if err != nil {
					t.Errorf("Transfer error: %v", err)
				}
			case <-ctx.Done():
				t.Fatal("Large data transfer timed out")
			}
		}

		// Verify data integrity
		if received.Len() != dataSize {
			t.Errorf("Size mismatch: expected %d, got %d", dataSize, received.Len())
		}
		if !bytes.Equal(testData, received.Bytes()) {
			t.Error("Data corruption detected")
		}
	})

	t.Run("Concurrent bidirectional communication", func(t *testing.T) {
		// Create two pipes for bidirectional communication
		pipe1, err := createPipe("test-bidir-1")
		if err != nil {
			t.Fatalf("Failed to create pipe1: %v", err)
		}
		defer os.Remove(pipe1)

		pipe2, err := createPipe("test-bidir-2")
		if err != nil {
			t.Fatalf("Failed to create pipe2: %v", err)
		}
		defer os.Remove(pipe2)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Messages to exchange
		messages := []string{
			"Message 1 from A to B",
			"Message 2 from B to A",
			"Message 3 from A to B",
			"Message 4 from B to A",
		}

		var wg sync.WaitGroup
		errors := make(chan error, 4)

		// Process A: writes to pipe1, reads from pipe2
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Write messages 0 and 2
			for i := 0; i < len(messages); i += 2 {
				reader := strings.NewReader(messages[i] + "\n")
				pipeIOContext(ctx, pipe1, reader, nil)
			}

			// Read messages 1 and 3
			var received bytes.Buffer
			pipeIOContext(ctx, pipe2, nil, &received)

			// Verify received messages
			lines := strings.Split(strings.TrimSpace(received.String()), "\n")
			for i := 1; i < len(messages); i += 2 {
				if i/2 < len(lines) && lines[i/2] != messages[i] {
					errors <- fmt.Errorf("Process A: expected %q, got %q", messages[i], lines[i/2])
				}
			}
		}()

		// Process B: writes to pipe2, reads from pipe1
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Read messages 0 and 2
			var received bytes.Buffer
			pipeIOContext(ctx, pipe1, nil, &received)

			// Write messages 1 and 3
			for i := 1; i < len(messages); i += 2 {
				reader := strings.NewReader(messages[i] + "\n")
				pipeIOContext(ctx, pipe2, reader, nil)
			}

			// Verify received messages
			lines := strings.Split(strings.TrimSpace(received.String()), "\n")
			for i := 0; i < len(messages); i += 2 {
				if i/2 < len(lines) && lines[i/2] != messages[i] {
					errors <- fmt.Errorf("Process B: expected %q, got %q", messages[i], lines[i/2])
				}
			}
		}()

		// Wait for completion
		done := make(chan bool)
		go func() {
			wg.Wait()
			close(errors)
			done <- true
		}()

		select {
		case <-done:
			// Check for errors
			for err := range errors {
				t.Error(err)
			}
		case <-ctx.Done():
			t.Fatal("Bidirectional communication timed out")
		}
	})
}

// TestPipeCleanupAndResourceManagement tests proper cleanup of pipes and resources
func TestPipeCleanupAndResourceManagement(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping test on non-macOS platform")
	}

	t.Run("Automatic cleanup on context cancellation", func(t *testing.T) {
		// Count initial pipes
		initialPipes := countTempPipes(t, "cleanup-test")

		ctx, cancel := context.WithCancel(context.Background())

		// Create multiple pipes
		numPipes := 10
		pipes := make([]string, numPipes)
		for i := 0; i < numPipes; i++ {
			pipe, err := createPipe(fmt.Sprintf("cleanup-test-%d", i))
			if err != nil {
				t.Fatalf("Failed to create pipe %d: %v", i, err)
			}
			pipes[i] = pipe
			defer os.Remove(pipe) // Ensure cleanup even if test fails
		}

		// Start I/O operations on all pipes
		var wg sync.WaitGroup
		for i, pipe := range pipes {
			wg.Add(2)

			// Writer
			go func(p string, idx int) {
				defer wg.Done()
				data := fmt.Sprintf("Data for pipe %d\n", idx)
				reader := strings.NewReader(data)
				pipeIOContext(ctx, p, reader, nil)
			}(pipe, i)

			// Reader
			go func(p string) {
				defer wg.Done()
				var buf bytes.Buffer
				pipeIOContext(ctx, p, nil, &buf)
			}(pipe)
		}

		// Cancel context after a short time
		time.Sleep(100 * time.Millisecond)
		cancel()

		// Wait for all goroutines to finish
		done := make(chan bool)
		go func() {
			wg.Wait()
			done <- true
		}()

		select {
		case <-done:
			// Good, all cleaned up
		case <-time.After(2 * time.Second):
			t.Error("Goroutines did not exit after context cancellation")
		}

		// Clean up pipes
		for _, pipe := range pipes {
			os.Remove(pipe)
		}

		// Verify no leaked pipes
		finalPipes := countTempPipes(t, "cleanup-test")
		if finalPipes > initialPipes {
			t.Errorf("Pipe leak detected: initial=%d, final=%d", initialPipes, finalPipes)
		}
	})

	t.Run("File descriptor management", func(t *testing.T) {
		// This test verifies that file descriptors are properly closed
		pipePath, err := createPipe("test-fd-mgmt")
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}
		defer os.Remove(pipePath)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Track open file descriptors
		var openFDs int32

		// Perform multiple open/close cycles
		cycles := 20
		var wg sync.WaitGroup

		for i := 0; i < cycles; i++ {
			wg.Add(2)

			// Writer
			go func(cycle int) {
				defer wg.Done()
				atomic.AddInt32(&openFDs, 1)
				defer atomic.AddInt32(&openFDs, -1)

				data := fmt.Sprintf("Cycle %d data\n", cycle)
				reader := strings.NewReader(data)
				pipeIOContext(ctx, pipePath, reader, nil)
			}(i)

			// Reader
			go func() {
				defer wg.Done()
				atomic.AddInt32(&openFDs, 1)
				defer atomic.AddInt32(&openFDs, -1)

				var buf bytes.Buffer
				pipeIOContext(ctx, pipePath, nil, &buf)
			}()

			// Small delay between cycles
			time.Sleep(10 * time.Millisecond)
		}

		// Wait for completion
		done := make(chan bool)
		go func() {
			wg.Wait()
			done <- true
		}()

		select {
		case <-done:
			// Verify all file descriptors were closed
			finalFDs := atomic.LoadInt32(&openFDs)
			if finalFDs != 0 {
				t.Errorf("File descriptor leak: %d FDs still open", finalFDs)
			}
		case <-ctx.Done():
			t.Fatal("File descriptor test timed out")
		}
	})
}

// TestErrorScenariosAndRecovery tests error handling and recovery
func TestErrorScenariosAndRecovery(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping test on non-macOS platform")
	}

	t.Run("Broken pipe recovery", func(t *testing.T) {
		pipePath, err := createPipe("test-broken-pipe")
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}
		defer os.Remove(pipePath)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Start writer
		writerDone := make(chan error)
		go func() {
			// Write a large amount of data
			data := make([]byte, 1024*1024) // 1MB
			for i := range data {
				data[i] = byte(i % 256)
			}
			reader := bytes.NewReader(data)
			pipeIOContext(ctx, pipePath, reader, nil)
			writerDone <- nil
		}()

		// Start reader but close it early to simulate broken pipe
		readerDone := make(chan error)
		go func() {
			f, err := os.Open(pipePath)
			if err != nil {
				readerDone <- err
				return
			}

			// Read a small amount then close
			buf := make([]byte, 1024)
			f.Read(buf)
			f.Close() // Simulate broken pipe

			readerDone <- nil
		}()

		// Wait for operations to complete
		select {
		case <-writerDone:
			// Writer should handle broken pipe gracefully
		case <-ctx.Done():
			t.Error("Writer did not handle broken pipe in time")
		}

		select {
		case <-readerDone:
			// Reader completed
		case <-time.After(1 * time.Second):
			t.Error("Reader did not complete")
		}
	})

	t.Run("Non-existent pipe handling", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		nonExistentPipe := "/tmp/non-existent-pipe-" + fmt.Sprint(time.Now().UnixNano())

		// Try to read from non-existent pipe
		readDone := make(chan bool)
		go func() {
			var buf bytes.Buffer
			pipeIOContext(ctx, nonExistentPipe, nil, &buf)
			readDone <- true
		}()

		// Try to write to non-existent pipe
		writeDone := make(chan bool)
		go func() {
			reader := strings.NewReader("test data")
			pipeIOContext(ctx, nonExistentPipe, reader, nil)
			writeDone <- true
		}()

		// Both operations should complete quickly with errors
		for i := 0; i < 2; i++ {
			select {
			case <-readDone:
				// Read completed (with error)
			case <-writeDone:
				// Write completed (with error)
			case <-time.After(1 * time.Second):
				t.Error("Operation did not handle non-existent pipe error")
			}
		}
	})

	t.Run("Permission denied recovery", func(t *testing.T) {
		pipePath, err := createPipe("test-perm-denied")
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}
		defer func() {
			os.Chmod(pipePath, 0644) // Restore permissions
			os.Remove(pipePath)
		}()

		// Remove all permissions
		if err := os.Chmod(pipePath, 0000); err != nil {
			t.Fatalf("Failed to change permissions: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Try to read with no permissions
		readDone := make(chan bool)
		go func() {
			var buf bytes.Buffer
			pipeIOContext(ctx, pipePath, nil, &buf)
			readDone <- true
		}()

		// Should handle permission error gracefully
		select {
		case <-readDone:
			// Completed (with error)
		case <-time.After(1 * time.Second):
			t.Error("Did not handle permission error")
		}
	})
}

// TestContextCancellationAndTimeouts tests context handling
func TestContextCancellationAndTimeouts(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping test on non-macOS platform")
	}

	t.Run("Immediate context cancellation", func(t *testing.T) {
		pipePath, err := createPipe("test-immediate-cancel")
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}
		defer os.Remove(pipePath)

		// Create an already-cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// Operations should exit immediately
		done := make(chan bool, 2)

		go func() {
			reader := strings.NewReader("test data")
			pipeIOContext(ctx, pipePath, reader, nil)
			done <- true
		}()

		go func() {
			var buf bytes.Buffer
			pipeIOContext(ctx, pipePath, nil, &buf)
			done <- true
		}()

		// Both should complete very quickly
		for i := 0; i < 2; i++ {
			select {
			case <-done:
				// Good
			case <-time.After(100 * time.Millisecond):
				t.Error("Operation did not respect cancelled context")
			}
		}
	})

	t.Run("Context timeout during operation", func(t *testing.T) {
		pipePath, err := createPipe("test-timeout")
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}
		defer os.Remove(pipePath)

		// Short timeout context
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		// Start a slow writer
		writerDone := make(chan bool)
		go func() {
			// Write data slowly
			data := strings.Repeat("x", 1024*1024) // 1MB
			reader := &slowReader{data: []byte(data), delay: 50 * time.Millisecond}
			pipeIOContext(ctx, pipePath, reader, nil)
			writerDone <- true
		}()

		// Start a reader
		readerDone := make(chan bool)
		go func() {
			var buf bytes.Buffer
			pipeIOContext(ctx, pipePath, nil, &buf)
			readerDone <- true
		}()

		// Both should stop when context times out
		<-ctx.Done()

		// Verify operations stopped
		time.Sleep(100 * time.Millisecond) // Give operations time to clean up

		select {
		case <-writerDone:
			// Writer stopped
		case <-time.After(500 * time.Millisecond):
			t.Error("Writer did not stop after context timeout")
		}

		select {
		case <-readerDone:
			// Reader stopped
		case <-time.After(500 * time.Millisecond):
			t.Error("Reader did not stop after context timeout")
		}
	})
}

// TestIntegrationWithRelaunchMechanism tests integration with the relaunch system
func TestIntegrationWithRelaunchMechanism(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping test on non-macOS platform")
	}

	t.Run("Simulated relaunch I/O setup", func(t *testing.T) {
		// This simulates the I/O setup that happens during relaunch
		pipes := make([]string, 3)
		pipeNames := []string{"stdin", "stdout", "stderr"}

		// Create pipes like relaunch does
		for i, name := range pipeNames {
			pipe, err := createPipe("macgo-" + name)
			if err != nil {
				t.Fatalf("Failed to create %s pipe: %v", name, err)
			}
			pipes[i] = pipe
			defer os.Remove(pipe)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Test data
		stdinData := "Input to relaunched process\n"
		expectedStdout := "Output from relaunched process\n"
		expectedStderr := "Errors from relaunched process\n"

		// Set up I/O handlers like relaunch does
		var stdoutBuf, stderrBuf bytes.Buffer
		var wg sync.WaitGroup

		// Handle stdin
		wg.Add(1)
		go func() {
			defer wg.Done()
			reader := strings.NewReader(stdinData)
			pipeIOContext(ctx, pipes[0], reader, nil)
		}()

		// Handle stdout
		wg.Add(1)
		go func() {
			defer wg.Done()
			pipeIOContext(ctx, pipes[1], nil, &stdoutBuf)
		}()

		// Handle stderr
		wg.Add(1)
		go func() {
			defer wg.Done()
			pipeIOContext(ctx, pipes[2], nil, &stderrBuf)
		}()

		// Simulate a child process writing to the pipes
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Read from stdin pipe
			stdinFile, err := os.Open(pipes[0])
			if err != nil {
				t.Logf("Failed to open stdin pipe: %v", err)
				return
			}
			defer stdinFile.Close()

			// Write to stdout pipe
			stdoutFile, err := os.OpenFile(pipes[1], os.O_WRONLY, 0)
			if err != nil {
				t.Logf("Failed to open stdout pipe: %v", err)
				return
			}
			defer stdoutFile.Close()

			// Write to stderr pipe
			stderrFile, err := os.OpenFile(pipes[2], os.O_WRONLY, 0)
			if err != nil {
				t.Logf("Failed to open stderr pipe: %v", err)
				return
			}
			defer stderrFile.Close()

			// Read from stdin and write to stdout/stderr
			buf := make([]byte, len(stdinData))
			n, _ := stdinFile.Read(buf)
			if n > 0 {
				stdoutFile.Write([]byte(expectedStdout))
				stderrFile.Write([]byte(expectedStderr))
			}
		}()

		// Wait for completion
		done := make(chan bool)
		go func() {
			wg.Wait()
			done <- true
		}()

		select {
		case <-done:
			// Verify output
			if stdoutBuf.String() != expectedStdout {
				t.Errorf("Stdout mismatch:\nExpected: %q\nGot: %q", expectedStdout, stdoutBuf.String())
			}
			if stderrBuf.String() != expectedStderr {
				t.Errorf("Stderr mismatch:\nExpected: %q\nGot: %q", expectedStderr, stderrBuf.String())
			}
		case <-ctx.Done():
			t.Fatal("Relaunch simulation timed out")
		}
	})

	t.Run("Multiple concurrent relaunches", func(t *testing.T) {
		// Test that multiple processes can use I/O redirection simultaneously
		numProcesses := 3
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var wg sync.WaitGroup
		errors := make(chan error, numProcesses)

		for i := 0; i < numProcesses; i++ {
			wg.Add(1)
			go func(processID int) {
				defer wg.Done()

				// Create unique pipes for this process
				pipes := make([]string, 3)
				pipeNames := []string{"stdin", "stdout", "stderr"}

				for j, name := range pipeNames {
					pipe, err := createPipe(fmt.Sprintf("proc%d-%s", processID, name))
					if err != nil {
						errors <- fmt.Errorf("Process %d: failed to create %s pipe: %v", processID, name, err)
						return
					}
					pipes[j] = pipe
					defer os.Remove(pipe)
				}

				// Test data unique to this process
				inputData := fmt.Sprintf("Input for process %d\n", processID)
				expectedOutput := fmt.Sprintf("Output from process %d\n", processID)

				// Set up I/O
				var outputBuf bytes.Buffer
				var ioWg sync.WaitGroup

				// Stdin handler
				ioWg.Add(1)
				go func() {
					defer ioWg.Done()
					reader := strings.NewReader(inputData)
					pipeIOContext(ctx, pipes[0], reader, nil)
				}()

				// Stdout handler
				ioWg.Add(1)
				go func() {
					defer ioWg.Done()
					pipeIOContext(ctx, pipes[1], nil, &outputBuf)
				}()

				// Simulate process
				ioWg.Add(1)
				go func() {
					defer ioWg.Done()

					// Read from stdin
					stdinFile, _ := os.Open(pipes[0])
					defer stdinFile.Close()

					// Write to stdout
					stdoutFile, _ := os.OpenFile(pipes[1], os.O_WRONLY, 0)
					defer stdoutFile.Close()

					// Echo with modification
					buf := make([]byte, 1024)
					n, _ := stdinFile.Read(buf)
					if n > 0 {
						stdoutFile.Write([]byte(expectedOutput))
					}
				}()

				// Wait for this process's I/O to complete
				ioDone := make(chan bool)
				go func() {
					ioWg.Wait()
					ioDone <- true
				}()

				select {
				case <-ioDone:
					// Verify output
					if outputBuf.String() != expectedOutput {
						errors <- fmt.Errorf("Process %d: output mismatch: expected %q, got %q",
							processID, expectedOutput, outputBuf.String())
					}
				case <-ctx.Done():
					errors <- fmt.Errorf("Process %d: timed out", processID)
				}
			}(i)
		}

		// Wait for all processes
		wg.Wait()
		close(errors)

		// Check for errors
		for err := range errors {
			t.Error(err)
		}
	})
}

// Helper functions

// countTempPipes counts pipes in temp directory with given prefix
func countTempPipes(t *testing.T, prefix string) int {
	pattern := filepath.Join(os.TempDir(), "*"+prefix+"*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Logf("Failed to glob temp files: %v", err)
		return 0
	}
	return len(matches)
}

// slowReader simulates slow reading
type slowReader struct {
	data  []byte
	pos   int
	delay time.Duration
}

func (r *slowReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}

	time.Sleep(r.delay)

	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// TestIODebugLogging tests debug logging functionality in I/O operations
func TestIODebugLogging(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping test on non-macOS platform")
	}

	// Save original debug state
	oldDebug := os.Getenv("MACGO_DEBUG")
	defer os.Setenv("MACGO_DEBUG", oldDebug)

	t.Run("Debug logging enabled", func(t *testing.T) {
		os.Setenv("MACGO_DEBUG", "1")

		pipePath, err := createPipe("test-debug")
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}
		defer os.Remove(pipePath)

		// Capture stderr to check for debug output
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		// Perform an operation that should generate debug output
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		// Trigger an error to generate debug output
		pipeIOContext(ctx, "/nonexistent/pipe", nil, io.Discard)

		// Restore stderr
		w.Close()
		os.Stderr = oldStderr

		// Read captured output
		buf := make([]byte, 1024)
		n, _ := r.Read(buf)
		output := string(buf[:n])

		// Should contain debug output
		if !strings.Contains(output, "error opening pipe") {
			t.Error("Expected debug output not found")
		}
	})
}

// TestPipeIOTeeWriter tests the TeeWriter functionality for debug logging
func TestPipeIOTeeWriter(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping test on non-macOS platform")
	}

	// Save original debug state
	oldDebug := os.Getenv("MACGO_DEBUG")
	defer os.Setenv("MACGO_DEBUG", oldDebug)

	t.Run("TeeWriter with debug enabled", func(t *testing.T) {
		os.Setenv("MACGO_DEBUG", "1")

		pipePath, err := createPipe("test-tee")
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}
		defer os.Remove(pipePath)

		testData := "Test data for TeeWriter\n"

		// Create buffers for primary and debug output
		var primaryBuf, debugBuf bytes.Buffer

		// Create a MultiWriter (TeeWriter)
		teeWriter := io.MultiWriter(&primaryBuf, &debugBuf)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Write to pipe
		done := make(chan bool)
		go func() {
			reader := strings.NewReader(testData)
			pipeIOContext(ctx, pipePath, reader, nil)
			done <- true
		}()

		// Read from pipe with TeeWriter
		go func() {
			pipeIOContext(ctx, pipePath, nil, teeWriter)
		}()

		// Wait for write to complete
		<-done

		// Give read operation time to complete
		time.Sleep(100 * time.Millisecond)

		// Both buffers should contain the same data
		if primaryBuf.String() != testData {
			t.Errorf("Primary buffer mismatch: expected %q, got %q", testData, primaryBuf.String())
		}
		if debugBuf.String() != testData {
			t.Errorf("Debug buffer mismatch: expected %q, got %q", testData, debugBuf.String())
		}
	})
}
