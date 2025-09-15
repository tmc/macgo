package macgo

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// BenchmarkAppLaunch benchmarks the full app launch process
func BenchmarkAppLaunch(b *testing.B) {
	if runtime.GOOS != "darwin" {
		b.Skip("App launch benchmarks only supported on macOS")
	}

	testExecPath := createTestExecutable(b)
	defer os.Remove(testExecPath)

	origConfig := DefaultConfig

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Create fresh config for each iteration
		DefaultConfig = &Config{
			Relaunch:     false, // We'll manually test launch
			Entitlements: make(map[Entitlement]bool),
			PlistEntries: make(map[string]any),
			AutoSign:     false,
		}
		DefaultConfig.AddEntitlement(EntAppSandbox)

		bundlePath := fmt.Sprintf("/tmp/macgo-bench-launch-%d-%d.app", os.Getpid(), i)
		DefaultConfig.CustomDestinationAppPath = bundlePath

		// Create bundle
		appPath, err := createBundle(testExecPath)
		if err != nil {
			b.Fatalf("Failed to create bundle: %v", err)
		}

		// Benchmark the launch process
		startTime := time.Now()
		cmd := exec.Command("open", "-a", appPath, "--wait-apps")
		cmd.Env = append(os.Environ(), "MACGO_NO_RELAUNCH=1")

		err = cmd.Run()
		launchTime := time.Since(startTime)

		if err != nil {
			b.Logf("Launch failed (expected in CI): %v", err)
		} else {
			b.ReportMetric(float64(launchTime.Nanoseconds()), "ns/launch")
		}

		// Clean up
		os.RemoveAll(appPath)
	}

	DefaultConfig = origConfig
}

// BenchmarkSignalHandling benchmarks signal handling performance
func BenchmarkSignalHandling(b *testing.B) {
	// This benchmark tests the signal forwarding mechanism
	testExecPath := createTestExecutable(b)
	defer os.Remove(testExecPath)

	origConfig := DefaultConfig

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		DefaultConfig = &Config{
			Relaunch:     false,
			Entitlements: make(map[Entitlement]bool),
			PlistEntries: make(map[string]any),
			AutoSign:     false,
		}

		bundlePath := fmt.Sprintf("/tmp/macgo-bench-signal-%d-%d.app", os.Getpid(), i)
		DefaultConfig.CustomDestinationAppPath = bundlePath

		appPath, err := createBundle(testExecPath)
		if err != nil {
			b.Fatalf("Failed to create bundle: %v", err)
		}

		// Simulate signal handling setup
		startTime := time.Now()

		// Create a mock process for signal forwarding
		cmd := exec.Command("sleep", "0.1")
		if err := cmd.Start(); err != nil {
			b.Fatalf("Failed to start mock process: %v", err)
		}

		// Test signal forwarding setup time
		forwardSignals(cmd.Process.Pid)

		signalTime := time.Since(startTime)
		b.ReportMetric(float64(signalTime.Nanoseconds()), "ns/signal-setup")

		// Clean up
		cmd.Process.Kill()
		cmd.Wait()
		os.RemoveAll(appPath)
	}

	DefaultConfig = origConfig
}

// BenchmarkIORedirection benchmarks I/O redirection performance
func BenchmarkIORedirection(b *testing.B) {
	pipeSizes := []struct {
		name      string
		dataSize  int
		pipeCount int
	}{
		{"Small", 1024, 1},        // 1KB, 1 pipe
		{"Medium", 64 * 1024, 3},  // 64KB, 3 pipes (stdin/stdout/stderr)
		{"Large", 1024 * 1024, 3}, // 1MB, 3 pipes
	}

	for _, size := range pipeSizes {
		b.Run(size.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Create test pipes
				pipes := make([]string, size.pipeCount)
				for j := 0; j < size.pipeCount; j++ {
					pipe, err := createPipe(fmt.Sprintf("bench-pipe-%d-%d", i, j))
					if err != nil {
						b.Fatalf("Failed to create pipe: %v", err)
					}
					pipes[j] = pipe
				}

				// Test I/O redirection performance
				startTime := time.Now()

				// Create test data
				testData := make([]byte, size.dataSize)
				for k := range testData {
					testData[k] = byte(k % 256)
				}

				// Test pipe I/O
				var wg sync.WaitGroup
				for _, pipe := range pipes {
					wg.Add(1)
					go func(p string) {
						defer wg.Done()
						benchmarkPipeIO(p, testData)
					}(pipe)
				}
				wg.Wait()

				ioTime := time.Since(startTime)
				b.ReportMetric(float64(ioTime.Nanoseconds()), "ns/io-redirect")

				// Clean up pipes
				for _, pipe := range pipes {
					os.Remove(pipe)
					os.RemoveAll(filepath.Dir(pipe))
				}
			}
		})
	}
}

