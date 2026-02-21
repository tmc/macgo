package launch

import (
	"bytes"
	"crypto/rand"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// TestServicesLauncher_LargeStdoutOutput verifies that V2 correctly handles
// multi-MB stdout output without truncation or deadlocks.
func TestServicesLauncher_LargeStdoutOutput(t *testing.T) {
	launcher := &ServicesLauncher{
		logger: NewLogger(),
	}

	testCases := []struct {
		name      string
		sizeBytes int64
		chunkSize int
	}{
		{"128KB output", 128 * 1024, 32 * 1024},
		{"1MB output", 1 * 1024 * 1024, 32 * 1024},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test pipe directory
			tmpDir, err := os.MkdirTemp("", "macgo-v2-large-stdout-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			stdoutPipe := filepath.Join(tmpDir, "stdout")

			// Generate test data
			testData := generateTestData(tc.sizeBytes)
			expectedSize := int64(len(testData))

			// Create FIFO (no open yet)
			mkFifo(t, stdoutPipe)

			// Start forwardStdout in a goroutine
			done := make(chan error, 1)
			captured := &bytes.Buffer{}

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			go func() {
				done <- launcher.forwardStdout(stdoutPipe)
			}()

			// Open pipe for writing (blocks until forwardStdout opens for reading)
			f, err := os.OpenFile(stdoutPipe, os.O_WRONLY, 0600)
			if err != nil {
				t.Fatalf("Failed to open stdout pipe: %v", err)
			}

			// Write large data in chunks in a goroutine
			go func() {
				defer f.Close()
				written := 0
				for written < len(testData) {
					end := written + tc.chunkSize
					if end > len(testData) {
						end = len(testData)
					}
					n, err := f.Write(testData[written:end])
					if err != nil {
						t.Logf("Write error: %v", err)
						return
					}
					written += n
					f.Sync()
				}
			}()

			// Capture the output in a goroutine
			captureDone := make(chan struct{})
			go func() {
				io.Copy(captured, r)
				close(captureDone)
			}()

			// Wait for forwardStdout to complete with timeout
			select {
			case err := <-done:
				w.Close()
				os.Stdout = oldStdout
				<-captureDone

				if err != nil {
					t.Errorf("forwardStdout failed: %v", err)
				}

				// Verify all data was captured
				capturedSize := int64(captured.Len())
				if capturedSize != expectedSize {
					t.Errorf("Data truncation detected!\nExpected: %d bytes\nGot: %d bytes\nLoss: %d bytes",
						expectedSize, capturedSize, expectedSize-capturedSize)
				} else {
					t.Logf("Successfully captured all %d bytes", capturedSize)
				}

				// Verify data integrity (first and last chunks)
				capturedData := captured.Bytes()
				if len(capturedData) >= 1024 {
					if !bytes.Equal(testData[:1024], capturedData[:1024]) {
						t.Error("Data corruption detected in first 1KB")
					}
					if !bytes.Equal(testData[len(testData)-1024:], capturedData[len(capturedData)-1024:]) {
						t.Error("Data corruption detected in last 1KB")
					}
				}

			case <-time.After(60 * time.Second):
				w.Close()
				os.Stdout = oldStdout
				t.Fatal("Test timed out - possible deadlock detected")
			}
		})
	}
}

// TestServicesLauncher_LargeStderrOutput verifies that V2 correctly handles
// multi-MB stderr output without truncation or deadlocks.
func TestServicesLauncher_LargeStderrOutput(t *testing.T) {
	launcher := &ServicesLauncher{
		logger: NewLogger(),
	}

	testCases := []struct {
		name      string
		sizeBytes int64
		chunkSize int
	}{
		{"128KB stderr", 128 * 1024, 32 * 1024},
		{"1MB stderr", 1 * 1024 * 1024, 32 * 1024},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test pipe directory
			tmpDir, err := os.MkdirTemp("", "macgo-v2-large-stderr-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			stderrPipe := filepath.Join(tmpDir, "stderr")

			// Generate test data
			testData := generateTestData(tc.sizeBytes)
			expectedSize := int64(len(testData))

			// Create FIFO
			mkFifo(t, stderrPipe)

			// Start forwardStderr in a goroutine
			done := make(chan error, 1)
			captured := &bytes.Buffer{}

			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			go func() {
				done <- launcher.forwardStderr(stderrPipe)
			}()

			// Open pipe for writing (blocks until forwardStderr opens for reading)
			f, err := os.OpenFile(stderrPipe, os.O_WRONLY, 0600)
			if err != nil {
				t.Fatalf("Failed to open stderr pipe: %v", err)
			}

			// Write large data in chunks
			go func() {
				defer f.Close()
				written := 0
				for written < len(testData) {
					end := written + tc.chunkSize
					if end > len(testData) {
						end = len(testData)
					}
					n, err := f.Write(testData[written:end])
					if err != nil {
						t.Logf("Write error: %v", err)
						return
					}
					written += n
					f.Sync()
				}
			}()

			// Capture the output
			captureDone := make(chan struct{})
			go func() {
				io.Copy(captured, r)
				close(captureDone)
			}()

			// Wait for forwardStderr to complete with timeout
			select {
			case err := <-done:
				w.Close()
				os.Stderr = oldStderr
				<-captureDone

				if err != nil {
					t.Errorf("forwardStderr failed: %v", err)
				}

				// Verify all data was captured
				capturedSize := int64(captured.Len())
				if capturedSize != expectedSize {
					t.Errorf("Data truncation detected!\nExpected: %d bytes\nGot: %d bytes\nLoss: %d bytes",
						expectedSize, capturedSize, expectedSize-capturedSize)
				} else {
					t.Logf("Successfully captured all %d bytes", capturedSize)
				}

			case <-time.After(60 * time.Second):
				w.Close()
				os.Stderr = oldStderr
				t.Fatal("Test timed out - possible deadlock detected")
			}
		})
	}
}

