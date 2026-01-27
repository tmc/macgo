package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/tmc/macgo"
)

var (
	testPrompt      = flag.Bool("prompt", false, "Test CLI prompts")
	testPassword    = flag.Bool("password", false, "Test password-style input (no echo)")
	testEOF         = flag.Bool("eof", false, "Test EOF handling")
	testLineBuffer  = flag.Bool("linebuffer", false, "Test line buffering")
	testControlChar = flag.Bool("control", false, "Test terminal control characters")
	testMultiLine   = flag.Bool("multiline", false, "Test multi-line input")
	testTimeout     = flag.Bool("timeout", false, "Test input timeout")
	verbose         = flag.Bool("verbose", false, "Enable verbose output")
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
		WithAppName("Stdin Test")
	cfg.BundleID = "com.example.stdin-test"
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
				os.Exit(0)
			case syscall.SIGTERM:
				fmt.Fprintf(os.Stderr, "\n%s Received SIGTERM (PID: %d), exiting gracefully\n", processType, os.Getpid())
				os.Exit(0)
			}
		}
	}()
}

// testCLIPrompt tests simple CLI prompts with line-based input
func testCLIPrompt() {
	fmt.Println("\n=== CLI Prompt Test ===")
	reader := bufio.NewReader(os.Stdin)

	// Test 1: Simple prompt
	fmt.Print("Enter your name: ")
	name, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading name: %v\n", err)
		return
	}
	name = strings.TrimSpace(name)
	fmt.Printf("Hello, %s!\n", name)

	// Test 2: Multiple prompts
	fmt.Print("Enter your age: ")
	age, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading age: %v\n", err)
		return
	}
	age = strings.TrimSpace(age)
	fmt.Printf("You are %s years old.\n", age)

	// Test 3: Yes/No prompt
	fmt.Print("Continue? (y/n): ")
	answer, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading answer: %v\n", err)
		return
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	if answer == "y" || answer == "yes" {
		fmt.Println("Continuing...")
	} else {
		fmt.Println("Stopping.")
	}
}

// testPasswordInput simulates password input (no echo simulation)
func testPasswordInput() {
	fmt.Println("\n=== Password Input Test ===")
	fmt.Println("Note: This test simulates password input without terminal echo control.")
	fmt.Println("In real scenarios, use golang.org/x/term for proper password input.")

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter password: ")
	password, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading password: %v\n", err)
		return
	}
	password = strings.TrimSpace(password)

	// Don't print the actual password
	fmt.Printf("Password received (length: %d characters)\n", len(password))

	// Verify by asking again
	fmt.Print("Confirm password: ")
	confirm, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading confirmation: %v\n", err)
		return
	}
	confirm = strings.TrimSpace(confirm)

	if password == confirm {
		fmt.Println("✓ Passwords match!")
	} else {
		fmt.Println("✗ Passwords do not match.")
	}
}

// testEOFHandling tests how stdin handles EOF
func testEOFHandling() {
	fmt.Println("\n=== EOF Handling Test ===")
	fmt.Println("Reading lines until EOF (Ctrl+D on Unix, Ctrl+Z on Windows)...")

	reader := bufio.NewReader(os.Stdin)
	lineNum := 0

	for {
		fmt.Printf("Line %d: ", lineNum+1)
		line, err := reader.ReadString('\n')

		if err == io.EOF {
			fmt.Println("\n✓ EOF detected")
			if line != "" {
				// Handle partial line before EOF
				fmt.Printf("  Partial line before EOF: %q\n", strings.TrimSpace(line))
			}
			break
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading line: %v\n", err)
			break
		}

		lineNum++
		fmt.Printf("  Received: %q\n", strings.TrimSpace(line))

		// Limit to prevent infinite loops in tests
		if lineNum >= 10 {
			fmt.Println("  (Limit reached, stopping)")
			break
		}
	}

	fmt.Printf("Total lines read: %d\n", lineNum)

	// Try reading after EOF
	fmt.Println("\nAttempting to read after EOF...")
	moreLine, err := reader.ReadString('\n')
	if err == io.EOF {
		fmt.Println("✓ Still getting EOF (expected)")
	} else if err != nil {
		fmt.Printf("✓ Got error: %v\n", err)
	} else {
		fmt.Printf("✗ Unexpectedly read: %q\n", strings.TrimSpace(moreLine))
	}
}

