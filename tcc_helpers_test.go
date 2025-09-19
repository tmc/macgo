package macgo_test

import (
	"os"
	"strings"
	"testing"

	"github.com/tmc/misc/macgo"
)

func TestCheckFullDiskAccess(t *testing.T) {
	// This test will only pass if the test binary has Full Disk Access
	hasFDA := macgo.CheckFullDiskAccess()
	t.Logf("Full Disk Access: %v", hasFDA)

	// We can't assert a specific value since it depends on the environment
	// Just ensure the function doesn't panic
}

func TestKnownTCCServices(t *testing.T) {
	services := macgo.KnownTCCServices()

	if len(services) == 0 {
		t.Error("Expected at least some known TCC services")
	}

	// Check for essential services
	essentialServices := []string{
		"kTCCServiceCamera",
		"kTCCServiceMicrophone",
		"kTCCServiceLocation",
	}

	serviceMap := make(map[string]bool)
	for _, s := range services {
		serviceMap[s.Name] = true
	}

	for _, essential := range essentialServices {
		if !serviceMap[essential] {
			t.Errorf("Essential service %s not in known services", essential)
		}
	}
}

func TestGetServiceDescription(t *testing.T) {
	tests := []struct {
		service  string
		expected string
	}{
		{"kTCCServiceCamera", "Camera access"},
		{"kTCCServiceMicrophone", "Microphone access"},
		{"kTCCServiceLocation", "Location services"},
		{"kTCCServiceUnknown", "Unknown"}, // Should handle unknown services gracefully
	}

	for _, tt := range tests {
		t.Run(tt.service, func(t *testing.T) {
			desc := macgo.GetServiceDescription(tt.service)
			if desc == "" {
				t.Errorf("GetServiceDescription(%s) returned empty string", tt.service)
			}
			if tt.expected != "Unknown" && desc != tt.expected {
				t.Errorf("GetServiceDescription(%s) = %q, want %q", tt.service, desc, tt.expected)
			}
		})
	}
}

func TestFormatTCCEntries(t *testing.T) {
	entries := []macgo.TCCEntry{
		{
			Service:    "kTCCServiceCamera",
			Client:     "com.example.app",
			ClientType: 0,
			Auth:       2,
			Allowed:    true,
		},
		{
			Service:    "kTCCServiceMicrophone",
			Client:     "/usr/local/bin/myapp",
			ClientType: 1,
			Auth:       0,
			Allowed:    false,
		},
	}

	// Test JSON format
	jsonOutput, err := macgo.FormatTCCEntries(entries, "json")
	if err != nil {
		t.Fatalf("FormatTCCEntries with json failed: %v", err)
	}

	if !strings.Contains(jsonOutput, "kTCCServiceCamera") {
		t.Error("JSON output doesn't contain expected service")
	}

	// Test table format
	tableOutput, err := macgo.FormatTCCEntries(entries, "table")
	if err != nil {
		t.Fatalf("FormatTCCEntries with table failed: %v", err)
	}

	if !strings.Contains(tableOutput, "Service") || !strings.Contains(tableOutput, "Client") {
		t.Error("Table output doesn't contain expected headers")
	}

	// Test default format (should be table)
	defaultOutput, err := macgo.FormatTCCEntries(entries, "")
	if err != nil {
		t.Fatalf("FormatTCCEntries with default format failed: %v", err)
	}

	if defaultOutput != tableOutput {
		t.Error("Default format should be table")
	}

	// Test invalid format
	_, err = macgo.FormatTCCEntries(entries, "invalid")
	if err == nil {
		t.Error("Expected error for invalid format")
	}
}

