// io-tree-child is a macgo-wrapped child program for testing I/O forwarding.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/tmc/macgo"
)

var (
	depth    = flag.Int("depth", 1, "Current depth in the tree")
	maxDepth = flag.Int("max-depth", 2, "Maximum depth")
)

func init() {
	if os.Getenv("MACGO_NOBUNDLE") == "1" {
		return
	}

	// Debug: show which config file we might use
	if os.Getenv("MACGO_DEBUG") == "1" {
		execPath, _ := os.Executable()
		fmt.Fprintf(os.Stderr, "[child depth=%s pid=%d] macgo: checking for config files... (exec=%s)\n", os.Getenv("TREE_DEPTH"), os.Getpid(), execPath)
	}

	cfg := macgo.NewConfig().
		FromEnv(). // Pick up MACGO_DEBUG from environment
		WithAppName("io-tree-child").
		WithPermissions(macgo.Files) // Request a TCC permission to trigger bundle relaunch
	cfg.BundleID = "com.example.io-tree-child"
	cfg.Version = "1.0.0"

	if err := macgo.Start(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "[child depth=%s] macgo.Start failed: %v\n", os.Getenv("TREE_DEPTH"), err)
		os.Exit(1)
	}

	// Debug: show that Start completed (this will go to pipe if redirected)
	if os.Getenv("MACGO_DEBUG") == "1" {
		execPath, _ := os.Executable()
		fmt.Fprintf(os.Stderr, "[child depth=%s pid=%d] macgo: Start completed (exec=%s)\n", os.Getenv("TREE_DEPTH"), os.Getpid(), execPath)
	}
}

func main() {
	defer macgo.Cleanup() // Write done file when exiting
	flag.Parse()

	// Override depth from environment if set
	if d := os.Getenv("TREE_DEPTH"); d != "" {
		if v, err := strconv.Atoi(d); err == nil {
			*depth = v
		}
	}

	prefix := fmt.Sprintf("[child depth=%d pid=%d]", *depth, os.Getpid())

	// Output to both streams
	fmt.Fprintf(os.Stderr, "%s STDERR: Starting\n", prefix)
	os.Stderr.Sync()
	fmt.Printf("%s STDOUT: Hello from child at depth %d\n", prefix, *depth)
	os.Stdout.Sync()

	// If we should spawn another level
	if *depth < *maxDepth {
		nextDepth := *depth + 1
		fmt.Fprintf(os.Stderr, "%s STDERR: Spawning grandchild at depth %d\n", prefix, nextDepth)

		// Use the installed binary name
		cmd := exec.Command("io-tree-child", fmt.Sprintf("-depth=%d", nextDepth), fmt.Sprintf("-max-depth=%d", *maxDepth))
		cmd.Env = append(os.Environ(), fmt.Sprintf("TREE_DEPTH=%d", nextDepth))

		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()

		if err := cmd.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "%s ERROR: Failed to start grandchild: %v\n", prefix, err)
			os.Exit(1)
		}

		// Forward output with prefix
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				fmt.Printf("%s grandchild-stdout: %s\n", prefix, scanner.Text())
			}
		}()
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				fmt.Fprintf(os.Stderr, "%s grandchild-stderr: %s\n", prefix, scanner.Text())
			}
		}()

		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()

		select {
		case err := <-done:
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s grandchild exited with error: %v\n", prefix, err)
			} else {
				fmt.Fprintf(os.Stderr, "%s grandchild completed successfully\n", prefix)
			}
		case <-time.After(60 * time.Second):
			fmt.Fprintf(os.Stderr, "%s grandchild timed out\n", prefix)
			cmd.Process.Kill()
		}
	} else {
		fmt.Fprintf(os.Stderr, "%s STDERR: At max depth, not spawning\n", prefix)
	}

	fmt.Printf("%s STDOUT: Goodbye from child at depth %d\n", prefix, *depth)
	os.Stdout.Sync()
	fmt.Fprintf(os.Stderr, "%s STDERR: Exiting\n", prefix)
	os.Stderr.Sync()
}
