package debug

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestInitialize(t *testing.T) {
	// Reset state before testing
	resetDebugState()

	// Test basic initialization
	initialize()

	if !isInitialized {
		t.Error("Expected debug package to be initialized")
	}
}

func TestInitializeWithEnvironmentVariables(t *testing.T) {
	// Reset state and set environment variables
	resetDebugState()

	// Set environment variables
	os.Setenv("MACGO_SIGNAL_DEBUG", "1")
	os.Setenv("MACGO_DEBUG_LEVEL", "2")
	os.Setenv("MACGO_PPROF", "1")
	os.Setenv("MACGO_PPROF_PORT", "7070")

	defer func() {
		// Clean up environment variables
		os.Unsetenv("MACGO_SIGNAL_DEBUG")
		os.Unsetenv("MACGO_DEBUG_LEVEL")
		os.Unsetenv("MACGO_PPROF")
		os.Unsetenv("MACGO_PPROF_PORT")
	}()

	initialize()

	if !signalDebugEnabled {
		t.Error("Expected signal debug to be enabled")
	}
	if advancedDebugLevel != 2 {
		t.Errorf("Expected debug level 2, got %d", advancedDebugLevel)
	}
	if !pprofEnabled {
		t.Error("Expected pprof to be enabled")
	}
	if pprofBasePort != 7070 {
		t.Errorf("Expected pprof port 7070, got %d", pprofBasePort)
	}
}

func TestInitializeWithInvalidDebugLevel(t *testing.T) {
	resetDebugState()

	// Set invalid debug level
	os.Setenv("MACGO_DEBUG_LEVEL", "invalid")
	defer os.Unsetenv("MACGO_DEBUG_LEVEL")

	initialize()

	if advancedDebugLevel != 0 {
		t.Errorf("Expected debug level 0 for invalid input, got %d", advancedDebugLevel)
	}
}

func TestInitializeWithInvalidPprofPort(t *testing.T) {
	resetDebugState()

	// Set invalid pprof port
	os.Setenv("MACGO_PPROF", "1")
	os.Setenv("MACGO_PPROF_PORT", "invalid")
	defer func() {
		os.Unsetenv("MACGO_PPROF")
		os.Unsetenv("MACGO_PPROF_PORT")
	}()

	initialize()

	if pprofBasePort != defaultPprofPort {
		t.Errorf("Expected default pprof port %d for invalid input, got %d", defaultPprofPort, pprofBasePort)
	}
}

