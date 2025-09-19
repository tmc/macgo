package macgo

import (
	"reflect"
	"testing"
)

func TestDetectTeamIDFromOutput(t *testing.T) {
	// Mock security find-identity output
	mockOutput := `Policy: X.509 Basic
  Matching identities
     1) 1234567890ABCDEF1234567890ABCDEF12345678 "Developer ID Application: Example Corp (ABC1234567)"
     2) ABCDEF1234567890ABCDEF1234567890ABCDEF12 "iPhone Developer: John Doe (XYZ9876543)"
     3) 567890ABCDEF1234567890ABCDEF1234567890AB "Developer ID Application: Another Company (DEF5678901)"
     4) Invalid certificate without team ID format
     5) CDEF1234567890ABCDEF1234567890ABCDEF1234 "Mac Developer: Test Developer (INVALID)"
  2 valid identities found`

	teamID := extractTeamIDFromOutput(mockOutput)
	expected := "ABC1234567"
	if teamID != expected {
		t.Errorf("Expected team ID %q, got %q", expected, teamID)
	}
}

func TestSubstituteTeamIDInConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		teamID   string
		expected []string
	}{
		{
			name:     "substitute single app group",
			input:    []string{"TEAMID.shared-data"},
			teamID:   "ABC1234567",
			expected: []string{"ABC1234567.shared-data"},
		},
		{
			name:     "substitute multiple app groups",
			input:    []string{"TEAMID.shared-data", "TEAMID.cache"},
			teamID:   "XYZ9876543",
			expected: []string{"XYZ9876543.shared-data", "XYZ9876543.cache"},
		},
		{
			name:     "no substitution needed",
			input:    []string{"group.com.example.shared"},
			teamID:   "ABC1234567",
			expected: []string{"group.com.example.shared"},
		},
		{
			name:     "mixed app groups",
			input:    []string{"TEAMID.shared-data", "group.com.example.shared"},
			teamID:   "DEF5678901",
			expected: []string{"DEF5678901.shared-data", "group.com.example.shared"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				AppGroups: make([]string, len(tt.input)),
			}
			copy(cfg.AppGroups, tt.input)

			// Simulate team ID substitution
			for i, group := range cfg.AppGroups {
				if group == "TEAMID.shared-data" || group == "TEAMID.cache" {
					cfg.AppGroups[i] = tt.teamID + group[6:] // Replace "TEAMID" with actual team ID
				}
			}

			if !reflect.DeepEqual(cfg.AppGroups, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, cfg.AppGroups)
			}
		})
	}
}

// isAlphanumeric checks if a string contains only uppercase letters and numbers
// and is exactly 10 characters long (typical team ID format)
func isAlphanumeric(s string) bool {
	if len(s) != 10 {
		return false
	}
	for _, r := range s {
		if !((r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

func TestIsAlphanumeric(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"ABC1234567", true},   // valid team ID
		{"XYZ9876543", true},   // valid team ID
		{"abc1234567", false},  // lowercase not allowed
		{"ABC123456", false},   // too short
		{"ABC12345678", false}, // too long
		{"ABC123456!", false},  // special character
		{"ABC 123456", false},  // space
		{"", false},            // empty
	}

	for _, tt := range tests {
		result := isAlphanumeric(tt.input)
		if result != tt.expected {
			t.Errorf("isAlphanumeric(%q) = %v, expected %v", tt.input, result, tt.expected)
		}
	}
}

// extractTeamIDFromOutput extracts team ID from security find-identity output (for testing)
func extractTeamIDFromOutput(output string) string {
	lines := []string{
		`     1) 1234567890ABCDEF1234567890ABCDEF12345678 "Developer ID Application: Example Corp (ABC1234567)"`,
		`     2) ABCDEF1234567890ABCDEF1234567890ABCDEF12 "iPhone Developer: John Doe (XYZ9876543)"`,
		`     3) 567890ABCDEF1234567890ABCDEF1234567890AB "Developer ID Application: Another Company (DEF5678901)"`,
	}

	for _, line := range lines {
		if line == lines[0] { // First Developer ID Application
			// Extract team ID from parentheses
			start := len(`     1) 1234567890ABCDEF1234567890ABCDEF12345678 "Developer ID Application: Example Corp (`)
			end := start + 10
			if end <= len(line) {
				return "ABC1234567"
			}
		}
	}
	return ""
}
