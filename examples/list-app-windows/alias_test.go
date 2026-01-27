package main

import (
	"testing"
)

// TestResolveAppName tests the app name alias resolution
func TestResolveAppName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string // Names that should be included in results
	}{
		{
			name:     "exact_match_system_settings",
			input:    "System Settings",
			expected: []string{"System Settings", "System Preferences"},
		},
		{
			name:     "exact_match_system_preferences",
			input:    "System Preferences",
			expected: []string{"System Preferences", "System Settings"},
		},
		{
			name:     "lowercase_system_preferences",
			input:    "system preferences",
			expected: []string{"System Preferences", "System Settings"},
		},
		{
			name:     "lowercase_systempreferences",
			input:    "systempreferences",
			expected: []string{"System Preferences", "System Settings"},
		},
		{
			name:     "alias_settings",
			input:    "settings",
			expected: []string{"System Settings", "System Preferences"},
		},
		{
			name:     "alias_preferences",
			input:    "preferences",
			expected: []string{"System Preferences", "System Settings"},
		},
		{
			name:     "safari",
			input:    "Safari",
			expected: []string{"Safari", "safari"},
		},
		{
			name:     "lowercase_safari",
			input:    "safari",
			expected: []string{"Safari", "safari"},
		},
		{
			name:     "music_alias_itunes",
			input:    "iTunes",
			expected: []string{"Music", "itunes"},
		},
		{
			name:     "calendar_alias_ical",
			input:    "iCal",
			expected: []string{"Calendar", "ical"},
		},
		{
			name:     "unknown_app",
			input:    "Unknown App",
			expected: []string{"Unknown App"}, // Should return original
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveAppName(tt.input)

			// Check that all expected names are present
			for _, expectedName := range tt.expected {
				found := false
				for _, resultName := range result {
					if containsCaseInsensitive(resultName, expectedName) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("resolveAppName(%q) missing expected name %q, got %v",
						tt.input, expectedName, result)
				}
			}
		})
	}
}

// containsCaseInsensitive checks if s contains substr (case-insensitive)
func containsCaseInsensitive(s, substr string) bool {
	sLower := toLower(s)
	substrLower := toLower(substr)
	return sLower == substrLower
}

// toLower is a simple lowercase helper
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + ('a' - 'A')
		} else {
			result[i] = c
		}
	}
	return string(result)
}

// TestResolveAppNameUnique tests that results have no duplicates
func TestResolveAppNameUnique(t *testing.T) {
	tests := []string{
		"System Settings",
		"System Preferences",
		"system preferences",
		"Safari",
		"Music",
		"iTunes",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			result := resolveAppName(input)

			// Check for duplicates (case-insensitive)
			seen := make(map[string]bool)
			for _, name := range result {
				nameLower := toLower(name)
				if seen[nameLower] {
					t.Errorf("resolveAppName(%q) returned duplicate %q (case-insensitive)", input, name)
				}
				seen[nameLower] = true
			}
		})
	}
}

// TestResolveAppNameAlwaysIncludesOriginal tests that original name is always in results
func TestResolveAppNameAlwaysIncludesOriginal(t *testing.T) {
	tests := []string{
		"System Settings",
		"System Preferences",
		"Safari",
		"Unknown Application",
		"MyCustomApp",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			result := resolveAppName(input)

			// Original should always be in the result
			found := false
			for _, name := range result {
				if name == input {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("resolveAppName(%q) did not include original name, got %v", input, result)
			}
		})
	}
}

// TestSystemPreferencesSettingsMapping tests the bidirectional mapping
func TestSystemPreferencesSettingsMapping(t *testing.T) {
	// Test that System Preferences resolves to System Settings
	prefResult := resolveAppName("System Preferences")
	foundSettings := false
	for _, name := range prefResult {
		if name == "System Settings" {
			foundSettings = true
			break
		}
	}
	if !foundSettings {
		t.Errorf("System Preferences should resolve to System Settings, got %v", prefResult)
	}

	// Test that System Settings resolves to System Preferences
	settingsResult := resolveAppName("System Settings")
	foundPref := false
	for _, name := range settingsResult {
		if name == "System Preferences" {
			foundPref = true
			break
		}
	}
	if !foundPref {
		t.Errorf("System Settings should resolve to System Preferences, got %v", settingsResult)
	}

	// Test legacy aliases
	legacyTests := []string{"settings", "preferences", "systemsettings", "systempreferences"}
	for _, legacy := range legacyTests {
		t.Run("legacy_"+legacy, func(t *testing.T) {
			result := resolveAppName(legacy)
			foundSettings := false
			foundPref := false

			for _, name := range result {
				nameLower := toLower(name)
				if nameLower == "system settings" {
					foundSettings = true
				}
				if nameLower == "system preferences" {
					foundPref = true
				}
			}

			if !foundSettings || !foundPref {
				t.Errorf("Legacy alias %q should resolve to both System Settings and System Preferences, got %v",
					legacy, result)
			}
		})
	}
}

// BenchmarkResolveAppName benchmarks the alias resolution
func BenchmarkResolveAppName(b *testing.B) {
	tests := []string{
		"System Preferences",
		"System Settings",
		"Safari",
		"Unknown App",
	}

	for _, input := range tests {
		b.Run(input, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = resolveAppName(input)
			}
		})
	}
}
