package security

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tmc/misc/macgo"
)

// CodeSigner implements the Signer interface.
// It provides code signing functionality for macOS app bundles.
type CodeSigner struct {
	pathValidator macgo.PathValidator
}

// NewCodeSigner creates a new code signer with the provided path validator.
func NewCodeSigner(pathValidator macgo.PathValidator) *CodeSigner {
	return &CodeSigner{
		pathValidator: pathValidator,
	}
}

// Sign signs an app bundle with the given identity.
// If identity is empty, ad-hoc signing ("-") is used.
func (s *CodeSigner) Sign(ctx context.Context, bundlePath, identity string) error {
	// Validate bundle path
	if err := s.pathValidator.Validate(bundlePath); err != nil {
		return fmt.Errorf("signing: invalid bundle path: %w", err)
	}

	// Use ad-hoc signing if no identity is provided
	if identity == "" {
		identity = "-"
	}

	// Validate signing identity if it's not ad-hoc
	if identity != "-" {
		if err := s.ValidateIdentity(identity); err != nil {
			return fmt.Errorf("signing: invalid identity: %w", err)
		}
	}

	// Check if bundle has entitlements
	entitlementsPath := filepath.Join(bundlePath, "Contents", "entitlements.plist")
	hasEntitlements := fileExists(entitlementsPath)

	// Build codesign command
	args := []string{
		"codesign",
		"--force",
		"--sign", identity,
	}

	// Add entitlements if they exist
	if hasEntitlements {
		args = append(args, "--entitlements", entitlementsPath)
	}

	// Add verbose flag for debugging
	args = append(args, "--verbose")

	// Add bundle path
	args = append(args, bundlePath)

	// Execute codesign command
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("signing: codesign failed: %w (output: %s)", err, string(output))
	}

	return nil
}

// Verify verifies the signature of an app bundle.
func (s *CodeSigner) Verify(bundlePath string) error {
	// Validate bundle path
	if err := s.pathValidator.Validate(bundlePath); err != nil {
		return fmt.Errorf("signing: invalid bundle path: %w", err)
	}

	// Execute codesign verification
	cmd := exec.Command("codesign", "--verify", "--deep", "--strict", "--verbose=2", bundlePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("signing: verification failed: %w (output: %s)", err, string(output))
	}

	return nil
}

// ValidateIdentity checks if a signing identity is valid.
// This implementation is extracted from the original bundle.go validateSigningIdentity function.
func (s *CodeSigner) ValidateIdentity(identity string) error {
	if identity == "" {
		return nil // Empty identity is valid (uses ad-hoc signing)
	}

	// Check for dangerous characters that could be used for command injection
	if strings.ContainsAny(identity, "\x00\r\n;|&`$(){}[]<>\"'\\") {
		return fmt.Errorf("signing: identity contains invalid characters")
	}

	// Check for obvious command injection attempts
	dangerousPatterns := []string{
		"--", "rm ", "mv ", "cp ", "cat ", "echo ", "sh ", "bash ", "/bin/", "/usr/bin/",
		"sudo", "su ", "chmod", "chown", "> ", "< ", "| ", "& ", "; ", "$(", "`",
	}

	lowerIdentity := strings.ToLower(identity)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerIdentity, pattern) {
			return fmt.Errorf("signing: identity contains potentially dangerous pattern: %s", pattern)
		}
	}

	// Length check to prevent resource exhaustion
	if len(identity) > 256 {
		return fmt.Errorf("signing: identity too long: %d characters", len(identity))
	}

	// Valid signing identities should match expected patterns
	validPatterns := []string{
		"Developer ID Application:",
		"Mac Developer:",
		"Apple Development:",
		"Apple Distribution:",
		"iPhone Developer:",
		"iPhone Distribution:",
		"3rd Party Mac Developer Application:",
		"3rd Party Mac Developer Installer:",
	}

	// Special case: ad-hoc signing
	if identity == "-" {
		return nil
	}

	// Check if it matches any valid signing identity pattern
	for _, pattern := range validPatterns {
		if strings.HasPrefix(identity, pattern) {
			return nil
		}
	}

	// If it doesn't match known patterns, it might be a certificate hash
	// Valid certificate hashes are 40-character hex strings
	if len(identity) == 40 {
		for _, c := range identity {
			if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F') || (c >= 'a' && c <= 'f')) {
				return fmt.Errorf("signing: invalid certificate hash format")
			}
		}
		return nil
	}

	return fmt.Errorf("signing: identity does not match known valid patterns")
}

// GetAvailableIdentities returns a list of available code signing identities.
func (s *CodeSigner) GetAvailableIdentities() ([]string, error) {
	cmd := exec.Command("security", "find-identity", "-v", "-p", "codesigning")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("signing: failed to list identities: %w", err)
	}

	var identities []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Developer ID Application") || strings.Contains(line, "Mac Developer") {
			// Extract identity name (everything after the first quote)
			if start := strings.Index(line, "\""); start != -1 {
				if end := strings.Index(line[start+1:], "\""); end != -1 {
					identity := line[start+1 : start+1+end]
					identities = append(identities, identity)
				}
			}
		}
	}

	return identities, nil
}

// fileExists checks if a file exists (helper function).
func fileExists(path string) bool {
	if _, err := exec.Command("test", "-f", path).Output(); err != nil {
		return false
	}
	return true
}

// Compile-time check that CodeSigner implements Signer
var _ macgo.Signer = (*CodeSigner)(nil)
