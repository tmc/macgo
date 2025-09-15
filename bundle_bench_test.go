package macgo

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// BenchmarkBundleCreation benchmarks the time it takes to create a new app bundle
func BenchmarkBundleCreation(b *testing.B) {
	// Create a test executable
	testExecPath := createTestExecutable(b)
	defer os.Remove(testExecPath)

	// Save original config
	origConfig := DefaultConfig

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Reset config to clean state
		DefaultConfig = &Config{
			Relaunch:     false, // Don't relaunch during benchmarks
			Entitlements: make(map[Entitlement]bool),
			PlistEntries: make(map[string]any),
			AutoSign:     false, // Disable signing for benchmark consistency
		}

		// Create unique bundle path to avoid conflicts
		bundlePath := fmt.Sprintf("/tmp/macgo-bench-%d-%d.app", os.Getpid(), i)
		DefaultConfig.CustomDestinationAppPath = bundlePath

		// Benchmark bundle creation
		appPath, err := createBundle(testExecPath)
		if err != nil {
			b.Fatalf("Failed to create bundle: %v", err)
		}

		// Clean up
		os.RemoveAll(appPath)
	}

	// Restore original config
	DefaultConfig = origConfig
}

// BenchmarkBundleCreationWithEntitlements benchmarks bundle creation with various entitlements
func BenchmarkBundleCreationWithEntitlements(b *testing.B) {
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
			name:         "Medium",
			entitlements: []Entitlement{EntAppSandbox, EntCamera, EntMicrophone, EntNetworkClient},
		},
		{
			name: "Large",
			entitlements: []Entitlement{
				EntAppSandbox, EntCamera, EntMicrophone, EntNetworkClient, EntNetworkServer,
				EntLocation, EntAddressBook, EntCalendars, EntPhotos, EntReminders,
				EntUserSelectedReadOnly, EntUserSelectedReadWrite, EntBluetooth,
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

				bundlePath := fmt.Sprintf("/tmp/macgo-bench-ent-%d-%d.app", os.Getpid(), i)
				DefaultConfig.CustomDestinationAppPath = bundlePath

				appPath, err := createBundle(testExecPath)
				if err != nil {
					b.Fatalf("Failed to create bundle: %v", err)
				}

				os.RemoveAll(appPath)
			}
		})
	}

	DefaultConfig = origConfig
}

