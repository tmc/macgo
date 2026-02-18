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
		`<key>NSHighResolutionCapable</key>`,
		`<true/>`,
		`</dict>`,
		`</plist>`,
	}

	// LSUIElement should NOT be present when BackgroundOnly is false
	// and no CustomKeys override it (this is UIModeRegular behavior).
	absentElements := []string{
		`<key>LSUIElement</key>`,
		`<key>LSBackgroundOnly</key>`,
	}

	for _, element := range requiredElements {
		if !strings.Contains(contentStr, element) {
			t.Errorf("Missing required element: %s", element)
		}
	}

	for _, element := range absentElements {
		if strings.Contains(contentStr, element) {
			t.Errorf("Unexpected element present: %s", element)
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

// TestInfoPlistBackgroundModes tests the LSBackgroundOnly and LSUIElement combinations
func TestInfoPlistBackgroundModes(t *testing.T) {
	tests := []struct {
		name             string
		backgroundOnly   bool
		showInDockEnv    string
		wantLSBackground bool   // expect LSBackgroundOnly=true
		wantLSUIElement  *bool  // nil=absent, true/false=present with value
	}{
		{
			name:             "default: no LSUIElement or LSBackgroundOnly (regular app)",
			backgroundOnly:   false,
			showInDockEnv:    "",
			wantLSBackground: false,
			wantLSUIElement:  nil,
		},
		{
			name:             "BackgroundOnly=true: LSBackgroundOnly=true (no LSUIElement)",
			backgroundOnly:   true,
			showInDockEnv:    "",
			wantLSBackground: true,
			wantLSUIElement:  nil, // should not be present
		},
		{
			name:             "BackgroundOnly=true ignores SHOW_IN_DOCK",
			backgroundOnly:   true,
			showInDockEnv:    "1",
			wantLSBackground: true,
			wantLSUIElement:  nil, // should not be present even with SHOW_IN_DOCK
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set/unset environment variable
			if tt.showInDockEnv != "" {
				os.Setenv("MACGO_SHOW_IN_DOCK", tt.showInDockEnv)
				defer os.Unsetenv("MACGO_SHOW_IN_DOCK")
			} else {
				os.Unsetenv("MACGO_SHOW_IN_DOCK")
			}

			tempDir := t.TempDir()
			plistPath := filepath.Join(tempDir, "Info.plist")

			cfg := InfoPlistConfig{
				AppName:        "TestApp",
				BundleID:       "com.example.testapp",
				ExecName:       "testapp",
				Version:        "1.0.0",
				BackgroundOnly: tt.backgroundOnly,
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

			// Check LSBackgroundOnly
			hasLSBackgroundOnly := strings.Contains(contentStr, "<key>LSBackgroundOnly</key>")
			hasLSBackgroundOnlyTrue := strings.Contains(contentStr, "<key>LSBackgroundOnly</key>") &&
				strings.Contains(contentStr, "<true/>")
			if tt.wantLSBackground {
				if !hasLSBackgroundOnly {
					t.Errorf("Expected LSBackgroundOnly but not found")
				}
				if !hasLSBackgroundOnlyTrue {
					t.Errorf("LSBackgroundOnly should be true")
				}
			} else {
				if hasLSBackgroundOnly {
					t.Errorf("Did not expect LSBackgroundOnly but found it")
				}
			}

			// Check LSUIElement
			hasLSUIElement := strings.Contains(contentStr, "<key>LSUIElement</key>")
			if tt.wantLSUIElement == nil {
				if hasLSUIElement {
					t.Errorf("Expected no LSUIElement but found it")
				}
			} else {
				if !hasLSUIElement {
					t.Errorf("Expected LSUIElement but not found")
				}
				// Check the value follows the key
				expectedValue := "<true/>"
				if !*tt.wantLSUIElement {
					expectedValue = "<false/>"
				}
				// Find LSUIElement key and check next line has expected value
				lines := strings.Split(contentStr, "\n")
				found := false
				for i, line := range lines {
					if strings.Contains(line, "<key>LSUIElement</key>") && i+1 < len(lines) {
						if strings.Contains(lines[i+1], expectedValue) {
							found = true
							break
						}
					}
				}
				if !found {
					t.Errorf("LSUIElement should be %v", *tt.wantLSUIElement)
				}
			}

			t.Logf("Content:\n%s", contentStr)
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
