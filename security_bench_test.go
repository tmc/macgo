package macgo

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// BenchmarkPathValidation benchmarks path validation performance
func BenchmarkPathValidation(b *testing.B) {
	testPaths := []struct {
		name string
		path string
	}{
		{"Valid-Temp", "/tmp/test.app"},
		{"Valid-Home", "/Users/test/Desktop/app.app"},
		{"Valid-System", "/System/Library/Test/app.app"},
		{"Invalid-Traversal", "../../../etc/passwd"},
		{"Invalid-Null", "/tmp/test\x00app"},
		{"Invalid-Long", strings.Repeat("a", 5000)},
		{"Valid-GOPATH", "/Users/test/go/bin/app.app"},
		{"Valid-VarFolders", "/var/folders/test/app.app"},
	}

	for _, testPath := range testPaths {
		b.Run(testPath.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := securePath(testPath.path)
				// Expected to fail for invalid paths
				_ = err
			}
		})
	}
}

// BenchmarkPathSanitization benchmarks path sanitization
func BenchmarkPathSanitization(b *testing.B) {
	testPaths := []string{
		"../../../etc/passwd",
		"/tmp/../etc/passwd",
		"/tmp/./test.app",
		"/tmp//test.app",
		"/tmp/test.app/",
		"./test.app",
		"test.app",
		"/tmp/test\x00app",
		"/tmp/test\rapp",
		"/tmp/test\napp",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, path := range testPaths {
			_, err := sanitizePath(path)
			// Expected to fail for some paths
			_ = err
		}
	}
}

// BenchmarkSecureJoin benchmarks secure path joining
func BenchmarkSecureJoin(b *testing.B) {
	basePaths := []string{
		"/tmp",
		"/Users/test/Desktop",
		"/var/folders/test",
		"/System/Library/Test",
	}

	pathElements := [][]string{
		{"test.app"},
		{"test", "app.app"},
		{"deep", "path", "to", "app.app"},
		{"../test.app"},        // Should fail
		{"/absolute/path.app"}, // Should fail
	}

	for i, base := range basePaths {
		for j, elements := range pathElements {
			testName := fmt.Sprintf("Base%d-Elements%d", i, j)
			b.Run(testName, func(b *testing.B) {
				b.ResetTimer()
				b.ReportAllocs()

				for k := 0; k < b.N; k++ {
					_, err := secureJoin(base, elements...)
					// Expected to fail for some combinations
					_ = err
				}
			})
		}
	}
}

// BenchmarkExecutablePathValidation benchmarks executable path validation
func BenchmarkExecutablePathValidation(b *testing.B) {
	// Create test executable
	testExec := createTestExecutable(b)
	defer os.Remove(testExec)

	testPaths := []struct {
		name string
		path string
	}{
		{"Valid-Executable", testExec},
		{"Invalid-NonExistent", "/tmp/non-existent-binary"},
		{"Invalid-Directory", "/tmp"},
		{"Invalid-Empty", ""},
		{"Invalid-Traversal", "../../../bin/sh"},
	}

	for _, testPath := range testPaths {
		b.Run(testPath.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				err := validateExecutablePath(testPath.path)
				// Expected to fail for invalid paths
				_ = err
			}
		})
	}
}

// BenchmarkCodeSigningValidation benchmarks code signing identity validation
func BenchmarkCodeSigningValidation(b *testing.B) {
	testIdentities := []struct {
		name     string
		identity string
	}{
		{"Valid-AdHoc", "-"},
		{"Valid-DeveloperID", "Developer ID Application: Test User"},
		{"Valid-MacDeveloper", "Mac Developer: Test User"},
		{"Valid-AppleDevelopment", "Apple Development: Test User"},
		{"Valid-CertHash", "1234567890ABCDEF1234567890ABCDEF12345678"},
		{"Invalid-Empty", ""},
		{"Invalid-Injection", "Test; rm -rf /"},
		{"Invalid-Command", "Developer ID Application: Test `rm -rf /`"},
		{"Invalid-TooLong", strings.Repeat("a", 300)},
		{"Invalid-NullByte", "Test\x00Identity"},
		{"Invalid-InvalidHash", "INVALID_HASH_FORMAT"},
	}

	for _, testIdentity := range testIdentities {
		b.Run(testIdentity.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				err := validateSigningIdentity(testIdentity.identity)
				// Expected to fail for invalid identities
				_ = err
			}
		})
	}
}

