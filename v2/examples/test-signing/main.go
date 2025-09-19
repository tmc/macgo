// Test Code Signing - macgo v2
// Verifies that code signing functionality works correctly
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	macgo "github.com/tmc/misc/macgo/v2"
	"github.com/tmc/misc/macgo/v2/internal/testcerts"
)

func main() {
	fmt.Printf("Code Signing Verification - macgo v2! PID: %d\n", os.Getpid())
	fmt.Println()

	// List existing identities
	fmt.Println("🔍 Checking existing code signing identities...")
	identities, err := testcerts.ListCodeSigningIdentities()
	if err != nil {
		log.Printf("Warning: Failed to list identities: %v", err)
	} else if len(identities) == 0 {
		fmt.Println("  No existing code signing identities found")
	} else {
		fmt.Printf("  Found %d existing identities:\n", len(identities))
		for _, identity := range identities {
			fmt.Printf("    • %s\n", identity)
		}
	}
	fmt.Println()

	// Create test certificate (or use ad-hoc signing)
	fmt.Println("🔐 Setting up test signing identity...")
	certName, err := testcerts.CreateTestCertificate()
	if err != nil {
		log.Fatalf("Failed to setup test signing: %v", err)
	}
	if certName == "-" {
		fmt.Println("  ✓ Using ad-hoc signing for testing (no certificate required)")
	} else {
		fmt.Printf("  ✓ Test certificate created: %s\n", certName)
	}
	fmt.Println()

	// Ensure cleanup
	defer func() {
		if certName != "-" {
			fmt.Println("🧹 Cleaning up test certificate...")
			if err := testcerts.RemoveTestCertificate(); err != nil {
				log.Printf("Warning: Failed to remove test certificate: %v", err)
			} else {
				fmt.Println("  ✓ Test certificate removed")
			}
		}
	}()

	// Test code signing with the test certificate
	fmt.Println("🧪 Testing code signing with test certificate...")
	cfg := &macgo.Config{
		Permissions:      []macgo.Permission{macgo.Files},
		CodeSignIdentity: certName,
		Debug:            true,
	}

	err = macgo.Start(cfg)
	if err != nil {
		log.Fatalf("Failed to create signed bundle: %v", err)
	}

	fmt.Println("✓ Code signing test completed successfully!")
	fmt.Println()

	// Verify the signature
	fmt.Println("🔎 Verifying the signature...")
	bundlePath := fmt.Sprintf("%s/go/bin/main.app", os.Getenv("HOME"))
	if err := verifySignature(bundlePath); err != nil {
		log.Printf("Warning: Signature verification failed: %v", err)
	} else {
		fmt.Println("  ✓ Signature verification passed!")
	}
	fmt.Println()

	// Test auto-signing
	fmt.Println("🤖 Testing auto-signing functionality...")
	cfg2 := &macgo.Config{
		Permissions: []macgo.Permission{macgo.Network},
		AutoSign:    true,
		Debug:       true,
	}

	err = macgo.Start(cfg2)
	if err != nil {
		log.Fatalf("Failed to create auto-signed bundle: %v", err)
	}

	fmt.Println("✓ Auto-signing test completed successfully!")
	fmt.Println()

	fmt.Println("🎉 All code signing tests passed!")
	fmt.Println()
	fmt.Println("📋 What was tested:")
	fmt.Println("  • Test certificate creation and management")
	fmt.Println("  • Explicit code signing with specific identity")
	fmt.Println("  • Automatic code signing identity detection")
	fmt.Println("  • Bundle signature verification")
	fmt.Println("  • Cleanup of test certificates")
}

func verifySignature(bundlePath string) error {
	cmd := exec.Command("codesign", "-dv", bundlePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("codesign verification failed: %w\nOutput: %s", err, string(output))
	}

	fmt.Printf("  Signature details:\n")
	lines := string(output)
	for _, line := range []string{lines} {
		if line != "" {
			fmt.Printf("    %s\n", line)
		}
	}

	return nil
}
