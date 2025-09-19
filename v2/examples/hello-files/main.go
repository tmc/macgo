// Hello Files - macgo v2
// Demonstrates file access permissions with Desktop listing
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	macgo "github.com/tmc/misc/macgo/v2"
)

func main() {
	fmt.Printf("Hello from macgo v2 with Files! PID: %d\n", os.Getpid())
	fmt.Println()

	// Request sandbox and file permissions - single line!
	err := macgo.Request(macgo.Sandbox, macgo.Files)
	if err != nil {
		log.Fatalf("Failed to request file permissions: %v", err)
	}

	fmt.Println("✓ File permissions granted!")
	fmt.Println()

	// Try to list files on Desktop
	// Note: With sandbox enabled, this will only work if:
	// 1. User has granted access via file picker dialog
	// 2. Or app is running without sandbox (MACGO_NO_SANDBOX=1)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Error getting home directory: %v", err)
		return
	}

	desktopPath := filepath.Join(homeDir, "Desktop")
	fmt.Printf("📁 Attempting to list files in %s:\n", desktopPath)
	fmt.Println(strings.Repeat("-", 60))

	entries, err := os.ReadDir(desktopPath)
	if err != nil {
		fmt.Printf("  ⚠️  Cannot access Desktop: %v\n", err)
		fmt.Println()
		fmt.Println("  This is expected with sandbox enabled!")
		fmt.Println("  To access Desktop files, you need to either:")
		fmt.Println("  1. Use a file picker dialog (recommended)")
		fmt.Println("  2. Run without sandbox: MACGO_NO_SANDBOX=1 go run main.go")
		fmt.Println("  3. Grant full disk access in System Settings")
		fmt.Println()

		// Try to access a file we can create ourselves
		fmt.Println("  Let's try accessing temp directory instead...")
		tempDir := os.TempDir()
		testFile := filepath.Join(tempDir, "macgo-test.txt")

		// Create a test file
		if err := os.WriteFile(testFile, []byte("Hello from macgo v2!"), 0644); err != nil {
			fmt.Printf("    ❌ Cannot write to temp: %v\n", err)
		} else {
			fmt.Printf("    ✓ Created test file: %s\n", testFile)

			// Read it back
			if data, err := os.ReadFile(testFile); err != nil {
				fmt.Printf("    ❌ Cannot read test file: %v\n", err)
			} else {
				fmt.Printf("    ✓ Read test file: %s\n", string(data))
			}

			// Clean up
			os.Remove(testFile)
		}
		return
	}

	if len(entries) == 0 {
		fmt.Println("  (Desktop is empty)")
	} else {
		count := 0
		for _, entry := range entries {
			// Skip hidden files
			if strings.HasPrefix(entry.Name(), ".") {
				continue
			}

			info, err := entry.Info()
			if err != nil {
				continue
			}

			icon := "📄"
			if entry.IsDir() {
				icon = "📁"
			} else if strings.HasSuffix(strings.ToLower(entry.Name()), ".png") ||
				strings.HasSuffix(strings.ToLower(entry.Name()), ".jpg") ||
				strings.HasSuffix(strings.ToLower(entry.Name()), ".jpeg") {
				icon = "🖼️"
			} else if strings.HasSuffix(strings.ToLower(entry.Name()), ".pdf") {
				icon = "📑"
			} else if strings.HasSuffix(strings.ToLower(entry.Name()), ".txt") ||
				strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
				icon = "📝"
			}

			size := formatBytes(info.Size())
			if entry.IsDir() {
				size = "-"
			}

			fmt.Printf("  %s %-30s %10s\n", icon, truncate(entry.Name(), 30), size)

			count++
			if count >= 10 {
				remaining := len(entries) - count
				if remaining > 0 {
					fmt.Printf("\n  ... and %d more items\n", remaining)
				}
				break
			}
		}
	}

	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("Total items shown: %d\n", min(len(entries), 10))
	fmt.Println()

	// Also try Downloads folder
	downloadsPath := filepath.Join(homeDir, "Downloads")
	fmt.Printf("📥 Checking Downloads folder access:\n")

	if entries, err := os.ReadDir(downloadsPath); err != nil {
		fmt.Printf("  ❌ Cannot access Downloads: %v\n", err)
	} else {
		fmt.Printf("  ✓ Downloads folder accessible (%d items)\n", len(entries))
	}

	fmt.Println()
	fmt.Println("🎯 Key points about v2 file access:")
	fmt.Println("  • Single permission: macgo.Files")
	fmt.Println("  • Replaces multiple v1 entitlements")
	fmt.Println("  • User can grant access to specific folders")
	fmt.Println("  • Works with sandbox restrictions")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
