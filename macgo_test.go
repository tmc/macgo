package macgo

import (
	"os"
	"runtime"
	"testing"

	"github.com/tmc/macgo/internal/system"
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
		"MACGO_APP_NAME":                        os.Getenv("MACGO_APP_NAME"),
		"MACGO_BUNDLE_ID":                       os.Getenv("MACGO_BUNDLE_ID"),
		"MACGO_DEBUG":                           os.Getenv("MACGO_DEBUG"),
		"MACGO_LOCAL_NETWORK_USAGE_DESCRIPTION": os.Getenv("MACGO_LOCAL_NETWORK_USAGE_DESCRIPTION"),
		"MACGO_BONJOUR_SERVICES":                os.Getenv("MACGO_BONJOUR_SERVICES"),
		"MACGO_CAMERA":                          os.Getenv("MACGO_CAMERA"),
		"MACGO_MICROPHONE":                      os.Getenv("MACGO_MICROPHONE"),
		"MACGO_PROVISIONING_PROFILE":            os.Getenv("MACGO_PROVISIONING_PROFILE"),
		"MACGO_ICON":                            os.Getenv("MACGO_ICON"),
	}
	defer func() {
		for k, v := range originalEnv {
			if v == "" {
				_ = os.Unsetenv(k)
			} else {
				_ = os.Setenv(k, v)
			}
		}
	}()

	// Test environment loading
	_ = os.Setenv("MACGO_APP_NAME", "TestApp")
	_ = os.Setenv("MACGO_BUNDLE_ID", "com.test.app")
	_ = os.Setenv("MACGO_DEBUG", "1")
	_ = os.Setenv("MACGO_LOCAL_NETWORK_USAGE_DESCRIPTION", "Discover nearby peers")
	_ = os.Setenv("MACGO_BONJOUR_SERVICES", "_test._tcp,_test._udp")
	_ = os.Setenv("MACGO_CAMERA", "1")
	_ = os.Setenv("MACGO_MICROPHONE", "1")
	_ = os.Setenv("MACGO_PROVISIONING_PROFILE", "/tmp/test.provisionprofile")
	_ = os.Setenv("MACGO_ICON", "/tmp/test.icns")

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
	if cfg.LocalNetworkUsageDescription != "Discover nearby peers" {
		t.Errorf("expected LocalNetworkUsageDescription to be loaded from env, got %q", cfg.LocalNetworkUsageDescription)
	}
	if len(cfg.BonjourServices) != 2 {
		t.Fatalf("expected 2 bonjour services, got %d", len(cfg.BonjourServices))
	}
	if cfg.BonjourServices[0] != "_test._tcp" || cfg.BonjourServices[1] != "_test._udp" {
		t.Errorf("unexpected bonjour services: %#v", cfg.BonjourServices)
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

	if cfg.ProvisioningProfile != "/tmp/test.provisionprofile" {
		t.Errorf("expected ProvisioningProfile=/tmp/test.provisionprofile, got %s", cfg.ProvisioningProfile)
	}
	if cfg.IconPath != "/tmp/test.icns" {
		t.Errorf("expected IconPath=/tmp/test.icns, got %s", cfg.IconPath)
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

func TestConfigPrepareDefaultIdentity(t *testing.T) {
	cfg := NewConfig()

	cfg.prepare("/tmp/peer-tool")

	if cfg.AppName != "peer-tool" {
		t.Fatalf("prepare should infer AppName from executable, got %q", cfg.AppName)
	}
	if cfg.BundleID == "" {
		t.Fatal("prepare should infer BundleID when unset")
	}
}

func TestConfigPrepareCameraUsage(t *testing.T) {
	cfg := NewConfig().WithCameraUsage("Take pictures.")

	cfg.prepare("/tmp/camera-tool")

	if len(cfg.Permissions) != 1 || cfg.Permissions[0] != Camera {
		t.Fatalf("prepare should auto-enable Camera permission, got %#v", cfg.Permissions)
	}
	if cfg.Info == nil {
		t.Fatal("prepare should initialize Info map for camera usage")
	}
	if got := cfg.Info["NSCameraUsageDescription"]; got != "Take pictures." {
		t.Fatalf("unexpected NSCameraUsageDescription: %#v", got)
	}
}

func TestConfigPrepareMicrophoneUsage(t *testing.T) {
	cfg := NewConfig().WithMicrophoneUsage("Record audio.")

	cfg.prepare("/tmp/mic-tool")

	if len(cfg.Permissions) != 1 || cfg.Permissions[0] != Microphone {
		t.Fatalf("prepare should auto-enable Microphone permission, got %#v", cfg.Permissions)
	}
	if cfg.Info == nil {
		t.Fatal("prepare should initialize Info map for microphone usage")
	}
	if got := cfg.Info["NSMicrophoneUsageDescription"]; got != "Record audio." {
		t.Fatalf("unexpected NSMicrophoneUsageDescription: %#v", got)
	}
}

func TestConfigPreparePermissionDefaultDescriptions(t *testing.T) {
	cfg := NewConfig().
		WithAppName("MediaTool").
		WithPermissions(Camera, Microphone)

	cfg.prepare("/tmp/media-tool")

	cameraDescription, ok := cfg.Info["NSCameraUsageDescription"].(string)
	if !ok || cameraDescription == "" {
		t.Fatal("prepare should add a default camera usage description")
	}
	if !contains(cameraDescription, "MediaTool") {
		t.Fatalf("default camera usage description should mention app name, got %q", cameraDescription)
	}

	microphoneDescription, ok := cfg.Info["NSMicrophoneUsageDescription"].(string)
	if !ok || microphoneDescription == "" {
		t.Fatal("prepare should add a default microphone usage description")
	}
	if !contains(microphoneDescription, "MediaTool") {
		t.Fatalf("default microphone usage description should mention app name, got %q", microphoneDescription)
	}
}

func TestConfigPrepareLocalNetwork(t *testing.T) {
	cfg := NewConfig().
		WithBonjourServices("_peer-tool._tcp", "_peer-tool._tcp", " ").
		WithLocalNetworkUsage("PeerTool discovers peers on the local network.")

	cfg.prepare("/tmp/peer-tool")

	if len(cfg.Permissions) != 1 || cfg.Permissions[0] != Network {
		t.Fatalf("prepare should auto-enable Network permission, got %#v", cfg.Permissions)
	}
	if cfg.Info == nil {
		t.Fatal("prepare should initialize Info map for local network settings")
	}
	if got := cfg.Info["NSLocalNetworkUsageDescription"]; got != "PeerTool discovers peers on the local network." {
		t.Fatalf("unexpected NSLocalNetworkUsageDescription: %#v", got)
	}
	services, ok := cfg.Info["NSBonjourServices"].([]string)
	if !ok {
		t.Fatalf("NSBonjourServices should be []string, got %T", cfg.Info["NSBonjourServices"])
	}
	if len(services) != 1 || services[0] != "_peer-tool._tcp" {
		t.Fatalf("unexpected NSBonjourServices: %#v", services)
	}
}

func TestConfigPrepareBonjourServicesDefaultDescription(t *testing.T) {
	cfg := NewConfig().
		WithAppName("PeerTool").
		WithBonjourServices("_peer-tool._tcp")

	cfg.prepare("/tmp/peer-tool")

	description, ok := cfg.Info["NSLocalNetworkUsageDescription"].(string)
	if !ok || description == "" {
		t.Fatal("prepare should add a default local network usage description")
	}
	if !contains(description, "PeerTool") {
		t.Fatalf("default local network usage description should mention app name, got %q", description)
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
	_ = os.Setenv("MACGO_NO_RELAUNCH", "1")
	defer func() {
		if originalEnv == "" {
			_ = os.Unsetenv("MACGO_NO_RELAUNCH")
		} else {
			_ = os.Setenv("MACGO_NO_RELAUNCH", originalEnv)
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Start(cfg)
	}
}
