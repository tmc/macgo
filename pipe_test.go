package macgo

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestCreatePipeDetailed tests the createPipe function
func TestCreatePipeDetailed(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping test on non-macOS platform")
	}

	t.Run("Basic pipe creation", func(t *testing.T) {
		pipePath, err := createPipe("test-pipe")
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}
		defer os.Remove(pipePath)

		// Verify the pipe exists
		info, err := os.Stat(pipePath)
		if err != nil {
			t.Fatalf("Failed to stat pipe: %v", err)
		}

		// Verify it's a named pipe (FIFO)
		if info.Mode()&os.ModeNamedPipe == 0 {
			t.Error("Created file is not a named pipe")
		}

		// Verify the path contains the prefix
		if !strings.Contains(filepath.Base(pipePath), "test-pipe") {
			t.Errorf("Pipe path doesn't contain prefix: %s", pipePath)
		}

		// Verify it's in the temp directory
		if !strings.HasPrefix(pipePath, os.TempDir()) {
			t.Errorf("Pipe not created in temp directory: %s", pipePath)
		}
	})

	t.Run("Multiple pipes with same prefix", func(t *testing.T) {
		pipes := make([]string, 5)
		for i := range pipes {
			pipe, err := createPipe("multi-test")
			if err != nil {
				t.Fatalf("Failed to create pipe %d: %v", i, err)
			}
			pipes[i] = pipe
		}

		// Clean up
		for _, pipe := range pipes {
			defer os.Remove(pipe)
		}

		// Verify all pipes are unique
		seen := make(map[string]bool)
		for _, pipe := range pipes {
			if seen[pipe] {
				t.Errorf("Duplicate pipe path: %s", pipe)
			}
			seen[pipe] = true
		}
	})

	t.Run("Empty prefix", func(t *testing.T) {
		pipePath, err := createPipe("")
		if err != nil {
			t.Fatalf("Failed to create pipe with empty prefix: %v", err)
		}
		defer os.Remove(pipePath)

		// Should still create a valid pipe
		info, err := os.Stat(pipePath)
		if err != nil {
			t.Fatalf("Failed to stat pipe: %v", err)
		}

		if info.Mode()&os.ModeNamedPipe == 0 {
			t.Error("Created file is not a named pipe")
		}
	})

	t.Run("Special characters in prefix", func(t *testing.T) {
		specialPrefixes := []string{
			"test-with-spaces ",
			"test_underscore",
			"test.dot",
			"test-123",
		}

		for _, prefix := range specialPrefixes {
			pipePath, err := createPipe(prefix)
			if err != nil {
				t.Errorf("Failed to create pipe with prefix %q: %v", prefix, err)
				continue
			}
			defer os.Remove(pipePath)

			// Verify pipe was created
			if _, err := os.Stat(pipePath); err != nil {
				t.Errorf("Pipe with prefix %q doesn't exist: %v", prefix, err)
			}
		}
	})
}

// TestPipeIO tests the pipeIO function
func TestPipeIO(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping test on non-macOS platform")
	}

	t.Run("Write to pipe", func(t *testing.T) {
		pipePath, err := createPipe("test-write")
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}
		defer os.Remove(pipePath)

		testData := []byte("Hello from pipeIO test")
		reader := bytes.NewReader(testData)

		// Start reading from the pipe in a goroutine
		readDone := make(chan []byte)
		go func() {
			data, err := os.ReadFile(pipePath)
			if err != nil {
				// Try opening and reading manually
				f, err := os.Open(pipePath)
				if err != nil {
					t.Logf("Failed to open pipe for reading: %v", err)
					readDone <- nil
					return
				}
				defer f.Close()
				
				buf := make([]byte, len(testData))
				n, err := f.Read(buf)
				if err != nil && err != io.EOF {
					t.Logf("Failed to read from pipe: %v", err)
					readDone <- nil
					return
				}
				readDone <- buf[:n]
			} else {
				readDone <- data
			}
		}()

		// Write to the pipe
		go pipeIO(pipePath, reader, nil)

		// Wait for data with timeout
		select {
		case data := <-readDone:
			if data != nil && !bytes.Equal(data, testData) {
				t.Errorf("Expected %q, got %q", testData, data)
			}
		case <-time.After(2 * time.Second):
			t.Error("Timeout waiting for pipe data")
		}
	})

	t.Run("Read from pipe", func(t *testing.T) {
		pipePath, err := createPipe("test-read")
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}
		defer os.Remove(pipePath)

		testData := []byte("Hello from pipe reader")
		var output bytes.Buffer

		// Start pipeIO reader in a goroutine
		readDone := make(chan bool)
		go func() {
			pipeIO(pipePath, nil, &output)
			readDone <- true
		}()

		// Give reader time to set up
		time.Sleep(100 * time.Millisecond)

		// Write to the pipe
		f, err := os.OpenFile(pipePath, os.O_WRONLY, 0)
		if err != nil {
			t.Fatalf("Failed to open pipe for writing: %v", err)
		}
		
		_, err = f.Write(testData)
		f.Close()
		if err != nil {
			t.Fatalf("Failed to write to pipe: %v", err)
		}

		// Wait for reader to finish
		select {
		case <-readDone:
			if !bytes.Equal(output.Bytes(), testData) {
				t.Errorf("Expected %q, got %q", testData, output.Bytes())
			}
		case <-time.After(2 * time.Second):
			t.Error("Timeout waiting for pipe reader")
		}
	})
}