// TestServicesLauncher_ConcurrentLargeOutput verifies that V2 correctly handles
// simultaneous large stdout and stderr output without deadlocks.
func TestServicesLauncher_ConcurrentLargeOutput(t *testing.T) {
	launcher := &ServicesLauncher{
		logger: NewLogger(),
	}

	// Test with 1MB on each stream simultaneously
	stdoutSize := int64(1 * 1024 * 1024)
	stderrSize := int64(1 * 1024 * 1024)

	// Create test pipe directory
	tmpDir, err := os.MkdirTemp("", "macgo-v2-concurrent-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	stdoutPipe := filepath.Join(tmpDir, "stdout")
	stderrPipe := filepath.Join(tmpDir, "stderr")

	// Generate test data
	stdoutData := generateTestData(stdoutSize)
	stderrData := generateTestData(stderrSize)

	// Create FIFOs
	mkFifo(t, stdoutPipe)
	mkFifo(t, stderrPipe)

	// Capture stdout
	stdoutCaptured := &bytes.Buffer{}
	oldStdout := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	// Capture stderr
	stderrCaptured := &bytes.Buffer{}
	oldStderr := os.Stderr
	rErr, wErr, _ := os.Pipe()
	os.Stderr = wErr

	// Start both forwarders concurrently
	stdoutDone := make(chan error, 1)
	stderrDone := make(chan error, 1)

	go func() {
		stdoutDone <- launcher.forwardStdout(stdoutPipe)
	}()

	go func() {
		stderrDone <- launcher.forwardStderr(stderrPipe)
	}()

	// Open pipes for writing (blocks until readers ready)
	fOut, err := os.OpenFile(stdoutPipe, os.O_WRONLY, 0600)
	if err != nil {
		t.Fatalf("Failed to open stdout pipe: %v", err)
	}
	fErr, err := os.OpenFile(stderrPipe, os.O_WRONLY, 0600)
	if err != nil {
		t.Fatalf("Failed to create stderr pipe: %v", err)
	}

	// Start capturing
	stdoutCaptureDone := make(chan struct{})
	go func() {
		io.Copy(stdoutCaptured, rOut)
		close(stdoutCaptureDone)
	}()

	stderrCaptureDone := make(chan struct{})
	go func() {
		io.Copy(stderrCaptured, rErr)
		close(stderrCaptureDone)
	}()

	// Write data to both pipes concurrently
	writeStdoutDone := make(chan struct{})
	writeStderrDone := make(chan struct{})

	go func() {
		defer fOut.Close()
		chunkSize := 64 * 1024
		written := 0
		for written < len(stdoutData) {
			end := written + chunkSize
			if end > len(stdoutData) {
				end = len(stdoutData)
			}
			n, _ := fOut.Write(stdoutData[written:end])
			written += n
			fOut.Sync()
		}
		close(writeStdoutDone)
	}()

	go func() {
		defer fErr.Close()
		chunkSize := 64 * 1024
		written := 0
		for written < len(stderrData) {
			end := written + chunkSize
			if end > len(stderrData) {
				end = len(stderrData)
			}
			n, _ := fErr.Write(stderrData[written:end])
			written += n
			fErr.Sync()
		}
		close(writeStderrDone)
	}()

	// Wait for both writes to complete
	<-writeStdoutDone
	<-writeStderrDone

	// Wait for both forwarders to complete with timeout
	testTimeout := time.After(60 * time.Second)
	stdoutComplete := false
	stderrComplete := false

	for !stdoutComplete || !stderrComplete {
		select {
		case err := <-stdoutDone:
			wOut.Close()
			os.Stdout = oldStdout
			<-stdoutCaptureDone

			if err != nil {
				t.Errorf("forwardStdout failed: %v", err)
			}

			stdoutCapturedSize := int64(stdoutCaptured.Len())
			if stdoutCapturedSize != stdoutSize {
				t.Errorf("Stdout truncation!\nExpected: %d bytes\nGot: %d bytes",
					stdoutSize, stdoutCapturedSize)
			}
			stdoutComplete = true

		case err := <-stderrDone:
			wErr.Close()
			os.Stderr = oldStderr
			<-stderrCaptureDone

			if err != nil {
				t.Errorf("forwardStderr failed: %v", err)
			}

			stderrCapturedSize := int64(stderrCaptured.Len())
			if stderrCapturedSize != stderrSize {
				t.Errorf("Stderr truncation!\nExpected: %d bytes\nGot: %d bytes",
					stderrSize, stderrCapturedSize)
			}
			stderrComplete = true

		case <-testTimeout:
			wOut.Close()
			wErr.Close()
			os.Stdout = oldStdout
			os.Stderr = oldStderr
			t.Fatal("Test timed out - possible deadlock with concurrent I/O")
		}
	}

	t.Logf("Successfully handled concurrent I/O: stdout=%d bytes, stderr=%d bytes",
		stdoutCaptured.Len(), stderrCaptured.Len())
}

// TestServicesLauncher_BufferBoundaryConditions tests edge cases around
// the 32KB io.Copy buffer boundary to ensure correct handling.
func TestServicesLauncher_BufferBoundaryConditions(t *testing.T) {
	launcher := &ServicesLauncher{
		logger: NewLogger(),
	}

	bufferSize := 32 * 1024

	testCases := []struct {
		name      string
		sizeBytes int
	}{
		{"exactly 32KB", bufferSize},
		{"32KB + 1 byte", bufferSize + 1},
		{"32KB - 1 byte", bufferSize - 1},
		{"2x buffer size", 2 * bufferSize},
		{"2.5x buffer size", 5 * bufferSize / 2},
		{"3x buffer size", 3 * bufferSize},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "macgo-boundary-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			stdoutPipe := filepath.Join(tmpDir, "stdout")
			testData := generateTestData(int64(tc.sizeBytes))

			mkFifo(t, stdoutPipe)

			done := make(chan error, 1)
			captured := &bytes.Buffer{}

			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			go func() {
				done <- launcher.forwardStdout(stdoutPipe)
			}()

			// Write all data then close to trigger EOF
			f, err := os.OpenFile(stdoutPipe, os.O_WRONLY, 0600)
			if err != nil {
				t.Fatalf("Failed to open FIFO for writing: %v", err)
			}
			go func() {
				defer f.Close()
				f.Write(testData)
			}()

			captureDone := make(chan struct{})
			go func() {
				io.Copy(captured, r)
				close(captureDone)
			}()

			select {
			case err := <-done:
				w.Close()
				os.Stdout = oldStdout
				<-captureDone

				if err != nil {
					t.Errorf("forwardStdout failed: %v", err)
				}

				if !bytes.Equal(testData, captured.Bytes()) {
					t.Errorf("data mismatch at buffer boundary: expected %d bytes, got %d",
						len(testData), captured.Len())
				}

			case <-time.After(30 * time.Second):
				w.Close()
				os.Stdout = oldStdout
				t.Fatal("test timed out")
			}
		})
	}
}

// generateTestData creates test data of specified size with recognizable patterns.
// Uses pseudo-random data with markers every 1KB to aid debugging.
func generateTestData(size int64) []byte {
	data := make([]byte, size)

	// Fill with crypto random data for realistic testing
	if _, err := rand.Read(data); err != nil {
		// Fallback to simple pattern if random fails
		for i := range data {
			data[i] = byte(i % 256)
		}
	}

	// Add markers every 1KB for debugging
	marker := []byte("MARKER")
	for i := int64(0); i < size; i += 1024 {
		if i+int64(len(marker)) < size {
			copy(data[i:], marker)
		}
	}

	return data
}

// mkFifo creates a named pipe (FIFO) at path.
func mkFifo(t *testing.T, path string) {
	if err := syscall.Mkfifo(path, 0600); err != nil {
		t.Fatalf("Failed to create FIFO at %s: %v", path, err)
	}
}
