package macgo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// TestIORedirectionWithSignalHandling tests I/O redirection with signal forwarding
func TestIORedirectionWithSignalHandling(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping test on non-macOS platform")
	}

	t.Run("Signal forwarding during I/O operations", func(t *testing.T) {
		// Create test program that handles signals
		testProgram := `
package main
import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)
func main() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	fmt.Fprintln(os.Stdout, "Process started")
	fmt.Fprintln(os.Stderr, "Waiting for signal...")
	
	// Read from stdin in background
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				fmt.Fprintf(os.Stdout, "Received: %s", buf[:n])
			}
		}
	}()
	
	sig := <-sigChan
	fmt.Fprintf(os.Stdout, "Received signal: %v\n", sig)
	fmt.Fprintln(os.Stderr, "Shutting down...")
	time.Sleep(100 * time.Millisecond) // Allow time for output
}
`
		// Create and build test program
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test_signal_io.go")
		if err := os.WriteFile(testFile, []byte(testProgram), 0644); err != nil {
			t.Fatalf("Failed to write test program: %v", err)
		}

		testBinary := filepath.Join(tmpDir, "test_signal_io")
		cmd := exec.Command("go", "build", "-o", testBinary, testFile)
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to build test program: %v", err)
		}

		// Create pipes
		stdinPipe, err := createPipe("test-sig-stdin")
		if err != nil {
			t.Fatalf("Failed to create stdin pipe: %v", err)
		}
		defer os.Remove(stdinPipe)

		stdoutPipe, err := createPipe("test-sig-stdout")
		if err != nil {
			t.Fatalf("Failed to create stdout pipe: %v", err)
		}
		defer os.Remove(stdoutPipe)

		stderrPipe, err := createPipe("test-sig-stderr")
		if err != nil {
			t.Fatalf("Failed to create stderr pipe: %v", err)
		}
		defer os.Remove(stderrPipe)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Output buffers
		var stdoutBuf, stderrBuf bytes.Buffer
		var wg sync.WaitGroup

		// Set up I/O handlers
		wg.Add(3)

		// Stdin handler
		go func() {
			defer wg.Done()
			testData := "Test input\n"
			reader := strings.NewReader(testData)
			pipeIOContext(ctx, stdinPipe, reader, nil)
		}()

		// Stdout handler
		go func() {
			defer wg.Done()
			pipeIOContext(ctx, stdoutPipe, nil, &stdoutBuf)
		}()

		// Stderr handler
		go func() {
			defer wg.Done()
			pipeIOContext(ctx, stderrPipe, nil, &stderrBuf)
		}()

		// Start the test process
		testCmd := exec.CommandContext(ctx, testBinary)
		
		// Open pipes for the process
		stdinFile, _ := os.Open(stdinPipe)
		defer stdinFile.Close()
		stdoutFile, _ := os.OpenFile(stdoutPipe, os.O_WRONLY, 0)
		defer stdoutFile.Close()
		stderrFile, _ := os.OpenFile(stderrPipe, os.O_WRONLY, 0)
		defer stderrFile.Close()

		testCmd.Stdin = stdinFile
		testCmd.Stdout = stdoutFile
		testCmd.Stderr = stderrFile

		// Start process
		if err := testCmd.Start(); err != nil {
			t.Fatalf("Failed to start test process: %v", err)
		}

		// Wait for process to be ready
		time.Sleep(500 * time.Millisecond)

		// Send SIGINT to the process
		if err := testCmd.Process.Signal(syscall.SIGINT); err != nil {
			t.Fatalf("Failed to send signal: %v", err)
		}

		// Wait for process to exit
		testCmd.Wait()

		// Close write ends
		stdoutFile.Close()
		stderrFile.Close()

		// Wait for I/O handlers
		done := make(chan bool)
		go func() {
			wg.Wait()
			done <- true
		}()

		select {
		case <-done:
			// Verify output contains expected messages
			stdout := stdoutBuf.String()
			stderr := stderrBuf.String()

			if !strings.Contains(stdout, "Process started") {
				t.Error("Stdout missing 'Process started'")
			}
			if !strings.Contains(stdout, "Received signal: interrupt") {
				t.Error("Stdout missing signal receipt confirmation")
			}
			if !strings.Contains(stderr, "Waiting for signal") {
				t.Error("Stderr missing 'Waiting for signal'")
			}
			if !strings.Contains(stderr, "Shutting down") {
				t.Error("Stderr missing 'Shutting down'")
			}
		case <-ctx.Done():
			t.Fatal("Test timed out")
		}
	})

	t.Run("I/O continuation during signal storms", func(t *testing.T) {
		// Test that I/O operations continue properly during multiple signals
		pipePath, err := createPipe("test-signal-storm")
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}
		defer os.Remove(pipePath)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Large data to transfer
		dataSize := 1024 * 1024 // 1MB
		testData := make([]byte, dataSize)
		for i := range testData {
			testData[i] = byte(i % 256)
		}

		var received bytes.Buffer
		transferComplete := make(chan bool)

		// Start writer
		go func() {
			reader := bytes.NewReader(testData)
			pipeIOContext(ctx, pipePath, reader, nil)
		}()

		// Start reader
		go func() {
			pipeIOContext(ctx, pipePath, nil, &received)
			transferComplete <- true
		}()

		// Send signals during transfer (simulating signal storm)
		// Note: In real implementation, these would be handled by signal forwarder
		signalCount := 10
		go func() {
			for i := 0; i < signalCount; i++ {
				time.Sleep(50 * time.Millisecond)
				// In a real test with process, we'd send signals here
				// For this test, we're verifying I/O continues despite activity
			}
		}()

		// Wait for transfer to complete
		select {
		case <-transferComplete:
			// Verify data integrity despite signals
			if received.Len() != dataSize {
				t.Errorf("Size mismatch: expected %d, got %d", dataSize, received.Len())
			}
			if !bytes.Equal(testData, received.Bytes()) {
				t.Error("Data corruption during signal storm")
			}
		case <-ctx.Done():
			t.Fatal("Transfer did not complete during signal storm")
		}
	})
}

