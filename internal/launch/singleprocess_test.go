package launch

import (
	"os"
	"testing"
)

func TestSingleProcessLauncher_writeEntitlements(t *testing.T) {
	launcher := &SingleProcessLauncher{logger: NewLogger()}

	tests := []struct {
		name        string
		cfg         *Config
		wantStrings []string
	}{
		{
			name: "camera permission",
			cfg: &Config{
				Permissions: []string{"camera"},
			},
			wantStrings: []string{
				"com.apple.security.device.camera",
			},
		},
		{
			name: "multiple permissions",
			cfg: &Config{
				Permissions: []string{"camera", "microphone", "network"},
			},
			wantStrings: []string{
				"com.apple.security.device.camera",
				"com.apple.security.device.microphone",
				"com.apple.security.network.client",
			},
		},
		{
			name: "custom entitlements",
			cfg: &Config{
				Entitlements: []string{"com.apple.security.virtualization"},
			},
			wantStrings: []string{
				"com.apple.security.virtualization",
			},
		},
		{
			name: "empty config gets get-task-allow",
			cfg:  &Config{},
			wantStrings: []string{
				"com.apple.security.get-task-allow",
			},
		},
		{
			name: "mixed permissions and entitlements",
			cfg: &Config{
				Permissions:  []string{"camera"},
				Entitlements: []string{"com.apple.security.virtualization"},
			},
			wantStrings: []string{
				"com.apple.security.device.camera",
				"com.apple.security.virtualization",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := launcher.writeEntitlements(tt.cfg)
			if err != nil {
				t.Fatalf("writeEntitlements() error: %v", err)
			}
			defer os.Remove(path)

			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile() error: %v", err)
			}
			content := string(data)

			for _, want := range tt.wantStrings {
				if !containsString(content, want) {
					t.Errorf("entitlements missing %q\ncontent:\n%s", want, content)
				}
			}

			// Verify valid plist structure
			if !containsString(content, "<?xml") {
				t.Error("missing XML declaration")
			}
			if !containsString(content, "<plist") {
				t.Error("missing plist element")
			}
			if !containsString(content, "<dict>") {
				t.Error("missing dict element")
			}
		})
	}
}

func TestPermissionToEntitlement(t *testing.T) {
	tests := []struct {
		perm string
		want string
	}{
		{"camera", "com.apple.security.device.camera"},
		{"microphone", "com.apple.security.device.microphone"},
		{"location", "com.apple.security.personal-information.location"},
		{"sandbox", "com.apple.security.app-sandbox"},
		{"files", "com.apple.security.files.user-selected.read-only"},
		{"network", "com.apple.security.network.client"},
		{"screen-recording", "com.apple.security.screen-capture"},
		{"accessibility", "com.apple.security.accessibility"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.perm, func(t *testing.T) {
			got := permissionToEntitlement(tt.perm)
			if got != tt.want {
				t.Errorf("permissionToEntitlement(%q) = %q, want %q", tt.perm, got, tt.want)
			}
		})
	}
}

func TestStrategy_SingleProcess_String(t *testing.T) {
	if got := StrategySingleProcess.String(); got != "single-process" {
		t.Errorf("StrategySingleProcess.String() = %q, want %q", got, "single-process")
	}
}

func TestDetermineStrategy_SingleProcess(t *testing.T) {
	manager := New()

	tests := []struct {
		name   string
		config *Config
		envKey string
		envVal string
		want   Strategy
	}{
		{
			name:   "single-process via config",
			config: &Config{SingleProcess: true},
			want:   StrategySingleProcess,
		},
		{
			name:   "single-process via env",
			config: &Config{},
			envKey: "MACGO_SINGLE_PROCESS",
			envVal: "1",
			want:   StrategySingleProcess,
		},
		{
			name:   "direct overrides when no single-process",
			config: &Config{ForceDirectExecution: true},
			want:   StrategyDirect,
		},
		{
			name:   "single-process takes precedence over direct",
			config: &Config{SingleProcess: true, ForceDirectExecution: true},
			want:   StrategySingleProcess,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envKey != "" {
				t.Setenv(tt.envKey, tt.envVal)
			}
			got := manager.determineStrategy(tt.config)
			if got != tt.want {
				t.Errorf("determineStrategy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSingleProcessLauncher_sentinelCheck(t *testing.T) {
	// Verify the sentinel constant is what we expect
	if singleProcessSentinel != "MACGO_SINGLE_PROCESS_ACTIVE" {
		t.Errorf("singleProcessSentinel = %q, want %q", singleProcessSentinel, "MACGO_SINGLE_PROCESS_ACTIVE")
	}
}

func containsString(haystack, needle string) bool {
	return len(haystack) > 0 && len(needle) > 0 &&
		len(haystack) >= len(needle) &&
		// Simple contains check
		stringContains(haystack, needle)
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
