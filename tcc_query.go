package macgo

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// PermissionQuery provides simpler TCC permission queries without SQL dependencies
type PermissionQuery struct {
	bundleID string
	execPath string
}

// NewPermissionQuery creates a new permission query helper for the current process
func NewPermissionQuery() (*PermissionQuery, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	pq := &PermissionQuery{
		execPath: execPath,
	}

	// Try to get bundle ID if we're in a bundle
	if bundlePath := GetContainingBundlePath(); bundlePath != "" {
		if cfg := getCurrentConfig(); cfg != nil && cfg.BundleID != "" {
			pq.bundleID = cfg.BundleID
		}
	}

	return pq, nil
}

// GetContainingBundlePath returns the path to the containing app bundle, if any
func GetContainingBundlePath() string {
	execPath, err := os.Executable()
	if err != nil {
		return ""
	}

	if !strings.Contains(execPath, ".app/") {
		return ""
	}

	// Find the .app directory
	parts := strings.Split(execPath, "/")
	for i, part := range parts {
		if strings.HasSuffix(part, ".app") {
			return strings.Join(parts[:i+1], "/")
		}
	}

	return ""
}

// HasCameraAccess checks if the app has camera access
func (pq *PermissionQuery) HasCameraAccess() bool {
	return pq.checkTCCAccess("kTCCServiceCamera")
}

// HasMicrophoneAccess checks if the app has microphone access
func (pq *PermissionQuery) HasMicrophoneAccess() bool {
	return pq.checkTCCAccess("kTCCServiceMicrophone")
}

// HasScreenCaptureAccess checks if the app has screen capture access
func (pq *PermissionQuery) HasScreenCaptureAccess() bool {
	return pq.checkTCCAccess("kTCCServiceScreenCapture")
}

// HasFullDiskAccess checks if the app has full disk access
func (pq *PermissionQuery) HasFullDiskAccess() bool {
	// Check by trying to access a protected location
	testPath := "/Library/Application Support/com.apple.TCC/TCC.db"
	file, err := os.Open(testPath)
	if err != nil {
		return false
	}
	file.Close()
	return true
}

// HasLocationAccess checks if the app has location services access
func (pq *PermissionQuery) HasLocationAccess() bool {
	return pq.checkTCCAccess("kTCCServiceLocation")
}

// HasContactsAccess checks if the app has contacts access
func (pq *PermissionQuery) HasContactsAccess() bool {
	return pq.checkTCCAccess("kTCCServiceAddressBook")
}

// HasCalendarAccess checks if the app has calendar access
func (pq *PermissionQuery) HasCalendarAccess() bool {
	return pq.checkTCCAccess("kTCCServiceCalendar")
}

// HasRemindersAccess checks if the app has reminders access
func (pq *PermissionQuery) HasRemindersAccess() bool {
	return pq.checkTCCAccess("kTCCServiceReminders")
}

// HasPhotosAccess checks if the app has photos access
func (pq *PermissionQuery) HasPhotosAccess() bool {
	return pq.checkTCCAccess("kTCCServicePhotos")
}

// checkTCCAccess is the internal method to check TCC access using tccutil
func (pq *PermissionQuery) checkTCCAccess(service string) bool {
	// This is a simplified check - in practice, we'd need to query the TCC database
	// or use a more sophisticated method. For now, we'll return false if we can't determine.

	// Try to use our TCC database helper if we have Full Disk Access
	if pq.HasFullDiskAccess() {
		if granted, err := CheckPermission(service); err == nil {
			return granted
		}
	}

	return false
}

// RequestCameraAccess triggers a camera access request dialog
func RequestCameraAccess() error {
	return Request(Camera)
}

// RequestMicrophoneAccess triggers a microphone access request dialog
func RequestMicrophoneAccess() error {
	return Request(Microphone)
}

// RequestLocationAccess triggers a location access request dialog
func RequestLocationAccess() error {
	return Request(Location)
}

// RequestFileAccess triggers file access permissions
func RequestFileAccess() error {
	return Request(Files)
}

// RequestNetworkAccess triggers network access permissions
func RequestNetworkAccess() error {
	return Request(Network)
}