// BenchmarkBundleCreationConcurrent benchmarks concurrent bundle creation
func BenchmarkBundleCreationConcurrent(b *testing.B) {
	testExecPath := createTestExecutable(b)
	defer os.Remove(testExecPath)

	concurrencyLevels := []int{1, 2, 4, 8, 16}

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

					// Each goroutine gets its own config copy
					config := &Config{
						Relaunch:     false,
						Entitlements: make(map[Entitlement]bool),
						PlistEntries: make(map[string]any),
						AutoSign:     false,
					}
					config.AddEntitlement(EntAppSandbox)
					config.AddEntitlement(EntCamera)

					bundlePath := fmt.Sprintf("/tmp/macgo-bench-concurrent-%d-%d.app", os.Getpid(), idx)
					config.CustomDestinationAppPath = bundlePath

					// Use temporary config for this goroutine
					savedConfig := DefaultConfig
					DefaultConfig = config

					appPath, err := createBundle(testExecPath)
					if err != nil {
						b.Errorf("Failed to create bundle: %v", err)
						return
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

// BenchmarkBundleCreationMemory benchmarks memory usage during bundle creation
func BenchmarkBundleCreationMemory(b *testing.B) {
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

		// Add medium set of entitlements
		DefaultConfig.AddEntitlement(EntAppSandbox)
		DefaultConfig.AddEntitlement(EntCamera)
		DefaultConfig.AddEntitlement(EntMicrophone)
		DefaultConfig.AddEntitlement(EntNetworkClient)

		bundlePath := fmt.Sprintf("/tmp/macgo-bench-mem-%d-%d.app", os.Getpid(), i)
		DefaultConfig.CustomDestinationAppPath = bundlePath

		appPath, err := createBundle(testExecPath)
		if err != nil {
			b.Fatalf("Failed to create bundle: %v", err)
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

// BenchmarkBundleFileOperations benchmarks file I/O operations during bundle creation
func BenchmarkBundleFileOperations(b *testing.B) {
	testExecPath := createTestExecutable(b)
	defer os.Remove(testExecPath)

	bundleSizes := []struct {
		name         string
		plistEntries int
		resources    int
	}{
		{"Small", 5, 1},
		{"Medium", 20, 5},
		{"Large", 50, 10},
	}

	origConfig := DefaultConfig

	for _, size := range bundleSizes {
		b.Run(size.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				DefaultConfig = &Config{
					Relaunch:     false,
					Entitlements: make(map[Entitlement]bool),
					PlistEntries: make(map[string]any),
					AutoSign:     false,
				}

				// Add many plist entries to increase file operations
				for j := 0; j < size.plistEntries; j++ {
					DefaultConfig.AddPlistEntry(fmt.Sprintf("TestKey%d", j), fmt.Sprintf("TestValue%d", j))
				}

				bundlePath := fmt.Sprintf("/tmp/macgo-bench-file-%d-%d.app", os.Getpid(), i)
				DefaultConfig.CustomDestinationAppPath = bundlePath

				appPath, err := createBundle(testExecPath)
				if err != nil {
					b.Fatalf("Failed to create bundle: %v", err)
				}

				os.RemoveAll(appPath)
			}
		})
	}

	DefaultConfig = origConfig
}

// BenchmarkBundleChecksumCalculation benchmarks checksum calculation performance
func BenchmarkBundleChecksumCalculation(b *testing.B) {
	// Create test files of different sizes
	fileSizes := []struct {
		name string
		size int64
	}{
		{"Small", 1024},             // 1KB
		{"Medium", 1024 * 1024},     // 1MB
		{"Large", 10 * 1024 * 1024}, // 10MB
	}

	for _, size := range fileSizes {
		b.Run(size.name, func(b *testing.B) {
			testFile := createTestFileWithSize(b, size.size)
			defer os.Remove(testFile)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := checksum(testFile)
				if err != nil {
					b.Fatalf("Failed to calculate checksum: %v", err)
				}
			}
		})
	}
}

// BenchmarkBundleCreationWithExistingCheck benchmarks bundle creation with existing bundle check
func BenchmarkBundleCreationWithExistingCheck(b *testing.B) {
	testExecPath := createTestExecutable(b)
	defer os.Remove(testExecPath)

	origConfig := DefaultConfig

	// Create an existing bundle first
	DefaultConfig = &Config{
		Relaunch:     false,
		Entitlements: make(map[Entitlement]bool),
		PlistEntries: make(map[string]any),
		AutoSign:     false,
	}

	bundlePath := fmt.Sprintf("/tmp/macgo-bench-existing-%d.app", os.Getpid())
	DefaultConfig.CustomDestinationAppPath = bundlePath

	// Create the bundle once
	appPath, err := createBundle(testExecPath)
	if err != nil {
		b.Fatalf("Failed to create initial bundle: %v", err)
	}
	defer os.RemoveAll(appPath)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// This should hit the existing bundle check path
		_, err := createBundle(testExecPath)
		if err != nil {
			b.Fatalf("Failed to check existing bundle: %v", err)
		}
	}

	DefaultConfig = origConfig
}

// BenchmarkCleanupManagerPerformance benchmarks cleanup manager operations
func BenchmarkCleanupManagerPerformance(b *testing.B) {
	initCleanupManager()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Create temporary file
		tempFile := fmt.Sprintf("/tmp/macgo-bench-cleanup-%d", i)
		f, err := os.Create(tempFile)
		if err != nil {
			b.Fatalf("Failed to create temp file: %v", err)
		}
		f.Close()

		// Schedule cleanup
		globalCleanupManager.scheduleCleanup(tempFile, 100*time.Millisecond, false)
	}

	// Give cleanup manager time to process
	time.Sleep(200 * time.Millisecond)
}

