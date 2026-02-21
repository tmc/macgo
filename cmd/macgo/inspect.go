package main

import (
	"fmt"
	"path/filepath"

	"github.com/tmc/macgo/codesign"
)

func runInspect(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: macgo inspect <bundle.app|binary>")
	}
	target := args[0]

	info, err := codesign.GetSignatureInfo(target)
	if err != nil {
		return fmt.Errorf("reading signature: %w", err)
	}

	fmt.Printf("Path:       %s\n", target)

	if id, ok := info["Identifier"]; ok {
		fmt.Printf("Identifier: %s\n", id)
	}

	// Determine signature type from Authority.
	if auth, ok := info["Authority"]; ok {
		fmt.Printf("Authority:  %s\n", auth)
	} else {
		fmt.Println("Signature:  ad-hoc")
	}

	if team, ok := info["TeamIdentifier"]; ok {
		fmt.Printf("Team ID:    %s\n", team)
	} else {
		fmt.Println("Team ID:    not set")
	}

	if format, ok := info["Format"]; ok {
		fmt.Printf("Format:     %s\n", format)
	}

	if cdhash, ok := info["CDHash"]; ok {
		fmt.Printf("CDHash:     %s\n", cdhash)
	}

	// Verify signature.
	absPath, _ := filepath.Abs(target)
	if err := codesign.VerifySignature(absPath); err != nil {
		fmt.Printf("Verified:   FAIL (%v)\n", err)
	} else {
		fmt.Println("Verified:   OK")
	}

	return nil
}
