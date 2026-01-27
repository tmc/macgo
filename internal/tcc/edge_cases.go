package tcc

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// EdgeCaseError represents a TCC permission edge case that requires special handling.
type EdgeCaseError struct {
	Type    EdgeCaseType
	Message string
	Service string
	BundleID string
	Recovery string // Recovery instructions for the user
}

// EdgeCaseType represents different types of TCC permission edge cases.
type EdgeCaseType int

const (
	// EdgeCaseUnknown represents an unknown edge case
	EdgeCaseUnknown EdgeCaseType = iota

	// EdgeCasePromptDismissed indicates the user dismissed the TCC prompt
	EdgeCasePromptDismissed

	// EdgeCasePermissionDenied indicates the user explicitly denied permission
	EdgeCasePermissionDenied

	// EdgeCaseMultipleDenials indicates permission was denied multiple times
	EdgeCaseMultipleDenials

	// EdgeCaseSettingsOpen indicates System Settings is already showing the TCC panel
	EdgeCaseSettingsOpen

	// EdgeCaseSettingsLocked indicates the TCC panel is locked (requires authentication)
	EdgeCaseSettingsLocked

	// EdgeCaseAppNotInList indicates the app isn't shown in System Settings TCC list
	EdgeCaseAppNotInList
)

func (e *EdgeCaseError) Error() string {
	return fmt.Sprintf("TCC edge case [%s]: %s\nRecovery: %s",
		e.Type.String(), e.Message, e.Recovery)
}

func (t EdgeCaseType) String() string {
	switch t {
	case EdgeCasePromptDismissed:
		return "PromptDismissed"
	case EdgeCasePermissionDenied:
		return "PermissionDenied"
	case EdgeCaseMultipleDenials:
		return "MultipleDenials"
	case EdgeCaseSettingsOpen:
		return "SettingsOpen"
	case EdgeCaseSettingsLocked:
		return "SettingsLocked"
	case EdgeCaseAppNotInList:
		return "AppNotInList"
	default:
		return "Unknown"
	}
}

// DetectEdgeCase attempts to detect common TCC permission edge cases.
// It checks System Settings state and provides appropriate recovery instructions.
func DetectEdgeCase(service, bundleID string) (*EdgeCaseError, error) {
	// Check if System Settings is already open to the TCC panel
	if isSettingsOpen, pane := isSystemSettingsOpenToTCC(service); isSettingsOpen {
		return &EdgeCaseError{
			Type:     EdgeCaseSettingsOpen,
			Message:  fmt.Sprintf("System Settings is already showing %s panel", service),
			Service:  service,
			BundleID: bundleID,
			Recovery: fmt.Sprintf("System Settings is already open to %s.\n"+
				"1. Check if the app is listed\n"+
				"2. If listed and unchecked, check the box to grant permission\n"+
				"3. If not listed, the app may need to trigger the prompt first\n"+
				"4. Try running the app again to trigger the TCC prompt", pane),
		}, nil
	}

	// Check if the settings panel is locked
	if isLocked, err := isSettingsPanelLocked(); err == nil && isLocked {
		return &EdgeCaseError{
			Type:     EdgeCaseSettingsLocked,
			Message:  "TCC settings panel is locked (requires authentication)",
			Service:  service,
			BundleID: bundleID,
			Recovery: "The privacy settings panel is locked.\n"+
				"1. Click the lock icon in System Settings\n"+
				"2. Authenticate with your administrator password\n"+
				"3. Then grant the permission",
		}, nil
	}

	return nil, nil
}

// isSystemSettingsOpenToTCC checks if System Settings is open and showing a TCC panel.
// Returns true and the pane name if open, false otherwise.
func isSystemSettingsOpenToTCC(service string) (bool, string) {
	// Use AppleScript to check if System Settings is running and which pane is visible
	script := `
tell application "System Events"
	if exists process "System Settings" then
		tell process "System Settings"
			if exists window 1 then
				try
					set windowTitle to title of window 1
					if windowTitle contains "Privacy" or windowTitle contains "Security" then
						return "open:" & windowTitle
					end if
				end try
			end if
		end tell
	end if
	return "closed"
end tell
`
	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		return false, ""
	}

	outputStr := strings.TrimSpace(string(output))
	if strings.HasPrefix(outputStr, "open:") {
		pane := strings.TrimPrefix(outputStr, "open:")
		return true, pane
	}

	return false, ""
}

