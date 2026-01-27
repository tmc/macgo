package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const lsregisterPath = "/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"

func usage(fs *flag.FlagSet) {
	fmt.Fprintf(os.Stderr, "Usage: %s <command> [options] [path...]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nCommands:\n")
	fmt.Fprintf(os.Stderr, "  dump        Dump Launch Services database\n")
	fmt.Fprintf(os.Stderr, "  register    Register application(s) at path(s)\n")
	fmt.Fprintf(os.Stderr, "  unregister  Unregister application(s) at path(s)\n")
	fmt.Fprintf(os.Stderr, "  lint        Check for plist errors\n")
	fmt.Fprintf(os.Stderr, "  gc          Garbage collect database\n")
	fmt.Fprintf(os.Stderr, "  seed        Rescan default locations\n")
	fmt.Fprintf(os.Stderr, "  reset       Delete database (requires reboot)\n")
	fmt.Fprintf(os.Stderr, "  search      Search database for apps (by name/id/path)\n")
	fmt.Fprintf(os.Stderr, "  info        Show details for a specific app (by name/id/path)\n")
	fmt.Fprintf(os.Stderr, "  list        List all registered application\n")
	fmt.Fprintf(os.Stderr, "\nOptions:\n")
	if fs != nil {
		fs.PrintDefaults()
	} else {
		flag.PrintDefaults()
	}
	os.Exit(1)
}

func main() {
	if _, err := os.Stat(lsregisterPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: lsregister not found at %s\n", lsregisterPath)
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		usage(nil)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var lsArgs []string

	// Parse common flags
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	verbose := fs.Bool("v", false, "Verbose output")
	dryRun := fs.Bool("n", false, "Dry run - print command without executing")
	recursive := fs.Bool("r", false, "Recursive directory scan (for register)")
	lazy := fs.Int("lazy", 0, "Sleep for n seconds before processing")
	jsonOutput := fs.Bool("json", false, "Output results as JSON (for list/search)")

	// Domain selection
	user := fs.Bool("user", false, "Include user domain")
	system := fs.Bool("system", false, "Include system domain")
	local := fs.Bool("local", false, "Include local domain")
	network := fs.Bool("network", false, "Include network domain")

	// Type selection for domains (maps to -apps, -libs, -all)
	// Default to -apps if nothing specified, or -all? lsregister defaults to searching for apps.
	apps := fs.Bool("apps", true, "Register/scan applications (default)")
	libs := fs.Bool("libs", false, "Register/scan libraries")
	all := fs.Bool("all", false, "Register/scan all types")

	fs.Parse(args)
	paths := fs.Args()

	// Helper to append domain args based on selection type
	appendDomains := func() {
		var domains []string
		if *user {
			domains = append(domains, "user")
		}
		if *system {
			domains = append(domains, "system")
		}
		if *local {
			domains = append(domains, "local")
		}
		if *network {
			domains = append(domains, "network")
		}

		if len(domains) > 0 {
			domainStr := strings.Join(domains, ",")
			if *all {
				lsArgs = append(lsArgs, "-all", domainStr)
			} else if *libs {
				lsArgs = append(lsArgs, "-libs", domainStr)
			} else if *apps {
				lsArgs = append(lsArgs, "-apps", domainStr)
			} else {
				// Default to apps if nothing specified but domains are present
				lsArgs = append(lsArgs, "-apps", domainStr)
			}
		}
	}

	// Helper for lazy
	if *lazy > 0 {
		lsArgs = append(lsArgs, "-lazy", fmt.Sprintf("%d", *lazy))
	}

	switch cmd {
	case "dump":
		lsArgs = append(lsArgs, "-dump")
	case "register":
		if len(paths) == 0 {
			fmt.Fprintf(os.Stderr, "Error: register requires at least one path\n")
			os.Exit(1)
		}
		lsArgs = append(lsArgs, "-f")
		if *recursive {
			lsArgs = append(lsArgs, "-R")
		}
		appendDomains()
		lsArgs = append(lsArgs, paths...)
	case "unregister":
		if len(paths) == 0 {
			fmt.Fprintf(os.Stderr, "Error: unregister requires at least one path\n")
			os.Exit(1)
		}
		lsArgs = append(lsArgs, "-u")
		appendDomains()
		lsArgs = append(lsArgs, paths...)
	case "lint":
		lsArgs = []string{"-lint"}
		if len(paths) > 0 {
			lsArgs = append(lsArgs, paths...)
		}
	case "gc":
		lsArgs = []string{"-gc"}
	case "seed":
		lsArgs = []string{"-seed"}
	case "reset":
		// Wraps -delete. Dangerous.
		fmt.Printf("WARNING: This will delete the Launch Services database. You MUST reboot immediately after.\n")
		fmt.Printf("Are you sure? [y/N] ")
		var response string
		if !*dryRun {
			fmt.Scanln(&response)
			if strings.ToLower(response) != "y" {
				fmt.Println("Aborted.")
				os.Exit(0)
			}
		} else {
			fmt.Println("(Dry run: skipping confirmation prompt)")
		}
		lsArgs = []string{"-delete"}
	case "search", "list", "info":
		if *dryRun {
			fmt.Println("Dry run not supported for search/list/info (these are read-only operations)")
			return
		}

		fmt.Println("Dumping database for analysis (this may take a moment)...")

		// Run dump command and pipe output
		cmdDump := exec.Command(lsregisterPath, "-dump")
		stdout, err := cmdDump.StdoutPipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating stdout pipe: %v\n", err)
			os.Exit(1)
		}

		if err := cmdDump.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting dump: %v\n", err)
			os.Exit(1)
		}

		records, err := ParseDump(stdout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing dump: %v\n", err)
			// Don't exit here necessarily, as scanner might error on EOF sometimes or partial reads
		}

		if err := cmdDump.Wait(); err != nil {
			if _, ok := err.(*exec.ExitError); !ok {
				// Only complain if it's not a normal exit (though dump usually exits 0)
				fmt.Fprintf(os.Stderr, "Error waiting for dump: %v\n", err)
			}
		}

		query := ""
		if len(paths) > 0 {
			query = strings.Join(paths, " ")
		}

		if cmd == "list" {
			query = ""
		}

		matches := filterRecords(records, query)

		if cmd == "info" {
			if len(matches) == 0 {
				if *jsonOutput {
					printJSON([]Record{})
				} else {
					fmt.Println("No matches found.")
				}
			} else if len(matches) == 1 {
				if *jsonOutput {
					printJSON(matches)
				} else {
					printRecord(matches[0])
				}
			} else {
				if *jsonOutput {
					printJSON(matches)
				} else {
					fmt.Printf("Multiple matches found (%d). Please refine your search:\n", len(matches))
					printTable(matches)
				}
			}
		} else {
			if *jsonOutput {
				printJSON(matches)
			} else {
				printTable(matches)
			}
		}
		return
	default:
		usage(fs)
	}

	if *verbose || *dryRun {
		// Use a simple quoting strategy for display
		displayArgs := make([]string, len(lsArgs))
		for i, arg := range lsArgs {
			if strings.Contains(arg, " ") {
				displayArgs[i] = fmt.Sprintf("%q", arg)
			} else {
				displayArgs[i] = arg
			}
		}
		fmt.Printf("Command: %s %s\n", lsregisterPath, strings.Join(displayArgs, " "))
	}

	if *dryRun {
		return
	}

	c := exec.Command(lsregisterPath, lsArgs...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin

	if err := c.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
