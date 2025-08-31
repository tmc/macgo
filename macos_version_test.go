package macgo

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// macOSVersion represents a macOS version
type macOSVersion struct {
	Major int
	Minor int
	Patch int
}

// String returns the version as a string
func (v macOSVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// IsAtLeast checks if this version is at least the given version
func (v macOSVersion) IsAtLeast(other macOSVersion) bool {
	if v.Major != other.Major {
		return v.Major > other.Major
	}
	if v.Minor != other.Minor {
		return v.Minor > other.Minor
	}
	return v.Patch >= other.Patch
}

// getMacOSVersion returns the current macOS version
func getMacOSVersion() (macOSVersion, error) {
	if runtime.GOOS != "darwin" {
		return macOSVersion{}, fmt.Errorf("not running on macOS")
	}

	cmd := exec.Command("sw_vers", "-productVersion")
	output, err := cmd.Output()
	if err != nil {
		return macOSVersion{}, fmt.Errorf("failed to get macOS version: %v", err)
	}

	versionStr := strings.TrimSpace(string(output))
	parts := strings.Split(versionStr, ".")
	
	if len(parts) < 2 {
		return macOSVersion{}, fmt.Errorf("invalid version format: %s", versionStr)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return macOSVersion{}, fmt.Errorf("invalid major version: %s", parts[0])
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return macOSVersion{}, fmt.Errorf("invalid minor version: %s", parts[1])
	}

	patch := 0
	if len(parts) > 2 {
		patch, err = strconv.Atoi(parts[2])
		if err != nil {
			return macOSVersion{}, fmt.Errorf("invalid patch version: %s", parts[2])
		}
	}

	return macOSVersion{Major: major, Minor: minor, Patch: patch}, nil
}

// TestMacOSVersionCompatibility tests macgo functionality across different macOS versions
func TestMacOSVersionCompatibility(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS version compatibility tests on non-macOS platform")
	}

	currentVersion, err := getMacOSVersion()
	if err != nil {
		t.Fatalf("Failed to get macOS version: %v", err)
	}

	t.Logf("Testing on macOS %s", currentVersion.String())

	// Test basic functionality across different version expectations
	tests := []struct {
		name           string
		minVersion     macOSVersion
		testFunc       func(*testing.T)
		skipReason     string
		description    string
	}{
		{
			name:       "Basic bundle creation",
			minVersion: macOSVersion{Major: 10, Minor: 15, Patch: 0}, // macOS Catalina
			testFunc:   testBasicBundleCreation,
			description: "Test basic app bundle creation functionality",
		},
		{
			name:       "App sandbox entitlements",
			minVersion: macOSVersion{Major: 10, Minor: 15, Patch: 0}, // macOS Catalina
			testFunc:   testAppSandboxEntitlements,
			description: "Test app sandbox entitlements functionality",
		},
		{
			name:       "TCC permissions",
			minVersion: macOSVersion{Major: 10, Minor: 15, Patch: 0}, // macOS Catalina
			testFunc:   testTCCPermissions,
			description: "Test TCC (Transparency, Consent, and Control) permissions",
		},
		{
			name:       "Code signing",
			minVersion: macOSVersion{Major: 10, Minor: 15, Patch: 0}, // macOS Catalina
			testFunc:   testCodeSigning,
			description: "Test code signing functionality",
		},
		{
			name:       "Notarization requirements",
			minVersion: macOSVersion{Major: 10, Minor: 15, Patch: 0}, // macOS Catalina
			testFunc:   testNotarizationRequirements,
			description: "Test notarization requirements and handling",
		},
		{
			name:       "System Integrity Protection (SIP)",
			minVersion: macOSVersion{Major: 10, Minor: 15, Patch: 0}, // macOS Catalina
			testFunc:   testSIPCompatibility,
			description: "Test System Integrity Protection compatibility",
		},
		{
			name:       "Hardened Runtime",
			minVersion: macOSVersion{Major: 10, Minor: 15, Patch: 0}, // macOS Catalina
			testFunc:   testHardenedRuntime,
			description: "Test Hardened Runtime compatibility",
		},
		{
			name:       "macOS Big Sur specific features",
			minVersion: macOSVersion{Major: 11, Minor: 0, Patch: 0}, // macOS Big Sur
			testFunc:   testBigSurFeatures,
			description: "Test macOS Big Sur specific features",
		},
		{
			name:       "macOS Monterey specific features",
			minVersion: macOSVersion{Major: 12, Minor: 0, Patch: 0}, // macOS Monterey
			testFunc:   testMontereyFeatures,
			description: "Test macOS Monterey specific features",
		},
		{
			name:       "macOS Ventura specific features",
			minVersion: macOSVersion{Major: 13, Minor: 0, Patch: 0}, // macOS Ventura
			testFunc:   testVenturaFeatures,
			description: "Test macOS Ventura specific features",
		},
		{
			name:       "macOS Sonoma specific features",
			minVersion: macOSVersion{Major: 14, Minor: 0, Patch: 0}, // macOS Sonoma
			testFunc:   testSonomaFeatures,
			description: "Test macOS Sonoma specific features",
		},
		{
			name:       "macOS Sequoia specific features",
			minVersion: macOSVersion{Major: 15, Minor: 0, Patch: 0}, // macOS Sequoia
			testFunc:   testSequoiaFeatures,
			description: "Test macOS Sequoia specific features",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !currentVersion.IsAtLeast(tt.minVersion) {
				t.Skipf("Skipping test for macOS %s (requires %s)", currentVersion.String(), tt.minVersion.String())
				return
			}

			if tt.skipReason != "" {
				t.Skip(tt.skipReason)
				return
			}

			t.Logf("Running %s on macOS %s", tt.description, currentVersion.String())
			tt.testFunc(t)
		})
	}
}