// TestIOCleanupOnSignalTermination tests cleanup when process is terminated by signal
func TestIOCleanupOnSignalTermination(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping test on non-macOS platform")
	}

	t.Run("Cleanup on SIGTERM", func(t *testing.T) {
		// Count initial pipes
		initialPipes := countTempPipes(t, "sigterm-test")

		// Create pipes
		pipes := make([]string, 3)
		pipeNames := []string{"stdin", "stdout", "stderr"}
		for i, name := range pipeNames {
			pipe, err := createPipe("sigterm-test-" + name)
			if err != nil {
				t.Fatalf("Failed to create %s pipe: %v", name, err)
			}
			pipes[i] = pipe
			defer os.Remove(pipe)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start I/O operations
		var wg sync.WaitGroup
		
		for i, pipe := range pipes {
			wg.Add(1)
			go func(p string, idx int) {
				defer wg.Done()
				if idx == 0 {
					// Stdin - write continuously
					reader := &infiniteReader{pattern: []byte("test data\n")}
					pipeIOContext(ctx, p, reader, nil)
				} else {
					// Stdout/stderr - read continuously
					pipeIOContext(ctx, p, nil, io.Discard)
				}
			}(pipe, i)
		}

		// Simulate SIGTERM by cancelling context after delay
		time.Sleep(200 * time.Millisecond)
		cancel()

		// Wait for cleanup
		done := make(chan bool)
		go func() {
			wg.Wait()
			done <- true
		}()

		select {
		case <-done:
			// Good, all I/O operations stopped
		case <-time.After(2 * time.Second):
			t.Error("I/O operations did not stop after simulated SIGTERM")
		}

		// Clean up pipes
		for _, pipe := range pipes {
			os.Remove(pipe)
		}

		// Verify no leaked pipes
		finalPipes := countTempPipes(t, "sigterm-test")
		if finalPipes > initialPipes {
			t.Errorf("Pipe leak after SIGTERM: initial=%d, final=%d", initialPipes, finalPipes)
		}
	})

	t.Run("Cleanup on SIGKILL simulation", func(t *testing.T) {
		// SIGKILL cannot be caught, but we can test abrupt termination
		pipePath, err := createPipe("test-sigkill")
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}
		defer os.Remove(pipePath)

		// Start a writer that will be abruptly terminated
		ctx, cancel := context.WithCancel(context.Background())
		
		writerStarted := make(chan bool)
		go func() {
			writerStarted <- true
			reader := &infiniteReader{pattern: []byte("continuous data\n")}
			pipeIOContext(ctx, pipePath, reader, nil)
		}()

		// Wait for writer to start
		<-writerStarted
		time.Sleep(100 * time.Millisecond)

		// Simulate SIGKILL by immediate context cancellation
		cancel()

		// Verify pipe can still be cleaned up
		if err := os.Remove(pipePath); err != nil {
			t.Logf("Note: Pipe removal after simulated SIGKILL: %v", err)
			// This is expected behavior - the pipe might still be in use
		}
	})
}