// TestPipeIOContextFunction tests the pipeIOContext function
func TestPipeIOContextFunction(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping test on non-macOS platform")
	}

	t.Run("Context cancellation during write", func(t *testing.T) {
		pipePath, err := createPipe("test-ctx-write")
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}
		defer os.Remove(pipePath)

		// Create a context that we'll cancel
		ctx, cancel := context.WithCancel(context.Background())

		// Create a reader that blocks
		blockingReader := &blockingReader{
			data: make([]byte, 1024*1024), // 1MB of data
		}

		// Start pipeIOContext in a goroutine
		done := make(chan bool)
		go func() {
			pipeIOContext(ctx, pipePath, blockingReader, nil)
			done <- true
		}()

		// Cancel the context after a short delay
		time.Sleep(100 * time.Millisecond)
		cancel()

		// The function should exit quickly after cancellation
		select {
		case <-done:
			// Good, it exited
		case <-time.After(1 * time.Second):
			t.Error("pipeIOContext did not respect context cancellation")
		}
	})

	t.Run("Context cancellation during read", func(t *testing.T) {
		pipePath, err := createPipe("test-ctx-read")
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}
		defer os.Remove(pipePath)

		// Create a context that we'll cancel
		ctx, cancel := context.WithCancel(context.Background())

		var output bytes.Buffer

		// Start pipeIOContext reader in a goroutine
		done := make(chan bool)
		go func() {
			pipeIOContext(ctx, pipePath, nil, &output)
			done <- true
		}()

		// Cancel the context after a short delay
		time.Sleep(100 * time.Millisecond)
		cancel()

		// The function should exit quickly after cancellation
		select {
		case <-done:
			// Good, it exited
		case <-time.After(1 * time.Second):
			t.Error("pipeIOContext did not respect context cancellation during read")
		}
	})

	t.Run("Normal completion with context", func(t *testing.T) {
		pipePath, err := createPipe("test-ctx-normal")
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}
		defer os.Remove(pipePath)

		ctx := context.Background()
		testData := []byte("Normal completion test")
		reader := bytes.NewReader(testData)

		// Reader goroutine
		readDone := make(chan []byte)
		go func() {
			f, err := os.Open(pipePath)
			if err != nil {
				readDone <- nil
				return
			}
			defer f.Close()
			
			buf := make([]byte, len(testData))
			n, _ := f.Read(buf)
			readDone <- buf[:n]
		}()

		// Writer goroutine
		go pipeIOContext(ctx, pipePath, reader, nil)

		// Should complete normally
		select {
		case data := <-readDone:
			if data != nil && !bytes.Equal(data, testData) {
				t.Errorf("Expected %q, got %q", testData, data)
			}
		case <-time.After(2 * time.Second):
			t.Error("Normal completion timed out")
		}
	})
}

