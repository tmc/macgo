package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/tmc/macgo"
)

func main() {
	// Force new instance to avoid attaching to zombies or existing instances during testing
	os.Setenv("MACGO_OPEN_NEW_INSTANCE", "1")

	// Initialize macgo (this causes the bundle relaunch dance)
	err := macgo.Start(&macgo.Config{
		AppName: "io-tree-test",
		Debug:   os.Getenv("DEBUG_TREE") == "1",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "macgo start error: %v\n", err)
		os.Exit(1)
	}

	var (
		depth    = flag.Int("depth", 0, "Current recursion depth")
		maxDepth = flag.Int("max-depth", 1, "Max recursion depth")
		sleep    = flag.Duration("sleep", 0, "Sleep duration before exit")
	)
	flag.Parse()

	prefix := fmt.Sprintf("[%d]", *depth)
	log(prefix, "Started. PID: %d, UID: %d, Sleep: %v", os.Getpid(), os.Getuid(), *sleep)

	// Read from Stdin
	inputData, err := io.ReadAll(os.Stdin)
	if err != nil {
		log(prefix, "Error reading stdin: %v", err)
	} else {
		log(prefix, "Read %d bytes from stdin: %q", len(inputData), string(inputData))
	}

	output := string(inputData)

	// Recurse if needed
	if *depth < *maxDepth {
		nextDepth := *depth + 1
		log(prefix, "Spawning child with depth %d...", nextDepth)

		// We want to call the wrapper binary (not the bundled one we might be running in)
		// Assumption: 'io-tree-test' is in PATH or we can find it.
		// For verification, we'll try to use the one in ~/go/bin/io-tree-test
		cmdName := "io-tree-test"
		if p, err := exec.LookPath("io-tree-test"); err == nil {
			cmdName = p
		} else {
			// Fallback to expecting it in go bin
			home, _ := os.UserHomeDir()
			cmdName = fmt.Sprintf("%s/go/bin/io-tree-test", home)
		}

		cmd := exec.Command(cmdName,
			"-depth", fmt.Sprintf("%d", nextDepth),
			"-max-depth", fmt.Sprintf("%d", *maxDepth),
			"-sleep", sleep.String(),
		)

		// Setup pipes
		cmd.Stdin = bytes.NewReader([]byte(output))
		var stdoutBuf, stderrBuf bytes.Buffer
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf

		// Forward specific env to debug
		cmd.Env = os.Environ()
		// Essential: Force LaunchServices to create a new instance, otherwise open hangs
		// waiting for Apple Events on the existing (but non-responsive to AEs) process.
		cmd.Env = append(cmd.Env, "MACGO_OPEN_NEW_INSTANCE=1")

		if os.Getenv("DEBUG_TREE") == "1" {
			cmd.Env = append(cmd.Env, "MACGO_DEBUG=1")
		}

		if err := cmd.Run(); err != nil {
			log(prefix, "Child run error: %v. Stderr: %s", err, stderrBuf.String())
			os.Exit(1)
		}

		childOut := stdoutBuf.String()
		log(prefix, "Child stdout: %q", childOut)
		output = fmt.Sprintf("Parent[%d]( %s )", *depth, childOut)
	} else {
		// Leaf node
		log(prefix, "Leaf node, processing data...")
		output = fmt.Sprintf("Leaf[%d]( %s )", *depth, output)
	}

	if *sleep > 0 {
		log(prefix, "Sleeping for %v...", *sleep)
		time.Sleep(*sleep)
	}

	// Final Output
	fmt.Print(output)
	// We also verify stderr propagation by logging something to stderr
	fmt.Fprintf(os.Stderr, "%s Done\n", prefix)
}

func log(prefix, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "%s %s\n", prefix, msg)
}