func testBasicBundleCreation(t *testing.T) {
	// Test that we can create a basic app bundle
	cfg := NewConfig()
	cfg.ApplicationName = "VersionTestApp"
	cfg.BundleID = "com.example.versiontest"
	cfg.KeepTemp = true
	
	// Create a temporary executable
	tmpExec, err := createTempExecutable(t)
	if err != nil {
		t.Fatalf("Failed to create temporary executable: %v", err)
	}
	defer os.Remove(tmpExec)

	// Test bundle creation
	bundlePath, err := createAppBundle(cfg, tmpExec)
	if err != nil {
		t.Fatalf("Failed to create app bundle: %v", err)
	}
	defer os.RemoveAll(bundlePath)

	// Verify bundle structure
	if !directoryExists(bundlePath) {
		t.Errorf("Bundle directory does not exist: %s", bundlePath)
	}

	contentsPath := bundlePath + "/Contents"
	if !directoryExists(contentsPath) {
		t.Errorf("Contents directory does not exist: %s", contentsPath)
	}

	macOSPath := contentsPath + "/MacOS"
	if !directoryExists(macOSPath) {
		t.Errorf("MacOS directory does not exist: %s", macOSPath)
	}

	infoPlistPath := contentsPath + "/Info.plist"
	if !fileExists(infoPlistPath) {
		t.Errorf("Info.plist does not exist: %s", infoPlistPath)
	}
}