// isSettingsPanelLocked checks if the TCC settings panel lock icon shows as locked.
func isSettingsPanelLocked() (bool, error) {
	script := `
tell application "System Events"
	if exists process "System Settings" then
		tell process "System Settings"
			if exists window 1 then
				try
					set lockButton to first button of window 1 whose description contains "lock"
					if exists lockButton then
						if description of lockButton contains "locked" then
							return "locked"
						else
							return "unlocked"
						end if
					end if
				end try
			end if
		end tell
	end if
	return "unknown"
end tell
`
	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(string(output)) == "locked", nil
}

// HandleEdgeCase provides user-friendly guidance for TCC permission edge cases.
// It analyzes the situation and provides clear recovery instructions.
func HandleEdgeCase(service, bundleID, appName string, debug bool) error {
	if debug {
		fmt.Fprintf(os.Stderr, "macgo: checking for TCC edge cases (service=%s, bundle=%s)\n", service, bundleID)
	}

	edgeCase, err := DetectEdgeCase(service, bundleID)
	if err != nil {
		if debug {
			fmt.Fprintf(os.Stderr, "macgo: error detecting edge case: %v\n", err)
		}
		return err
	}

	if edgeCase != nil {
		return edgeCase
	}

	return nil
}

// WaitForPermissionGrant waits for the user to grant permission in System Settings.
// Returns true if permission was granted, false if timeout or denied.
func WaitForPermissionGrant(service, bundleID string, timeout time.Duration, debug bool) (bool, error) {
	if debug {
		fmt.Fprintf(os.Stderr, "macgo: waiting for permission grant (timeout=%v)\n", timeout)
	}

	deadline := time.Now().Add(timeout)
	checkInterval := 1 * time.Second

	for time.Now().Before(deadline) {
		// Check if permission has been granted
		granted, err := checkPermissionStatus(service, bundleID)
		if err != nil {
			if debug {
				fmt.Fprintf(os.Stderr, "macgo: error checking permission: %v\n", err)
			}
		} else if granted {
			if debug {
				fmt.Fprintf(os.Stderr, "macgo: permission granted!\n")
			}
			return true, nil
		}

		// Check for edge cases that would prevent granting
		edgeCase, err := DetectEdgeCase(service, bundleID)
		if err == nil && edgeCase != nil {
			// Return the edge case as an error for the caller to handle
			return false, edgeCase
		}

		time.Sleep(checkInterval)
	}

	if debug {
		fmt.Fprintf(os.Stderr, "macgo: timeout waiting for permission grant\n")
	}

	return false, fmt.Errorf("timeout waiting for permission grant after %v", timeout)
}

// checkPermissionStatus checks if a permission has been granted using tccutil or similar.
// This is a helper function - actual implementation would use system APIs or command-line tools.
func checkPermissionStatus(service, bundleID string) (bool, error) {
	// For Camera, Microphone, and some other services, we can check using tccutil
	// Note: tccutil doesn't work for all services, and the output format varies by macOS version

	// Try using sqlite3 to query the TCC database directly (requires Full Disk Access)
	// This is more reliable but requires additional permissions
	tccDB := os.ExpandEnv("$HOME/Library/Application Support/com.apple.TCC/TCC.db")

	// Map common service names to TCC service identifiers
	serviceMap := map[string]string{
		"camera":           "kTCCServiceCamera",
		"microphone":       "kTCCServiceMicrophone",
		"screen-recording": "kTCCServiceScreenCapture",
		"accessibility":    "kTCCServiceAccessibility",
	}

	tccService, ok := serviceMap[strings.ToLower(service)]
	if !ok {
		// Unknown service, can't check status
		return false, fmt.Errorf("unknown TCC service: %s", service)
	}

	// Try to query the TCC database (this may fail without Full Disk Access)
	query := fmt.Sprintf("SELECT allowed FROM access WHERE service='%s' AND client='%s'", tccService, bundleID)
	cmd := exec.Command("sqlite3", tccDB, query)
	output, err := cmd.Output()
	if err != nil {
		// Can't check status (likely don't have Full Disk Access)
		return false, nil
	}

	// Parse the output: 1 means granted, 0 means denied
	allowed := strings.TrimSpace(string(output))
	return allowed == "1", nil
}

