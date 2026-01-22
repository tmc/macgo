package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/tmc/macgo"
)

var (
	interactive = flag.Bool("interactive", false, "Test interactive input")
	testPipe    = flag.Bool("pipe", false, "Test pipe redirection")
	verbose     = flag.Bool("verbose", false, "Enable verbose output")
	hang        = flag.Bool("hang", false, "Hang forever to test timeout")
)

func init() {
	// Check MACGO_NOBUNDLE environment variable to skip bundle setup
	if os.Getenv("MACGO_NOBUNDLE") == "1" {
		fmt.Fprintf(os.Stderr, "[parent] Running without bundle (PID: %d)\n", os.Getpid())
		return
	}

	// Always show parent process output
	fmt.Fprintf(os.Stderr, "[parent] Starting macgo initialization (PID: %d)\n", os.Getpid())

	// Initialize macgo configuration in init() to ensure proper re-execution
	cfg := macgo.NewConfig().
		WithAppName("IO Test")
	cfg.BundleID = "com.example.io-test"
	cfg.Version = "1.0.0"

	// This will re-execute the program if needed and exit the current process
	if err := macgo.Start(cfg); err != nil {
		log.Fatal("Failed to start macgo:", err)
	}

	// If we get here, we're running in the bundle
	fmt.Fprintf(os.Stderr, "[child] Running in bundle (PID: %d)\n", os.Getpid())
}

func setupSignalHandlers() {
	// Create buffered channel for signals
	sigChan := make(chan os.Signal, 1)

	// Register for signals
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	processType := "[parent]"
	if os.Getenv("MACGO_NOBUNDLE") != "1" {
		processType = "[child]"
	}

	go func() {
		for sig := range sigChan {
			switch sig {
			case syscall.SIGQUIT:
				// Print stack traces for all goroutines
				fmt.Fprintf(os.Stderr, "\n%s Received SIGQUIT (PID: %d)\n", processType, os.Getpid())
				fmt.Fprintf(os.Stderr, "%s Printing stack traces:\n", processType)
				buf := make([]byte, 1<<20) // 1MB buffer
				stackLen := runtime.Stack(buf, true)
				fmt.Fprintf(os.Stderr, "%s\n", buf[:stackLen])
				fmt.Fprintf(os.Stderr, "%s Stack trace complete (PID: %d)\n", processType, os.Getpid())
			case syscall.SIGINT:
				fmt.Fprintf(os.Stderr, "\n%s Received SIGINT (PID: %d), exiting gracefully\n", processType, os.Getpid())
				// Start force-kill timer in case we're blocked on I/O
				go func() {
					time.Sleep(100 * time.Millisecond)
					fmt.Fprintf(os.Stderr, "%s Forcing exit after grace period (PID: %d)\n", processType, os.Getpid())
					syscall.Kill(os.Getpid(), syscall.SIGKILL)
				}()
				// Try graceful exit
				os.Exit(0)
			case syscall.SIGTERM:
				fmt.Fprintf(os.Stderr, "\n%s Received SIGTERM (PID: %d), exiting gracefully\n", processType, os.Getpid())
				// Start force-kill timer in case we're blocked on I/O
				go func() {
					time.Sleep(100 * time.Millisecond)
					fmt.Fprintf(os.Stderr, "%s Forcing exit after grace period (PID: %d)\n", processType, os.Getpid())
					syscall.Kill(os.Getpid(), syscall.SIGKILL)
				}()
				// Try graceful exit
				os.Exit(0)
			}
		}
	}()
}

func main() {
	flag.Parse()

	// Set up signal handlers early
	setupSignalHandlers()

	// Show which process we're in
	if os.Getenv("MACGO_NOBUNDLE") == "1" {
		fmt.Fprintf(os.Stderr, "[parent] Main function started (PID: %d)\n", os.Getpid())
	} else {
		fmt.Fprintf(os.Stderr, "[child] Main function started (PID: %d)\n", os.Getpid())
	}

	// Test 1: Basic output (now after macgo.Start() if bundled)
	fmt.Printf("=== IO Test Running (PID: %d) ===\n", os.Getpid())
	fmt.Fprintf(os.Stdout, "STDOUT: Test output (PID: %d)\n", os.Getpid())
	fmt.Fprintf(os.Stderr, "STDERR: Test output (PID: %d)\n", os.Getpid())

	// Test 3: Logging
	log.Println("LOG: Standard log output")

	// Test 4: Hang test (simulates hung process)
	if *hang {
		fmt.Println("\n=== Hang Test - Sleeping Forever ===")
		for {
			time.Sleep(1 * time.Hour)
		}
	}

	// Test 5: Time-based output to test buffering
	if *verbose {
		fmt.Println("\n=== Time-based Output Test ===")
		for i := 1; i <= 3; i++ {
			fmt.Printf("Count %d to stdout\n", i)
			fmt.Fprintf(os.Stderr, "Count %d to stderr\n", i)
			time.Sleep(500 * time.Millisecond)
		}
	}

	// Test 6: Interactive input
	if *interactive {
		fmt.Println("\n=== Interactive Input Test ===")
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Enter your name: ")
		name, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		} else {
			fmt.Printf("Hello, %s", name)
		}
	}

	// Test 7: Pipe detection and handling
	if *testPipe {
		fmt.Println("\n=== Pipe Test ===")

		// Check if stdin is a pipe
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			fmt.Println("Input is from a pipe")
			scanner := bufio.NewScanner(os.Stdin)
			lineNum := 0
			for scanner.Scan() {
				lineNum++
				fmt.Printf("Line %d: %s\n", lineNum, scanner.Text())
			}
		} else {
			fmt.Println("Input is from terminal")
		}

		// Check if stdout is a pipe
		stat, _ = os.Stdout.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			fmt.Fprintln(os.Stderr, "Output is to a pipe")
		} else {
			fmt.Fprintln(os.Stderr, "Output is to terminal")
		}
	}

	// Test 8: Exit codes
	exitCode := 0
	if len(flag.Args()) > 0 && flag.Args()[0] == "fail" {
		exitCode = 1
		fmt.Fprintf(os.Stderr, "Exiting with error code %d\n", exitCode)
	} else {
		fmt.Println("\n=== Test Complete ===")
	}

	os.Exit(exitCode)
}