// BenchmarkNamedPipeCreation benchmarks named pipe creation
func BenchmarkNamedPipeCreation(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		pipe, err := createPipe(fmt.Sprintf("bench-pipe-%d", i))
		if err != nil {
			b.Fatalf("Failed to create pipe: %v", err)
		}

		// Clean up
		os.Remove(pipe)
		os.RemoveAll(filepath.Dir(pipe))
	}
}

// BenchmarkPipeCleanup benchmarks pipe cleanup performance
func BenchmarkPipeCleanup(b *testing.B) {
	// Create pipes first
	pipes := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		pipe, err := createPipe(fmt.Sprintf("bench-cleanup-%d", i))
		if err != nil {
			b.Fatalf("Failed to create pipe: %v", err)
		}
		pipes[i] = pipe
	}

	b.ResetTimer()
	b.ReportAllocs()

	// Benchmark cleanup
	for i := 0; i < b.N; i++ {
		os.Remove(pipes[i])
		os.RemoveAll(filepath.Dir(pipes[i]))
	}
}

// BenchmarkRelaunchWithIORedirection benchmarks the full relaunch with I/O process
func BenchmarkRelaunchWithIORedirection(b *testing.B) {
	if runtime.GOOS != "darwin" {
		b.Skip("Relaunch benchmarks only supported on macOS")
	}

	testExecPath := createTestExecutable(b)
	defer os.Remove(testExecPath)

	origConfig := DefaultConfig

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		DefaultConfig = &Config{
			Relaunch:     false, // We'll manually test relaunch
			Entitlements: make(map[Entitlement]bool),
			PlistEntries: make(map[string]any),
			AutoSign:     false,
		}
		DefaultConfig.AddEntitlement(EntAppSandbox)

		bundlePath := fmt.Sprintf("/tmp/macgo-bench-relaunch-%d-%d.app", os.Getpid(), i)
		DefaultConfig.CustomDestinationAppPath = bundlePath

		appPath, err := createBundle(testExecPath)
		if err != nil {
			b.Fatalf("Failed to create bundle: %v", err)
		}

		// Benchmark relaunch setup (without actually relaunching)
		startTime := time.Now()

		// Create pipes for IO redirection
		pipes := make([]string, 3)
		for j, name := range []string{"stdin", "stdout", "stderr"} {
			pipe, err := createPipe(fmt.Sprintf("bench-relaunch-%s-%d", name, i))
			if err != nil {
				b.Fatalf("Failed to create %s pipe: %v", name, err)
			}
			pipes[j] = pipe
		}

		// Prepare open command arguments (setup overhead)
		args := []string{
			"-a", appPath,
			"--wait-apps",
			"--stdin", pipes[0],
			"--stdout", pipes[1],
			"--stderr", pipes[2],
		}

		relaunchTime := time.Since(startTime)
		b.ReportMetric(float64(relaunchTime.Nanoseconds()), "ns/relaunch-setup")

		// Clean up
		for _, pipe := range pipes {
			os.Remove(pipe)
			os.RemoveAll(filepath.Dir(pipe))
		}
		os.RemoveAll(appPath)

		// Avoid unused variable warning
		_ = args
	}

	DefaultConfig = origConfig
}

// BenchmarkImprovedSignalHandling benchmarks improved signal handling
func BenchmarkImprovedSignalHandling(b *testing.B) {
	testExecPath := createTestExecutable(b)
	defer os.Remove(testExecPath)

	origConfig := DefaultConfig

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		DefaultConfig = &Config{
			Relaunch:     false,
			Entitlements: make(map[Entitlement]bool),
			PlistEntries: make(map[string]any),
			AutoSign:     false,
		}

		bundlePath := fmt.Sprintf("/tmp/macgo-bench-improved-signal-%d-%d.app", os.Getpid(), i)
		DefaultConfig.CustomDestinationAppPath = bundlePath

		appPath, err := createBundle(testExecPath)
		if err != nil {
			b.Fatalf("Failed to create bundle: %v", err)
		}

		// Test improved signal handling setup
		startTime := time.Now()

		// Simulate improved signal handling setup
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)

		// Mock the improved signal handling process
		cmd := exec.Command("sleep", "0.1")
		if err := cmd.Start(); err != nil {
			cancel()
			b.Fatalf("Failed to start mock process: %v", err)
		}

		// Test the context-based signal handling
		select {
		case <-ctx.Done():
			// Context timeout
		case <-time.After(50 * time.Millisecond):
			// Normal completion
		}

		improvedSignalTime := time.Since(startTime)
		b.ReportMetric(float64(improvedSignalTime.Nanoseconds()), "ns/improved-signal")

		// Clean up
		cmd.Process.Kill()
		cmd.Wait()
		cancel()
		os.RemoveAll(appPath)
	}

	DefaultConfig = origConfig
}