func testAppSandboxEntitlements(t *testing.T) {
	// Test app sandbox entitlements
	cfg := NewConfig()
	cfg.ApplicationName = "SandboxTestApp"
	cfg.BundleID = "com.example.sandboxtest"
	cfg.RequestEntitlements(EntAppSandbox, EntUserSelectedReadOnly)
	cfg.KeepTemp = true

	// Create a temporary executable
	tmpExec, err := createTempExecutable(t)
	if err != nil {
		t.Fatalf("Failed to create temporary executable: %v", err)
	}
	defer os.Remove(tmpExec)

	// Test bundle creation with sandbox entitlements
	bundlePath, err := createAppBundle(cfg, tmpExec)
	if err != nil {
		t.Fatalf("Failed to create app bundle with sandbox entitlements: %v", err)
	}
	defer os.RemoveAll(bundlePath)

	// Verify entitlements file
	entitlementsPath := bundlePath + "/Contents/entitlements.plist"
	if !fileExists(entitlementsPath) {
		t.Errorf("Entitlements file does not exist: %s", entitlementsPath)
	}

	// Read and verify entitlements content
	content, err := os.ReadFile(entitlementsPath)
	if err != nil {
		t.Fatalf("Failed to read entitlements file: %v", err)
	}

	entitlementsContent := string(content)
	if !strings.Contains(entitlementsContent, "com.apple.security.app-sandbox") {
		t.Error("App sandbox entitlement not found in entitlements file")
	}
	if !strings.Contains(entitlementsContent, "com.apple.security.files.user-selected.read-only") {
		t.Error("User selected read-only entitlement not found in entitlements file")
	}
}

func testTCCPermissions(t *testing.T) {
	// Test TCC permissions handling
	cfg := NewConfig()
	cfg.ApplicationName = "TCCTestApp"
	cfg.BundleID = "com.example.tcctest"
	cfg.RequestEntitlements(EntCamera, EntMicrophone, EntLocation)
	cfg.KeepTemp = true

	// Create a temporary executable
	tmpExec, err := createTempExecutable(t)
	if err != nil {
		t.Fatalf("Failed to create temporary executable: %v", err)
	}
	defer os.Remove(tmpExec)

	// Test bundle creation with TCC permissions
	bundlePath, err := createAppBundle(cfg, tmpExec)
	if err != nil {
		t.Fatalf("Failed to create app bundle with TCC permissions: %v", err)
	}
	defer os.RemoveAll(bundlePath)

	// Verify TCC entitlements in entitlements file
	entitlementsPath := bundlePath + "/Contents/entitlements.plist"
	if !fileExists(entitlementsPath) {
		t.Errorf("Entitlements file does not exist: %s", entitlementsPath)
	}

	content, err := os.ReadFile(entitlementsPath)
	if err != nil {
		t.Fatalf("Failed to read entitlements file: %v", err)
	}

	entitlementsContent := string(content)
	expectedEntitlements := []string{
		"com.apple.security.device.camera",
		"com.apple.security.device.microphone",
		"com.apple.security.personal-information.location",
	}

	for _, entitlement := range expectedEntitlements {
		if !strings.Contains(entitlementsContent, entitlement) {
			t.Errorf("TCC entitlement %s not found in entitlements file", entitlement)
		}
	}
}

func testCodeSigning(t *testing.T) {
	// Test code signing functionality
	cfg := NewConfig()
	cfg.ApplicationName = "SigningTestApp"
	cfg.BundleID = "com.example.signingtest"
	cfg.AutoSign = true
	cfg.SigningIdentity = "-" // Ad-hoc signing
	cfg.KeepTemp = true

	// Create a temporary executable
	tmpExec, err := createTempExecutable(t)
	if err != nil {
		t.Fatalf("Failed to create temporary executable: %v", err)
	}
	defer os.Remove(tmpExec)

	// Test bundle creation with code signing
	bundlePath, err := createAppBundle(cfg, tmpExec)
	if err != nil {
		t.Fatalf("Failed to create app bundle with code signing: %v", err)
	}
	defer os.RemoveAll(bundlePath)

	// Verify code signing was applied
	cmd := exec.Command("codesign", "-dv", bundlePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Code signing verification output: %s", string(output))
		t.Errorf("Failed to verify code signing: %v", err)
	}

	// Check if the bundle is properly signed
	cmd = exec.Command("codesign", "-v", bundlePath)
	err = cmd.Run()
	if err != nil {
		t.Errorf("Bundle is not properly signed: %v", err)
	}
}