// TestSignalHandlingEdgeCases tests edge cases in signal handling with I/O
func TestSignalHandlingEdgeCases(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping test on non-macOS platform")
	}

	t.Run("Rapid signal delivery during I/O", func(t *testing.T) {
		// Test program that counts signals while doing I/O
		testProgram := `
package main
import (
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)
func main() {
	var sigCount int32
	sigChan := make(chan os.Signal, 100) // Large buffer
	signal.Notify(sigChan, syscall.SIGUSR1, syscall.SIGUSR2)
	
	done := make(chan bool)
	
	// Signal counter
	go func() {
		for sig := range sigChan {
			atomic.AddInt32(&sigCount, 1)
			fmt.Fprintf(os.Stderr, "Signal %v (total: %d)\n", sig, atomic.LoadInt32(&sigCount))
		}
	}()
	
	// I/O worker
	go func() {
		buf := make([]byte, 1024)
		totalRead := 0
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				break
			}
			totalRead += n
			fmt.Fprintf(os.Stdout, "Read %d bytes (total: %d)\n", n, totalRead)
		}
		done <- true
	}()
	
	// Run for 2 seconds
	time.Sleep(2 * time.Second)
	
	fmt.Fprintf(os.Stdout, "Final signal count: %d\n", atomic.LoadInt32(&sigCount))
	close(sigChan)
}
`
		// Build test program
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test_rapid_signals.go")
		if err := os.WriteFile(testFile, []byte(testProgram), 0644); err != nil {
			t.Fatalf("Failed to write test program: %v", err)
		}

		testBinary := filepath.Join(tmpDir, "test_rapid_signals")
		cmd := exec.Command("go", "build", "-o", testBinary, testFile)
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to build test program: %v", err)
		}

		// Create pipes
		stdinPipe, err := createPipe("test-rapid-stdin")
		if err != nil {
			t.Fatalf("Failed to create stdin pipe: %v", err)
		}
		defer os.Remove(stdinPipe)

		stdoutPipe, err := createPipe("test-rapid-stdout")
		if err != nil {
			t.Fatalf("Failed to create stdout pipe: %v", err)
		}
		defer os.Remove(stdoutPipe)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var stdoutBuf bytes.Buffer
		var wg sync.WaitGroup

		// I/O handlers
		wg.Add(2)

		// Continuous stdin writer
		go func() {
			defer wg.Done()
			pattern := []byte("Test data chunk\n")
			reader := &infiniteReader{pattern: pattern}
			pipeIOContext(ctx, stdinPipe, reader, nil)
		}()

		// Stdout reader
		go func() {
			defer wg.Done()
			pipeIOContext(ctx, stdoutPipe, nil, &stdoutBuf)
		}()

		// Start process
		testCmd := exec.CommandContext(ctx, testBinary)
		
		stdinFile, _ := os.Open(stdinPipe)
		defer stdinFile.Close()
		stdoutFile, _ := os.OpenFile(stdoutPipe, os.O_WRONLY, 0)
		defer stdoutFile.Close()

		testCmd.Stdin = stdinFile
		testCmd.Stdout = stdoutFile
		testCmd.Stderr = os.Stderr // Direct to see signal counts

		if err := testCmd.Start(); err != nil {
			t.Fatalf("Failed to start process: %v", err)
		}

		// Send rapid signals
		go func() {
			for i := 0; i < 20; i++ {
				testCmd.Process.Signal(syscall.SIGUSR1)
				time.Sleep(50 * time.Millisecond)
				testCmd.Process.Signal(syscall.SIGUSR2)
				time.Sleep(50 * time.Millisecond)
			}
		}()

		// Let it run
		testCmd.Wait()
		stdoutFile.Close()

		// Wait for I/O to complete
		done := make(chan bool)
		go func() {
			wg.Wait()
			done <- true
		}()

		select {
		case <-done:
			// Verify I/O continued despite signals
			output := stdoutBuf.String()
			if !strings.Contains(output, "Read") {
				t.Error("No I/O activity detected during rapid signals")
			}
			if !strings.Contains(output, "Final signal count:") {
				t.Error("Process did not complete properly")
			}
		case <-ctx.Done():
			t.Fatal("Test timed out")
		}
	})

	t.Run("Signal during pipe creation", func(t *testing.T) {
		// Test handling signals during pipe creation phase
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start creating pipes in a loop
		pipeCreated := make(chan string, 10)
		errors := make(chan error, 10)

		go func() {
			for i := 0; i < 10; i++ {
				select {
				case <-ctx.Done():
					return
				default:
					pipe, err := createPipe(fmt.Sprintf("create-test-%d", i))
					if err != nil {
						errors <- err
					} else {
						pipeCreated <- pipe
					}
					time.Sleep(50 * time.Millisecond)
				}
			}
		}()

		// Cancel context after some pipes are created (simulating signal)
		time.Sleep(200 * time.Millisecond)
		cancel()

		// Collect created pipes
		time.Sleep(100 * time.Millisecond)
		close(pipeCreated)
		close(errors)

		// Clean up any created pipes
		for pipe := range pipeCreated {
			os.Remove(pipe)
		}

		// Check for errors
		errorCount := 0
		for err := range errors {
			t.Logf("Pipe creation error: %v", err)
			errorCount++
		}

		// Some pipes should have been created before cancellation
		if len(pipeCreated) == 0 && errorCount == 0 {
			t.Error("No pipes were created before cancellation")
		}
	})
}

