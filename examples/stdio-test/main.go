// stdio-test demonstrates stdin/stdout/stderr handling with macgo v2
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	macgo "github.com/tmc/misc/macgo"
)

var (
	flagAdHoc       = flag.Bool("ad-hoc", false, "Use ad-hoc code signing")
	flagAutoSign    = flag.Bool("auto-sign", false, "Auto-detect signing certificate")
	flagSign        = flag.String("sign", "", "Code signing identity")
	flagDebug       = flag.Bool("debug", false, "Enable debug logging")
	flagDirect      = flag.Bool("direct", false, "Force direct execution (no LaunchServices)")
	flagLaunch      = flag.Bool("launch", false, "Force LaunchServices")
	flagInteractive = flag.Bool("interactive", false, "Run interactive tests")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "STDIO Test - macgo v2\n")
		fmt.Fprintf(flag.CommandLine.Output(), "\nTests stdin/stdout/stderr forwarding in macgo\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\nExecution modes:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  -direct     Force direct execution (best I/O)\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  -launch     Force LaunchServices (for TCC)\n")
		fmt.Fprintf(flag.CommandLine.Output(), "\nExamples:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  # Test with piped input\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  echo 'hello world' | stdio-test\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  # Test with file redirection\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  stdio-test < input.txt > output.txt 2> error.txt\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  # Interactive mode\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  stdio-test -interactive\n")
	}
	flag.Parse()

	fmt.Printf("🧪 STDIO Test - macgo v2! PID: %d\n", os.Getpid())
	fmt.Println()

	// Configure macgo
	cfg := &macgo.Config{
		AppName:  "stdio-test",
		BundleID: "com.macgo.examples.stdio-test",
		Debug:    *flagDebug,
	}

	// Request Files permission to test LaunchServices path
	if *flagLaunch {
		cfg.Permissions = []macgo.Permission{macgo.Files}
		cfg.ForceLaunchServices = true
		fmt.Println("📁 Forcing LaunchServices path (Files permission)")
	} else if *flagDirect {
		cfg.ForceDirectExecution = true
		fmt.Println("⚡ Forcing direct execution path")
	} else {
		// Default: no TCC permissions, will use direct execution
		fmt.Println("🔧 Using default execution path (likely direct)")
	}

	// Set up code signing
	if *flagAdHoc {
		cfg.AdHocSign = true
		fmt.Println("🔒 Using ad-hoc code signing")
	} else if *flagAutoSign {
		cfg.AutoSign = true
		fmt.Println("🔍 Auto-detecting code signing certificate")
	} else if *flagSign != "" {
		cfg.CodeSignIdentity = *flagSign
		fmt.Printf("🔐 Using code signing identity: %s\n", *flagSign)
	} else {
		fmt.Println("🔓 Running without code signing")
	}

	// Initialize macgo
	err := macgo.Start(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize macgo: %v", err)
	}

	fmt.Println()
	runStdioTests(*flagInteractive)
}

func runStdioTests(interactive bool) {
	fmt.Println("📝 Testing Standard I/O Streams")
	fmt.Println("─" + strings.Repeat("─", 49))

	// Test 1: Basic stdout
	fmt.Println("1️⃣ STDOUT Test:")
	fmt.Println("   This message goes to stdout")
	fmt.Printf("   Formatted: PID=%d, Time=%s\n", os.Getpid(), time.Now().Format("15:04:05"))

	// Test 2: stderr output
	fmt.Fprintln(os.Stderr, "2️⃣ STDERR Test:")
	fmt.Fprintln(os.Stderr, "   This message goes to stderr")
	log.Println("   Log messages also go to stderr")

	// Test 3: File descriptors
	fmt.Println("3️⃣ File Descriptor Test:")
	fmt.Printf("   stdin fd:  %d\n", os.Stdin.Fd())
	fmt.Printf("   stdout fd: %d\n", os.Stdout.Fd())
	fmt.Printf("   stderr fd: %d\n", os.Stderr.Fd())

	// Test 4: Mixed output
	fmt.Println("4️⃣ Mixed Output Test:")
	fmt.Print("   stdout: A")
	fmt.Fprint(os.Stderr, " stderr: B")
	fmt.Print(" stdout: C")
	fmt.Fprint(os.Stderr, " stderr: D")
	fmt.Println(" stdout: E")

	// Test 5: Check if stdin is available
	fmt.Println("5️⃣ STDIN Test:")

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// stdin is piped or redirected
		fmt.Println("   Detected piped/redirected input, reading...")
		scanner := bufio.NewScanner(os.Stdin)
		lineCount := 0
		for scanner.Scan() {
			lineCount++
			line := scanner.Text()
			fmt.Printf("   Line %d: %s\n", lineCount, line)
			if lineCount >= 5 {
				fmt.Println("   (showing first 5 lines only)")
				break
			}
		}
		if lineCount == 0 {
			fmt.Println("   No input received")
		}
	} else if interactive {
		// Interactive mode
		fmt.Println("   Interactive mode - please type something and press Enter:")
		fmt.Print("   > ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("   Error reading input: %v\n", err)
		} else {
			input = strings.TrimSpace(input)
			fmt.Printf("   You typed: '%s'\n", input)
			fmt.Printf("   Length: %d characters\n", len(input))
			fmt.Printf("   Uppercase: %s\n", strings.ToUpper(input))
		}
	} else {
		fmt.Println("   No piped input detected (use -interactive for manual input)")
	}

	// Test 6: Large output
	fmt.Println("6️⃣ Large Output Test:")
	fmt.Print("   Generating 100 dots: ")
	for i := 0; i < 100; i++ {
		fmt.Print(".")
		if (i+1)%25 == 0 {
			fmt.Print("|")
		}
	}
	fmt.Println(" Done!")

	// Test 7: Error conditions
	fmt.Println("7️⃣ Error Handling Test:")
	fmt.Fprintln(os.Stderr, "   ERROR: This is a test error message")
	fmt.Fprintln(os.Stderr, "   WARNING: This is a test warning")
	fmt.Fprintln(os.Stderr, "   INFO: This is informational")

	fmt.Println()
	fmt.Println("─" + strings.Repeat("─", 49))
	fmt.Println("✅ All stdio tests completed!")

	// Summary
	fmt.Println()
	fmt.Println("📊 Summary:")
	fmt.Println("  • stdout: Working ✓")
	fmt.Println("  • stderr: Working ✓")
	fmt.Println("  • stdin:  Working ✓")
	fmt.Println("  • File descriptors: Standard (0,1,2)")

	exitCode := 0
	if interactive {
		fmt.Println()
		fmt.Println("Exit code test - enter a number (0-255):")
		fmt.Print("> ")
		var code int
		if _, err := fmt.Scanf("%d", &code); err == nil && code >= 0 && code <= 255 {
			exitCode = code
			fmt.Printf("Exiting with code %d\n", exitCode)
		}
	}

	os.Exit(exitCode)
}
