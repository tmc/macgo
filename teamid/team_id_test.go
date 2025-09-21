package teamid

import (
	"reflect"
	"testing"
)


func TestSubstituteTeamIDInGroups(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		teamID   string
		expected []string
		count    int
	}{
		{
			name:     "substitute single app group",
			input:    []string{"TEAMID.shared-data"},
			teamID:   "ABC1234567",
			expected: []string{"ABC1234567.shared-data"},
			count:    1,
		},
		{
			name:     "substitute multiple app groups",
			input:    []string{"TEAMID.shared-data", "TEAMID.cache"},
			teamID:   "XYZ9876543",
			expected: []string{"XYZ9876543.shared-data", "XYZ9876543.cache"},
			count:    2,
		},
		{
			name:     "no substitution needed",
			input:    []string{"group.com.example.shared"},
			teamID:   "ABC1234567",
			expected: []string{"group.com.example.shared"},
			count:    0,
		},
		{
			name:     "mixed app groups",
			input:    []string{"TEAMID.shared-data", "group.com.example.shared"},
			teamID:   "DEF5678901",
			expected: []string{"DEF5678901.shared-data", "group.com.example.shared"},
			count:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := make([]string, len(tt.input))
			copy(groups, tt.input)

			count := SubstituteTeamIDInGroups(groups, tt.teamID)

			if count != tt.count {
				t.Errorf("Expected %d substitutions, got %d", tt.count, count)
			}

			if !reflect.DeepEqual(groups, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, groups)
			}
		})
	}
}

func TestIsValidTeamID(t *testing.T) {
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
		result := IsValidTeamID(tt.input)
		if result != tt.expected {
			t.Errorf("IsValidTeamID(%q) = %v, expected %v", tt.input, result, tt.expected)
		}
	}
}
