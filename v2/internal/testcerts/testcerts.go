// Package testcerts provides utilities for creating test code signing certificates.
// This is used for testing and verifying macgo's code signing functionality.
package testcerts

import (
	"fmt"
	"os/exec"
	"strings"
)

// CreateTestCertificate creates a self-signed certificate for testing code signing.
// Returns the certificate name that can be used with codesign.
func CreateTestCertificate() (string, error) {
	certName := "macgo Test Code Signing Certificate"

	// Check if certificate already exists
	if exists, err := certificateExists(certName); err != nil {
		return "", fmt.Errorf("failed to check certificate existence: %w", err)
	} else if exists {
		return certName, nil
	}

	// For testing purposes, we'll use ad-hoc signing which doesn't require certificates
	// This validates the code signing flow without needing real certificates
	// Ad-hoc signing is indicated by the "-" identity
	return "-", nil
}

// RemoveTestCertificate removes the test certificate from the keychain.
// For ad-hoc signing, this is a no-op.
func RemoveTestCertificate() error {
	// Ad-hoc signing doesn't create certificates to remove
	return nil
}

// certificateExists checks if a certificate with the given name exists in the keychain.
func certificateExists(name string) (bool, error) {
	cmd := exec.Command("security", "find-identity", "-v", "-p", "codesigning")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to query certificates: %w", err)
	}

	return strings.Contains(string(output), name), nil
}

// ListCodeSigningIdentities returns a list of available code signing identities.
func ListCodeSigningIdentities() ([]string, error) {
	cmd := exec.Command("security", "find-identity", "-v", "-p", "codesigning")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to query identities: %w", err)
	}

	var identities []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, `"`) && !strings.Contains(line, "valid identities found") {
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