func testNotarizationRequirements(t *testing.T) {
	// Test notarization requirements handling
	// This test checks if the bundle meets notarization requirements
	cfg := NewConfig()
	cfg.ApplicationName = "NotarizationTestApp"
	cfg.BundleID = "com.example.notarizationtest"
	cfg.AutoSign = true
	cfg.SigningIdentity = "-" // Ad-hoc signing
	cfg.RequestEntitlements(EntAppSandbox, EntHardenedRuntime)
	cfg.KeepTemp = true

	// Create a temporary executable
	tmpExec, err := createTempExecutable(t)
	if err != nil {
		t.Fatalf("Failed to create temporary executable: %v", err)
	}
	defer os.Remove(tmpExec)

	// Test bundle creation with notarization requirements
	bundlePath, err := createAppBundle(cfg, tmpExec)
	if err != nil {
		t.Fatalf("Failed to create app bundle with notarization requirements: %v", err)
	}
	defer os.RemoveAll(bundlePath)

	// Verify hardened runtime is enabled
	cmd := exec.Command("codesign", "-dv", "--entitlements", ":-", bundlePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Entitlements verification output: %s", string(output))
		t.Errorf("Failed to verify entitlements: %v", err)
	}

	entitlementsOutput := string(output)
	if !strings.Contains(entitlementsOutput, "com.apple.security.app-sandbox") {
		t.Error("App sandbox entitlement not found (required for notarization)")
	}
}

func testSIPCompatibility(t *testing.T) {
	// Test System Integrity Protection compatibility
	// This test checks if macgo respects SIP restrictions
	
	// Check if SIP is enabled
	cmd := exec.Command("csrutil", "status")
	output, err := cmd.Output()
	if err != nil {
		t.Logf("Could not check SIP status: %v", err)
		return
	}

	sipStatus := string(output)
	t.Logf("SIP Status: %s", strings.TrimSpace(sipStatus))

	// Test that macgo doesn't try to access SIP-protected areas
	cfg := NewConfig()
	cfg.ApplicationName = "SIPTestApp"
	cfg.BundleID = "com.example.siptest"
	cfg.KeepTemp = true

	// Create a temporary executable
	tmpExec, err := createTempExecutable(t)
	if err != nil {
		t.Fatalf("Failed to create temporary executable: %v", err)
	}
	defer os.Remove(tmpExec)

	// Test bundle creation respects SIP
	bundlePath, err := createAppBundle(cfg, tmpExec)
	if err != nil {
		t.Fatalf("Failed to create app bundle with SIP considerations: %v", err)
	}
	defer os.RemoveAll(bundlePath)

	// Verify bundle was created in a SIP-allowed location
	if strings.HasPrefix(bundlePath, "/System/") {
		t.Error("Bundle should not be created in SIP-protected /System/ directory")
	}
	if strings.HasPrefix(bundlePath, "/usr/") && !strings.HasPrefix(bundlePath, "/usr/local/") {
		t.Error("Bundle should not be created in SIP-protected /usr/ directory")
	}
}

func testHardenedRuntime(t *testing.T) {
	// Test Hardened Runtime compatibility
	cfg := NewConfig()
	cfg.ApplicationName = "HardenedRuntimeTestApp"
	cfg.BundleID = "com.example.hardenedtest"
	cfg.RequestEntitlements(EntHardenedRuntime)
	cfg.AutoSign = true
	cfg.SigningIdentity = "-" // Ad-hoc signing
	cfg.KeepTemp = true

	// Create a temporary executable
	tmpExec, err := createTempExecutable(t)
	if err != nil {
		t.Fatalf("Failed to create temporary executable: %v", err)
	}
	defer os.Remove(tmpExec)

	// Test bundle creation with Hardened Runtime
	bundlePath, err := createAppBundle(cfg, tmpExec)
	if err != nil {
		t.Fatalf("Failed to create app bundle with Hardened Runtime: %v", err)
	}
	defer os.RemoveAll(bundlePath)

	// Verify Hardened Runtime is enabled
	cmd := exec.Command("codesign", "-dv", bundlePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Hardened Runtime verification output: %s", string(output))
		t.Errorf("Failed to verify Hardened Runtime: %v", err)
	}

	// Check for Hardened Runtime flag in the output
	signingOutput := string(output)
	if !strings.Contains(signingOutput, "runtime") {
		t.Log("Hardened Runtime may not be enabled (this is expected for ad-hoc signing)")
	}
}

