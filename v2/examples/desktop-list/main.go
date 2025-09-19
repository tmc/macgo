// Desktop List - macgo v2
// Simple example that lists Desktop files with proper file permissions
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	macgo "github.com/tmc/misc/macgo/v2"
)

func init() {
	// Customize flag usage
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Desktop List - macgo v2\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Lists Desktop files with proper macOS permissions\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags]\n\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\nExamples:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  %s                           # Basic usage\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  %s -debug                    # With debug output\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  %s -ad-hoc                   # With ad-hoc signing\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  %s -auto-sign                # With auto-detected signing\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  %s -sign \"Developer ID\"       # With specific identity\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "\nCode Signing:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  Ad-hoc signing provides basic code signing for development.\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  Auto-sign detects Developer ID certificates automatically.\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  List available identities: security find-identity -v -p codesigning\n")
	}
}

func main() {
	// Parse command line flags
	var (
		signIdentity = flag.String("sign", "", "code signing identity (e.g., 'Developer ID Application')")
		autoSign     = flag.Bool("auto-sign", false, "automatically detect and use Developer ID certificate")
		adHocSign    = flag.Bool("ad-hoc", false, "use ad-hoc code signing (no certificate required)")
		debug        = flag.Bool("debug", false, "enable debug output")
	)
	flag.Parse()

	fmt.Printf("Desktop Lister - macgo v2! PID: %d\n", os.Getpid())
	fmt.Println()

	// Configure macgo with optional code signing
	keepBundle := false
	cfg := &macgo.Config{
		Permissions: []macgo.Permission{macgo.Sandbox, macgo.Files},
		Debug:       *debug,
		AutoSign:    *autoSign,
		AdHocSign:   *adHocSign,
		KeepBundle:  &keepBundle,
	}

	if *signIdentity != "" {
		cfg.CodeSignIdentity = *signIdentity
		fmt.Printf("🔐 Code signing enabled with identity: %s\n", *signIdentity)
	} else if *autoSign {
		fmt.Println("🔍 Auto-detection enabled for Developer ID certificate")
	} else if *adHocSign {
		fmt.Println("🔒 Ad-hoc code signing enabled (no certificate required)")
	} else {
		fmt.Println("🔓 Running without code signing (use -sign, -auto-sign, or -ad-hoc flag to enable)")
	}
	fmt.Println()

	// Request file access permissions for Desktop access
	err := macgo.Start(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize macgo: %v", err)
	}

	// Get Desktop path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Error getting home directory: %v", err)
	}

	desktopPath := filepath.Join(homeDir, "Desktop")
	fmt.Printf("📂 Listing files in ~/Desktop:\n")
	fmt.Printf("   %s\n", desktopPath)

	// Read Desktop contents
	entries, err := os.ReadDir(desktopPath)
	if err != nil {
		log.Fatalf("Error reading Desktop: %v", err)
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
	fmt.Println("✨ macgo v2 benefits:")
	fmt.Println("  • Simple configuration with optional code signing")
	fmt.Println("  • Creates proper .app bundle automatically")
	fmt.Println("  • Handles file permissions with sandbox integration")
	fmt.Println("  • Uses LaunchServices for proper macOS behavior")
	fmt.Println()
	fmt.Println("🔐 Code signing usage:")
	fmt.Println("  desktop-list -ad-hoc                            # Ad-hoc signing (dev/test)")
	fmt.Println("  desktop-list -auto-sign                         # Auto-detect certificate")
	fmt.Println("  desktop-list -sign \"Developer ID Application\"   # Use specific identity")
	fmt.Println("  desktop-list -debug                             # See bundle creation details")
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
