package bundle

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tmc/misc/macgo/helpers"
)

// codeSignBundle signs the app bundle with the configured identity and options.
// This function handles both regular code signing and ad-hoc signing.
func codeSignBundle(bundlePath string, cfg *Config) error {
	args := []string{
		"--sign", cfg.CodeSignIdentity,
		"--force",
	}

	if cfg.CodeSignIdentity != "-" {
		args = append(args, "--timestamp")
		args = append(args, "--options", "runtime")
	}

	// Add identifier - use custom identifier if specified, otherwise use bundle ID
	identifier := cfg.CodeSigningIdentifier
	if cfg.Debug {
		fmt.Printf("macgo: codesign identifier from config: %q\n", identifier)
	}
	if identifier == "" {
		// Read bundle ID from Info.plist
		plistPath := filepath.Join(bundlePath, "Contents", "Info.plist")
		if bundleID, err := readBundleIDFromPlist(plistPath); err == nil && bundleID != "" {
			identifier = bundleID
			if cfg.Debug {
				fmt.Printf("macgo: using bundle ID as identifier: %q\n", identifier)
			}
		} else if cfg.Debug {
			fmt.Printf("macgo: failed to read bundle ID: %v\n", err)
		}
	}
	if identifier != "" {
		args = append(args, "--identifier", identifier)
		if cfg.Debug {
			fmt.Printf("macgo: codesign will use identifier: %q\n", identifier)
		}
	}

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
	identity := helpers.FindDeveloperID()
	if debug && identity != "" {
		if strings.Contains(identity, "Developer ID Application") {
			fmt.Fprintf(os.Stderr, "macgo: found Developer ID: %s\n", identity)
		} else {
			fmt.Fprintf(os.Stderr, "macgo: found fallback identity: %s\n", identity)
		}
	}
	return identity
}

// readBundleIDFromPlist reads the CFBundleIdentifier from an Info.plist file
// using the plutil command-line utility.
func readBundleIDFromPlist(plistPath string) (string, error) {
	cmd := exec.Command("plutil", "-extract", "CFBundleIdentifier", "raw", plistPath)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// validateCodeSignIdentity checks if the provided code signing identity is valid
// and available in the system keychain.
func validateCodeSignIdentity(identity string) error {
	return helpers.ValidateCodeSignIdentity(identity)
}

// listAvailableIdentities returns a list of available code signing identities
// from the system keychain. Useful for debugging and identity selection.
func listAvailableIdentities() ([]string, error) {
	return helpers.ListAvailableIdentities()
}

// VerifySignature verifies that a bundle is properly code signed.
// Returns nil if the signature is valid, error otherwise.
func VerifySignature(bundlePath string) error {
	return helpers.VerifySignature(bundlePath)
}

// GetSignatureInfo retrieves detailed information about the bundle's code signature.
// Returns the signing identity, team ID, and other signature details.
func GetSignatureInfo(bundlePath string) (map[string]string, error) {
	return helpers.GetSignatureInfo(bundlePath)
}