func TestTCCDatabaseOperations(t *testing.T) {
	// Skip this test if we don't have Full Disk Access
	if !macgo.CheckFullDiskAccess() {
		t.Skip("Skipping TCC database test - Full Disk Access not available")
	}

	// Test opening database for reading
	db, err := macgo.OpenTCCDatabase()
	if err != nil {
		t.Fatalf("Failed to open TCC database: %v", err)
	}
	defer db.Close()

	// Test listing permissions (won't fail even if empty)
	entries, err := db.ListAllPermissions()
	if err != nil {
		t.Errorf("Failed to list permissions: %v", err)
	}

	t.Logf("Found %d TCC entries", len(entries))

	// Test permission status check
	status, err := db.GetPermissionStatus("kTCCServiceCamera", "com.example.test")
	if err != nil {
		t.Errorf("Failed to get permission status: %v", err)
	}

	if status.Service != "kTCCServiceCamera" {
		t.Errorf("Wrong service in status: got %s, want kTCCServiceCamera", status.Service)
	}
}

func TestCheckPermission(t *testing.T) {
	// This test may fail if Full Disk Access is not available
	// We'll just ensure it doesn't panic

	granted, err := macgo.CheckPermission("kTCCServiceCamera")
	t.Logf("Camera permission: granted=%v, err=%v", granted, err)

	// If we get an FDA error, that's expected
	if err != nil && strings.Contains(err.Error(), "Full Disk Access") {
		t.Log("Expected error: need Full Disk Access to check permissions")
		return
	}
}

func TestGetCurrentAppPermissions(t *testing.T) {
	// This test may fail if Full Disk Access is not available
	entries, err := macgo.GetCurrentAppPermissions()

	if err != nil {
		if strings.Contains(err.Error(), "Full Disk Access") {
			t.Skip("Skipping test - Full Disk Access not available")
		}
		t.Errorf("Failed to get current app permissions: %v", err)
	}

	t.Logf("Current app has %d permission entries", len(entries))
}

func TestGetContainingBundlePath(t *testing.T) {
	// Test when not in a bundle
	bundlePath := macgo.GetContainingBundlePath()

	// In test environment, we're likely not in a bundle
	if bundlePath != "" {
		t.Logf("Running from bundle: %s", bundlePath)
	} else {
		t.Log("Not running from a bundle (expected in test environment)")
	}
}

func TestPermissionQuery(t *testing.T) {
	pq, err := macgo.NewPermissionQuery()
	if err != nil {
		t.Fatalf("Failed to create PermissionQuery: %v", err)
	}

	// Test various permission checks
	// These will likely all return false in test environment
	perms := map[string]bool{
		"Camera":         pq.HasCameraAccess(),
		"Microphone":     pq.HasMicrophoneAccess(),
		"Location":       pq.HasLocationAccess(),
		"Contacts":       pq.HasContactsAccess(),
		"Calendar":       pq.HasCalendarAccess(),
		"Reminders":      pq.HasRemindersAccess(),
		"Photos":         pq.HasPhotosAccess(),
		"ScreenCapture":  pq.HasScreenCaptureAccess(),
		"FullDiskAccess": pq.HasFullDiskAccess(),
	}

	for name, granted := range perms {
		t.Logf("%s permission: %v", name, granted)
	}
}

func TestGetAllPermissions(t *testing.T) {
	status, err := macgo.GetAllPermissions()
	if err != nil {
		t.Fatalf("Failed to get all permissions: %v", err)
	}

	if status.CheckedAt.IsZero() {
		t.Error("CheckedAt time should not be zero")
	}

	// Log the status for debugging
	t.Logf("Permission status: Camera=%v, Microphone=%v, FDA=%v",
		status.Camera, status.Microphone, status.FullDiskAccess)
}

func TestResetServicePermission(t *testing.T) {
	// Skip this test if not running with appropriate permissions
	if os.Geteuid() != 0 && !macgo.CheckFullDiskAccess() {
		t.Skip("Skipping reset test - requires root or Full Disk Access")
	}

	// Try to reset a harmless permission
	// This might fail if we don't have the right permissions, which is OK
	err := macgo.ResetServicePermission("camera")
	if err != nil {
		t.Logf("Reset permission returned error (may be expected): %v", err)
	}
}