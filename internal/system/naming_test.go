package system

import (
	"testing"

	"github.com/tmc/misc/macgo/helpers/bundle"
)

func TestModulePathToBundleID(t *testing.T) {
	tests := []struct {
		name       string
		modulePath string
		appName    string
		expected   string
	}{
		{
			name:       "github_module",
			modulePath: "github.com/user/repo",
			appName:    "myapp",
			expected:   "com.github.user.repo.myapp",
		},
		{
			name:       "gitlab_module",
			modulePath: "gitlab.com/company/project",
			appName:    "tool",
			expected:   "com.gitlab.company.project.tool",
		},
		{
			name:       "custom_domain",
			modulePath: "example.com/project",
			appName:    "service",
			expected:   "com.example.project.service",
		},
		{
			name:       "simple_module",
			modulePath: "local/project",
			appName:    "app",
			expected:   "local.project.app",
		},
		{
			name:       "deep_path",
			modulePath: "github.com/org/repo/cmd/tool",
			appName:    "binary",
			expected:   "com.github.org.repo.cmd.tool.binary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bundle.ModulePathToBundleID(tt.modulePath, tt.appName)
			if result != tt.expected {
				t.Errorf("modulePathToBundleID(%q, %q) = %q, want %q",
					tt.modulePath, tt.appName, result, tt.expected)
			}
		})
	}
}

func TestInferFallbackBundleID(t *testing.T) {
	tests := []struct {
		name    string
		appName string
	}{
		{"simple_app", "myapp"},
		{"hyphenated_app", "my-app"},
		{"empty_name", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bundle.InferFallbackBundleID(tt.appName)

			// Should contain the app name (or "app" for empty)
			expectedName := tt.appName
			if expectedName == "" {
				expectedName = "app"
			}

			if !containsSubstring(result, expectedName) {
				t.Errorf("inferFallbackBundleID(%q) = %q, should contain %q",
					tt.appName, result, expectedName)
			}

			// Should be a valid bundle ID
			if err := ValidateBundleID(result); err != nil {
				t.Errorf("inferFallbackBundleID(%q) = %q, invalid bundle ID: %v",
					tt.appName, result, err)
			}

			// Should not contain "com.macgo"
			if containsSubstring(result, "com.macgo") {
				t.Errorf("inferFallbackBundleID(%q) = %q, should not contain 'com.macgo'",
					tt.appName, result)
			}
		})
	}
}

func TestSanitizeComponent(t *testing.T) {
	tests := []struct {
		name      string
		component string
		expected  string
	}{
		{"simple", "user", "user"},
		{"uppercase", "USER", "user"},
		{"with_spaces", "john doe", "john-doe"},
		{"with_numbers", "user123", "user123"},
		{"leading_number", "123user", "user123user"},
		{"special_chars", "user@domain", "user-domain"},
		{"empty", "", "user"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bundle.SanitizeComponent(tt.component)
			if result != tt.expected {
				t.Errorf("sanitizeComponent(%q) = %q, want %q",
					tt.component, result, tt.expected)
			}
		})
	}
}