// ResetPermissions resets all TCC permissions for the current app
func ResetPermissions() error {
	// Get the client identifier
	clientID := ""
	if pq, err := NewPermissionQuery(); err == nil {
		if pq.bundleID != "" {
			clientID = pq.bundleID
		} else {
			clientID = pq.execPath
		}
	}

	if clientID == "" {
		return fmt.Errorf("could not determine client identifier")
	}

	// Use tccutil to reset permissions
	cmd := exec.Command("tccutil", "reset", "All", clientID)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to reset permissions: %w (output: %s)", err, output)
	}

	return nil
}

// ResetServicePermission resets a specific TCC service permission for the current app
func ResetServicePermission(service string) error {
	// Get the client identifier
	clientID := ""
	if pq, err := NewPermissionQuery(); err == nil {
		if pq.bundleID != "" {
			clientID = pq.bundleID
		} else {
			clientID = pq.execPath
		}
	}

	if clientID == "" {
		return fmt.Errorf("could not determine client identifier")
	}

	// Map our permission constants to TCC service names
	tccService := ""
	switch service {
	case "camera":
		tccService = "Camera"
	case "microphone":
		tccService = "Microphone"
	case "location":
		tccService = "Location"
	case "contacts", "addressbook":
		tccService = "AddressBook"
	case "calendar":
		tccService = "Calendar"
	case "reminders":
		tccService = "Reminders"
	case "photos":
		tccService = "Photos"
	case "screencapture":
		tccService = "ScreenCapture"
	case "accessibility":
		tccService = "Accessibility"
	default:
		tccService = service // Use as-is if not mapped
	}

	// Use tccutil to reset specific service
	cmd := exec.Command("tccutil", "reset", tccService, clientID)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to reset %s permission: %w (output: %s)", service, err, output)
	}

	return nil
}

// PermissionStatus represents the status of all common permissions
type PermissionStatus struct {
	Camera         bool      `json:"camera"`
	Microphone     bool      `json:"microphone"`
	Location       bool      `json:"location"`
	Contacts       bool      `json:"contacts"`
	Calendar       bool      `json:"calendar"`
	Reminders      bool      `json:"reminders"`
	Photos         bool      `json:"photos"`
	ScreenCapture  bool      `json:"screen_capture"`
	FullDiskAccess bool      `json:"full_disk_access"`
	CheckedAt      time.Time `json:"checked_at"`
}

// GetAllPermissions returns the status of all common permissions
func GetAllPermissions() (*PermissionStatus, error) {
	pq, err := NewPermissionQuery()
	if err != nil {
		return nil, err
	}

	return &PermissionStatus{
		Camera:         pq.HasCameraAccess(),
		Microphone:     pq.HasMicrophoneAccess(),
		Location:       pq.HasLocationAccess(),
		Contacts:       pq.HasContactsAccess(),
		Calendar:       pq.HasCalendarAccess(),
		Reminders:      pq.HasRemindersAccess(),
		Photos:         pq.HasPhotosAccess(),
		ScreenCapture:  pq.HasScreenCaptureAccess(),
		FullDiskAccess: pq.HasFullDiskAccess(),
		CheckedAt:      time.Now(),
	}, nil
}

// WaitForPermission waits for a specific permission to be granted
func WaitForPermission(permission Permission, timeout time.Duration) error {
	pq, err := NewPermissionQuery()
	if err != nil {
		return err
	}

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		var hasPermission bool

		switch permission {
		case Camera:
			hasPermission = pq.HasCameraAccess()
		case Microphone:
			hasPermission = pq.HasMicrophoneAccess()
		case Location:
			hasPermission = pq.HasLocationAccess()
		case Files:
			hasPermission = pq.HasFullDiskAccess()
		default:
			return fmt.Errorf("unsupported permission type: %v", permission)
		}

		if hasPermission {
			return nil
		}

		select {
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for %v permission", permission)
			}
		}
	}
}

// getCurrentConfig returns the current macgo configuration if available
func getCurrentConfig() *Config {
	// This would need to be implemented to track the current config
	// For now, return nil
	return nil
}