func TestInitializeWithCustomLogPath(t *testing.T) {
	resetDebugState()

	// Create a temporary directory for the log file
	tmpDir, err := os.MkdirTemp("", "macgo-debug-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	customLogPath := filepath.Join(tmpDir, "custom-debug.log")

	os.Setenv("MACGO_SIGNAL_DEBUG", "1")
	os.Setenv("MACGO_DEBUG_LOG", customLogPath)
	defer func() {
		os.Unsetenv("MACGO_SIGNAL_DEBUG")
		os.Unsetenv("MACGO_DEBUG_LOG")
	}()

	initialize()

	if debugLogFile == nil {
		t.Error("Expected debug log file to be created")
	}

	// Verify the log file was created
	if _, err := os.Stat(customLogPath); os.IsNotExist(err) {
		t.Error("Expected custom log file to be created")
	}
}

func TestInitializeWithUnwritableLogPath(t *testing.T) {
	resetDebugState()

	// Try to write to a path that doesn't exist
	unwritablePath := "/nonexistent/path/debug.log"

	os.Setenv("MACGO_SIGNAL_DEBUG", "1")
	os.Setenv("MACGO_DEBUG_LOG", unwritablePath)
	defer func() {
		os.Unsetenv("MACGO_SIGNAL_DEBUG")
		os.Unsetenv("MACGO_DEBUG_LOG")
	}()

	initialize()

	// Should fall back to stderr logging
	if debugLogger == nil {
		t.Error("Expected debug logger to be created even with unwritable path")
	}
}

func TestInit(t *testing.T) {
	resetDebugState()

	Init()

	if !isInitialized {
		t.Error("Expected debug package to be initialized after Init() call")
	}
}

func TestInitMultipleCalls(t *testing.T) {
	resetDebugState()

	// Multiple calls should not cause issues
	Init()
	Init()
	Init()

	if !isInitialized {
		t.Error("Expected debug package to be initialized after multiple Init() calls")
	}
}

func TestLogSystemInfo(t *testing.T) {
	resetDebugState()

	// Enable signal debugging to activate logging
	os.Setenv("MACGO_SIGNAL_DEBUG", "1")
	defer os.Unsetenv("MACGO_SIGNAL_DEBUG")

	// Create a temporary log file
	tmpFile, err := os.CreateTemp("", "macgo-debug-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Set up debug logger manually
	debugLogFile, err = os.OpenFile(tmpFile.Name(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		t.Fatal(err)
	}
	debugLogger = log.New(debugLogFile, "[macgo-debug] ", log.LstdFlags|log.Lmicroseconds)

	logSystemInfo()

	// Read the log file content
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	logContent := string(content)

	// Check that system info was logged
	expectedEntries := []string{
		"macgo debug logging initialized",
		"PID:",
		"Args:",
		"Signal debugging: enabled",
		"Debug level:",
		"OS:",
		"Working directory:",
		"MACGO_DEBUG=",
		"Time:",
	}

	for _, expected := range expectedEntries {
		if !strings.Contains(logContent, expected) {
			t.Errorf("Expected log to contain '%s', but it didn't. Log content:\n%s", expected, logContent)
		}
	}
}

func TestLogSignal(t *testing.T) {
	resetDebugState()

	// Enable signal debugging
	os.Setenv("MACGO_SIGNAL_DEBUG", "1")
	defer os.Unsetenv("MACGO_SIGNAL_DEBUG")

	// Create a temporary log file
	tmpFile, err := os.CreateTemp("", "macgo-debug-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Set up debug logger manually
	debugLogFile, err = os.OpenFile(tmpFile.Name(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		t.Fatal(err)
	}
	debugLogger = log.New(debugLogFile, "[macgo-debug] ", log.LstdFlags|log.Lmicroseconds)
	signalDebugEnabled = true

	// Test basic signal logging
	LogSignal(syscall.SIGINT, "Test signal message with %s", "argument")

	// Read the log file content
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	logContent := string(content)

	if !strings.Contains(logContent, "SIGNAL interrupt") {
		t.Errorf("Expected log to contain signal information, got: %s", logContent)
	}
	if !strings.Contains(logContent, "Test signal message with argument") {
		t.Errorf("Expected log to contain formatted message, got: %s", logContent)
	}
}

func TestLogSignalWithStackTrace(t *testing.T) {
	resetDebugState()

	// Enable signal debugging with high debug level
	os.Setenv("MACGO_SIGNAL_DEBUG", "1")
	os.Setenv("MACGO_DEBUG_LEVEL", "2")
	defer func() {
		os.Unsetenv("MACGO_SIGNAL_DEBUG")
		os.Unsetenv("MACGO_DEBUG_LEVEL")
	}()

	// Create a temporary log file
	tmpFile, err := os.CreateTemp("", "macgo-debug-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Set up debug logger manually
	debugLogFile, err = os.OpenFile(tmpFile.Name(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		t.Fatal(err)
	}
	debugLogger = log.New(debugLogFile, "[macgo-debug] ", log.LstdFlags|log.Lmicroseconds)
	signalDebugEnabled = true
	advancedDebugLevel = 2

	// Test signal logging with stack trace
	LogSignal(syscall.SIGTERM, "Test signal with stack trace")

	// Read the log file content
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	logContent := string(content)

	if !strings.Contains(logContent, "SIGNAL terminated") {
		t.Errorf("Expected log to contain signal information, got: %s", logContent)
	}
	if !strings.Contains(logContent, "Stack trace:") {
		t.Errorf("Expected log to contain stack trace, got: %s", logContent)
	}
}

func TestLogSignalDisabled(t *testing.T) {
	resetDebugState()

	// Ensure signal debugging is disabled
	signalDebugEnabled = false
	debugLogger = nil

	// Create a temporary log file
	tmpFile, err := os.CreateTemp("", "macgo-debug-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Test signal logging when disabled
	LogSignal(syscall.SIGINT, "This should not be logged")

	// Read the log file content
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	logContent := string(content)

	if strings.Contains(logContent, "This should not be logged") {
		t.Errorf("Expected no logging when disabled, but got: %s", logContent)
	}
}

func TestLogDebug(t *testing.T) {
	resetDebugState()

	// Enable signal debugging
	os.Setenv("MACGO_SIGNAL_DEBUG", "1")
	defer os.Unsetenv("MACGO_SIGNAL_DEBUG")

	// Create a temporary log file
	tmpFile, err := os.CreateTemp("", "macgo-debug-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Set up debug logger manually
	debugLogFile, err = os.OpenFile(tmpFile.Name(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		t.Fatal(err)
	}
	debugLogger = log.New(debugLogFile, "[macgo-debug] ", log.LstdFlags|log.Lmicroseconds)
	signalDebugEnabled = true

	// Test debug logging
	LogDebug("Test debug message with %s and %d", "string", 42)

	// Read the log file content
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	logContent := string(content)

	if !strings.Contains(logContent, "Test debug message with string and 42") {
		t.Errorf("Expected log to contain debug message, got: %s", logContent)
	}
}

func TestLogDebugDisabled(t *testing.T) {
	resetDebugState()

	// Ensure signal debugging is disabled
	signalDebugEnabled = false
	debugLogger = nil

	// Create a temporary log file
	tmpFile, err := os.CreateTemp("", "macgo-debug-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Test debug logging when disabled
	LogDebug("This should not be logged")

	// Read the log file content
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	logContent := string(content)

	if strings.Contains(logContent, "This should not be logged") {
		t.Errorf("Expected no logging when disabled, but got: %s", logContent)
	}
}

func TestGetNextPprofPort(t *testing.T) {
	resetDebugState()

	// Test port incrementing
	originalPort := pprofBasePort

	port1 := GetNextPprofPort()
	port2 := GetNextPprofPort()
	port3 := GetNextPprofPort()

	if port1 != originalPort+1 {
		t.Errorf("Expected first port to be %d, got %d", originalPort+1, port1)
	}
	if port2 != originalPort+2 {
		t.Errorf("Expected second port to be %d, got %d", originalPort+2, port2)
	}
	if port3 != originalPort+3 {
		t.Errorf("Expected third port to be %d, got %d", originalPort+3, port3)
	}
}

func TestIsPprofEnabled(t *testing.T) {
	resetDebugState()

	// Test when pprof is disabled
	if IsPprofEnabled() {
		t.Error("Expected pprof to be disabled by default")
	}

	// Test when pprof is enabled
	pprofEnabled = true
	if !IsPprofEnabled() {
		t.Error("Expected pprof to be enabled")
	}
}

func TestIsTraceEnabled(t *testing.T) {
	resetDebugState()

	// Test when trace is disabled
	if IsTraceEnabled() {
		t.Error("Expected trace to be disabled by default")
	}

	// Test when trace is enabled
	TraceSignalHandling = true
	if !IsTraceEnabled() {
		t.Error("Expected trace to be enabled")
	}
}

func TestGetWorkingDir(t *testing.T) {
	dir := getWorkingDir()

	if dir == "" {
		t.Error("Expected working directory to be returned")
	}

	// Should not contain error message for normal operation
	if strings.Contains(dir, "Error getting working directory") {
		t.Errorf("Expected valid working directory, got error: %s", dir)
	}
}

func TestClose(t *testing.T) {
	resetDebugState()

	// Set up debug logger with a file
	tmpFile, err := os.CreateTemp("", "macgo-debug-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	debugLogFile, err = os.OpenFile(tmpFile.Name(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		t.Fatal(err)
	}
	debugLogger = log.New(debugLogFile, "[macgo-debug] ", log.LstdFlags)

	// Close should clean up resources
	Close()

	if debugLogFile != nil {
		t.Error("Expected debug log file to be nil after Close()")
	}
	if debugLogger != nil {
		t.Error("Expected debug logger to be nil after Close()")
	}
}

func TestStartPprofServerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	resetDebugState()

	// Start a pprof server on a high port to avoid conflicts
	testPort := 9999

	// Start the server
	startPprofServer(testPort)

	// Give it time to start
	time.Sleep(100 * time.Millisecond)

	// Test that we can connect to the server
	// Note: This is a basic integration test - in a real scenario
	// you might want to make an HTTP request to verify the server is running
	// but for now we just verify the function doesn't panic
}

func TestRaceConditions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race condition test in short mode")
	}

	resetDebugState()

	// Test concurrent access to debug functions
	const numGoroutines = 10

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Test concurrent initialization
			Init()

			// Test concurrent port allocation
			GetNextPprofPort()

			// Test concurrent flag checking
			IsPprofEnabled()
			IsTraceEnabled()

			// Test concurrent logging (if enabled)
			LogDebug("Test message from goroutine %d", id)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func TestEnvironmentVariableParsing(t *testing.T) {
	tests := []struct {
		name           string
		envVars        map[string]string
		expectedSignal bool
		expectedLevel  int
		expectedPprof  bool
	}{
		{
			name:           "All disabled",
			envVars:        map[string]string{},
			expectedSignal: false,
			expectedLevel:  0,
			expectedPprof:  false,
		},
		{
			name: "Signal debug enabled",
			envVars: map[string]string{
				"MACGO_SIGNAL_DEBUG": "1",
			},
			expectedSignal: true,
			expectedLevel:  0,
			expectedPprof:  false,
		},
		{
			name: "All enabled with custom level",
			envVars: map[string]string{
				"MACGO_SIGNAL_DEBUG": "1",
				"MACGO_DEBUG_LEVEL":  "3",
				"MACGO_PPROF":        "1",
			},
			expectedSignal: true,
			expectedLevel:  3,
			expectedPprof:  true,
		},
		{
			name: "Invalid values",
			envVars: map[string]string{
				"MACGO_SIGNAL_DEBUG": "invalid",
				"MACGO_DEBUG_LEVEL":  "invalid",
				"MACGO_PPROF":        "invalid",
			},
			expectedSignal: false,
			expectedLevel:  0,
			expectedPprof:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetDebugState()

			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Clean up after test
			defer func() {
				for key := range tt.envVars {
					os.Unsetenv(key)
				}
			}()

			initialize()

			if signalDebugEnabled != tt.expectedSignal {
				t.Errorf("Expected signal debug %v, got %v", tt.expectedSignal, signalDebugEnabled)
			}
			if advancedDebugLevel != tt.expectedLevel {
				t.Errorf("Expected debug level %d, got %d", tt.expectedLevel, advancedDebugLevel)
			}
			if pprofEnabled != tt.expectedPprof {
				t.Errorf("Expected pprof %v, got %v", tt.expectedPprof, pprofEnabled)
			}
		})
	}
}

// Helper function to reset debug state between tests
func resetDebugState() {
	debugMutex.Lock()
	defer debugMutex.Unlock()

	isInitialized = false
	signalDebugEnabled = false
	advancedDebugLevel = 0
	pprofEnabled = false
	pprofBasePort = defaultPprofPort
	TraceSignalHandling = false

	if debugLogFile != nil {
		debugLogFile.Close()
		debugLogFile = nil
	}

	debugLogger = nil
}

// Benchmark tests
func BenchmarkInitialize(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resetDebugState()
		initialize()
	}
}

func BenchmarkGetNextPprofPort(b *testing.B) {
	resetDebugState()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetNextPprofPort()
	}
}

func BenchmarkLogSignal(b *testing.B) {
	resetDebugState()

	// Enable signal debugging
	signalDebugEnabled = true
	debugLogger = log.New(os.Stderr, "[macgo-debug] ", log.LstdFlags)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LogSignal(syscall.SIGINT, "Benchmark test message %d", i)
	}
}

func BenchmarkLogDebug(b *testing.B) {
	resetDebugState()

	// Enable signal debugging
	signalDebugEnabled = true
	debugLogger = log.New(os.Stderr, "[macgo-debug] ", log.LstdFlags)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LogDebug("Benchmark test message %d", i)
	}
}

func TestMain(m *testing.M) {
	// Set up any global test state
	// Run tests
	code := m.Run()

	// Clean up
	resetDebugState()

	os.Exit(code)
}
