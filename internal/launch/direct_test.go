package launch

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDirectLauncher_getBundleExecutablePath(t *testing.T) {
	tests := []struct {
		name       string
		bundlePath string
		execPath   string
		config     *Config
		want       string
		wantErr    bool
	}{
		{
			name:       "app name specified in config",
			bundlePath: "/path/to/MyApp.app",
			execPath:   "/bin/some-executable",
			config: &Config{
				AppName: "CustomName",
			},
			want: "/path/to/MyApp.app/Contents/MacOS/CustomName",
		},
		{
			name:       "app name from executable path",
			bundlePath: "/path/to/MyApp.app",
			execPath:   "/bin/my-binary",
			config:     &Config{},
			want:       "/path/to/MyApp.app/Contents/MacOS/my-binary",
		},
		{
			name:       "app name from executable with extension",
			bundlePath: "/path/to/MyApp.app",
			execPath:   "/bin/my-binary.exe",
			config:     &Config{},
			want:       "/path/to/MyApp.app/Contents/MacOS/my-binary",
		},
		{
			name:       "app name from config with path",
			bundlePath: "/path/to/MyApp.app",
			execPath:   "/bin/some-executable",
			config: &Config{
				AppName: "/some/path/AppName",
			},
			want: "/path/to/MyApp.app/Contents/MacOS/AppName",
		},
	}

	launcher := &DirectLauncher{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := launcher.getBundleExecutablePath(tt.bundlePath, tt.execPath, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("DirectLauncher.getBundleExecutablePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DirectLauncher.getBundleExecutablePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDirectLauncher_getBundleExecutablePath_EmptyName(t *testing.T) {
	launcher := &DirectLauncher{}
	config := &Config{
		AppName: "",
	}

	// Test with empty executable path (edge case)
	_, err := launcher.getBundleExecutablePath("/path/to/App.app", "", config)
	if err == nil {
		t.Error("Expected error for empty executable name, got nil")
	}

	if !strings.Contains(err.Error(), "could not determine executable name") {
		t.Errorf("Expected error about executable name, got: %v", err)
	}
}

// TestDirectLauncher_Launch_Integration tests the launcher with a mock setup
// This test doesn't actually execute anything but verifies the command setup
func TestDirectLauncher_Launch_Integration(t *testing.T) {
	// Skip this test if we can't create temporary files
	tmpDir, err := os.MkdirTemp("", "macgo-test-*")
	if err != nil {
		t.Skipf("Cannot create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a mock bundle structure
	bundlePath := filepath.Join(tmpDir, "TestApp.app")
	macosDir := filepath.Join(bundlePath, "Contents", "MacOS")
	if err := os.MkdirAll(macosDir, 0755); err != nil {
		t.Fatalf("Failed to create bundle structure: %v", err)
	}

	// Create a mock executable (just an empty file for path testing)
	execName := "testapp"
	bundleExec := filepath.Join(macosDir, execName)
	if err := os.WriteFile(bundleExec, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("Failed to create mock executable: %v", err)
	}

	launcher := &DirectLauncher{}
	config := &Config{
		AppName: execName,
		Debug:   false, // Disable debug to avoid output in tests
	}

	// Test getBundleExecutablePath works correctly
	got, err := launcher.getBundleExecutablePath(bundlePath, "/some/path/"+execName, config)
	if err != nil {
		t.Fatalf("getBundleExecutablePath failed: %v", err)
	}

	expected := bundleExec
	if got != expected {
		t.Errorf("getBundleExecutablePath() = %v, want %v", got, expected)
	}

	// Verify the file exists at the expected path
	if _, err := os.Stat(got); err != nil {
		t.Errorf("Bundle executable does not exist at %s: %v", got, err)
	}
}

func TestDirectLauncher_Launch_NonExistentExecutable(t *testing.T) {
	launcher := &DirectLauncher{}
	config := &Config{
		AppName: "nonexistent",
		Debug:   false,
	}

	ctx := context.Background()
	err := launcher.Launch(ctx, "/nonexistent/bundle.app", "/nonexistent/exec", config)

	if err == nil {
		t.Error("Expected error for nonexistent bundle executable, got nil")
	}

	if !strings.Contains(err.Error(), "bundle executable not found") {
		t.Errorf("Expected error about bundle executable not found, got: %v", err)
	}
}
