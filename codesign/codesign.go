package codesign

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/tmc/misc/macgo/teamid"
)

// FindDeveloperID attempts to find a Developer ID Application certificate
// by querying the system keychain for available code signing identities.
//
// This function searches for "Developer ID Application" certificates first,
// which are preferred for distribution outside the Mac App Store. If none
// are found, it falls back to any valid code signing identity.
//
// Returns the certificate name/identity string, or empty string if none found.
func FindDeveloperID() string {
	cmd := exec.Command("security", "find-identity", "-v", "-p", "codesigning")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(output), "\n")

	// First pass: look for Developer ID Application certificates (preferred)
	for _, line := range lines {
		if strings.Contains(line, "Developer ID Application") {
			if start := strings.Index(line, `"`); start != -1 {
				if end := strings.LastIndex(line, `"`); end != -1 && end > start {
					identity := line[start+1 : end]
					return identity
				}
			}
		}
	}

	// Second pass: look for any valid identity as fallback
	for _, line := range lines {
		if strings.Contains(line, "valid identities found") {
			continue
		}
		if strings.Contains(line, `"`) && !strings.Contains(line, "invalid") {
			if start := strings.Index(line, `"`); start != -1 {
				if end := strings.LastIndex(line, `"`); end != -1 && end > start {
					identity := line[start+1 : end]
					return identity
				}
			}
		}
	}

	return ""
}

// ValidateCodeSignIdentity checks if the provided code signing identity is valid
// and available in the system keychain.
//
// The special identity "-" (ad-hoc signing) is always considered valid.
// For other identities, this function verifies they exist in the keychain.
func ValidateCodeSignIdentity(identity string) error {
	if identity == "" {
		return fmt.Errorf("empty code signing identity")
	}

	// Ad-hoc signing doesn't need validation
	if identity == "-" {
		return nil
	}

	// Check if identity exists in keychain
	cmd := exec.Command("security", "find-identity", "-v", "-p", "codesigning")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to query keychain: %w", err)
	}

	if !strings.Contains(string(output), identity) {
		return fmt.Errorf("code signing identity not found in keychain: %s", identity)
	}

	return nil
}

// ListAvailableIdentities returns a list of available code signing identities
// from the system keychain. Useful for debugging and identity selection.
//
// Returns a slice of identity strings that can be used with code signing tools.
func ListAvailableIdentities() ([]string, error) {
	cmd := exec.Command("security", "find-identity", "-v", "-p", "codesigning")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to query keychain: %w", err)
	}

	var identities []string
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if strings.Contains(line, "valid identities found") {
			continue
		}
		if strings.Contains(line, `"`) && !strings.Contains(line, "invalid") {
			if start := strings.Index(line, `"`); start != -1 {
				if end := strings.LastIndex(line, `"`); end != -1 && end > start {
					identity := line[start+1 : end]
					identities = append(identities, identity)
				}
			}
		}
	}

	return identities, nil
}

// VerifySignature verifies that a bundle is properly code signed.
// Returns nil if the signature is valid, error otherwise.
//
// This function performs a deep verification of the code signature,
// including all embedded frameworks and resources.
func VerifySignature(bundlePath string) error {
	cmd := exec.Command("codesign", "--verify", "--deep", "--strict", bundlePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("signature verification failed: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// GetSignatureInfo retrieves detailed information about the bundle's code signature.
// Returns a map containing signature details such as signing identity, team ID,
// and other code signing attributes.
//
// The returned map may contain keys such as:
//   - "Authority": The signing authority/certificate name
//   - "TeamIdentifier": The developer team ID
//   - "Identifier": The bundle identifier used for signing
//   - "Format": The signature format
func GetSignatureInfo(bundlePath string) (map[string]string, error) {
	cmd := exec.Command("codesign", "--display", "--verbose", bundlePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get signature info: %w", err)
	}

	info := make(map[string]string)
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				info[key] = value
			}
		}
	}

	return info, nil
}

// HasDeveloperIDCertificate checks if the system has any Developer ID certificates
// installed. This is useful for determining whether automatic code signing is possible.
//
// Returns true if at least one Developer ID certificate is found.
func HasDeveloperIDCertificate() bool {
	identity := FindDeveloperID()
	return identity != "" && strings.Contains(identity, "Developer ID")
}

// ExtractTeamIDFromCertificate attempts to extract the team ID from a certificate
// identity string. Developer ID certificates typically include the team ID in
// parentheses at the end of the certificate name.
//
// Example: "Developer ID Application: Company Name (ABC123DEF4)" -> "ABC123DEF4"
//
// Returns the team ID if found, empty string otherwise.
func ExtractTeamIDFromCertificate(identity string) string {
	// Look for team ID in parentheses
	start := strings.LastIndex(identity, "(")
	end := strings.LastIndex(identity, ")")
	if start != -1 && end != -1 && end > start {
		teamID := identity[start+1 : end]
		// Validate team ID format (10 characters, alphanumeric)
		if teamid.IsValidTeamID(teamID) {
			return teamID
		}
	}
	return ""
}

// GetCertificateTeamID retrieves the team ID from the first available
// Developer ID certificate. This combines FindDeveloperID and
// ExtractTeamIDFromCertificate for convenience.
//
// Returns the team ID from the certificate, or empty string if no
// Developer ID certificate is found or no team ID can be extracted.
func GetCertificateTeamID() string {
	identity := FindDeveloperID()
	if identity == "" {
		return ""
	}
	return ExtractTeamIDFromCertificate(identity)
}