// Helper function for substring checking
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestCleanAppName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "clean name unchanged",
			input: "MyApp",
			want:  "MyApp",
		},
		{
			name:  "replace filesystem characters",
			input: "My/App\\With:Bad*Chars",
			want:  "My-App-With-Bad-Chars",
		},
		{
			name:  "remove non-printable characters",
			input: "MyApp\x00\x01\x02",
			want:  "MyApp",
		},
		{
			name:  "handle spaces and punctuation",
			input: "My App (v1.0)",
			want:  "My App (v1.0)",
		},
		{
			name:  "trim and collapse hyphens",
			input: "--My--App--",
			want:  "My-App",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only bad characters",
			input: "/\\:*?\"<>|",
			want:  "",
		},
		{
			name:  "mixed valid and invalid",
			input: "Good/Bad\\Mix:App",
			want:  "Good-Bad-Mix-App",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanAppName(tt.input)
			if got != tt.want {
				t.Errorf("CleanAppName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInferBundleID(t *testing.T) {
	tests := []struct {
		name    string
		appName string
		want    string
	}{
		{
			name:    "simple app name",
			appName: "MyApp",
			want:    "", // Can't predict exact value due to build info dependency
		},
		{
			name:    "app name with spaces",
			appName: "My App",
			want:    "", // Can't predict exact value
		},
		{
			name:    "empty app name",
			appName: "",
			want:    "", // Can't predict exact value, depends on module info or environment
		},
		{
			name:    "app name with special chars",
			appName: "My-App_v2",
			want:    "", // Can't predict exact value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InferBundleID(tt.appName)

			if tt.want != "" {
				if got != tt.want {
					t.Errorf("InferBundleID() = %q, want %q", got, tt.want)
				}
			} else {
				// For cases where we can't predict the exact result,
				// just check that we get a valid-looking bundle ID
				if got == "" {
					t.Errorf("InferBundleID() returned empty string")
				}
				if !containsString([]string{"com.macgo.", "."}, func() string {
					for _, prefix := range []string{"com.macgo.", "."} {
						if len(got) >= len(prefix) && got[:len(prefix)] == prefix {
							return prefix
						}
					}
					return ""
				}()) {
					// Should either start with com.macgo. or contain a dot
					if !containsChar(got, '.') {
						t.Errorf("InferBundleID() = %q, should contain a dot", got)
					}
				}
			}
		})
	}
}

func TestSanitizeBundleID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "valid bundle ID unchanged",
			input: "com.example.app",
			want:  "com.example.app",
		},
		{
			name:  "uppercase to lowercase",
			input: "Com.Example.App",
			want:  "com.example.app",
		},
		{
			name:  "replace invalid characters",
			input: "com.example.app@test",
			want:  "com.example.app-test",
		},
		{
			name:  "trim dots and hyphens",
			input: "..com.example.app..",
			want:  "com.example.app",
		},
		{
			name:  "handle numbers at start",
			input: "1com.example.app",
			want:  "app1com.example.app",
		},
		{
			name:  "empty string",
			input: "",
			want:  "app",
		},
		{
			name:  "collapse consecutive separators",
			input: "com...example---app",
			want:  "com.example-app",
		},
		{
			name:  "only invalid characters",
			input: "@#$%",
			want:  "app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bundle.SanitizeBundleID(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeBundleID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractAppNameFromPath(t *testing.T) {
	tests := []struct {
		name     string
		execPath string
		want     string
	}{
		{
			name:     "simple binary name",
			execPath: "/usr/local/bin/myapp",
			want:     "myapp",
		},
		{
			name:     "binary with extension",
			execPath: "/path/to/myapp.exe",
			want:     "myapp",
		},
		{
			name:     "complex path",
			execPath: "/Users/user/go/bin/my-awesome-app",
			want:     "my-awesome-app",
		},
		{
			name:     "empty path",
			execPath: "",
			want:     "",
		},
		{
			name:     "just filename",
			execPath: "myapp",
			want:     "myapp",
		},
		{
			name:     "problematic characters",
			execPath: "/path/to/my app (v1.0)",
			want:     "my app (v1.0)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractAppNameFromPath(tt.execPath)
			if got != tt.want {
				t.Errorf("ExtractAppNameFromPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateBundleID(t *testing.T) {
	tests := []struct {
		name        string
		bundleID    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid bundle ID",
			bundleID: "com.example.app",
			wantErr:  false,
		},
		{
			name:     "valid with hyphens",
			bundleID: "com.example.my-app",
			wantErr:  false,
		},
		{
			name:        "empty bundle ID",
			bundleID:    "",
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name:        "no dots",
			bundleID:    "comexampleapp",
			wantErr:     true,
			errContains: "at least one dot",
		},
		{
			name:        "invalid characters",
			bundleID:    "com.example.app@test",
			wantErr:     true,
			errContains: "alphanumeric characters",
		},
		{
			name:        "starts with dot",
			bundleID:    ".com.example.app",
			wantErr:     true,
			errContains: "cannot start or end",
		},
		{
			name:        "ends with dot",
			bundleID:    "com.example.app.",
			wantErr:     true,
			errContains: "cannot start or end",
		},
		{
			name:        "consecutive dots",
			bundleID:    "com..example.app",
			wantErr:     true,
			errContains: "consecutive dots",
		},
		{
			name:        "component starts with number",
			bundleID:    "com.1example.app",
			wantErr:     true,
			errContains: "cannot start with a number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBundleID(tt.bundleID)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateBundleID() expected error but got none")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateBundleID() error = %v, want to contain %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateBundleID() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestValidateAppName(t *testing.T) {
	tests := []struct {
		name        string
		appName     string
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid app name",
			appName: "MyApp",
			wantErr: false,
		},
		{
			name:    "app name with spaces",
			appName: "My Great App",
			wantErr: false,
		},
		{
			name:        "empty app name",
			appName:     "",
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name:        "too long app name",
			appName:     string(make([]byte, 256)),
			wantErr:     true,
			errContains: "too long",
		},
		{
			name:        "app name with slash",
			appName:     "My/App",
			wantErr:     true,
			errContains: "cannot contain character: /",
		},
		{
			name:        "app name with backslash",
			appName:     "My\\App",
			wantErr:     true,
			errContains: "cannot contain character: \\",
		},
		{
			name:        "app name with colon",
			appName:     "My:App",
			wantErr:     true,
			errContains: "cannot contain character: :",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAppName(tt.appName)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateAppName() expected error but got none")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateAppName() error = %v, want to contain %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateAppName() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestLimitAppNameLength(t *testing.T) {
	tests := []struct {
		name      string
		appName   string
		maxLength int
		want      string
	}{
		{
			name:      "short name unchanged",
			appName:   "MyApp",
			maxLength: 10,
			want:      "MyApp",
		},
		{
			name:      "exact length unchanged",
			appName:   "MyApp",
			maxLength: 5,
			want:      "MyApp",
		},
		{
			name:      "long name truncated",
			appName:   "MyVeryLongApplicationName",
			maxLength: 10,
			want:      "MyVeryLong",
		},
		{
			name:      "zero length",
			appName:   "MyApp",
			maxLength: 0,
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LimitAppNameLength(tt.appName, tt.maxLength)
			if got != tt.want {
				t.Errorf("LimitAppNameLength() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Helper functions
func containsChar(s string, c rune) bool {
	for _, r := range s {
		if r == c {
			return true
		}
	}
	return false
}
