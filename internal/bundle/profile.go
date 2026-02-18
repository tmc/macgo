package bundle

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/tmc/macgo/codesign"
)

// ProfileEntitlements holds entitlement values extracted from a provisioning profile.
type ProfileEntitlements struct {
	ApplicationIdentifier string
	TeamIdentifier        string
}

// decodeProvisioningProfile strips the PKCS7 envelope from a provisioning profile
// using security cms and returns the raw XML plist.
func decodeProvisioningProfile(path string) ([]byte, error) {
	cmd := exec.Command("security", "cms", "-D", "-i", path)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("security cms: %w", err)
	}
	return out, nil
}

// extractProfileEntitlements parses XML plist data from a decoded provisioning
// profile and extracts com.apple.application-identifier and
// com.apple.developer.team-identifier from the Entitlements dict.
//
// Uses a line-scan state machine following the pattern in system.GetBundleID.
func extractProfileEntitlements(xmlData []byte) ProfileEntitlements {
	var pe ProfileEntitlements
	lines := strings.Split(string(xmlData), "\n")

	// State machine: find <key>Entitlements</key>, enter the dict,
	// extract target string values, exit on </dict>.
	const (
		stateSearchEntitlements = iota
		stateExpectDict
		stateInDict
		stateReadAppID
		stateReadTeamID
	)
	state := stateSearchEntitlements
	depth := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch state {
		case stateSearchEntitlements:
			if trimmed == "<key>Entitlements</key>" {
				state = stateExpectDict
			}
		case stateExpectDict:
			if trimmed == "<dict>" {
				state = stateInDict
				depth = 1
			}
		case stateInDict:
			if trimmed == "<dict>" {
				depth++
				continue
			}
			if trimmed == "</dict>" {
				depth--
				if depth == 0 {
					return pe
				}
				continue
			}
			// Only match keys at the top level of the Entitlements dict.
			if depth != 1 {
				continue
			}
			if trimmed == "<key>com.apple.application-identifier</key>" {
				state = stateReadAppID
			} else if trimmed == "<key>com.apple.developer.team-identifier</key>" {
				state = stateReadTeamID
			}
		case stateReadAppID:
			if strings.HasPrefix(trimmed, "<string>") && strings.HasSuffix(trimmed, "</string>") {
				pe.ApplicationIdentifier = strings.TrimSuffix(strings.TrimPrefix(trimmed, "<string>"), "</string>")
			}
			state = stateInDict
		case stateReadTeamID:
			if strings.HasPrefix(trimmed, "<string>") && strings.HasSuffix(trimmed, "</string>") {
				pe.TeamIdentifier = strings.TrimSuffix(strings.TrimPrefix(trimmed, "<string>"), "</string>")
			}
			state = stateInDict
		}
	}
	return pe
}

// readProvisioningProfileEntitlements decodes a provisioning profile and
// extracts its entitlement values.
func readProvisioningProfileEntitlements(path string) (ProfileEntitlements, error) {
	data, err := decodeProvisioningProfile(path)
	if err != nil {
		return ProfileEntitlements{}, err
	}
	return extractProfileEntitlements(data), nil
}

// deriveStringEntitlements returns string-valued entitlements that can be
// automatically derived from the bundle's provisioning profile or signing
// identity. User-provided WithCustomString values take precedence and are
// NOT overwritten by the caller.
func (b *Bundle) deriveStringEntitlements() map[string]string {
	// Try provisioning profile first.
	if b.Config.ProvisioningProfile != "" {
		pe, err := readProvisioningProfileEntitlements(b.Config.ProvisioningProfile)
		if err != nil {
			if b.Config.Debug {
				fmt.Fprintf(os.Stderr, "macgo: warning: failed to read provisioning profile entitlements: %v\n", err)
			}
			return nil
		}
		m := make(map[string]string)
		if pe.ApplicationIdentifier != "" {
			m["com.apple.application-identifier"] = pe.ApplicationIdentifier
		}
		if pe.TeamIdentifier != "" {
			m["com.apple.developer.team-identifier"] = pe.TeamIdentifier
		}
		if len(m) > 0 {
			return m
		}
	}

	// Fall back: derive from signing identity.
	identity := b.Config.CodeSignIdentity
	if identity == "" && b.Config.AutoSign {
		identity = codesign.FindDeveloperID()
	}
	if identity == "" || identity == "-" {
		return nil
	}

	teamID := codesign.ExtractTeamIDFromCertificate(identity)
	if teamID == "" {
		return nil
	}
	appID := teamID + "." + b.bundleID
	return map[string]string{
		"com.apple.application-identifier":      appID,
		"com.apple.developer.team-identifier": teamID,
	}
}
