package launch

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"
)

// TestIOBaseline_FIFOStdout tests stdout forwarding via FIFO.
// Baseline: measure current behavior before refactoring.
func TestIOBaseline_FIFOStdout(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "macgo-io-baseline-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	stdoutPipe := filepath.Join(tmpDir, "stdout")
	if err := syscall.Mkfifo(stdoutPipe, 0600); err != nil {
		t.Fatalf("Failed to create FIFO: %v", err)
	}

	testData := "line1\nline2\nline3\n"
	var captured bytes.Buffer

	// Writer goroutine (simulates child process)
	writerDone := make(chan error, 1)
	go func() {
		// Small delay to ensure reader is ready
		time.Sleep(50 * time.Millisecond)
		w, err := os.OpenFile(stdoutPipe, os.O_WRONLY, 0)
		if err != nil {
			writerDone <- fmt.Errorf("open fifo for write: %w", err)
			return
		}
		defer w.Close()
		_, err = w.WriteString(testData)
		writerDone <- err
	}()

	// Reader (simulates parent process forwarding)
	start := time.Now()
	r, err := os.OpenFile(stdoutPipe, os.O_RDONLY, 0)
	if err != nil {
		t.Fatalf("Failed to open FIFO for read: %v", err)
	}
	_, err = io.Copy(&captured, r)
	r.Close()
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("io.Copy failed: %v", err)
	}

	if err := <-writerDone; err != nil {
		t.Fatalf("Writer failed: %v", err)
	}

	if captured.String() != testData {
		t.Errorf("Data mismatch: got %q, want %q", captured.String(), testData)
	}

	t.Logf("FIFO stdout: %d bytes in %v", len(testData), elapsed)
}

// TestIOBaseline_FIFOStdin tests stdin forwarding via FIFO.
func TestIOBaseline_FIFOStdin(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "macgo-io-baseline-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	stdinPipe := filepath.Join(tmpDir, "stdin")
	if err := syscall.Mkfifo(stdinPipe, 0600); err != nil {
		t.Fatalf("Failed to create FIFO: %v", err)
	}

	testData := "input line 1\ninput line 2\n"
	var captured bytes.Buffer

	// Reader goroutine (simulates child process reading stdin)
	readerDone := make(chan error, 1)
	go func() {
		r, err := os.OpenFile(stdinPipe, os.O_RDONLY, 0)
		if err != nil {
			readerDone <- fmt.Errorf("open fifo for read: %w", err)
			return
		}
		defer r.Close()
		_, err = io.Copy(&captured, r)
		readerDone <- err
	}()

	// Writer (simulates parent forwarding stdin)
	start := time.Now()

	// Open with O_NONBLOCK to avoid blocking if reader not ready
	var w *os.File
	for i := 0; i < 50; i++ {
		w, err = os.OpenFile(stdinPipe, os.O_WRONLY|syscall.O_NONBLOCK, 0)
		if err == nil {
			break
		}
		if pathErr, ok := err.(*os.PathError); ok {
			if errno, ok := pathErr.Err.(syscall.Errno); ok && errno == syscall.ENXIO {
				time.Sleep(10 * time.Millisecond)
				continue
			}
		}
		t.Fatalf("Failed to open FIFO for write: %v", err)
	}
	if w == nil {
		t.Fatalf("Timed out waiting for FIFO reader")
	}

	// Switch to blocking mode
	syscall.SetNonblock(int(w.Fd()), false)

	_, err = w.WriteString(testData)
	w.Close()
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if err := <-readerDone; err != nil {
		t.Fatalf("Reader failed: %v", err)
	}

	if captured.String() != testData {
		t.Errorf("Data mismatch: got %q, want %q", captured.String(), testData)
	}

	t.Logf("FIFO stdin: %d bytes in %v", len(testData), elapsed)
}