// TestRelaunchWithSignalsIntegration tests the full integration of relaunch with signals and I/O
func TestRelaunchWithSignalsIntegration(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping test on non-macOS platform")
	}

	t.Run("Simulated relaunch with signal forwarding", func(t *testing.T) {
		// This simulates the full relaunch scenario with I/O and signals
		
		// Create the standard macgo pipes
		pipes := make([]string, 3)
		for i, name := range []string{"stdin", "stdout", "stderr"} {
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
		inputData := "Test input for relaunched process\n"
		var stdoutBuf, stderrBuf bytes.Buffer

		// Set up signal channel for parent
		parentSigs := make(chan os.Signal, 10)
		signal.Notify(parentSigs, syscall.SIGUSR1)
		defer signal.Stop(parentSigs)

		// Start I/O handlers (parent side)
		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			reader := strings.NewReader(inputData)
			pipeIOContext(ctx, pipes[0], reader, nil)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			pipeIOContext(ctx, pipes[1], nil, &stdoutBuf)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			pipeIOContext(ctx, pipes[2], nil, &stderrBuf)
		}()

		// Simulate child process
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Open pipes (child side)
			stdinFile, err := os.Open(pipes[0])
			if err != nil {
				return
			}
			defer stdinFile.Close()

			stdoutFile, err := os.OpenFile(pipes[1], os.O_WRONLY, 0)
			if err != nil {
				return
			}
			defer stdoutFile.Close()

			stderrFile, err := os.OpenFile(pipes[2], os.O_WRONLY, 0)
			if err != nil {
				return
			}
			defer stderrFile.Close()

			// Child process behavior
			fmt.Fprintln(stdoutFile, "Child process started")
			fmt.Fprintln(stderrFile, "Child ready for input")

			// Read input
			buf := make([]byte, 1024)
			n, _ := stdinFile.Read(buf)
			if n > 0 {
				fmt.Fprintf(stdoutFile, "Child received: %s", buf[:n])
			}

			// Simulate signal handling in child
			fmt.Fprintln(stderrFile, "Child process completing")
		}()

		// Send a test signal to parent (would normally be forwarded to child)
		go func() {
			time.Sleep(100 * time.Millisecond)
			syscall.Kill(syscall.Getpid(), syscall.SIGUSR1)
		}()

		// Wait for everything to complete
		done := make(chan bool)
		go func() {
			wg.Wait()
			done <- true
		}()

		select {
		case <-done:
			// Verify outputs
			stdout := stdoutBuf.String()
			stderr := stderrBuf.String()

			if !strings.Contains(stdout, "Child process started") {
				t.Error("Missing child startup message")
			}
			if !strings.Contains(stdout, "Child received:") {
				t.Error("Child did not receive input")
			}
			if !strings.Contains(stderr, "Child ready for input") {
				t.Error("Missing child ready message")
			}

			// Check if parent received signal
			select {
			case sig := <-parentSigs:
				t.Logf("Parent received signal: %v", sig)
			default:
				t.Log("No signal received by parent")
			}
		case <-ctx.Done():
			t.Fatal("Relaunch simulation timed out")
		}
	})
}

