// Desktop List - macgo v2
// Simple example that lists Desktop files without sandbox
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
	fmt.Printf("Desktop Lister - macgo v2! PID: %d\n", os.Getpid())
	fmt.Println()

	// Simple setup - no sandbox, just proper app bundling
	// This allows Desktop access without file picker dialogs
	err := macgo.Request()  // No permissions = no sandbox
	if err != nil {
		log.Fatalf("Failed to initialize macgo: %v", err)
	}

	// Try Desktop first (requires permissions)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Error getting home directory: %v", err)
	}

	desktopPath := filepath.Join(homeDir, "Desktop")
	targetDir := desktopPath

	// Try to read Desktop
	entries, err := os.ReadDir(desktopPath)
	if err != nil {
		fmt.Printf("⚠️  Cannot access Desktop: %v\n", err)
		fmt.Println("   Note: Terminal/iTerm may need Full Disk Access in System Settings")
		fmt.Println()

		// Fall back to home directory (showing accessible folders)
		fmt.Println("📂 Listing accessible directories in home instead:")

		// List what we CAN access in home directory
		homeEntries, _ := os.ReadDir(homeDir)
		accessible := []string{}
		for _, e := range homeEntries {
			if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				testPath := filepath.Join(homeDir, e.Name())
				if _, err := os.ReadDir(testPath); err == nil {
					accessible = append(accessible, e.Name())
				}
			}
		}

		if len(accessible) > 0 {
			fmt.Printf("   ✓ Accessible: %s\n", strings.Join(accessible, ", "))
		}

		// Use the macgo project directory for demonstration
		targetDir = "/Volumes/tmc/go/src/github.com/tmc/misc/macgo/v2"
		fmt.Printf("\n📂 Listing macgo v2 directory to demonstrate:\n")
		fmt.Printf("   %s\n", targetDir)

		entries, err = os.ReadDir(targetDir)
		if err != nil {
			// Final fallback to current directory
			targetDir, _ = os.Getwd()
			entries, err = os.ReadDir(targetDir)
			if err != nil {
				log.Fatalf("Error reading directory: %v", err)
			}
		}
	} else {
		fmt.Printf("📂 Listing files in ~/Desktop:\n")
		fmt.Printf("   %s\n", targetDir)
	}

	fmt.Println(strings.Repeat("─", 70))

	if len(entries) == 0 {
		fmt.Println("  (Desktop is empty)")
	} else {
		fileCount := 0
		dirCount := 0
		totalSize := int64(0)

		for i, entry := range entries {
			// Skip hidden files
			if strings.HasPrefix(entry.Name(), ".") {
				continue
			}

			info, err := entry.Info()
			if err != nil {
				continue
			}

			// Determine icon based on type
			icon := "📄"
			if entry.IsDir() {
				icon = "📁"
				dirCount++
			} else {
				fileCount++
				totalSize += info.Size()

				// Set icon based on extension
				name := strings.ToLower(entry.Name())
				switch {
				case strings.HasSuffix(name, ".png"),
				     strings.HasSuffix(name, ".jpg"),
				     strings.HasSuffix(name, ".jpeg"),
				     strings.HasSuffix(name, ".gif"):
					icon = "🖼️"
				case strings.HasSuffix(name, ".pdf"):
					icon = "📑"
				case strings.HasSuffix(name, ".txt"),
				     strings.HasSuffix(name, ".md"):
					icon = "📝"
				case strings.HasSuffix(name, ".zip"),
				     strings.HasSuffix(name, ".tar"),
				     strings.HasSuffix(name, ".gz"):
					icon = "📦"
				case strings.HasSuffix(name, ".mp3"),
				     strings.HasSuffix(name, ".wav"),
				     strings.HasSuffix(name, ".m4a"):
					icon = "🎵"
				case strings.HasSuffix(name, ".mp4"),
				     strings.HasSuffix(name, ".mov"),
				     strings.HasSuffix(name, ".avi"):
					icon = "🎬"
				case strings.HasSuffix(name, ".go"):
					icon = "🐹"
				case strings.HasSuffix(name, ".js"),
				     strings.HasSuffix(name, ".ts"):
					icon = "📜"
				case strings.HasSuffix(name, ".html"),
				     strings.HasSuffix(name, ".css"):
					icon = "🌐"
				}
			}

			// Format size
			size := formatBytes(info.Size())
			if entry.IsDir() {
				size = "–"
			}

			// Print entry
			fmt.Printf("  %s %-40s %10s\n", icon, truncate(entry.Name(), 40), size)

			// Limit output
			if i >= 19 && len(entries) > 20 {
				remaining := len(entries) - 20
				fmt.Printf("\n  ... and %d more items\n", remaining)
				break
			}
		}

		// Print summary
		fmt.Println(strings.Repeat("─", 70))
		fmt.Printf("  📊 Summary: %d files (%s), %d folders\n",
			fileCount, formatBytes(totalSize), dirCount)
	}

	fmt.Println()
	fmt.Println("✨ Benefits of macgo v2:")
	fmt.Println("  • Simple one-line setup: macgo.Request()")
	fmt.Println("  • No sandbox = direct file access")
	fmt.Println("  • Still creates proper .app bundle")
	fmt.Println("  • Add sandbox with: macgo.Request(macgo.Sandbox)")
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