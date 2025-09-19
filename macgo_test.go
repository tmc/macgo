package macgo

import (
	"os"
	"runtime"
	"testing"

	"github.com/tmc/misc/macgo/internal/system"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want string // Description of expected behavior
	}{
		{
			name: "zero config is valid",
			cfg:  &Config{},
			want: "should work with defaults",
		},
		{
			name: "with permissions",
			cfg:  &Config{Permissions: []Permission{Camera, Microphone}},
			want: "should request camera and microphone",
		},
		{
			name: "with custom entitlements",
			cfg:  &Config{Custom: []string{"com.apple.security.device.bluetooth"}},
			want: "should include custom entitlements",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that config is valid (doesn't panic)
			if tt.cfg == nil {
				t.Fatal("config should not be nil")
			}

			// Test builder pattern
			cfg := new(Config).WithPermissions(Camera).WithDebug()
			if len(cfg.Permissions) != 1 || cfg.Permissions[0] != Camera {
				t.Error("WithPermissions should add camera permission")
			}
			if !cfg.Debug {
				t.Error("WithDebug should enable debug")
			}
		})
	}
}

func TestFromEnv(t *testing.T) {
	// Save original environment
	originalEnv := map[string]string{
		"MACGO_APP_NAME":   os.Getenv("MACGO_APP_NAME"),
		"MACGO_BUNDLE_ID":  os.Getenv("MACGO_BUNDLE_ID"),
		"MACGO_DEBUG":      os.Getenv("MACGO_DEBUG"),
		"MACGO_CAMERA":     os.Getenv("MACGO_CAMERA"),
		"MACGO_MICROPHONE": os.Getenv("MACGO_MICROPHONE"),
	}
	defer func() {
		for k, v := range originalEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	// Test environment loading
	os.Setenv("MACGO_APP_NAME", "TestApp")
	os.Setenv("MACGO_BUNDLE_ID", "com.test.app")
	os.Setenv("MACGO_DEBUG", "1")
	os.Setenv("MACGO_CAMERA", "1")
	os.Setenv("MACGO_MICROPHONE", "1")

	cfg := new(Config).FromEnv()

	if cfg.AppName != "TestApp" {
		t.Errorf("expected AppName=TestApp, got %s", cfg.AppName)
	}
	if cfg.BundleID != "com.test.app" {
		t.Errorf("expected BundleID=com.test.app, got %s", cfg.BundleID)
	}
	if !cfg.Debug {
		t.Error("expected Debug=true")
	}

	expectedPerms := []Permission{Camera, Microphone}
	if len(cfg.Permissions) != len(expectedPerms) {
		t.Errorf("expected %d permissions, got %d", len(expectedPerms), len(cfg.Permissions))
	}

	for i, expected := range expectedPerms {
		if i >= len(cfg.Permissions) || cfg.Permissions[i] != expected {
			t.Errorf("expected permission %s at index %d", expected, i)
		}
	}
}

func TestStartOnNonDarwin(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("skipping non-darwin test on darwin")
	}

	// Should be no-op on non-darwin platforms
	err := Start(&Config{Permissions: []Permission{Camera}})
	if err != nil {
		t.Errorf("Start should be no-op on non-darwin, got error: %v", err)
	}

	err = Request(Camera, Microphone)
	if err != nil {
		t.Errorf("Request should be no-op on non-darwin, got error: %v", err)
	}

	err = Auto()
	if err != nil {
		t.Errorf("Auto should be no-op on non-darwin, got error: %v", err)
	}
}

func TestPermissionConstants(t *testing.T) {
	// Test that permission constants are reasonable
	permissions := []Permission{Camera, Microphone, Location, Files, Network, Sandbox}

	for _, perm := range permissions {
		if string(perm) == "" {
			t.Errorf("permission %s should not be empty", perm)
		}
		if len(string(perm)) > 50 {
			t.Errorf("permission %s is too long: %s", perm, string(perm))
		}
	}
}

func TestCleanAppName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal", "normal"},
		{"with/slash", "with-slash"},
		{"with\\backslash", "with-backslash"},
		{"with:colon", "with-colon"},
		{"with*asterisk", "with-asterisk"},
		{"with?question", "with-question"},
		{"with\"quote", "with-quote"},
		{"with<bracket", "with-bracket"},
		{"with>bracket", "with-bracket"},
		{"with|pipe", "with-pipe"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := system.CleanAppName(tt.input)
			if result != tt.expected {
				t.Errorf("CleanAppName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestInferBundleID(t *testing.T) {
	tests := []struct {
		name     string
		appName  string
		expected string // We'll check that it contains this substring
	}{
		{"valid_app_name", "myapp", "myapp"},
		{"hyphenated_name", "test-app", "test-app"},
		{"empty_name", "", "app"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := system.InferBundleID(tt.appName)

			// Should always contain dots (reverse DNS format)
			if !contains(result, ".") {
				t.Errorf("InferBundleID(%q) = %q, should contain dots for bundle ID format", tt.appName, result)
			}

			// Should contain the expected app name component
			if !contains(result, tt.expected) {
				t.Errorf("InferBundleID(%q) = %q, should contain %q", tt.appName, result, tt.expected)
			}

			// Should not contain the old "com.macgo" prefix
			if contains(result, "com.macgo") {
				t.Errorf("InferBundleID(%q) = %q, should not contain old 'com.macgo' prefix", tt.appName, result)
			}

			// Should be a valid bundle ID format
			if err := system.ValidateBundleID(result); err != nil {
				t.Errorf("InferBundleID(%q) = %q, invalid bundle ID: %v", tt.appName, result, err)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsInMiddle(s, substr))))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// BenchmarkStart measures the performance of the Start function.
func BenchmarkStart(b *testing.B) {
	if runtime.GOOS != "darwin" {
		b.Skip("benchmark only relevant on darwin")
	}

	cfg := &Config{
		Permissions: []Permission{Camera},
		Debug:       false,
	}

	// Set environment to prevent actual relaunch
	originalEnv := os.Getenv("MACGO_NO_RELAUNCH")
	os.Setenv("MACGO_NO_RELAUNCH", "1")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("MACGO_NO_RELAUNCH")
		} else {
			os.Setenv("MACGO_NO_RELAUNCH", originalEnv)
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Start(cfg)
	}
}