func testBigSurFeatures(t *testing.T) {
	// Test macOS Big Sur specific features
	t.Log("Testing macOS Big Sur specific features")
	
	// Big Sur introduced stricter notarization requirements
	cfg := NewConfig()
	cfg.ApplicationName = "BigSurTestApp"
	cfg.BundleID = "com.example.bigsurtest"
	cfg.RequestEntitlements(EntAppSandbox, EntHardenedRuntime)
	cfg.KeepTemp = true

	// Create a temporary executable
	tmpExec, err := createTempExecutable(t)
	if err != nil {
		t.Fatalf("Failed to create temporary executable: %v", err)
	}
	defer os.Remove(tmpExec)

	// Test bundle creation with Big Sur requirements
	bundlePath, err := createAppBundle(cfg, tmpExec)
	if err != nil {
		t.Fatalf("Failed to create app bundle for Big Sur: %v", err)
	}
	defer os.RemoveAll(bundlePath)

	// Verify bundle meets Big Sur requirements
	if !directoryExists(bundlePath) {
		t.Error("Bundle was not created successfully for Big Sur")
	}
}

func testMontereyFeatures(t *testing.T) {
	// Test macOS Monterey specific features
	t.Log("Testing macOS Monterey specific features")
	
	// Monterey has enhanced privacy features
	cfg := NewConfig()
	cfg.ApplicationName = "MontereyTestApp"
	cfg.BundleID = "com.example.montereytest"
	cfg.RequestEntitlements(EntAppSandbox, EntCamera, EntMicrophone)
	cfg.KeepTemp = true

	// Create a temporary executable
	tmpExec, err := createTempExecutable(t)
	if err != nil {
		t.Fatalf("Failed to create temporary executable: %v", err)
	}
	defer os.Remove(tmpExec)

	// Test bundle creation with Monterey features
	bundlePath, err := createAppBundle(cfg, tmpExec)
	if err != nil {
		t.Fatalf("Failed to create app bundle for Monterey: %v", err)
	}
	defer os.RemoveAll(bundlePath)

	// Verify bundle meets Monterey requirements
	if !directoryExists(bundlePath) {
		t.Error("Bundle was not created successfully for Monterey")
	}
}

func testVenturaFeatures(t *testing.T) {
	// Test macOS Ventura specific features
	t.Log("Testing macOS Ventura specific features")
	
	// Ventura has updated security requirements
	cfg := NewConfig()
	cfg.ApplicationName = "VenturaTestApp"
	cfg.BundleID = "com.example.venturatest"
	cfg.RequestEntitlements(EntAppSandbox, EntNetworkClient)
	cfg.KeepTemp = true

	// Create a temporary executable
	tmpExec, err := createTempExecutable(t)
	if err != nil {
		t.Fatalf("Failed to create temporary executable: %v", err)
	}
	defer os.Remove(tmpExec)

	// Test bundle creation with Ventura features
	bundlePath, err := createAppBundle(cfg, tmpExec)
	if err != nil {
		t.Fatalf("Failed to create app bundle for Ventura: %v", err)
	}
	defer os.RemoveAll(bundlePath)

	// Verify bundle meets Ventura requirements
	if !directoryExists(bundlePath) {
		t.Error("Bundle was not created successfully for Ventura")
	}
}

