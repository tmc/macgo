// Sandboxed File Access Example - macgo v2
// Demonstrates sandbox restrictions and file access permissions
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	macgo "github.com/tmc/misc/macgo"
)

func main() {
	fmt.Println("macgo v2 - Sandboxed File Access Example")
	fmt.Println("========================================")
	fmt.Println()

	// Explicit configuration - no magic imports!
	cfg := &macgo.Config{
		AppName:  "SandboxedFileExec",
		BundleID: "com.example.sandboxed-file-exec",

		// Enable app sandbox and user-selected file access
		// This replaces v1's EntAppSandbox + EntUserSelectedReadOnly
		Permissions: []macgo.Permission{
			macgo.Sandbox, // App sandbox isolation
			macgo.Files,   // User-selected file access
		},

		Debug: true,
	}

	// Start macgo with sandbox configuration
	if err := macgo.Start(cfg); err != nil {
		log.Fatalf("Failed to start macgo: %v", err)
	}

	fmt.Println("✓ Running in sandbox mode")
	fmt.Println("✓ File access is restricted to user-selected files")
	fmt.Println()

	// Get home directory
	homeDir, _ := os.UserHomeDir()
	fmt.Printf("Home directory: %s\n", homeDir)

	// Try to access various directories
	dirsToTry := []string{
		homeDir,
		filepath.Join(homeDir, "Documents"),
		filepath.Join(homeDir, "Desktop"),
		filepath.Join(homeDir, "Downloads"),
		"/tmp", // This might be accessible
	}

	fmt.Println("\n1. Attempting to access directories (sandboxed):")
	fmt.Println("------------------------------------------------")
	for _, dir := range dirsToTry {
		fmt.Printf("Reading %s: ", dir)
		files, err := os.ReadDir(dir)
		if err != nil {
			fmt.Printf("❌ BLOCKED: %v\n", err)
			continue
		}

		fmt.Printf("✓ SUCCESS! Found %d files\n", len(files))
		for i, f := range files {
			if i >= 3 {
				fmt.Println("  ...")
				break
			}
			fmt.Printf("  - %s\n", f.Name())
		}
	}

	// Try to execute commands
	fmt.Println("\n2. Attempting to execute commands (sandboxed):")
	fmt.Println("----------------------------------------------")

	commands := []struct {
		name string
		args []string
	}{
		{"ls", []string{"-la", "/tmp"}},
		{"echo", []string{"Hello from sandboxed subprocess"}},
		{"whoami", []string{}},
		{"date", []string{}},
	}

	for _, cmd := range commands {
		fmt.Printf("Executing: %s %v - ", cmd.name, cmd.args)

		execCmd := exec.Command(cmd.name, cmd.args...)
		output, err := execCmd.CombinedOutput()

		if err != nil {
			fmt.Printf("❌ ERROR: %v\n", err)
			continue
		}

		fmt.Printf("✓ SUCCESS!\n")
		outputStr := string(output)
		if len(outputStr) > 100 {
			fmt.Printf("  Output: %s...\n", outputStr[:100])
		} else {
			fmt.Printf("  Output: %s", outputStr)
			if outputStr[len(outputStr)-1] != '\n' {
				fmt.Println()
			}
		}
	}

	fmt.Println("\n3. Key Differences from v1:")
	fmt.Println("--------------------------")
	fmt.Println("• v1: Uses magic import '_ \"github.com/tmc/misc/macgo/auto/sandbox\"'")
	fmt.Println("• v2: Explicit config with 'macgo.Files' permission")
	fmt.Println()
	fmt.Println("• v1: Global state modified at import time")
	fmt.Println("• v2: Configuration passed explicitly to Start()")
	fmt.Println()
	fmt.Println("• v1: Need to know multiple entitlement constants")
	fmt.Println("• v2: Single 'Files' permission for sandboxed file access")

	fmt.Println("\nPress Enter to exit...")
	fmt.Scanln()
}

// Alternative: Even simpler for basic sandbox
func simpleVersion() {
	// One line to enable sandboxed file access
	if err := macgo.Request(macgo.Files); err != nil {
		log.Fatal(err)
	}

	// Your sandboxed app code here...
}
