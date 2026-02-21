package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/tmc/macgo/codesign"
)

func runSign(args []string) error {
	fs := flag.NewFlagSet("sign", flag.ExitOnError)
	identity := fs.String("identity", "", "signing identity (default: best available, use - for ad-hoc)")
	entitlements := fs.String("entitlements", "", "path to entitlements plist")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: macgo sign [flags] <bundle.app>\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		fs.Usage()
		return fmt.Errorf("missing bundle path")
	}
	bundlePath := fs.Arg(0)

	// Resolve identity.
	id := *identity
	if id == "" {
		id = codesign.FindBestIdentity()
		if id == "" {
			id = "-"
			fmt.Fprintf(os.Stderr, "macgo sign: no identity found, using ad-hoc\n")
		} else {
			fmt.Fprintf(os.Stderr, "macgo sign: using %s\n", id)
		}
	}

	// Build codesign args.
	csArgs := []string{"--sign", id, "--force"}
	if id != "-" {
		csArgs = append(csArgs, "--timestamp", "--options", "runtime")
	}
	if *entitlements != "" {
		csArgs = append(csArgs, "--entitlements", *entitlements)
	}
	csArgs = append(csArgs, bundlePath)

	fmt.Fprintf(os.Stderr, "macgo sign: codesign %s\n", strings.Join(csArgs, " "))

	cmd := exec.Command("codesign", csArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("codesign: %w", err)
	}

	// Verify.
	if err := codesign.VerifySignature(bundlePath); err != nil {
		fmt.Fprintf(os.Stderr, "macgo sign: warning: verification failed: %v\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "macgo sign: verified OK\n")
	}

	return nil
}
