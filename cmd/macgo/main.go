// Command macgo provides signing diagnostics, bundle inspection, and code
// signing for macOS app bundles.
//
// Usage:
//
//	macgo doctor          signing environment diagnostics
//	macgo sign <path>     sign a bundle
//	macgo inspect <path>  show bundle/signature info
//	macgo version         print version
package main

import (
	"fmt"
	"os"
)

// version is set by -ldflags at build time.
var version = "dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "doctor":
		err = runDoctor()
	case "sign":
		err = runSign(os.Args[2:])
	case "inspect":
		err = runInspect(os.Args[2:])
	case "version":
		fmt.Println("macgo", version)
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "macgo: unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "macgo %s: %v\n", os.Args[1], err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: macgo <command> [arguments]

Commands:
  doctor          signing environment diagnostics
  sign <path>     sign a bundle
  inspect <path>  show bundle/signature info
  version         print version
`)
}
