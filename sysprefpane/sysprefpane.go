// Package sysprefpane provides functions to open macOS System Settings panes.
package sysprefpane

import (
	"os/exec"
)

// Pane represents a System Settings pane identifier.
type Pane string

// Privacy panes accessible via the Security preference pane.
const (
	Accessibility    Pane = "Privacy_Accessibility"
	Automation       Pane = "Privacy_Automation"
	ScreenRecording  Pane = "Privacy_ScreenCapture"
	Camera           Pane = "Privacy_Camera"
	Microphone       Pane = "Privacy_Microphone"
	FullDiskAccess   Pane = "Privacy_AllFiles"
	FilesAndFolders  Pane = "Privacy_FilesAndFolders"
	InputMonitoring  Pane = "Privacy_ListenEvent"
	Location         Pane = "Privacy_LocationServices"
	Contacts         Pane = "Privacy_Contacts"
	Calendars        Pane = "Privacy_Calendars"
	Photos           Pane = "Privacy_Photos"
	DesktopFolder    Pane = "Privacy_DesktopFolder"
	DocumentsFolder  Pane = "Privacy_DocumentsFolder"
	DownloadsFolder  Pane = "Privacy_DownloadsFolder"
)

// Top-level panes.
const (
	Security Pane = "" // Opens Security & Privacy main pane
)

// URL returns the x-apple.systempreferences URL for the pane.
func (p Pane) URL() string {
	base := "x-apple.systempreferences:com.apple.preference.security"
	if p == "" {
		return base
	}
	return base + "?" + string(p)
}

// Open opens the pane in System Settings.
func (p Pane) Open() error {
	return exec.Command("open", p.URL()).Run()
}

// Open opens a System Settings pane by its identifier.
func Open(pane Pane) error {
	return pane.Open()
}

// URL returns the URL for a pane.
func URL(pane Pane) string {
	return pane.URL()
}