// TestPipeIOConcurrency tests concurrent pipe operations
func TestPipeIOConcurrency(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping test on non-macOS platform")
	}

	t.Run("Multiple writers", func(t *testing.T) {
		pipePath, err := createPipe("test-concurrent-write")
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}
		defer os.Remove(pipePath)

		// Note: Multiple writers to the same pipe can cause issues
		// This test documents that behavior
		t.Log("Multiple concurrent writers to the same pipe may block or fail")
	})

	t.Run("Multiple readers", func(t *testing.T) {
		pipePath, err := createPipe("test-concurrent-read")
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}
		defer os.Remove(pipePath)

		// Note: Multiple readers from the same pipe can cause issues
		// This test documents that behavior
		t.Log("Multiple concurrent readers from the same pipe may receive partial data")
	})

	t.Run("Separate pipes", func(t *testing.T) {
		// This is the recommended pattern - each connection gets its own pipe
		numPipes := 5
		var wg sync.WaitGroup

		for i := 0; i < numPipes; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				pipePath, err := createPipe("test-separate")
				if err != nil {
					t.Errorf("Failed to create pipe %d: %v", idx, err)
					return
				}
				defer os.Remove(pipePath)

				// Each pipe can be used independently
				testData := []byte("Data for pipe " + string(rune('0'+idx)))
				reader := bytes.NewReader(testData)

				// This should work without interference
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
				defer cancel()

				done := make(chan bool)
				go func() {
					pipeIOContext(ctx, pipePath, reader, nil)
					done <- true
				}()

				select {
				case <-done:
					// Success
				case <-ctx.Done():
					t.Errorf("Pipe %d operation timed out", idx)
				}
			}(i)
		}

		wg.Wait()
	})
}

// TestPipeIOErrorHandling tests error handling in pipe operations
func TestPipeIOErrorHandling(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping test on non-macOS platform")
	}

	t.Run("Non-existent pipe", func(t *testing.T) {
		nonExistentPipe := "/tmp/non-existent-pipe-12345"
		
		// Ensure it doesn't exist
		os.Remove(nonExistentPipe)

		var output bytes.Buffer
		
		// This should handle the error gracefully
		done := make(chan bool)
		go func() {
			pipeIO(nonExistentPipe, nil, &output)
			done <- true
		}()

		select {
		case <-done:
			// Should complete quickly due to error
		case <-time.After(1 * time.Second):
			t.Error("pipeIO didn't handle non-existent pipe error")
		}
	})

	t.Run("Permission denied", func(t *testing.T) {
		// Create a pipe and change permissions
		pipePath, err := createPipe("test-perm")
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}
		defer os.Remove(pipePath)

		// Make it unreadable
		if err := os.Chmod(pipePath, 0000); err != nil {
			t.Fatalf("Failed to change permissions: %v", err)
		}
		
		// Restore permissions for cleanup
		defer os.Chmod(pipePath, 0644)

		var output bytes.Buffer

		// This should handle the permission error gracefully
		done := make(chan bool)
		go func() {
			pipeIO(pipePath, nil, &output)
			done <- true
		}()

		select {
		case <-done:
			// Should complete quickly due to error
		case <-time.After(1 * time.Second):
			t.Error("pipeIO didn't handle permission error")
		}
	})
}

// TestPipeCleanup tests that pipes are properly cleaned up
func TestPipeCleanup(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping test on non-macOS platform")
	}

	// Get initial temp directory state
	tempDir := os.TempDir()
	_, err := filepath.Glob(filepath.Join(tempDir, "*-pipe-*"))
	if err != nil {
		t.Fatalf("Failed to list temp files: %v", err)
	}

	// Create and remove several pipes
	for i := 0; i < 10; i++ {
		pipePath, err := createPipe("cleanup-test")
		if err != nil {
			t.Fatalf("Failed to create pipe %d: %v", i, err)
		}
		
		// Use the pipe briefly
		go func() {
			f, _ := os.OpenFile(pipePath, os.O_WRONLY, 0)
			if f != nil {
				f.Close()
			}
		}()

		// Clean up
		os.Remove(pipePath)
	}

	// Check final state
	finalFiles, err := filepath.Glob(filepath.Join(tempDir, "*cleanup-test*"))
	if err != nil {
		t.Fatalf("Failed to list temp files: %v", err)
	}

	// There should be no leftover cleanup-test pipes
	if len(finalFiles) > 0 {
		t.Errorf("Found %d leftover pipe files: %v", len(finalFiles), finalFiles)
	}
}