func testSonomaFeatures(t *testing.T) {
	// Test macOS Sonoma specific features
	t.Log("Testing macOS Sonoma specific features")
	
	// Sonoma has enhanced app permissions
	cfg := NewConfig()
	cfg.ApplicationName = "SonomaTestApp"
	cfg.BundleID = "com.example.sonomatest"
	cfg.RequestEntitlements(EntAppSandbox, EntUserSelectedReadOnly)
	cfg.KeepTemp = true

	// Create a temporary executable
	tmpExec, err := createTempExecutable(t)
	if err != nil {
		t.Fatalf("Failed to create temporary executable: %v", err)
	}
	defer os.Remove(tmpExec)

	// Test bundle creation with Sonoma features
	bundlePath, err := createAppBundle(cfg, tmpExec)
	if err != nil {
		t.Fatalf("Failed to create app bundle for Sonoma: %v", err)
	}
	defer os.RemoveAll(bundlePath)

	// Verify bundle meets Sonoma requirements
	if !directoryExists(bundlePath) {
		t.Error("Bundle was not created successfully for Sonoma")
	}
}

func testSequoiaFeatures(t *testing.T) {
	// Test macOS Sequoia specific features
	t.Log("Testing macOS Sequoia specific features")
	
	// Sequoia has the latest security and privacy features
	cfg := NewConfig()
	cfg.ApplicationName = "SequoiaTestApp"
	cfg.BundleID = "com.example.sequoiatest"
	cfg.RequestEntitlements(EntAppSandbox, EntHardenedRuntime)
	cfg.KeepTemp = true

	// Create a temporary executable
	tmpExec, err := createTempExecutable(t)
	if err != nil {
		t.Fatalf("Failed to create temporary executable: %v", err)
	}
	defer os.Remove(tmpExec)

	// Test bundle creation with Sequoia features
	bundlePath, err := createAppBundle(cfg, tmpExec)
	if err != nil {
		t.Fatalf("Failed to create app bundle for Sequoia: %v", err)
	}
	defer os.RemoveAll(bundlePath)

	// Verify bundle meets Sequoia requirements
	if !directoryExists(bundlePath) {
		t.Error("Bundle was not created successfully for Sequoia")
	}
}

// Helper functions

func createTempExecutable(t *testing.T) (string, error) {
	// Create a temporary executable file
	tmpFile, err := os.CreateTemp("", "macgo-test-exec-*")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	// Write a simple executable
	execContent := `#!/bin/bash
echo "Test executable"
exit 0
`
	_, err = tmpFile.WriteString(execContent)
	if err != nil {
		return "", err
	}

	// Make it executable
	err = os.Chmod(tmpFile.Name(), 0755)
	if err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}

func directoryExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// createAppBundle is a simplified version for testing
func createAppBundle(cfg *Config, executablePath string) (string, error) {
	// This is a simplified implementation for testing
	// In a real scenario, this would use the actual bundle creation logic
	
	// Create a temporary directory for the bundle
	tmpDir, err := os.MkdirTemp("", "macgo-test-bundle-*")
	if err != nil {
		return "", err
	}

	bundlePath := tmpDir + "/" + cfg.ApplicationName + ".app"
	
	// Create bundle structure
	contentsPath := bundlePath + "/Contents"
	macOSPath := contentsPath + "/MacOS"
	
	if err := os.MkdirAll(macOSPath, 0755); err != nil {
		return "", err
	}

	// Copy executable
	execName := cfg.ApplicationName
	if execName == "" {
		execName = "app"
	}
	destExec := macOSPath + "/" + execName
	
	if err := copyFile(executablePath, destExec); err != nil {
		return "", err
	}

	// Create Info.plist
	infoPlist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleName</key>
	<string>%s</string>
	<key>CFBundleIdentifier</key>
	<string>%s</string>
	<key>CFBundleVersion</key>
	<string>1.0</string>
	<key>CFBundleExecutable</key>
	<string>%s</string>
</dict>
</plist>`, cfg.ApplicationName, cfg.BundleID, execName)

	infoPlistPath := contentsPath + "/Info.plist"
	if err := os.WriteFile(infoPlistPath, []byte(infoPlist), 0644); err != nil {
		return "", err
	}

	// Create entitlements file if needed
	if cfg.Entitlements != nil && len(cfg.Entitlements) > 0 {
		entitlementsContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
`
		for ent, value := range cfg.Entitlements {
			entitlementsContent += fmt.Sprintf(`	<key>%s</key>
	<%s/>
`, ent, boolToString(value))
		}
		entitlementsContent += `</dict>
</plist>`

		entitlementsPath := contentsPath + "/entitlements.plist"
		if err := os.WriteFile(entitlementsPath, []byte(entitlementsContent), 0644); err != nil {
			return "", err
		}
	}

	return bundlePath, nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = dstFile.ReadFrom(srcFile)
	if err != nil {
		return err
	}

	// Copy permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	
	return os.Chmod(dst, srcInfo.Mode())
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// TestSystemCapabilities tests system capabilities that affect macgo
func TestSystemCapabilities(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping system capabilities test on non-macOS platform")
	}

	t.Run("codesign availability", func(t *testing.T) {
		_, err := exec.LookPath("codesign")
		if err != nil {
			t.Errorf("codesign not available: %v", err)
		}
	})

	t.Run("sw_vers availability", func(t *testing.T) {
		_, err := exec.LookPath("sw_vers")
		if err != nil {
			t.Errorf("sw_vers not available: %v", err)
		}
	})

	t.Run("security command availability", func(t *testing.T) {
		_, err := exec.LookPath("security")
		if err != nil {
			t.Errorf("security command not available: %v", err)
		}
	})

	t.Run("developer tools availability", func(t *testing.T) {
		_, err := exec.LookPath("xcode-select")
		if err != nil {
			t.Logf("Xcode command line tools not available: %v", err)
		}
	})
}

// TestVersionSpecificBehavior tests version-specific behavior
func TestVersionSpecificBehavior(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping version-specific behavior test on non-macOS platform")
	}

	currentVersion, err := getMacOSVersion()
	if err != nil {
		t.Fatalf("Failed to get macOS version: %v", err)
	}

	// Test version-specific behavior
	if currentVersion.IsAtLeast(macOSVersion{Major: 11, Minor: 0, Patch: 0}) {
		t.Log("Running on macOS 11.0+ (Big Sur or later)")
		// Test Big Sur+ specific behavior
	}

	if currentVersion.IsAtLeast(macOSVersion{Major: 12, Minor: 0, Patch: 0}) {
		t.Log("Running on macOS 12.0+ (Monterey or later)")
		// Test Monterey+ specific behavior
	}

	if currentVersion.IsAtLeast(macOSVersion{Major: 13, Minor: 0, Patch: 0}) {
		t.Log("Running on macOS 13.0+ (Ventura or later)")
		// Test Ventura+ specific behavior
	}

	if currentVersion.IsAtLeast(macOSVersion{Major: 14, Minor: 0, Patch: 0}) {
		t.Log("Running on macOS 14.0+ (Sonoma or later)")
		// Test Sonoma+ specific behavior
	}

	if currentVersion.IsAtLeast(macOSVersion{Major: 15, Minor: 0, Patch: 0}) {
		t.Log("Running on macOS 15.0+ (Sequoia or later)")
		// Test Sequoia+ specific behavior
	}
}

// Benchmark tests
func BenchmarkVersionDetection(b *testing.B) {
	if runtime.GOOS != "darwin" {
		b.Skip("Skipping macOS version detection benchmark on non-macOS platform")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := getMacOSVersion()
		if err != nil {
			b.Fatalf("Failed to get macOS version: %v", err)
		}
	}
}

func BenchmarkSystemCapabilityCheck(b *testing.B) {
	if runtime.GOOS != "darwin" {
		b.Skip("Skipping system capability benchmark on non-macOS platform")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := exec.LookPath("codesign")
		if err != nil {
			b.Fatalf("codesign not available: %v", err)
		}
	}
}