// testLineBuffering tests line buffering behavior
func testLineBuffering() {
	fmt.Println("\n=== Line Buffering Test ===")
	fmt.Println("Testing different buffering modes...")

	// Test 1: Buffered line reading
	fmt.Println("\n1. Buffered line reading:")
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter a line: ")
	line, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	fmt.Printf("Read buffered line: %q\n", strings.TrimSpace(line))

	// Test 2: Byte-by-byte reading
	fmt.Println("\n2. Byte-by-byte reading (5 bytes):")
	fmt.Print("Enter at least 5 characters: ")
	buf := make([]byte, 5)
	n, err := io.ReadFull(os.Stdin, buf)
	if err != nil && err != io.ErrUnexpectedEOF {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	fmt.Printf("Read %d bytes: %q\n", n, string(buf[:n]))

	// Clear rest of line
	reader.ReadString('\n')

	// Test 3: Scanner (token-based)
	fmt.Println("\n3. Scanner word-by-word:")
	fmt.Print("Enter multiple words: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(bufio.ScanWords)

	words := []string{}
	for i := 0; i < 3 && scanner.Scan(); i++ {
		words = append(words, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Scanner error: %v\n", err)
		return
	}

	fmt.Printf("Read %d words: %v\n", len(words), words)
}

// testControlCharacters tests handling of terminal control characters
func testControlCharacters() {
	fmt.Println("\n=== Control Characters Test ===")
	fmt.Println("Testing various control characters...")

	reader := bufio.NewReader(os.Stdin)

	// Test 1: Tab character
	fmt.Println("\n1. Tab character test:")
	fmt.Print("Enter text with tabs (\\t): ")
	line, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	line = strings.TrimRight(line, "\n\r")
	fmt.Printf("  Raw: %q\n", line)
	fmt.Printf("  Has tabs: %v\n", strings.Contains(line, "\t"))
	fmt.Printf("  Tab count: %d\n", strings.Count(line, "\t"))

	// Test 2: Carriage return
	fmt.Println("\n2. Carriage return test:")
	fmt.Print("Enter text (any input): ")
	line, err = reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	fmt.Printf("  Raw bytes: %v\n", []byte(line))
	fmt.Printf("  Has CR: %v\n", strings.Contains(line, "\r"))
	fmt.Printf("  Has LF: %v\n", strings.Contains(line, "\n"))

	// Test 3: Backspace/Delete detection
	fmt.Println("\n3. Special character detection:")
	fmt.Print("Enter text: ")
	line, err = reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	line = strings.TrimSpace(line)
	fmt.Printf("  Final text: %q\n", line)
	fmt.Printf("  Length: %d bytes\n", len(line))

	// Check for various control characters
	hasControl := false
	for _, r := range line {
		if r < 32 && r != '\t' && r != '\n' && r != '\r' {
			hasControl = true
			break
		}
	}
	fmt.Printf("  Contains control chars: %v\n", hasControl)
}

// testMultiLineInput tests multi-line input handling
func testMultiLineInput() {
	fmt.Println("\n=== Multi-Line Input Test ===")
	fmt.Println("Enter multiple lines. Type 'END' on a line by itself to finish.")

	reader := bufio.NewReader(os.Stdin)
	lines := []string{}

	for {
		fmt.Printf("Line %d: ", len(lines)+1)
		line, err := reader.ReadString('\n')

		if err == io.EOF {
			fmt.Println("\n✓ EOF detected")
			if line != "" {
				lines = append(lines, strings.TrimSpace(line))
			}
			break
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading line: %v\n", err)
			break
		}

		line = strings.TrimSpace(line)

		if line == "END" {
			fmt.Println("✓ End marker detected")
			break
		}

		lines = append(lines, line)

		// Safety limit
		if len(lines) >= 20 {
			fmt.Println("(Limit reached)")
			break
		}
	}

	fmt.Printf("\nReceived %d lines:\n", len(lines))
	for i, line := range lines {
		fmt.Printf("  %d: %q\n", i+1, line)
	}
}

// testInputTimeout tests input with timeout
func testInputTimeout() {
	fmt.Println("\n=== Input Timeout Test ===")
	fmt.Println("You have 5 seconds to enter text...")

	type result struct {
		line string
		err  error
	}

	resultChan := make(chan result, 1)

	// Read in goroutine
	go func() {
		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		resultChan <- result{line: line, err: err}
	}()

	// Wait with timeout
	select {
	case res := <-resultChan:
		if res.err != nil {
			fmt.Fprintf(os.Stderr, "Error reading: %v\n", res.err)
		} else {
			fmt.Printf("✓ Received: %q\n", strings.TrimSpace(res.line))
		}
	case <-time.After(5 * time.Second):
		fmt.Println("✗ Timeout! No input received in 5 seconds.")
	}
}

// testStdinProperties checks stdin properties and capabilities
func testStdinProperties() {
	fmt.Println("\n=== Stdin Properties ===")

	// Check if stdin is a pipe or terminal
	stat, err := os.Stdin.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error statting stdin: %v\n", err)
		return
	}

	mode := stat.Mode()
	fmt.Printf("File mode: %v\n", mode)
	fmt.Printf("Is character device: %v\n", (mode&os.ModeCharDevice) != 0)
	fmt.Printf("Is pipe: %v\n", (mode&os.ModeNamedPipe) != 0)
	fmt.Printf("Is regular file: %v\n", mode.IsRegular())

	// Check file descriptor
	fmt.Printf("Stdin fd: %d\n", os.Stdin.Fd())

	// Try to get terminal size (may fail if not a terminal)
	if (mode & os.ModeCharDevice) != 0 {
		fmt.Println("Stdin appears to be a terminal/character device")
	} else if mode.IsRegular() {
		fmt.Println("Stdin appears to be a regular file")
		// Try to get size
		size := stat.Size()
		fmt.Printf("File size: %d bytes\n", size)
	} else {
		fmt.Println("Stdin appears to be a pipe or special file")
	}
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

	fmt.Printf("=== Stdin Test Running (PID: %d) ===\n", os.Getpid())

	// Always show stdin properties
	testStdinProperties()

	// Run selected tests
	if *testPrompt {
		testCLIPrompt()
	}

	if *testPassword {
		testPasswordInput()
	}

	if *testEOF {
		testEOFHandling()
	}

	if *testLineBuffer {
		testLineBuffering()
	}

	if *testControlChar {
		testControlCharacters()
	}

	if *testMultiLine {
		testMultiLineInput()
	}

	if *testTimeout {
		testInputTimeout()
	}

	// If no specific test selected, run a simple default test
	if !*testPrompt && !*testPassword && !*testEOF && !*testLineBuffer &&
		!*testControlChar && !*testMultiLine && !*testTimeout {
		fmt.Println("\nNo specific test selected. Running simple prompt test...")
		testCLIPrompt()
	}

	fmt.Println("\n=== Test Complete ===")
	os.Exit(0)
}