// TestIOBaseline_LargeData tests forwarding larger amounts of data.
func TestIOBaseline_LargeData(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "macgo-io-baseline-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	stdoutPipe := filepath.Join(tmpDir, "stdout")
	if err := syscall.Mkfifo(stdoutPipe, 0600); err != nil {
		t.Fatalf("Failed to create FIFO: %v", err)
	}

	// Generate 1MB of test data
	dataSize := 1024 * 1024
	testData := make([]byte, dataSize)
	for i := range testData {
		testData[i] = byte('A' + (i % 26))
	}

	var captured bytes.Buffer

	// Writer goroutine
	writerDone := make(chan error, 1)
	go func() {
		time.Sleep(50 * time.Millisecond)
		w, err := os.OpenFile(stdoutPipe, os.O_WRONLY, 0)
		if err != nil {
			writerDone <- err
			return
		}
		defer w.Close()
		_, err = w.Write(testData)
		writerDone <- err
	}()

	// Reader
	start := time.Now()
	r, err := os.OpenFile(stdoutPipe, os.O_RDONLY, 0)
	if err != nil {
		t.Fatalf("Failed to open FIFO: %v", err)
	}
	n, err := io.Copy(&captured, r)
	r.Close()
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("io.Copy failed: %v", err)
	}

	if err := <-writerDone; err != nil {
		t.Fatalf("Writer failed: %v", err)
	}

	if int(n) != dataSize {
		t.Errorf("Size mismatch: got %d, want %d", n, dataSize)
	}

	throughput := float64(dataSize) / elapsed.Seconds() / 1024 / 1024
	t.Logf("Large data: %d bytes in %v (%.1f MB/s)", dataSize, elapsed, throughput)
}

// TestIOBaseline_Bidirectional tests simultaneous stdin/stdout forwarding.
func TestIOBaseline_Bidirectional(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "macgo-io-baseline-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	stdinPipe := filepath.Join(tmpDir, "stdin")
	stdoutPipe := filepath.Join(tmpDir, "stdout")

	if err := syscall.Mkfifo(stdinPipe, 0600); err != nil {
		t.Fatalf("Failed to create stdin FIFO: %v", err)
	}
	if err := syscall.Mkfifo(stdoutPipe, 0600); err != nil {
		t.Fatalf("Failed to create stdout FIFO: %v", err)
	}

	stdinData := "request data\n"
	stdoutData := "response data\n"

	var wg sync.WaitGroup
	var stdinCaptured, stdoutCaptured bytes.Buffer
	var stdinErr, stdoutErr error

	// Simulate child process: read stdin, write stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Read from stdin pipe
		r, err := os.OpenFile(stdinPipe, os.O_RDONLY, 0)
		if err != nil {
			stdinErr = err
			return
		}
		io.Copy(&stdinCaptured, r)
		r.Close()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(50 * time.Millisecond)
		// Write to stdout pipe
		w, err := os.OpenFile(stdoutPipe, os.O_WRONLY, 0)
		if err != nil {
			stdoutErr = err
			return
		}
		w.WriteString(stdoutData)
		w.Close()
	}()

	// Simulate parent: write to stdin, read from stdout
	start := time.Now()

	// Write stdin (with retry for ENXIO)
	var stdinW *os.File
	for i := 0; i < 50; i++ {
		stdinW, err = os.OpenFile(stdinPipe, os.O_WRONLY|syscall.O_NONBLOCK, 0)
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if stdinW == nil {
		t.Fatalf("Failed to open stdin pipe: %v", err)
	}
	syscall.SetNonblock(int(stdinW.Fd()), false)
	stdinW.WriteString(stdinData)
	stdinW.Close()

	// Read stdout
	stdoutR, err := os.OpenFile(stdoutPipe, os.O_RDONLY, 0)
	if err != nil {
		t.Fatalf("Failed to open stdout pipe: %v", err)
	}
	io.Copy(&stdoutCaptured, stdoutR)
	stdoutR.Close()

	elapsed := time.Since(start)
	wg.Wait()

	if stdinErr != nil {
		t.Errorf("Stdin error: %v", stdinErr)
	}
	if stdoutErr != nil {
		t.Errorf("Stdout error: %v", stdoutErr)
	}
	if stdinCaptured.String() != stdinData {
		t.Errorf("Stdin mismatch: got %q, want %q", stdinCaptured.String(), stdinData)
	}
	if stdoutCaptured.String() != stdoutData {
		t.Errorf("Stdout mismatch: got %q, want %q", stdoutCaptured.String(), stdoutData)
	}

	t.Logf("Bidirectional: completed in %v", elapsed)
}

