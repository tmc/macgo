package bundle

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tmc/misc/macgo/helpers/codesign"
	"github.com/tmc/misc/macgo/internal/system"
)

// codeSignBundle signs the app bundle with the configured identity and options.
// This function handles both regular code signing and ad-hoc signing.
func codeSignBundle(bundlePath string, cfg *Config) error {
	args := []string{
		"--sign", cfg.CodeSignIdentity,
		"--force",
		"--deep",
	}

	if cfg.CodeSignIdentity != "-" {
		args = append(args, "--timestamp")
		args = append(args, "--options", "runtime")
	}

	// Always read bundle ID from Info.plist and use it as the identifier
	bundleID := system.GetBundleID(bundlePath)
	if bundleID == "" {
		return fmt.Errorf("failed to read bundle ID from Info.plist for signing")
	}

	// Use custom identifier if specified, otherwise use bundle ID
	identifier := cfg.CodeSigningIdentifier
	if identifier == "" {
		identifier = bundleID
	}

	if cfg.Debug {
		if cfg.CodeSigningIdentifier != "" {
			fmt.Printf("macgo: using custom codesign identifier: %q\n", identifier)
		} else {
			fmt.Printf("macgo: using bundle ID as identifier: %q\n", identifier)
		}
		fmt.Printf("macgo: codesign will use identifier: %q\n", identifier)
	}

	// Always add the identifier flag
	args = append(args, "--identifier", identifier)

	if cfg.CodeSignIdentity != "-" {
		entitlementsPath := filepath.Join(bundlePath, "Contents", "entitlements.plist")
		if _, err := os.Stat(entitlementsPath); err == nil {
			args = append(args, "--entitlements", entitlementsPath)
		}
	}

	args = append(args, bundlePath)

	cmd := exec.Command("codesign", args...)
	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "macgo: running: codesign %s\n", strings.Join(args, " "))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("codesign failed: %w\nOutput: %s", err, string(output))
	}

	if cfg.Debug && len(output) > 0 {
		fmt.Fprintf(os.Stderr, "macgo: codesign output: %s\n", string(output))
	}

	return nil
}

// findDeveloperID attempts to find a Developer ID Application certificate
// by querying the system keychain for available code signing identities.
func findDeveloperID(debug bool) string {
	identity := codesign.FindDeveloperID()
	if debug && identity != "" {
		if strings.Contains(identity, "Developer ID Application") {
			fmt.Fprintf(os.Stderr, "macgo: found Developer ID: %s\n", identity)
		} else {
			fmt.Fprintf(os.Stderr, "macgo: found fallback identity: %s\n", identity)
		}
	}
	return identity
}

// validateCodeSignIdentity checks if the provided code signing identity is valid
// and available in the system keychain.
func validateCodeSignIdentity(identity string) error {
	return codesign.ValidateCodeSignIdentity(identity)
}

// listAvailableIdentities returns a list of available code signing identities
// from the system keychain. Useful for debugging and identity selection.
func listAvailableIdentities() ([]string, error) {
	return codesign.ListAvailableIdentities()
}

// VerifySignature verifies that a bundle is properly code signed.
// Returns nil if the signature is valid, error otherwise.
func VerifySignature(bundlePath string) error {
	return codesign.VerifySignature(bundlePath)
}

// GetSignatureInfo retrieves detailed information about the bundle's code signature.
// Returns the signing identity, team ID, and other signature details.
func GetSignatureInfo(bundlePath string) (map[string]string, error) {
	return codesign.GetSignatureInfo(bundlePath)
}
