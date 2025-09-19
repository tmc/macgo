package bundle

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		execPath string
		config   *Config
		wantErr  bool
	}{
		{
			name:     "minimal config",
			execPath: "/usr/bin/test",
			config:   nil,
			wantErr:  false,
		},
		{
			name:     "with app name",
			execPath: "/usr/bin/test",
			config: &Config{
				AppName: "TestApp",
			},
			wantErr: false,
		},
		{
			name:     "with bundle ID",
			execPath: "/usr/bin/test",
			config: &Config{
				BundleID: "com.example.test",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bundle, err := New(tt.execPath, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if bundle == nil {
				t.Errorf("New() returned nil bundle")
				return
			}
			if bundle.execPath != tt.execPath {
				t.Errorf("New() execPath = %v, want %v", bundle.execPath, tt.execPath)
			}
		})
	}
}


func TestBundle_Create(t *testing.T) {
	// Create a temporary executable for testing
	tempDir, err := os.MkdirTemp("", "bundle-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	execPath := filepath.Join(tempDir, "testexec")
	if err := os.WriteFile(execPath, []byte("#!/bin/bash\necho test"), 0755); err != nil {
		t.Fatal(err)
	}

	config := &Config{
		AppName:  "TestApp",
		BundleID: "com.example.test",
		Version:  "1.2.3",
		Debug:    true,
	}

	bundle, err := New(execPath, config)
	if err != nil {
		t.Fatal(err)
	}

	// Override the bundle path to use temp directory
	bundle.Path = filepath.Join(tempDir, "TestApp.app")

	err = bundle.Create()
	if err != nil {
		t.Fatal(err)
	}

	// Check bundle structure
	contentsDir := filepath.Join(bundle.Path, "Contents")
	macosDir := filepath.Join(contentsDir, "MacOS")
	plistPath := filepath.Join(contentsDir, "Info.plist")
	execInBundle := filepath.Join(macosDir, "TestApp")

	// Verify directory structure
	if _, err := os.Stat(contentsDir); err != nil {
		t.Errorf("Contents directory not created: %v", err)
	}
	if _, err := os.Stat(macosDir); err != nil {
		t.Errorf("MacOS directory not created: %v", err)
	}
	if _, err := os.Stat(plistPath); err != nil {
		t.Errorf("Info.plist not created: %v", err)
	}
	if _, err := os.Stat(execInBundle); err != nil {
		t.Errorf("Executable not copied: %v", err)
	}

	// Check executable permissions
	if info, err := os.Stat(execInBundle); err != nil {
		t.Errorf("Cannot stat executable: %v", err)
	} else if info.Mode()&0111 == 0 {
		t.Errorf("Executable is not executable: %v", info.Mode())
	}
}

func TestBundle_Validate(t *testing.T) {
	// Create a temporary bundle for testing
	tempDir, err := os.MkdirTemp("", "bundle-validate-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	bundlePath := filepath.Join(tempDir, "TestApp.app")
	contentsDir := filepath.Join(bundlePath, "Contents")
	macosDir := filepath.Join(contentsDir, "MacOS")

	bundle := &Bundle{
		Path: bundlePath,
		Config: &Config{
			AppName: "TestApp",
		},
		appName: "TestApp",
	}

	// Test validation with missing bundle
	err = bundle.Validate()
	if err == nil {
		t.Error("Validate() should fail for non-existent bundle")
	}

	// Create partial structure
	if err := os.MkdirAll(macosDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Test validation with missing Info.plist
	err = bundle.Validate()
	if err == nil {
		t.Error("Validate() should fail for bundle missing Info.plist")
	}

	// Create Info.plist
	plistPath := filepath.Join(contentsDir, "Info.plist")
	if err := os.WriteFile(plistPath, []byte(`<?xml version="1.0"?>
<plist version="1.0">
<dict>
	<key>CFBundleIdentifier</key>
	<string>com.example.test</string>
</dict>
</plist>`), 0644); err != nil {
		t.Fatal(err)
	}

	// Now validation should pass
	err = bundle.Validate()
	if err != nil {
		t.Errorf("Validate() should pass for complete bundle: %v", err)
	}
}

func TestBundle_Getters(t *testing.T) {
	config := &Config{
		AppName:  "TestApp",
		BundleID: "com.example.test",
		Version:  "1.2.3",
	}

	bundle := &Bundle{
		Path:     "/tmp/TestApp.app",
		Config:   config,
		appName:  "TestApp",
		bundleID: "com.example.test",
		version:  "1.2.3",
	}

	if bundle.CleanName() != "TestApp" {
		t.Errorf("CleanName() = %v, want TestApp", bundle.CleanName())
	}

	if bundle.BundleID() != "com.example.test" {
		t.Errorf("BundleID() = %v, want com.example.test", bundle.BundleID())
	}

	if bundle.Version() != "1.2.3" {
		t.Errorf("Version() = %v, want 1.2.3", bundle.Version())
	}

	expectedExecPath := "/tmp/TestApp.app/Contents/MacOS/TestApp"
	if bundle.ExecutablePath() != expectedExecPath {
		t.Errorf("ExecutablePath() = %v, want %v", bundle.ExecutablePath(), expectedExecPath)
	}
}

func TestConfig_shouldKeepBundle(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name:     "nil pointer defaults to true",
			config:   &Config{KeepBundle: nil},
			expected: true,
		},
		{
			name:     "explicit true",
			config:   &Config{KeepBundle: &[]bool{true}[0]},
			expected: true,
		},
		{
			name:     "explicit false",
			config:   &Config{KeepBundle: &[]bool{false}[0]},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.shouldKeepBundle()
			if result != tt.expected {
				t.Errorf("shouldKeepBundle() = %v, want %v", result, tt.expected)
			}
		})
	}
}