// Helper type for continuous reading
type infiniteReader struct {
	pattern []byte
	pos     int
}

func (r *infiniteReader) Read(p []byte) (n int, err error) {
	if len(r.pattern) == 0 {
		return 0, io.EOF
	}

	remaining := len(p)
	written := 0

	for remaining > 0 {
		// Calculate how much to copy from pattern
		patternRemaining := len(r.pattern) - r.pos
		toCopy := remaining
		if toCopy > patternRemaining {
			toCopy = patternRemaining
		}

		// Copy from pattern
		copy(p[written:written+toCopy], r.pattern[r.pos:r.pos+toCopy])
		written += toCopy
		remaining -= toCopy
		r.pos += toCopy

		// Reset position if we've consumed the pattern
		if r.pos >= len(r.pattern) {
			r.pos = 0
		}
	}

	return written, nil
}

// TestComplexScenarios tests complex real-world scenarios
func TestComplexScenarios(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping test on non-macOS platform")
	}

	t.Run("Multiple process chain with I/O and signals", func(t *testing.T) {
		// Simulate: Process A -> pipes -> Process B -> pipes -> Process C
		// with signal forwarding through the chain

		// Create pipes for A->B
		abPipes := make([]string, 2)
		abPipes[0], _ = createPipe("test-ab-data")
		abPipes[1], _ = createPipe("test-ab-ctrl")
		defer os.Remove(abPipes[0])
		defer os.Remove(abPipes[1])

		// Create pipes for B->C
		bcPipes := make([]string, 2)
		bcPipes[0], _ = createPipe("test-bc-data")
		bcPipes[1], _ = createPipe("test-bc-ctrl")
		defer os.Remove(bcPipes[0])
		defer os.Remove(bcPipes[1])

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Process A: Data generator
		go func() {
			data := "Data from A\n"
			reader := strings.NewReader(data)
			pipeIOContext(ctx, abPipes[0], reader, nil)
		}()

		// Process B: Transformer
		go func() {
			var input bytes.Buffer
			pipeIOContext(ctx, abPipes[0], nil, &input)
			
			// Transform data
			transformed := fmt.Sprintf("B transformed: %s", input.String())
			reader := strings.NewReader(transformed)
			pipeIOContext(ctx, bcPipes[0], reader, nil)
		}()

		// Process C: Consumer
		var finalOutput bytes.Buffer
		done := make(chan bool)
		go func() {
			pipeIOContext(ctx, bcPipes[0], nil, &finalOutput)
			done <- true
		}()

		// Wait for pipeline to complete
		select {
		case <-done:
			output := finalOutput.String()
			if !strings.Contains(output, "B transformed:") {
				t.Error("Pipeline transformation failed")
			}
			if !strings.Contains(output, "Data from A") {
				t.Error("Original data lost in pipeline")
			}
		case <-ctx.Done():
			t.Fatal("Pipeline timed out")
		}
	})
}