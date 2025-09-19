package bundle_test

import (
	"os"
	"testing"

	"github.com/tmc/misc/macgo/helpers/bundle"
)

func TestCleanAppName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"simple name", "MyApp", "MyApp"},
		{"name with bad characters", "My/App:Bad*Name", "My-App-Bad-Name"},
		{"name with spaces", "My App Name", "My App Name"},
		{"name with leading/trailing hyphens", "--My-App--", "My-App"},
		{"name with consecutive hyphens", "My---App", "My-App"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bundle.CleanAppName(tt.input)
			if result != tt.expected {
				t.Errorf("CleanAppName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCleanAppNameWithPrefix(t *testing.T) {
	// Save original value
	originalPrefix := os.Getenv("MACGO_APP_NAME_PREFIX")
	defer func() {
		if originalPrefix == "" {
			os.Unsetenv("MACGO_APP_NAME_PREFIX")
		} else {
			os.Setenv("MACGO_APP_NAME_PREFIX", originalPrefix)
		}
	}()

	tests := []struct {
		name     string
		prefix   string
		input    string
		expected string
	}{
		{"no prefix", "", "MyApp", "MyApp"},
		{"simple prefix", "Dev-", "MyApp", "Dev-MyApp"},
		{"prefix with special chars", "Test_", "My/App", "Test_My-App"},
		{"empty input with prefix", "Prefix-", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prefix == "" {
				os.Unsetenv("MACGO_APP_NAME_PREFIX")
			} else {
				os.Setenv("MACGO_APP_NAME_PREFIX", tt.prefix)
			}

			result := bundle.CleanAppName(tt.input)
			if result != tt.expected {
				t.Errorf("CleanAppName(%q) with prefix %q = %q, want %q", tt.input, tt.prefix, result, tt.expected)
			}
		})
	}
}

func TestInferBundleID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string // We can't predict exact output due to module path, but we can check structure
	}{
		{"empty input", "", ""},
		{"simple name", "MyApp", ""},
		{"name with spaces", "My App", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bundle.InferBundleID(tt.input)

			if tt.input == "" && result == "" {
				return // Expected for empty input
			}

			// Check that result contains at least one dot (proper bundle ID format)
			if result != "" && !contains(result, ".") {
				t.Errorf("InferBundleID(%q) = %q, expected bundle ID format with dots", tt.input, result)
			}

			// Check that it doesn't contain invalid characters
			if containsInvalidBundleIDChars(result) {
				t.Errorf("InferBundleID(%q) = %q, contains invalid characters", tt.input, result)
			}
		})
	}
}

func TestInferBundleIDWithPrefix(t *testing.T) {
	// Save original value
	originalPrefix := os.Getenv("MACGO_BUNDLE_ID_PREFIX")
	defer func() {
		if originalPrefix == "" {
			os.Unsetenv("MACGO_BUNDLE_ID_PREFIX")
		} else {
			os.Setenv("MACGO_BUNDLE_ID_PREFIX", originalPrefix)
		}
	}()

	tests := []struct {
		name           string
		prefix         string
		input          string
		expectedPrefix string // What we expect the result to start with
	}{
		{"no prefix", "", "MyApp", ""},
		{"simple prefix", "dev", "MyApp", "dev."},
		{"prefix with dot", "test.", "MyApp", "test."},
		{"complex prefix", "development.staging", "MyApp", "development.staging."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prefix == "" {
				os.Unsetenv("MACGO_BUNDLE_ID_PREFIX")
			} else {
				os.Setenv("MACGO_BUNDLE_ID_PREFIX", tt.prefix)
			}

			result := bundle.InferBundleID(tt.input)

			if tt.expectedPrefix != "" {
				if !startsWith(result, tt.expectedPrefix) {
					t.Errorf("InferBundleID(%q) with prefix %q = %q, expected to start with %q", tt.input, tt.prefix, result, tt.expectedPrefix)
				}
			}

			// Always check that result is a valid bundle ID format
			if result != "" {
				if !contains(result, ".") {
					t.Errorf("InferBundleID(%q) = %q, expected bundle ID format with dots", tt.input, result)
				}
			}
		})
	}
}

func TestValidateBundleID(t *testing.T) {
	tests := []struct {
		name      string
		bundleID  string
		wantError bool
	}{
		{"valid bundle ID", "com.example.app", false},
		{"valid with hyphens", "com.example.my-app", false},
		{"empty bundle ID", "", true},
		{"no dots", "comexampleapp", true},
		{"starts with dot", ".com.example.app", true},
		{"ends with dot", "com.example.app.", true},
		{"consecutive dots", "com..example.app", true},
		{"component starts with number", "com.1example.app", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := bundle.ValidateBundleID(tt.bundleID)
			if tt.wantError && err == nil {
				t.Errorf("ValidateBundleID(%q) expected error but got none", tt.bundleID)
			}
			if !tt.wantError && err != nil {
				t.Errorf("ValidateBundleID(%q) unexpected error: %v", tt.bundleID, err)
			}
		})
	}
}

func TestExtractAppNameFromPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"empty path", "", ""},
		{"simple binary", "/usr/local/bin/myapp", "myapp"},
		{"binary with extension", "/path/to/myapp.exe", "myapp"},
		{"just filename", "myapp", "myapp"},
		{"complex path", "/Users/user/go/bin/my-app", "my-app"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bundle.ExtractAppNameFromPath(tt.path)
			if result != tt.expected {
				t.Errorf("ExtractAppNameFromPath(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

// Helper functions
func contains(s, substr string) bool {
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

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func containsInvalidBundleIDChars(s string) bool {
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '.' || r == '-') {
			return true
		}
	}
	return false
}