// TestIOBaseline_ContextCancellation tests that forwarding respects context cancellation.
func TestIOBaseline_ContextCancellation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "macgo-io-baseline-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	stdinPipe := filepath.Join(tmpDir, "stdin")
	if err := syscall.Mkfifo(stdinPipe, 0600); err != nil {
		t.Fatalf("Failed to create FIFO: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Start reader that will block (no writer)
	done := make(chan error, 1)
	go func() {
		// This should block until context is cancelled
		r, err := os.OpenFile(stdinPipe, os.O_RDONLY, 0)
		if err != nil {
			done <- err
			return
		}
		defer r.Close()

		// Use a goroutine so we can check context
		readDone := make(chan struct{})
		go func() {
			io.Copy(io.Discard, r)
			close(readDone)
		}()

		select {
		case <-ctx.Done():
			done <- ctx.Err()
		case <-readDone:
			done <- nil
		}
	}()

	// Start writer to unblock the reader open
	go func() {
		time.Sleep(20 * time.Millisecond)
		w, _ := os.OpenFile(stdinPipe, os.O_WRONLY|syscall.O_NONBLOCK, 0)
		if w != nil {
			// Keep open but don't write - simulates hanging input
			<-ctx.Done()
			w.Close()
		}
	}()

	start := time.Now()
	err = <-done
	elapsed := time.Since(start)

	if err != context.DeadlineExceeded {
		t.Logf("Got error: %v (expected context.DeadlineExceeded)", err)
	}

	if elapsed > 200*time.Millisecond {
		t.Errorf("Cancellation took too long: %v", elapsed)
	}

	t.Logf("Context cancellation: completed in %v", elapsed)
}

// TestIOBaseline_ServicesForwardStdout tests the actual ServicesLauncher.forwardStdout
func TestIOBaseline_ServicesForwardStdout(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "macgo-io-services-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	stdoutPipe := filepath.Join(tmpDir, "stdout")
	if err := syscall.Mkfifo(stdoutPipe, 0600); err != nil {
		t.Fatalf("Failed to create FIFO: %v", err)
	}

	testData := "test output from child\n"

	// Writer goroutine (simulates child process)
	writerDone := make(chan error, 1)
	go func() {
		time.Sleep(100 * time.Millisecond)
		w, err := os.OpenFile(stdoutPipe, os.O_WRONLY, 0)
		if err != nil {
			writerDone <- err
			return
		}
		defer w.Close()
		_, err = w.WriteString(testData)
		writerDone <- err
	}()

	// Use the actual launcher
	launcher := &ServicesLauncher{
		logger: NewLogger(),
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	start := time.Now()
	done := make(chan error, 1)
	go func() {
		done <- launcher.forwardStdout(stdoutPipe)
	}()

	// Wait for writer to complete
	if err := <-writerDone; err != nil {
		t.Fatalf("Writer failed: %v", err)
	}

	// Wait for forwarder (should complete after writer closes FIFO)
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("forwardStdout failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("forwardStdout timed out")
	}

	elapsed := time.Since(start)

	// Restore stdout and read captured output
	w.Close()
	os.Stdout = oldStdout
	var captured bytes.Buffer
	io.Copy(&captured, r)

	if captured.String() != testData {
		t.Errorf("Data mismatch: got %q, want %q", captured.String(), testData)
	}

	t.Logf("ServicesLauncher.forwardStdout: %d bytes in %v", len(testData), elapsed)
}

// BenchmarkIO_PureGo benchmarks pure Go io.Copy via FIFO
func BenchmarkIO_PureGo(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "macgo-io-bench-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	stdoutPipe := filepath.Join(tmpDir, "stdout")
	if err := syscall.Mkfifo(stdoutPipe, 0600); err != nil {
		b.Fatalf("Failed to create FIFO: %v", err)
	}

	// 64KB test data
	testData := make([]byte, 64*1024)
	for i := range testData {
		testData[i] = byte('A' + (i % 26))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Writer goroutine
		go func() {
			w, _ := os.OpenFile(stdoutPipe, os.O_WRONLY, 0)
			if w != nil {
				w.Write(testData)
				w.Close()
			}
		}()

		// Reader
		r, _ := os.OpenFile(stdoutPipe, os.O_RDONLY, 0)
		io.Copy(io.Discard, r)
		r.Close()
	}
	b.SetBytes(int64(len(testData)))
}

// BenchmarkIO_CatProcess benchmarks using cat subprocess
func BenchmarkIO_CatProcess(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "macgo-io-bench-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	stdoutPipe := filepath.Join(tmpDir, "stdout")
	if err := syscall.Mkfifo(stdoutPipe, 0600); err != nil {
		b.Fatalf("Failed to create FIFO: %v", err)
	}

	// 64KB test data
	testData := make([]byte, 64*1024)
	for i := range testData {
		testData[i] = byte('A' + (i % 26))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Writer goroutine
		go func() {
			w, _ := os.OpenFile(stdoutPipe, os.O_WRONLY, 0)
			if w != nil {
				w.Write(testData)
				w.Close()
			}
		}()

		// Reader via cat
		cmd := exec.Command("cat", "-u", stdoutPipe)
		cmd.Stdout = io.Discard
		cmd.Run()
	}
	b.SetBytes(int64(len(testData)))
}