// OpenSystemSettingsToTCC opens System Settings to the appropriate TCC panel for the service.
// Returns an error with recovery instructions if edge cases are detected.
func OpenSystemSettingsToTCC(service, bundleID, appName string, debug bool) error {
	// Map service names to System Settings pane URLs
	paneURLs := map[string]string{
		"camera":           "x-apple.systempreferences:com.apple.preference.security?Privacy_Camera",
		"microphone":       "x-apple.systempreferences:com.apple.preference.security?Privacy_Microphone",
		"location":         "x-apple.systempreferences:com.apple.preference.security?Privacy_LocationServices",
		"screen-recording": "x-apple.systempreferences:com.apple.preference.security?Privacy_ScreenCapture",
		"accessibility":    "x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility",
		"automation":       "x-apple.systempreferences:com.apple.preference.security?Privacy_Automation",
		"files":            "x-apple.systempreferences:com.apple.preference.security?Privacy_AllFiles",
	}

	paneURL, ok := paneURLs[strings.ToLower(service)]
	if !ok {
		return fmt.Errorf("unknown TCC service: %s", service)
	}

	// Check for edge cases before opening
	if edgeCase, err := DetectEdgeCase(service, bundleID); err == nil && edgeCase != nil {
		// System Settings is already open - just provide guidance
		return edgeCase
	}

	// Open System Settings to the correct pane
	if debug {
		fmt.Fprintf(os.Stderr, "macgo: opening System Settings to %s panel\n", service)
	}

	cmd := exec.Command("open", paneURL)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open System Settings: %w", err)
	}

	// Wait a moment for System Settings to open
	time.Sleep(2 * time.Second)

	// Provide clear instructions
	fmt.Printf("\n")
	fmt.Printf("═══════════════════════════════════════════════════════════\n")
	fmt.Printf("  TCC Permission Required: %s\n", service)
	fmt.Printf("═══════════════════════════════════════════════════════════\n")
	fmt.Printf("\n")
	fmt.Printf("System Settings has been opened to the %s panel.\n", service)
	fmt.Printf("\n")
	fmt.Printf("To grant permission:\n")
	fmt.Printf("  1. Look for '%s' in the list\n", appName)
	fmt.Printf("  2. Check the box next to it to grant permission\n")
	fmt.Printf("  3. If the panel is locked, click the lock icon and authenticate\n")
	fmt.Printf("\n")
	fmt.Printf("If the app is not in the list:\n")
	fmt.Printf("  • The app may need to trigger the prompt first\n")
	fmt.Printf("  • Try running the app again after this setup\n")
	fmt.Printf("  • The prompt should appear automatically\n")
	fmt.Printf("\n")
	fmt.Printf("If you accidentally dismissed the prompt:\n")
	fmt.Printf("  • Check if the app appears in the list (it may be unchecked)\n")
	fmt.Printf("  • Simply check the box to grant permission\n")
	fmt.Printf("  • No need to remove and re-add the app\n")
	fmt.Printf("\n")
	fmt.Printf("If you denied permission multiple times:\n")
	fmt.Printf("  • The app should still appear in the list\n")
	fmt.Printf("  • Check the box to grant permission\n")
	fmt.Printf("  • You may need to restart the app\n")
	fmt.Printf("\n")
	fmt.Printf("═══════════════════════════════════════════════════════════\n")
	fmt.Printf("\n")

	return nil
}