// BenchmarkLaunchWithDifferentEntitlements benchmarks launch with various entitlements
func BenchmarkLaunchWithDifferentEntitlements(b *testing.B) {
	if runtime.GOOS != "darwin" {
		b.Skip("Launch benchmarks only supported on macOS")
	}

	testExecPath := createTestExecutable(b)
	defer os.Remove(testExecPath)

	entitlementSets := []struct {
		name         string
		entitlements []Entitlement
	}{
		{
			name:         "Basic",
			entitlements: []Entitlement{EntAppSandbox},
		},
		{
			name:         "TCC",
			entitlements: []Entitlement{EntAppSandbox, EntCamera, EntMicrophone},
		},
		{
			name:         "Network",
			entitlements: []Entitlement{EntAppSandbox, EntNetworkClient, EntNetworkServer},
		},
		{
			name: "Full",
			entitlements: []Entitlement{
				EntAppSandbox, EntCamera, EntMicrophone, EntNetworkClient,
				EntLocation, EntAddressBook, EntUserSelectedReadOnly,
			},
		},
	}

	origConfig := DefaultConfig

	for _, entSet := range entitlementSets {
		b.Run(entSet.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				DefaultConfig = &Config{
					Relaunch:     false,
					Entitlements: make(map[Entitlement]bool),
					PlistEntries: make(map[string]any),
					AutoSign:     false,
				}

				// Add entitlements
				for _, ent := range entSet.entitlements {
					DefaultConfig.AddEntitlement(ent)
				}

				bundlePath := fmt.Sprintf("/tmp/macgo-bench-launch-ent-%d-%d.app", os.Getpid(), i)
				DefaultConfig.CustomDestinationAppPath = bundlePath

				appPath, err := createBundle(testExecPath)
				if err != nil {
					b.Fatalf("Failed to create bundle: %v", err)
				}

				// Benchmark launch preparation
				startTime := time.Now()

				// Create pipes for launch
				pipes := make([]string, 3)
				for j, name := range []string{"stdin", "stdout", "stderr"} {
					pipe, err := createPipe(fmt.Sprintf("bench-launch-ent-%s-%d", name, i))
					if err != nil {
						b.Fatalf("Failed to create %s pipe: %v", name, err)
					}
					pipes[j] = pipe
				}

				launchPrepTime := time.Since(startTime)
				b.ReportMetric(float64(launchPrepTime.Nanoseconds()), "ns/launch-prep")

				// Clean up
				for _, pipe := range pipes {
					os.Remove(pipe)
					os.RemoveAll(filepath.Dir(pipe))
				}
				os.RemoveAll(appPath)
			}
		})
	}

	DefaultConfig = origConfig
}

// BenchmarkLaunchConcurrency benchmarks concurrent launch operations
func BenchmarkLaunchConcurrency(b *testing.B) {
	testExecPath := createTestExecutable(b)
	defer os.Remove(testExecPath)

	concurrencyLevels := []int{1, 2, 4, 8}

	origConfig := DefaultConfig

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency-%d", concurrency), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			var wg sync.WaitGroup
			semaphore := make(chan struct{}, concurrency)

			for i := 0; i < b.N; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()
					semaphore <- struct{}{}
					defer func() { <-semaphore }()

					config := &Config{
						Relaunch:     false,
						Entitlements: make(map[Entitlement]bool),
						PlistEntries: make(map[string]any),
						AutoSign:     false,
					}
					config.AddEntitlement(EntAppSandbox)

					bundlePath := fmt.Sprintf("/tmp/macgo-bench-launch-concurrent-%d-%d.app", os.Getpid(), idx)
					config.CustomDestinationAppPath = bundlePath

					savedConfig := DefaultConfig
					DefaultConfig = config

					appPath, err := createBundle(testExecPath)
					if err != nil {
						b.Errorf("Failed to create bundle: %v", err)
						return
					}

					// Simulate launch preparation
					pipes := make([]string, 3)
					for j, name := range []string{"stdin", "stdout", "stderr"} {
						pipe, err := createPipe(fmt.Sprintf("bench-launch-concurrent-%s-%d", name, idx))
						if err != nil {
							b.Errorf("Failed to create %s pipe: %v", name, err)
							return
						}
						pipes[j] = pipe
					}

					// Clean up
					for _, pipe := range pipes {
						os.Remove(pipe)
						os.RemoveAll(filepath.Dir(pipe))
					}
					os.RemoveAll(appPath)
					DefaultConfig = savedConfig
				}(i)
			}

			wg.Wait()
		})
	}

	DefaultConfig = origConfig
}

