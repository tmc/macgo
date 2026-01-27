// Package main provides a utility to find and kill orphaned xpcproxy processes.
//
// Orphaned xpcproxy processes can occur when macgo-created app bundles are terminated
// improperly, leaving behind xpcproxy processes that continue to consume system resources.
// This utility identifies such processes and terminates them.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var (
	dryRun  = flag.Bool("dry-run", false, "show what would be killed without actually killing")
	verbose = flag.Bool("v", false, "verbose output")
	force   = flag.Bool("f", false, "force kill all xpcproxy processes owned by current user")
)

// ProcessInfo holds information about a running process.
type ProcessInfo struct {
	PID     int
	User    string
	Command string
	Args    []string
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		return fmt.Errorf("cannot determine current user")
	}

	// Find all xpcproxy processes owned by current user
	processes, err := findXPCProxyProcesses(currentUser)
	if err != nil {
		return fmt.Errorf("finding xpcproxy processes: %w", err)
	}

	if len(processes) == 0 {
		if *verbose {
			fmt.Println("No xpcproxy processes found")
		}
		return nil
	}

	// Filter to find orphaned processes (ones related to temporary macgo bundles)
	orphaned := filterOrphaned(processes)

	if len(orphaned) == 0 && !*force {
		if *verbose {
			fmt.Printf("Found %d xpcproxy processes, but none appear to be orphaned\n", len(processes))
			fmt.Println("Use -f to force kill all xpcproxy processes")
		}
		return nil
	}

	// Determine which processes to kill
	toKill := orphaned
	if *force {
		toKill = processes
		fmt.Printf("Force mode: will kill all %d xpcproxy processes\n", len(toKill))
	}

	// Display what will be killed
	fmt.Printf("Found %d orphaned xpcproxy process(es):\n", len(toKill))
	for _, p := range toKill {
		fmt.Printf("  PID %d: %s\n", p.PID, p.Command)
	}

	if *dryRun {
		fmt.Println("\nDry-run mode: no processes killed")
		return nil
	}

	// Kill the processes
	killed := 0
	failed := 0
	for _, p := range toKill {
		if err := killProcess(p.PID); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to kill PID %d: %v\n", p.PID, err)
			failed++
		} else {
			if *verbose {
				fmt.Printf("Killed PID %d\n", p.PID)
			}
			killed++
		}
	}

	fmt.Printf("\nKilled %d process(es)", killed)
	if failed > 0 {
		fmt.Printf(", %d failed", failed)
	}
	fmt.Println()

	return nil
}

// findXPCProxyProcesses finds all xpcproxy processes owned by the specified user.
func findXPCProxyProcesses(user string) ([]ProcessInfo, error) {
	cmd := exec.Command("ps", "aux")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("running ps: %w", err)
	}

	var processes []ProcessInfo
	scanner := bufio.NewScanner(bytes.NewReader(output))

	// Skip header line
	if !scanner.Scan() {
		return processes, nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 11 {
			continue
		}

		// Check if this is an xpcproxy process owned by the user
		if fields[0] != user {
			continue
		}

		command := strings.Join(fields[10:], " ")
		if !strings.Contains(command, "xpcproxy") {
			continue
		}

		pid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}

		processes = append(processes, ProcessInfo{
			PID:     pid,
			User:    fields[0],
			Command: command,
			Args:    fields[10:],
		})
	}

	return processes, scanner.Err()
}

// filterOrphaned filters processes to find those that appear to be orphaned.
// Orphaned processes are identified as xpcproxy processes associated with
// macgo bundles (containing macgo in bundle ID or temporary paths).
func filterOrphaned(processes []ProcessInfo) []ProcessInfo {
	var orphaned []ProcessInfo

	for _, p := range processes {
		// Look for processes associated with macgo bundles
		// These can be identified by:
		// 1. Temporary directories (/tmp/ or /var/folders/)
		// 2. Bundle identifiers containing "macgo"
		// 3. Application names containing "macgo"
		if strings.Contains(p.Command, "/tmp/") ||
			strings.Contains(p.Command, "/var/folders/") ||
			strings.Contains(p.Command, ".macgo.") ||
			strings.Contains(p.Command, "-macgo") {
			orphaned = append(orphaned, p)
		}
	}

	return orphaned
}

// killProcess sends SIGTERM to a process.
func killProcess(pid int) error {
	cmd := exec.Command("kill", strconv.Itoa(pid))
	return cmd.Run()
}
