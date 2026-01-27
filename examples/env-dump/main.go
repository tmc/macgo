// env-dump: Simple test app that writes environment variables to a file
// Used to verify whether `open --env` passes env vars to .app bundles
package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/tmc/macgo"
)

func main() {
	// Initialize macgo to get the .app bundle behavior
	if err := macgo.Start(nil); err != nil {
		fmt.Fprintf(os.Stderr, "macgo.Start failed: %v\n", err)
		os.Exit(1)
	}

	outputFile := "/tmp/env-dump-output.txt"
	if len(os.Args) > 1 {
		outputFile = os.Args[1]
	}

	// Collect all environment variables
	envVars := os.Environ()
	sort.Strings(envVars)

	// Build output
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== ENV DUMP at %s ===\n", time.Now().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("PID: %d\n", os.Getpid()))
	sb.WriteString(fmt.Sprintf("Args: %v\n", os.Args))
	sb.WriteString(fmt.Sprintf("Total env vars: %d\n\n", len(envVars)))

	// Write all env vars
	sb.WriteString("=== ALL ENVIRONMENT VARIABLES ===\n")
	for _, env := range envVars {
		sb.WriteString(env + "\n")
	}

	// Specifically check for test markers
	sb.WriteString("\n=== TEST MARKERS ===\n")
	testMarkers := []string{
		"TEST_VAR",
		"MACGO_PIPE",
		"MACGO_DEBUG_MARKER",
		"MACGO_STDOUT_PIPE",
		"MACGO_STDERR_PIPE",
	}
	for _, marker := range testMarkers {
		val := os.Getenv(marker)
		if val != "" {
			sb.WriteString(fmt.Sprintf("%s=%s (FOUND!)\n", marker, val))
		} else {
			sb.WriteString(fmt.Sprintf("%s= (NOT SET)\n", marker))
		}
	}

	// Write to file
	if err := os.WriteFile(outputFile, []byte(sb.String()), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write output: %v\n", err)
		os.Exit(1)
	}

	// Also print to stdout/stderr for -o/--stderr testing
	fmt.Printf("STDOUT: env-dump completed, wrote to %s\n", outputFile)
	fmt.Fprintf(os.Stderr, "STDERR: env-dump completed, wrote to %s\n", outputFile)

	fmt.Println("Done!")
}