// Helper functions

func createTestExecutable(b *testing.B) string {
	b.Helper()

	// Create a temporary Go file
	tempDir := b.TempDir()
	goFile := filepath.Join(tempDir, "test.go")

	goCode := `package main

import "fmt"

func main() {
	fmt.Println("Hello from test executable")
}
`

	err := os.WriteFile(goFile, []byte(goCode), 0644)
	if err != nil {
		b.Fatalf("Failed to create test Go file: %v", err)
	}

	// Compile it
	execPath := filepath.Join(tempDir, "test")
	if runtime.GOOS == "windows" {
		execPath += ".exe"
	}

	// Use go build to create the executable
	cmd := fmt.Sprintf("cd %s && go build -o %s %s", tempDir, execPath, goFile)
	err = runCommand(cmd)
	if err != nil {
		b.Fatalf("Failed to compile test executable: %v", err)
	}

	return execPath
}

func createTestFileWithSize(b *testing.B, size int64) string {
	b.Helper()

	tempFile := fmt.Sprintf("/tmp/macgo-bench-file-%d", time.Now().UnixNano())
	f, err := os.Create(tempFile)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}
	defer f.Close()

	// Write random data
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}

	_, err = f.Write(data)
	if err != nil {
		b.Fatalf("Failed to write test data: %v", err)
	}

	return tempFile
}

func runCommand(cmd string) error {
	// Simple command runner using os/exec
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	// Extract cd command if present
	if len(parts) >= 2 && parts[0] == "cd" {
		// Skip cd command for now, just run the rest
		if len(parts) > 3 && parts[2] == "&&" {
			parts = parts[3:]
		} else {
			return nil
		}
	}

	if len(parts) == 0 {
		return nil
	}

	cmd2 := exec.Command(parts[0], parts[1:]...)
	return cmd2.Run()
}

// BenchmarkPathSecurity benchmarks path security validation
func BenchmarkPathSecurity(b *testing.B) {
	testPaths := []string{
		"/tmp/test.app",
		"/Users/test/Desktop/app.app",
		"/System/Library/Test/app.app",
		"../test.app",
		"test/app.app",
		"/var/folders/test/app.app",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, path := range testPaths {
			_, err := securePath(path)
			if err != nil {
				// Expected for some paths
				continue
			}
		}
	}
}

// BenchmarkPlistWriting benchmarks plist file writing performance
func BenchmarkPlistWriting(b *testing.B) {
	plistSizes := []struct {
		name    string
		entries int
	}{
		{"Small", 10},
		{"Medium", 100},
		{"Large", 1000},
	}

	for _, size := range plistSizes {
		b.Run(size.name, func(b *testing.B) {
			// Create test data
			data := make(map[string]any)
			for i := 0; i < size.entries; i++ {
				data[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				tempFile := fmt.Sprintf("/tmp/macgo-bench-plist-%d.plist", i)
				err := writePlist(tempFile, data)
				if err != nil {
					b.Fatalf("Failed to write plist: %v", err)
				}
				os.Remove(tempFile)
			}
		})
	}
}

// BenchmarkContextualOperations benchmarks operations with context
func BenchmarkContextualOperations(b *testing.B) {
	testExecPath := createTestExecutable(b)
	defer os.Remove(testExecPath)

	origConfig := DefaultConfig

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		DefaultConfig = &Config{
			Relaunch:     false,
			Entitlements: make(map[Entitlement]bool),
			PlistEntries: make(map[string]any),
			AutoSign:     false,
		}

		bundlePath := fmt.Sprintf("/tmp/macgo-bench-ctx-%d-%d.app", os.Getpid(), i)
		DefaultConfig.CustomDestinationAppPath = bundlePath

		appPath, err := createBundle(testExecPath)
		if err != nil {
			cancel()
			b.Fatalf("Failed to create bundle: %v", err)
		}

		os.RemoveAll(appPath)
		cancel()
	}

	DefaultConfig = origConfig
}
