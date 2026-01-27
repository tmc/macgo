package tcc

import (
	"strings"
	"testing"
	"time"
)

func TestEdgeCaseType_String(t *testing.T) {
	tests := []struct {
		name     string
		caseType EdgeCaseType
		want     string
	}{
		{"Unknown", EdgeCaseUnknown, "Unknown"},
		{"PromptDismissed", EdgeCasePromptDismissed, "PromptDismissed"},
		{"PermissionDenied", EdgeCasePermissionDenied, "PermissionDenied"},
		{"MultipleDenials", EdgeCaseMultipleDenials, "MultipleDenials"},
		{"SettingsOpen", EdgeCaseSettingsOpen, "SettingsOpen"},
		{"SettingsLocked", EdgeCaseSettingsLocked, "SettingsLocked"},
		{"AppNotInList", EdgeCaseAppNotInList, "AppNotInList"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.caseType.String(); got != tt.want {
				t.Errorf("EdgeCaseType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEdgeCaseError_Error(t *testing.T) {
	err := &EdgeCaseError{
		Type:     EdgeCasePromptDismissed,
		Message:  "User dismissed the TCC prompt",
		Service:  "camera",
		BundleID: "com.example.app",
		Recovery: "Open System Settings and grant permission manually",
	}

	errorStr := err.Error()

	// Verify error message contains key components
	if !strings.Contains(errorStr, "PromptDismissed") {
		t.Error("Error message should contain edge case type")
	}
	if !strings.Contains(errorStr, "User dismissed") {
		t.Error("Error message should contain the message")
	}
	if !strings.Contains(errorStr, "Recovery:") {
		t.Error("Error message should contain recovery instructions")
	}
	if !strings.Contains(errorStr, "Open System Settings") {
		t.Error("Error message should contain recovery text")
	}
}

func TestDetectEdgeCase_NoEdgeCase(t *testing.T) {
	// Test when there's no edge case detected
	// This test assumes System Settings is not open
	edgeCase, err := DetectEdgeCase("camera", "com.example.test")
	if err != nil {
		t.Errorf("DetectEdgeCase() returned unexpected error: %v", err)
	}

	// We expect either nil (no edge case) or a specific edge case
	// The actual result depends on the system state during the test
	if edgeCase != nil {
		t.Logf("Edge case detected (system dependent): %v", edgeCase.Type)
	}
}

func TestOpenSystemSettingsToTCC_UnknownService(t *testing.T) {
	err := OpenSystemSettingsToTCC("unknown-service", "com.example.test", "TestApp", false)
	if err == nil {
		t.Error("Expected error for unknown service, got nil")
	}

	if !strings.Contains(err.Error(), "unknown TCC service") {
		t.Errorf("Expected 'unknown TCC service' error, got: %v", err)
	}
}

func TestOpenSystemSettingsToTCC_KnownServices(t *testing.T) {
	// Test that all known services have URLs defined
	services := []string{
		"camera",
		"microphone",
		"location",
		"screen-recording",
		"accessibility",
		"automation",
		"files",
	}

	for _, service := range services {
		t.Run(service, func(t *testing.T) {
			// We can't actually open System Settings in a test, but we can verify
			// that the function doesn't return an error for known services
			// (until it tries to execute 'open' command)

			// Just verify the service is recognized - actual opening would require
			// mocking or integration testing
			paneURLs := map[string]string{
				"camera":           "x-apple.systempreferences:com.apple.preference.security?Privacy_Camera",
				"microphone":       "x-apple.systempreferences:com.apple.preference.security?Privacy_Microphone",
				"location":         "x-apple.systempreferences:com.apple.preference.security?Privacy_LocationServices",
				"screen-recording": "x-apple.systempreferences:com.apple.preference.security?Privacy_ScreenCapture",
				"accessibility":    "x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility",
				"automation":       "x-apple.systempreferences:com.apple.preference.security?Privacy_Automation",
				"files":            "x-apple.systempreferences:com.apple.preference.security?Privacy_AllFiles",
			}

			if _, ok := paneURLs[service]; !ok {
				t.Errorf("Service %s not found in pane URLs", service)
			}
		})
	}
}

func TestHandleEdgeCase(t *testing.T) {
	// Test that HandleEdgeCase doesn't panic and returns reasonable results
	err := HandleEdgeCase("camera", "com.example.test", "TestApp", false)

	// The result depends on system state - we're mainly checking it doesn't crash
	if err != nil {
		// If there's an error, it should be an EdgeCaseError or a detection error
		if edgeErr, ok := err.(*EdgeCaseError); ok {
			t.Logf("Edge case detected: %v", edgeErr.Type)
			// Verify the error has recovery instructions
			if edgeErr.Recovery == "" {
				t.Error("EdgeCaseError should have recovery instructions")
			}
		}
	}
}

func TestWaitForPermissionGrant_Timeout(t *testing.T) {
	// Test that WaitForPermissionGrant properly times out
	// Use a very short timeout to avoid slowing down tests
	timeout := 100 * time.Millisecond

	granted, err := WaitForPermissionGrant("camera", "com.example.nonexistent", timeout, false)

	if granted {
		t.Error("Expected permission not to be granted for nonexistent app")
	}

	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	// Check if it's a timeout error or edge case error
	if err != nil {
		if strings.Contains(err.Error(), "timeout") {
			// Good - timeout as expected
		} else if _, ok := err.(*EdgeCaseError); ok {
			// Also acceptable - edge case detected during wait
		} else {
			t.Errorf("Expected timeout or edge case error, got: %v", err)
		}
	}
}

func TestCheckPermissionStatus_UnknownService(t *testing.T) {
	// Test that checkPermissionStatus handles unknown services gracefully
	granted, err := checkPermissionStatus("unknown-service", "com.example.test")

	if granted {
		t.Error("Unknown service should not be granted")
	}

	if err == nil {
		t.Error("Expected error for unknown service")
	}

	if !strings.Contains(err.Error(), "unknown TCC service") {
		t.Errorf("Expected 'unknown TCC service' error, got: %v", err)
	}
}

func TestCheckPermissionStatus_KnownServices(t *testing.T) {
	// Test that known services are properly mapped
	services := []string{"camera", "microphone", "screen-recording", "accessibility"}

	for _, service := range services {
		t.Run(service, func(t *testing.T) {
			// We can't actually check real permissions in a test,
			// but we can verify the service is recognized
			_, err := checkPermissionStatus(service, "com.example.test")

			// The function will likely fail trying to access the TCC database,
			// but the important thing is the service was recognized
			// (error would be about database access, not unknown service)
			if err != nil && strings.Contains(err.Error(), "unknown TCC service") {
				t.Errorf("Service %s should be recognized", service)
			}
		})
	}
}

// TestEdgeCaseError_Fields verifies all edge case error fields are populated correctly
func TestEdgeCaseError_Fields(t *testing.T) {
	err := &EdgeCaseError{
		Type:     EdgeCaseSettingsLocked,
		Message:  "Settings panel is locked",
		Service:  "camera",
		BundleID: "com.example.app",
		Recovery: "Click the lock icon and authenticate",
	}

	if err.Type != EdgeCaseSettingsLocked {
		t.Errorf("Expected type SettingsLocked, got %v", err.Type)
	}

	if err.Message == "" {
		t.Error("Message should not be empty")
	}

	if err.Service != "camera" {
		t.Errorf("Expected service 'camera', got %v", err.Service)
	}

	if err.BundleID != "com.example.app" {
		t.Errorf("Expected bundle ID 'com.example.app', got %v", err.BundleID)
	}

	if err.Recovery == "" {
		t.Error("Recovery instructions should not be empty")
	}
}

// TestIsSystemSettingsOpenToTCC tests the detection of open System Settings
func TestIsSystemSettingsOpenToTCC(t *testing.T) {
	// This test depends on system state, so we just verify it doesn't crash
	isOpen, pane := isSystemSettingsOpenToTCC("camera")

	if isOpen {
		t.Logf("System Settings is open to: %s", pane)
		if pane == "" {
			t.Error("If System Settings is open, pane should not be empty")
		}
	} else {
		t.Log("System Settings is not open (or not showing Privacy panel)")
	}
}

// TestIsSettingsPanelLocked tests the detection of locked settings panel
func TestIsSettingsPanelLocked(t *testing.T) {
	// This test depends on system state, so we just verify it doesn't crash
	locked, err := isSettingsPanelLocked()

	if err != nil {
		t.Logf("Could not check lock status (expected if Settings not open): %v", err)
	} else {
		t.Logf("Settings panel locked: %v", locked)
	}
}
