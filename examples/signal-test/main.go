// signal-test is a comprehensive test program for signal handling with macgo.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tmc/macgo"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

var stdinTest = flag.Bool("stdin", false, "Enable interactive stdin test (echo input back)")

func main() {
	flag.Parse()

	cfg := &macgo.Config{Debug: os.Getenv("MACGO_DEBUG") == "1"}
	if err := macgo.Start(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "macgo.Start: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("=== Signal Test ===\n")
	fmt.Printf("PID: %d\n", os.Getpid())
	fmt.Printf("PPID: %d\n", os.Getppid())
	fmt.Println()

	// Display stdin/TTY info
	printStdinInfo()

	// Set up our own signal monitoring (for logging, not handling)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGHUP,
		syscall.SIGWINCH,
		syscall.SIGCONT,
		syscall.SIGTSTP,
		syscall.SIGUSR1,
		syscall.SIGUSR2,
	)

	// Log received signals
	go func() {
		for sig := range sigChan {
			fmt.Fprintf(os.Stderr, "\n>>> Received signal: %v (%d)\n", sig, sig)

			// For SIGTSTP, we need to stop ourselves
			if sig == syscall.SIGTSTP {
				fmt.Fprintf(os.Stderr, ">>> Stopping process (SIGTSTP)\n")
				signal.Stop(sigChan)
				syscall.Kill(os.Getpid(), syscall.SIGSTOP)
			}
		}
	}()

	fmt.Println("Test scenarios:")
	fmt.Println("  1. SIGINT:   kill -INT <pid>   or Ctrl+C")
	fmt.Println("  2. SIGTERM:  kill -TERM <pid>")
	fmt.Println("  3. SIGQUIT:  kill -QUIT <pid>  or Ctrl+\\")
	fmt.Println("  4. SIGHUP:   kill -HUP <pid>")
	fmt.Println("  5. SIGWINCH: (resize terminal)")
	fmt.Println("  6. SIGTSTP:  kill -TSTP <pid> or Ctrl+Z")
	fmt.Println("  7. SIGUSR1:  kill -USR1 <pid>")
	fmt.Println()
	fmt.Println("Running... (dots indicate alive)")

	// Start worker goroutines for stack dump visibility
	go worker("alpha")
	go worker("beta")

	// Start stdin reader if requested
	if *stdinTest {
		go stdinReader()
	}

	// Main heartbeat loop
	tick := 0
	for {
		fmt.Print(".")
		if tick%60 == 59 {
			fmt.Printf(" [%ds]\n", tick+1)
		}
		tick++
		time.Sleep(time.Second)
	}
}

func worker(name string) {
	for {
		doWork(name)
		time.Sleep(500 * time.Millisecond)
	}
}

func doWork(name string) {
	// Nested call for stack trace visibility
	innerWork(name)
}

func innerWork(name string) {
	time.Sleep(100 * time.Millisecond)
}

// printStdinInfo displays TTY and process group information for debugging stdin handling.
func printStdinInfo() {
	fd := int(os.Stdin.Fd())

	fmt.Println("=== Stdin Info ===")
	fmt.Printf("PGRP: %d\n", unix.Getpgrp())

	isTTY := term.IsTerminal(fd)
	fmt.Printf("IsTerminal(stdin): %v\n", isTTY)

	fpgrp, err := unix.IoctlGetInt(fd, unix.TIOCGPGRP)
	if err != nil {
		fmt.Printf("Foreground PGRP: (error: %v)\n", err)
	} else {
		fmt.Printf("Foreground PGRP: %d\n", fpgrp)
		inForeground := unix.Getpgrp() == fpgrp
		fmt.Printf("In foreground: %v\n", inForeground)
		if isTTY && !inForeground {
			fmt.Println("WARNING: TTY stdin but NOT in foreground - stdin disabled to avoid SIGTTIN")
		}
	}

	stat, _ := os.Stdin.Stat()
	mode := stat.Mode()
	stdinType := "unknown"
	switch {
	case mode&os.ModeNamedPipe != 0:
		stdinType = "pipe"
	case mode.IsRegular():
		stdinType = "file"
	case mode&os.ModeCharDevice != 0 && isTTY:
		stdinType = "TTY"
	case mode&os.ModeCharDevice != 0:
		stdinType = "char device (e.g., /dev/null)"
	}
	fmt.Printf("Stdin type: %s\n", stdinType)
	fmt.Println()
}

// stdinReader reads lines from stdin and echoes them back.
func stdinReader() {
	fmt.Println("[stdin] Reader started. Type lines and press Enter:")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Printf("\n[stdin] Read: %q\n", line)
		if line == "quit" || line == "exit" {
			fmt.Println("[stdin] Exiting...")
			os.Exit(0)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "[stdin] Error: %v\n", err)
	} else {
		fmt.Println("[stdin] EOF received")
	}
}