// TestPipePerformance tests performance characteristics
func TestPipePerformance(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping test on non-macOS platform")
	}

	t.Run("Large data transfer", func(t *testing.T) {
		pipePath, err := createPipe("test-perf")
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}
		defer os.Remove(pipePath)

		// Create 1MB of test data
		testData := make([]byte, 1024*1024)
		for i := range testData {
			testData[i] = byte(i % 256)
		}
		reader := bytes.NewReader(testData)

		var output bytes.Buffer
		done := make(chan bool)

		// Reader
		go func() {
			f, err := os.Open(pipePath)
			if err != nil {
				done <- false
				return
			}
			defer f.Close()
			
			io.Copy(&output, f)
			done <- true
		}()

		// Writer
		start := time.Now()
		go pipeIO(pipePath, reader, nil)

		// Wait for completion
		select {
		case success := <-done:
			elapsed := time.Since(start)
			if success {
				t.Logf("Transferred 1MB in %v", elapsed)
				if output.Len() != len(testData) {
					t.Errorf("Expected %d bytes, got %d", len(testData), output.Len())
				}
			}
		case <-time.After(10 * time.Second):
			t.Error("Large data transfer timed out")
		}
	})
}

// Helper type for testing
type blockingReader struct {
	data []byte
	pos  int
	mu   sync.Mutex
}

func (b *blockingReader) Read(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Simulate slow reading
	time.Sleep(10 * time.Millisecond)

	if b.pos >= len(b.data) {
		return 0, io.EOF
	}

	n = copy(p, b.data[b.pos:])
	b.pos += n
	return n, nil
}

// BenchmarkCreatePipe benchmarks pipe creation
func BenchmarkCreatePipe(b *testing.B) {
	if runtime.GOOS != "darwin" {
		b.Skip("Skipping benchmark on non-macOS platform")
	}

	pipes := make([]string, 0, b.N)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pipe, err := createPipe("bench")
		if err != nil {
			b.Fatalf("Failed to create pipe: %v", err)
		}
		pipes = append(pipes, pipe)
	}
	b.StopTimer()

	// Cleanup
	for _, pipe := range pipes {
		os.Remove(pipe)
	}
}

// BenchmarkPipeIO benchmarks pipe I/O operations
func BenchmarkPipeIO(b *testing.B) {
	if runtime.GOOS != "darwin" {
		b.Skip("Skipping benchmark on non-macOS platform")
	}

	pipePath, err := createPipe("bench-io")
	if err != nil {
		b.Fatalf("Failed to create pipe: %v", err)
	}
	defer os.Remove(pipePath)

	testData := []byte("Benchmark test data for pipe I/O operations")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(testData)
		
		// Reader goroutine
		done := make(chan bool)
		go func() {
			f, _ := os.Open(pipePath)
			if f != nil {
				io.Copy(io.Discard, f)
				f.Close()
			}
			done <- true
		}()

		// Writer
		go func() {
			f, _ := os.OpenFile(pipePath, os.O_WRONLY, 0)
			if f != nil {
				io.Copy(f, reader)
				f.Close()
			}
		}()

		<-done
	}
}

// TestPipeIODebugging tests debug logging in pipe operations
func TestPipeIODebugging(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping test on non-macOS platform")
	}

	// Save original debug state
	oldDebug := os.Getenv("MACGO_DEBUG")
	defer os.Setenv("MACGO_DEBUG", oldDebug)

	// Enable debug
	os.Setenv("MACGO_DEBUG", "1")

	t.Run("Debug output on error", func(t *testing.T) {
		// Use a non-existent pipe to trigger error logging
		nonExistentPipe := "/tmp/debug-test-pipe-99999"
		os.Remove(nonExistentPipe) // Ensure it doesn't exist

		// This should log debug output about the error
		done := make(chan bool)
		go func() {
			pipeIO(nonExistentPipe, nil, io.Discard)
			done <- true
		}()

		select {
		case <-done:
			// Expected to complete with error
		case <-time.After(1 * time.Second):
			t.Error("Debug test timed out")
		}
	})
}