// Helper functions for launch benchmarks

func benchmarkPipeIO(pipePath string, testData []byte) {
	// Write test data to pipe
	f, err := os.OpenFile(pipePath, os.O_WRONLY, 0)
	if err != nil {
		return
	}
	defer f.Close()

	// Write in chunks to simulate real I/O
	chunkSize := 1024
	for i := 0; i < len(testData); i += chunkSize {
		end := i + chunkSize
		if end > len(testData) {
			end = len(testData)
		}
		f.Write(testData[i:end])
	}
}

// BenchmarkProcessGroupSetup benchmarks process group setup
func BenchmarkProcessGroupSetup(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Simulate process group setup
		cmd := exec.Command("sleep", "0.01")

		// This is the setup we benchmark
		startTime := time.Now()

		// Process group setup
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
			Pgid:    0,
		}

		setupTime := time.Since(startTime)
		b.ReportMetric(float64(setupTime.Nanoseconds()), "ns/process-group-setup")
	}
}

// BenchmarkLaunchMemoryUsage benchmarks memory usage during launch
func BenchmarkLaunchMemoryUsage(b *testing.B) {
	testExecPath := createTestExecutable(b)
	defer os.Remove(testExecPath)

	origConfig := DefaultConfig

	b.ResetTimer()
	b.ReportAllocs()

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	for i := 0; i < b.N; i++ {
		DefaultConfig = &Config{
			Relaunch:     false,
			Entitlements: make(map[Entitlement]bool),
			PlistEntries: make(map[string]any),
			AutoSign:     false,
		}
		DefaultConfig.AddEntitlement(EntAppSandbox)
		DefaultConfig.AddEntitlement(EntCamera)

		bundlePath := fmt.Sprintf("/tmp/macgo-bench-launch-mem-%d-%d.app", os.Getpid(), i)
		DefaultConfig.CustomDestinationAppPath = bundlePath

		appPath, err := createBundle(testExecPath)
		if err != nil {
			b.Fatalf("Failed to create bundle: %v", err)
		}

		// Simulate launch preparation
		pipes := make([]string, 3)
		for j, name := range []string{"stdin", "stdout", "stderr"} {
			pipe, err := createPipe(fmt.Sprintf("bench-launch-mem-%s-%d", name, i))
			if err != nil {
				b.Fatalf("Failed to create %s pipe: %v", name, err)
			}
			pipes[j] = pipe
		}

		// Clean up
		for _, pipe := range pipes {
			os.Remove(pipe)
			os.RemoveAll(filepath.Dir(pipe))
		}
		os.RemoveAll(appPath)
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Report memory usage
	b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "bytes/op")
	b.ReportMetric(float64(m2.TotalAlloc-m1.TotalAlloc)/float64(b.N), "total-bytes/op")

	DefaultConfig = origConfig
}

// BenchmarkIORedirectionContext benchmarks I/O redirection with context
func BenchmarkIORedirectionContext(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Create test pipe
		pipe, err := createPipe(fmt.Sprintf("bench-io-ctx-%d", i))
		if err != nil {
			b.Fatalf("Failed to create pipe: %v", err)
		}

		// Test I/O redirection with context
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)

		// Test data
		testData := strings.NewReader("test data for I/O redirection")

		// Simulate pipeIOContext
		go func() {
			pipeIOContext(ctx, pipe, testData, io.Discard)
		}()

		// Wait for context or completion
		select {
		case <-ctx.Done():
			// Context completed
		case <-time.After(50 * time.Millisecond):
			// Normal completion
		}

		cancel()
		os.Remove(pipe)
		os.RemoveAll(filepath.Dir(pipe))
	}
}
