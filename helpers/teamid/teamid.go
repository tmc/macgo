// Package teamid provides Apple Developer Team ID detection and validation.
package teamid

import (
	"fmt"
	"os/exec"
	"strings"
)

// DetectTeamID attempts to automatically detect the Apple Developer Team ID
// from installed code signing certificates.
//
// It searches for Developer ID Application certificates in the system keychain
// and extracts the team ID from the certificate information. The team ID is
// typically a 10-character alphanumeric string in parentheses.
//
// Returns the detected team ID or an error if no valid certificate is found.
func DetectTeamID() (string, error) {
	cmd := exec.Command("security", "find-identity", "-v", "-p", "codesigning")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run security find-identity: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		// Look for Developer ID Application certificates
		// Format: "Developer ID Application: Company Name (TEAMID123456)"
		if strings.Contains(line, "Developer ID Application:") {
			// Extract team ID from parentheses
			start := strings.LastIndex(line, "(")
			end := strings.LastIndex(line, ")")
			if start != -1 && end != -1 && end > start {
				teamID := line[start+1 : end]
				// Validate team ID format (10 characters, alphanumeric)
				if IsValidTeamID(teamID) {
					return teamID, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no Developer ID Application certificate found with valid team ID")
}

// IsValidTeamID checks if a string is a valid Apple Developer Team ID.
// Team IDs are 10-character alphanumeric strings containing only uppercase letters and digits.
func IsValidTeamID(teamID string) bool {
	if len(teamID) != 10 {
		return false
	}
	for _, r := range teamID {
		if !((r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

// SubstituteTeamIDInGroups replaces "TEAMID" placeholders in app group identifiers
// with the provided team ID. This is useful for creating app groups that work
// across different developer accounts.
//
// The function modifies the groups slice in place and returns the number of substitutions made.
func SubstituteTeamIDInGroups(groups []string, teamID string) int {
	if teamID == "" {
		return 0
	}

	substitutions := 0
	for i, group := range groups {
		if strings.Contains(group, "TEAMID") {
			groups[i] = strings.ReplaceAll(group, "TEAMID", teamID)
			substitutions++
		}
	}
	return substitutions
}

// AutoSubstituteTeamIDInGroups automatically detects the team ID and substitutes
// "TEAMID" placeholders in app group identifiers. This combines DetectTeamID
// and SubstituteTeamIDInGroups for convenience.
//
// Returns the detected team ID and the number of substitutions made, or an error
// if team ID detection fails.
func AutoSubstituteTeamIDInGroups(groups []string) (string, int, error) {
	// Check if any groups need substitution
	needsSubstitution := false
	for _, group := range groups {
		if strings.Contains(group, "TEAMID") {
			needsSubstitution = true
			break
		}
	}

	if !needsSubstitution {
		return "", 0, nil
	}

	// Detect team ID
	teamID, err := DetectTeamID()
	if err != nil {
		return "", 0, fmt.Errorf("team ID detection failed: %w", err)
	}

	// Substitute in groups
	substitutions := SubstituteTeamIDInGroups(groups, teamID)
	return teamID, substitutions, nil
}