// BenchmarkXMLEscaping benchmarks XML escaping for plist generation
func BenchmarkXMLEscaping(b *testing.B) {
	testStrings := []struct {
		name string
		str  string
	}{
		{"Simple", "Hello World"},
		{"WithAmpersand", "AT&T Corporation"},
		{"WithAngles", "<script>alert('xss')</script>"},
		{"WithQuotes", `"Hello" and 'World'`},
		{"Complex", `<test>&"hello"&</test>`},
		{"Empty", ""},
		{"OnlySpecialChars", "&<>\"'"},
		{"Long", strings.Repeat("Hello & World", 100)},
	}

	for _, testStr := range testStrings {
		b.Run(testStr.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				escaped := escapeXML(testStr.str)
				// Verify escaping worked
				if strings.Contains(escaped, "<") && !strings.Contains(escaped, "&lt;") {
					b.Errorf("XML escaping failed for: %s", testStr.str)
				}
			}
		})
	}
}

// BenchmarkChecksumCalculation benchmarks checksum calculation for different file sizes
func BenchmarkChecksumCalculation(b *testing.B) {
	fileSizes := []struct {
		name string
		size int64
	}{
		{"Tiny", 100},               // 100 bytes
		{"Small", 1024},             // 1KB
		{"Medium", 1024 * 1024},     // 1MB
		{"Large", 10 * 1024 * 1024}, // 10MB
	}

	for _, size := range fileSizes {
		b.Run(size.name, func(b *testing.B) {
			// Create test file
			testFile := createTestFileWithChecksum(b, size.size)
			defer os.Remove(testFile)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := checksum(testFile)
				if err != nil {
					b.Fatalf("Checksum calculation failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkChecksumComparison benchmarks checksum comparison for bundle validation
func BenchmarkChecksumComparison(b *testing.B) {
	// Create test files with different content
	testFile1 := createTestFileWithChecksum(b, 1024)
	testFile2 := createTestFileWithChecksum(b, 1024)
	defer os.Remove(testFile1)
	defer os.Remove(testFile2)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		hash1, err := checksum(testFile1)
		if err != nil {
			b.Fatalf("Checksum calculation failed: %v", err)
		}

		hash2, err := checksum(testFile2)
		if err != nil {
			b.Fatalf("Checksum calculation failed: %v", err)
		}

		// Compare checksums
		equal := hash1 == hash2
		_ = equal
	}
}

// BenchmarkBundleSigningValidation benchmarks bundle signing validation
func BenchmarkBundleSigningValidation(b *testing.B) {
	if runtime.GOOS != "darwin" {
		b.Skip("Bundle signing only supported on macOS")
	}

	testExecPath := createTestExecutable(b)
	defer os.Remove(testExecPath)

	origConfig := DefaultConfig

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Create test bundle
		DefaultConfig = &Config{
			Relaunch:        false,
			Entitlements:    make(map[Entitlement]bool),
			PlistEntries:    make(map[string]any),
			AutoSign:        false, // We'll test signing validation separately
			SigningIdentity: "-",   // Ad-hoc signing
		}

		bundlePath := fmt.Sprintf("/tmp/macgo-bench-signing-%d-%d.app", os.Getpid(), i)
		DefaultConfig.CustomDestinationAppPath = bundlePath

		appPath, err := createBundle(testExecPath)
		if err != nil {
			b.Fatalf("Failed to create bundle: %v", err)
		}

		// Test signing validation
		err = validateSigningIdentity(DefaultConfig.SigningIdentity)
		if err != nil {
			b.Errorf("Signing identity validation failed: %v", err)
		}

		// Clean up
		os.RemoveAll(appPath)
	}

	DefaultConfig = origConfig
}

// BenchmarkSecurityPathOperations benchmarks security-related path operations
func BenchmarkSecurityPathOperations(b *testing.B) {
	operations := []struct {
		name string
		fn   func() error
	}{
		{
			name: "ValidateTempPath",
			fn: func() error {
				_, err := securePath("/tmp/test.app")
				return err
			},
		},
		{
			name: "ValidateHomePath",
			fn: func() error {
				home, _ := os.UserHomeDir()
				testPath := filepath.Join(home, "test.app")
				_, err := securePath(testPath)
				return err
			},
		},
		{
			name: "ValidateGOPATHPath",
			fn: func() error {
				gopath := os.Getenv("GOPATH")
				if gopath == "" {
					gopath = "/tmp/go"
				}
				testPath := filepath.Join(gopath, "bin", "test.app")
				_, err := securePath(testPath)
				return err
			},
		},
		{
			name: "RejectTraversalPath",
			fn: func() error {
				_, err := securePath("../../../etc/passwd")
				// This should fail
				return err
			},
		},
		{
			name: "JoinSecurePaths",
			fn: func() error {
				_, err := secureJoin("/tmp", "test", "app.app")
				return err
			},
		},
	}

	for _, op := range operations {
		b.Run(op.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				err := op.fn()
				// Some operations are expected to fail
				_ = err
			}
		})
	}
}

// BenchmarkCleanupManagerSecurity benchmarks cleanup manager security operations
func BenchmarkCleanupManagerSecurity(b *testing.B) {
	initCleanupManager()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Create temporary file
		tempFile := fmt.Sprintf("/tmp/macgo-bench-security-cleanup-%d", i)
		f, err := os.Create(tempFile)
		if err != nil {
			b.Fatalf("Failed to create temp file: %v", err)
		}
		f.Close()

		// Test secure cleanup scheduling
		globalCleanupManager.scheduleCleanup(tempFile, 100*time.Millisecond, false)

		// Test path validation in cleanup
		cleanupEntry := cleanupEntry{
			path:      tempFile,
			cleanupAt: time.Now().Add(50 * time.Millisecond),
			isDir:     false,
		}

		// Simulate safe removal validation
		if _, err := securePath(cleanupEntry.path); err != nil {
			// Expected for some paths
			continue
		}

		// Remove the file manually to avoid cleanup manager interference
		os.Remove(tempFile)
	}
}

// BenchmarkEntitlementSecurity benchmarks entitlement security validation
func BenchmarkEntitlementSecurity(b *testing.B) {
	testEntitlements := []struct {
		name        string
		entitlement Entitlement
		dangerous   bool
	}{
		{"Safe-AppSandbox", EntAppSandbox, false},
		{"Safe-Camera", EntCamera, false},
		{"Safe-Microphone", EntMicrophone, false},
		{"Dangerous-JIT", EntAllowJIT, true},
		{"Dangerous-UnsignedMemory", EntAllowUnsignedExecutableMemory, true},
		{"Dangerous-DisableValidation", EntDisableLibraryValidation, true},
		{"Custom-Safe", Entitlement("com.mycompany.safe.entitlement"), false},
		{"Custom-Dangerous", Entitlement("com.apple.security.cs.allow-jit"), true},
	}

	for _, testEnt := range testEntitlements {
		b.Run(testEnt.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Validate entitlement string
				entStr := string(testEnt.entitlement)

				// Check for dangerous patterns
				dangerous := strings.Contains(entStr, "allow-jit") ||
					strings.Contains(entStr, "allow-unsigned-executable-memory") ||
					strings.Contains(entStr, "disable-library-validation") ||
					strings.Contains(entStr, "disable-executable-page-protection")

				if dangerous != testEnt.dangerous {
					b.Errorf("Dangerous entitlement detection failed for %s", entStr)
				}
			}
		})
	}
}

