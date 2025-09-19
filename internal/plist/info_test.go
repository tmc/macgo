package plist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteInfoPlist(t *testing.T) {
	tempDir := t.TempDir()
	plistPath := filepath.Join(tempDir, "Info.plist")

	cfg := InfoPlistConfig{
		AppName:  "TestApp",
		BundleID: "com.example.testapp",
		ExecName: "testapp",
		Version:  "1.0.0",
	}

	err := WriteInfoPlist(plistPath, cfg)
	if err != nil {
		t.Fatalf("WriteInfoPlist failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		t.Fatal("Info.plist file was not created")
	}

	// Read and verify content
	content, err := os.ReadFile(plistPath)
	if err != nil {
		t.Fatalf("Failed to read plist file: %v", err)
	}

	contentStr := string(content)

	// Check for required XML elements
	requiredElements := []string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<!DOCTYPE plist`,
		`<plist version="1.0">`,
		`<dict>`,
		`<key>CFBundleDisplayName</key>`,
		`<string>TestApp</string>`,
		`<key>CFBundleExecutable</key>`,
		`<string>testapp</string>`,
		`<key>CFBundleIdentifier</key>`,
		`<string>com.example.testapp</string>`,
		`<key>CFBundleName</key>`,
		`<string>TestApp</string>`,
		`<key>CFBundlePackageType</key>`,
		`<string>APPL</string>`,
		`<key>CFBundleVersion</key>`,
		`<string>1.0.0</string>`,
		`<key>CFBundleShortVersionString</key>`,
		`<string>1.0.0</string>`,
		`<key>LSUIElement</key>`,
		`<true/>`,
		`<key>NSHighResolutionCapable</key>`,
		`<true/>`,
		`</dict>`,
		`</plist>`,
	}

	for _, element := range requiredElements {
		if !strings.Contains(contentStr, element) {
			t.Errorf("Missing required element: %s", element)
		}
	}
}

func TestWriteInfoPlistWithCustomKeys(t *testing.T) {
	tempDir := t.TempDir()
	plistPath := filepath.Join(tempDir, "Info.plist")

	cfg := InfoPlistConfig{
		AppName:  "TestApp",
		BundleID: "com.example.testapp",
		ExecName: "testapp",
		Version:  "1.0.0",
		CustomKeys: map[string]interface{}{
			"LSUIElement":                          false, // Override default
			"NSHumanReadableCopyright":             "Copyright © 2024 Example Corp",
			"CFBundleGetInfoString":                "TestApp v1.0.0",
			"LSMinimumSystemVersion":               "10.15.0",
			"NSSupportsAutomaticGraphicsSwitching": true,
			"LSApplicationCategoryType":            []string{"public.app-category.productivity"},
		},
	}

	err := WriteInfoPlist(plistPath, cfg)
	if err != nil {
		t.Fatalf("WriteInfoPlist failed: %v", err)
	}

	content, err := os.ReadFile(plistPath)
	if err != nil {
		t.Fatalf("Failed to read plist file: %v", err)
	}

	contentStr := string(content)

	// Check custom keys
	customChecks := []string{
		`<key>LSUIElement</key>`,
		`<false/>`, // Should be false, not true
		`<key>NSHumanReadableCopyright</key>`,
		`<string>Copyright © 2024 Example Corp</string>`,
		`<key>CFBundleGetInfoString</key>`,
		`<string>TestApp v1.0.0</string>`,
		`<key>LSMinimumSystemVersion</key>`,
		`<string>10.15.0</string>`,
		`<key>NSSupportsAutomaticGraphicsSwitching</key>`,
		`<true/>`,
		`<key>LSApplicationCategoryType</key>`,
		`<array>`,
		`<string>public.app-category.productivity</string>`,
		`</array>`,
	}

	for _, check := range customChecks {
		if !strings.Contains(contentStr, check) {
			t.Errorf("Missing custom element: %s", check)
		}
	}
}

func TestValidateInfoPlistConfig(t *testing.T) {
	tests := []struct {
		name      string
		cfg       InfoPlistConfig
		shouldErr bool
		errorMsg  string
	}{
		{
			name: "valid config",
			cfg: InfoPlistConfig{
				AppName:  "TestApp",
				BundleID: "com.example.testapp",
				ExecName: "testapp",
				Version:  "1.0.0",
			},
			shouldErr: false,
		},
		{
			name: "missing app name",
			cfg: InfoPlistConfig{
				BundleID: "com.example.testapp",
				ExecName: "testapp",
				Version:  "1.0.0",
			},
			shouldErr: true,
			errorMsg:  "app name is required",
		},
		{
			name: "missing bundle ID",
			cfg: InfoPlistConfig{
				AppName:  "TestApp",
				ExecName: "testapp",
				Version:  "1.0.0",
			},
			shouldErr: true,
			errorMsg:  "bundle ID is required",
		},
		{
			name: "missing executable name",
			cfg: InfoPlistConfig{
				AppName:  "TestApp",
				BundleID: "com.example.testapp",
				Version:  "1.0.0",
			},
			shouldErr: true,
			errorMsg:  "executable name is required",
		},
		{
			name: "missing version",
			cfg: InfoPlistConfig{
				AppName:  "TestApp",
				BundleID: "com.example.testapp",
				ExecName: "testapp",
			},
			shouldErr: true,
			errorMsg:  "version is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInfoPlistConfig(tt.cfg)
			if tt.shouldErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestGenerateDefaultBundleID(t *testing.T) {
	tests := []struct {
		name    string
		appName string
	}{
		{
			name:    "simple name",
			appName: "MyApp",
		},
		{
			name:    "name with spaces",
			appName: "My Great App",
		},
		{
			name:    "name with hyphens and underscores",
			appName: "my-cool_app",
		},
		{
			name:    "name with special characters",
			appName: "My App! @#$%",
		},
		{
			name:    "name with numbers",
			appName: "App2024",
		},
		{
			name:    "empty name",
			appName: "",
		},
		{
			name:    "name with only special characters",
			appName: "!@#$%",
		},
		{
			name:    "mixed case with numbers",
			appName: "TestApp123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateDefaultBundleID(tt.appName)

			// Should return a valid bundle ID (contains at least one dot)
			if !strings.Contains(result, ".") {
				t.Errorf("GenerateDefaultBundleID(%q) = %q, should contain at least one dot", tt.appName, result)
			}

			// Should not be empty
			if result == "" {
				t.Errorf("GenerateDefaultBundleID(%q) returned empty string", tt.appName)
			}

			// Should not contain com.macgo anymore
			if strings.Contains(result, "com.macgo") {
				t.Errorf("GenerateDefaultBundleID(%q) = %q, should not contain 'com.macgo'", tt.appName, result)
			}
		})
	}
}

func TestInfoPlistXMLEscaping(t *testing.T) {
	tempDir := t.TempDir()
	plistPath := filepath.Join(tempDir, "Info.plist")

	cfg := InfoPlistConfig{
		AppName:  "Test & App <XML>",
		BundleID: "com.example.test&app",
		ExecName: "test-app",
		Version:  "1.0.0 \"beta\"",
	}

	err := WriteInfoPlist(plistPath, cfg)
	if err != nil {
		t.Fatalf("WriteInfoPlist failed: %v", err)
	}

	content, err := os.ReadFile(plistPath)
	if err != nil {
		t.Fatalf("Failed to read plist file: %v", err)
	}

	contentStr := string(content)

	// Check that special characters are properly escaped
	expectedEscapes := []string{
		`<string>Test &amp; App &lt;XML&gt;</string>`,
		`<string>com.example.test&amp;app</string>`,
		`<string>1.0.0 &quot;beta&quot;</string>`,
	}

	for _, escape := range expectedEscapes {
		if !strings.Contains(contentStr, escape) {
			t.Errorf("Missing escaped content: %s", escape)
		}
	}

	// Check that unescaped characters are not present
	forbiddenStrings := []string{
		`<string>Test & App <XML></string>`,
		`<string>com.example.test&app</string>`,
		`<string>1.0.0 "beta"</string>`,
	}

	for _, forbidden := range forbiddenStrings {
		if strings.Contains(contentStr, forbidden) {
			t.Errorf("Found unescaped content: %s", forbidden)
		}
	}
}
