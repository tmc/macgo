// io-tree-test tests that a tree of macgo programs works correctly with stdin/stdout/stderr.
// It verifies that I/O flows properly through multiple levels of macgo-wrapped programs.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/tmc/macgo"
)

var (
	depth      = flag.Int("depth", 0, "Current depth in the tree (0 = root)")
	maxDepth   = flag.Int("max-depth", 2, "Maximum depth of the tree")
	echoStdin  = flag.Bool("echo-stdin", false, "Echo stdin to stdout")
	childCount = flag.Int("children", 1, "Number of children to spawn at each level")
	timeout    = flag.Duration("timeout", 30*time.Second, "Timeout for the entire tree")
)

func init() {
	// Skip macgo if MACGO_NOBUNDLE is set (for testing without bundle)
	if os.Getenv("MACGO_NOBUNDLE") == "1" {
		return
	}

	cfg := macgo.NewConfig().
		WithAppName(fmt.Sprintf("IO Tree Test (depth %s)", os.Getenv("IO_TREE_DEPTH")))
	cfg.BundleID = "com.example.io-tree-test"
	cfg.Version = "1.0.0"

	if err := macgo.Start(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "[depth=%s] Failed to start macgo: %v\n", os.Getenv("IO_TREE_DEPTH"), err)
		os.Exit(1)
	}
}

func main() {
	flag.Parse()

	// Set depth from environment if not set via flag
	if envDepth := os.Getenv("IO_TREE_DEPTH"); envDepth != "" && *depth == 0 {
		if d, err := strconv.Atoi(envDepth); err == nil {
			*depth = d
		}
	}

	prefix := fmt.Sprintf("[depth=%d pid=%d]", *depth, os.Getpid())

	fmt.Fprintf(os.Stderr, "%s Starting\n", prefix)
	fmt.Printf("%s STDOUT: Hello from depth %d\n", prefix, *depth)

	// If we should echo stdin, do that
	if *echoStdin {
		fmt.Fprintf(os.Stderr, "%s Echoing stdin...\n", prefix)
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Printf("%s ECHO: %s\n", prefix, line)
		}
		if err := scanner.Err(); err != nil && err != io.EOF {
			fmt.Fprintf(os.Stderr, "%s Error reading stdin: %v\n", prefix, err)
		}
		fmt.Fprintf(os.Stderr, "%s Done echoing stdin\n", prefix)
		return
	}

	// If we haven't reached max depth, spawn children
	if *depth < *maxDepth {
		fmt.Fprintf(os.Stderr, "%s Spawning %d child(ren)...\n", prefix, *childCount)

		for i := 0; i < *childCount; i++ {
			childPrefix := fmt.Sprintf("%s child-%d:", prefix, i)

			// Get our own executable path
			execPath, err := os.Executable()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s Failed to get executable: %v\n", childPrefix, err)
				continue
			}

			// Build child command with incremented depth
			childDepth := *depth + 1
			args := []string{
				fmt.Sprintf("-depth=%d", childDepth),
				fmt.Sprintf("-max-depth=%d", *maxDepth),
				fmt.Sprintf("-children=%d", *childCount),
			}

			cmd := exec.Command(execPath, args...)
			cmd.Env = append(os.Environ(), fmt.Sprintf("IO_TREE_DEPTH=%d", childDepth))

			// Set up pipes
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s Failed to create stdout pipe: %v\n", childPrefix, err)
				continue
			}
			stderr, err := cmd.StderrPipe()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s Failed to create stderr pipe: %v\n", childPrefix, err)
				continue
			}

			// Start the child
			fmt.Fprintf(os.Stderr, "%s Starting child process...\n", childPrefix)
			if err := cmd.Start(); err != nil {
				fmt.Fprintf(os.Stderr, "%s Failed to start: %v\n", childPrefix, err)
				continue
			}

			// Forward child output with prefix
			go func(prefix string, r io.Reader, w io.Writer, name string) {
				scanner := bufio.NewScanner(r)
				for scanner.Scan() {
					fmt.Fprintf(w, "%s %s: %s\n", prefix, name, scanner.Text())
				}
			}(childPrefix, stdout, os.Stdout, "stdout")

			go func(prefix string, r io.Reader, w io.Writer, name string) {
				scanner := bufio.NewScanner(r)
				for scanner.Scan() {
					fmt.Fprintf(w, "%s %s: %s\n", prefix, name, scanner.Text())
				}
			}(childPrefix, stderr, os.Stderr, "stderr")

			// Wait for child with timeout
			done := make(chan error, 1)
			go func() {
				done <- cmd.Wait()
			}()

			select {
			case err := <-done:
				if err != nil {
					fmt.Fprintf(os.Stderr, "%s Child exited with error: %v\n", childPrefix, err)
				} else {
					fmt.Fprintf(os.Stderr, "%s Child completed successfully\n", childPrefix)
				}
			case <-time.After(*timeout):
				fmt.Fprintf(os.Stderr, "%s Child timed out, killing...\n", childPrefix)
				cmd.Process.Kill()
			}
		}
	} else {
		fmt.Fprintf(os.Stderr, "%s Reached max depth, not spawning children\n", prefix)
	}

	// Write some final output
	fmt.Printf("%s STDOUT: Goodbye from depth %d\n", prefix, *depth)
	fmt.Fprintf(os.Stderr, "%s Exiting\n", prefix)
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if strings.Contains(a, e) {
			return true
		}
	}
	return false
}