// BenchmarkPlistSecurityValidation benchmarks plist security validation
func BenchmarkPlistSecurityValidation(b *testing.B) {
	testPlistData := []struct {
		name string
		data map[string]any
	}{
		{
			name: "Safe",
			data: map[string]any{
				"CFBundleName":            "TestApp",
				"CFBundleIdentifier":      "com.test.app",
				"LSUIElement":             true,
				"NSHighResolutionCapable": true,
			},
		},
		{
			name: "WithSpecialChars",
			data: map[string]any{
				"CFBundleName":       "Test & App",
				"CFBundleIdentifier": "com.test.app",
				"CustomKey":          "<script>alert('xss')</script>",
				"QuoteKey":           `"Hello" and 'World'`,
			},
		},
		{
			name: "Large",
			data: func() map[string]any {
				data := make(map[string]any)
				for i := 0; i < 100; i++ {
					key := fmt.Sprintf("Key%d", i)
					value := fmt.Sprintf("Value%d with special chars: &<>\"'", i)
					data[key] = value
				}
				return data
			}(),
		},
	}

	for _, testData := range testPlistData {
		b.Run(testData.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Validate plist data by attempting to write it
				tempFile := fmt.Sprintf("/tmp/macgo-bench-plist-security-%d.plist", i)
				err := writePlist(tempFile, testData.data)
				if err != nil {
					b.Fatalf("Failed to write plist: %v", err)
				}

				// Verify the file was created securely
				if _, err := os.Stat(tempFile); err != nil {
					b.Errorf("Plist file not created properly: %v", err)
				}

				// Clean up
				os.Remove(tempFile)
			}
		})
	}
}

