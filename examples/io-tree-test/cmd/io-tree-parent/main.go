// io-tree-parent is a macgo-wrapped parent program for testing I/O forwarding across programs.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/tmc/macgo"
)

var (
	maxDepth = flag.Int("max-depth", 2, "Maximum depth for the tree")
)

func init() {
	if os.Getenv("MACGO_NOBUNDLE") == "1" {
		return
	}

	cfg := macgo.NewConfig().
		WithAppName("io-tree-parent").
		WithPermissions(macgo.Files) // Request a TCC permission to trigger bundle relaunch
	cfg.BundleID = "com.example.io-tree-parent"
	cfg.Version = "1.0.0"

	if err := macgo.Start(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "[parent] macgo.Start failed: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	defer macgo.Cleanup() // Write done file when exiting
	flag.Parse()

	prefix := fmt.Sprintf("[parent pid=%d]", os.Getpid())

	fmt.Fprintf(os.Stderr, "%s STDERR: Starting\n", prefix)
	os.Stderr.Sync()
	fmt.Printf("%s STDOUT: Hello from parent\n", prefix)
	os.Stdout.Sync()

	fmt.Fprintf(os.Stderr, "%s STDERR: Spawning io-tree-child\n", prefix)

	cmd := exec.Command("io-tree-child", fmt.Sprintf("-max-depth=%d", *maxDepth))
	cmd.Env = append(os.Environ(), "TREE_DEPTH=1")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s ERROR: stdout pipe: %v\n", prefix, err)
		os.Exit(1)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s ERROR: stderr pipe: %v\n", prefix, err)
		os.Exit(1)
	}

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "%s ERROR: start child: %v\n", prefix, err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "%s STDERR: Child started, forwarding I/O...\n", prefix)

	// Forward output
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			fmt.Printf("%s child-stdout: %s\n", prefix, scanner.Text())
		}
	}()
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Fprintf(os.Stderr, "%s child-stderr: %s\n", prefix, scanner.Text())
		}
	}()

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s child exited with error: %v\n", prefix, err)
		} else {
			fmt.Fprintf(os.Stderr, "%s child completed successfully\n", prefix)
		}
	case <-time.After(120 * time.Second):
		fmt.Fprintf(os.Stderr, "%s child timed out\n", prefix)
		cmd.Process.Kill()
	}

	fmt.Printf("%s STDOUT: Goodbye from parent\n", prefix)
	os.Stdout.Sync()
	fmt.Fprintf(os.Stderr, "%s STDERR: Exiting\n", prefix)
	os.Stderr.Sync()
}
