package sysprefs

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OpenSystemPreferences attempts to open macOS Privacy & Security settings.
// Tries to open Full Disk Access directly, falls back to general Privacy pane.
// Useful when your app needs Full Disk Access or other manual permission grants.
func OpenSystemPreferences() error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("system preferences only available on macOS")
	}

	// Try opening the Full Disk Access pane directly
	cmd := exec.Command("open", "x-apple.systempreferences:com.apple.preference.security?Privacy_AllFiles")
	if err := cmd.Run(); err != nil {
		// Fallback to general Privacy & Security
		cmd = exec.Command("open", "x-apple.systempreferences:com.apple.preference.security")
		return cmd.Run()
	}
	return nil
}

// ShowFullDiskAccessInstructions provides instructions for granting Full Disk Access.
// Optionally opens System Preferences if openSettings is true.
func ShowFullDiskAccessInstructions(openSettings bool) {
	if openSettings {
		// Open System Settings
		_ = OpenSystemPreferences()
	}
}
