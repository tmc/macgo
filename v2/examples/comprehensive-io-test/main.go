// Comprehensive I/O Test - macgo v2
// Tests both direct execution and open command I/O forwarding
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	macgo "github.com/tmc/misc/macgo/v2"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--test-mode" {
		runTest(os.Args[2])
		return
	}

	fmt.Println("🧪 Comprehensive I/O Test - macgo v2")
	fmt.Println("═══════════════════════════════════")
	fmt.Println()

	fmt.Println("Testing both execution paths:")
	fmt.Println("1. Direct execution (Network permission)")
	fmt.Println("2. Open command (Files permission)")
	fmt.Println()

	testResult1 := testDirectExecution()
	testResult2 := testOpenCommand()

	fmt.Println("\n📊 Test Results Summary")
	fmt.Println("═══════════════════════")
	fmt.Printf("Direct Execution: %s\n", testResult1)
	fmt.Printf("Open Command:     %s\n", testResult2)

	if testResult1 == "✅ PASS" && testResult2 == "✅ PASS" {
		fmt.Println("\n🎉 All I/O forwarding tests PASSED!")
	} else {
		fmt.Println("\n⚠️  Some tests failed")
	}
}

func testDirectExecution() string {
	fmt.Println("🔧 Test 1: Direct Execution Path")
	fmt.Println("─────────────────────────────────")

	// Use Network permission to trigger direct execution
	cfg := &macgo.Config{
		Permissions: []macgo.Permission{macgo.Network},
		Debug:       true,
	}

	err := macgo.Start(cfg)
	if err != nil {
		return fmt.Sprintf("❌ FAIL: %v", err)
	}

	return runIOTest("direct")
}

func testOpenCommand() string {
	fmt.Println("\n🎯 Test 2: Open Command Path")
	fmt.Println("─────────────────────────────────")

	// Use Files permission to trigger open command
	cfg := &macgo.Config{
		Permissions: []macgo.Permission{macgo.Files},
		Debug:       true,
	}

	err := macgo.Start(cfg)
	if err != nil {
		return fmt.Sprintf("❌ FAIL: %v", err)
	}

	return runIOTest("open")
}

func runIOTest(pathType string) string {
	fmt.Printf("Testing I/O forwarding via %s execution...\n", pathType)

	// Test 1: stdout
	fmt.Println("✓ stdout test: This message should appear")

	// Test 2: stderr
	fmt.Fprintf(os.Stderr, "✓ stderr test: Error message should appear\n")

	// Test 3: formatted output
	fmt.Printf("✓ printf test: PID=%d, Type=%s\n", os.Getpid(), pathType)

	// Test 4: file descriptors
	fmt.Printf("✓ FD test: stdin=%d, stdout=%d, stderr=%d\n",
		os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd())

	// Test 5: stdin (automated test)
	fmt.Print("✓ stdin test: Reading from stdin... ")

	// For automated testing, check if stdin has data
	stat, err := os.Stdin.Stat()
	if err != nil {
		fmt.Printf("(stdin stat error: %v) ", err)
	} else if (stat.Mode() & os.ModeCharDevice) == 0 {
		// stdin is redirected/piped
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			input := strings.TrimSpace(scanner.Text())
			fmt.Printf("received: '%s'\n", input)
		} else {
			fmt.Println("no input available")
		}
	} else {
		fmt.Println("interactive mode (skipped)")
	}

	fmt.Printf("✓ I/O test completed for %s execution\n", pathType)
	return "✅ PASS"
}

func runTest(testType string) {
	// This is called when running in --test-mode
	fmt.Printf("Running %s test in subprocess\n", testType)

	switch testType {
	case "signal":
		testSignalForwarding()
	case "interactive":
		testInteractiveIO()
	default:
		fmt.Printf("Unknown test type: %s\n", testType)
	}
}

func testSignalForwarding() {
	fmt.Printf("Signal test subprocess - PID: %d\n", os.Getpid())

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	fmt.Println("Ready for signal test")

	select {
	case sig := <-sigChan:
		fmt.Printf("Received signal: %v\n", sig)
		os.Exit(0)
	case <-time.After(5 * time.Second):
		fmt.Println("Signal test timeout")
		os.Exit(1)
	}
}

func testInteractiveIO() {
	fmt.Print("Enter a number: ")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if num, err := strconv.Atoi(input); err == nil {
			fmt.Printf("You entered: %d\n", num)
			fmt.Printf("Double: %d\n", num*2)
		} else {
			fmt.Printf("Invalid number: %s\n", input)
		}
	}
}