// Helper functions for security benchmarks

func createTestFileWithChecksum(b *testing.B, size int64) string {
	b.Helper()

	tempFile := fmt.Sprintf("/tmp/macgo-bench-checksum-%d-%d", time.Now().UnixNano(), size)
	f, err := os.Create(tempFile)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}
	defer f.Close()

	// Write random data for more realistic checksum calculation
	data := make([]byte, size)
	_, err = rand.Read(data)
	if err != nil {
		b.Fatalf("Failed to generate random data: %v", err)
	}

	_, err = f.Write(data)
	if err != nil {
		b.Fatalf("Failed to write test data: %v", err)
	}

	return tempFile
}

// BenchmarkSecurityOverhead benchmarks overall security overhead
func BenchmarkSecurityOverhead(b *testing.B) {
	testExecPath := createTestExecutable(b)
	defer os.Remove(testExecPath)

	origConfig := DefaultConfig

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Create config with security-sensitive operations
		DefaultConfig = &Config{
			Relaunch:     false,
			Entitlements: make(map[Entitlement]bool),
			PlistEntries: make(map[string]any),
			AutoSign:     false,
		}

		// Add potentially dangerous entitlements
		DefaultConfig.AddEntitlement(EntAppSandbox)
		DefaultConfig.AddEntitlement(EntAllowJIT)
		DefaultConfig.AddEntitlement(EntDisableLibraryValidation)

		// Add plist entries with special characters
		DefaultConfig.AddPlistEntry("TestKey", "Value with & special < chars >")
		DefaultConfig.AddPlistEntry("QuoteKey", `"Hello" and 'World'`)

		bundlePath := fmt.Sprintf("/tmp/macgo-bench-security-overhead-%d-%d.app", os.Getpid(), i)
		DefaultConfig.CustomDestinationAppPath = bundlePath

		// This will trigger all security validations
		appPath, err := createBundle(testExecPath)
		if err != nil {
			b.Fatalf("Failed to create bundle with security overhead: %v", err)
		}

		// Clean up
		os.RemoveAll(appPath)
	}

	DefaultConfig = origConfig
}

// BenchmarkSecurityValidationCaching benchmarks security validation with caching
func BenchmarkSecurityValidationCaching(b *testing.B) {
	// Common paths that would be validated repeatedly
	commonPaths := []string{
		"/tmp/test.app",
		"/Users/test/Desktop/app.app",
		"/var/folders/test/app.app",
		"/System/Library/Test/app.app",
	}

	// Cache for validation results
	validationCache := make(map[string]error)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, path := range commonPaths {
			// Check cache first
			if cachedErr, exists := validationCache[path]; exists {
				_ = cachedErr
				continue
			}

			// Perform validation and cache result
			_, err := securePath(path)
			validationCache[path] = err
		}
	}
}

// BenchmarkSecurityAuditLog benchmarks security audit logging
func BenchmarkSecurityAuditLog(b *testing.B) {
	// Enable debug mode for audit logging
	originalDebug := os.Getenv("MACGO_DEBUG")
	os.Setenv("MACGO_DEBUG", "1")
	defer os.Setenv("MACGO_DEBUG", originalDebug)

	testOperations := []struct {
		name string
		fn   func()
	}{
		{
			name: "PathValidation",
			fn: func() {
				_, err := securePath("/tmp/test.app")
				if err != nil {
					debugf("Path validation failed: %v", err)
				}
			},
		},
		{
			name: "BundleCreation",
			fn: func() {
				debugf("Creating bundle with security validation")
			},
		},
		{
			name: "SigningValidation",
			fn: func() {
				err := validateSigningIdentity("Developer ID Application: Test")
				if err != nil {
					debugf("Signing validation failed: %v", err)
				}
			},
		},
	}

	for _, op := range testOperations {
		b.Run(op.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				op.fn()
			}
		})
	}
}
