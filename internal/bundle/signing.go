package bundle

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tmc/macgo/codesign"
	"github.com/tmc/macgo/internal/system"
)

// codeSignBundle signs the app bundle with the configured identity and options.
// This function handles both regular code signing and ad-hoc signing.
//
// Before signing, entitlements.plist is moved out of Contents/ to a temp
// directory. codesign treats files directly in Contents/ as unsigned
// subcomponents and refuses to sign. The entitlements file is referenced
// via --entitlements from the temp path.
//
// Note: .source_hash lives in Contents/Resources/ where codesign seals it
// as a standard bundle resource, so it survives signing and enables bundle
// reuse detection on subsequent launches.
func codeSignBundle(bundlePath string, cfg *Config) error {
	contentsDir := filepath.Join(bundlePath, "Contents")

	// Move entitlements.plist out of Contents/ before signing.
	// codesign treats files directly in Contents/ as subcomponents and fails
	// if they aren't signed Mach-O binaries or standard bundle directories.
	tmpDir, err := os.MkdirTemp("", "macgo-sign-*")
	if err != nil {
		return fmt.Errorf("create temp dir for signing: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	entSrc := filepath.Join(contentsDir, "entitlements.plist")
	entTmp := filepath.Join(tmpDir, "entitlements.plist")
	if _, err := os.Stat(entSrc); err == nil {
		if err := os.Rename(entSrc, entTmp); err != nil {
			return fmt.Errorf("move entitlements out of bundle: %w", err)
		}
	}

	args := []string{
		"--sign", cfg.CodeSignIdentity,
		"--force",
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

	// Reference entitlements from temp path (outside the bundle)
	if _, err := os.Stat(entTmp); err == nil {
		args = append(args, "--entitlements", entTmp)
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

// findBestIdentity returns the strongest available signing identity.
// Preference: Developer ID Application > Apple Development > "".
func findBestIdentity(debug bool) string {
	identity := codesign.FindBestIdentity()
	if debug && identity != "" {
		fmt.Fprintf(os.Stderr, "macgo: best identity: %s\n", identity)
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
