// Package main demonstrates a CLI file processing tool with proper macOS file access.
// This example shows how to build a tool that can read and write files with sandbox permissions.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/tmc/misc/macgo"
)

func main() {
	// Parse command-line flags
	var (
		inputPath  = flag.String("input", "", "Input file or directory path")
		outputPath = flag.String("output", "", "Output directory path")
		pattern    = flag.String("pattern", "*.txt", "File pattern to process")
		transform  = flag.String("transform", "uppercase", "Transform to apply (uppercase, lowercase, reverse)")
	)
	flag.Parse()

	// Configure macgo for file access
	macgo.SetAppName("FileProcessor")
	macgo.SetBundleID("com.example.fileprocessor")

	// Request file access permissions
	macgo.RequestEntitlements(
		macgo.EntAppSandbox,
		macgo.EntUserSelectedReadWrite, // Allow user to select files for read/write
		macgo.EntFilesDownloads,        // Access to Downloads folder
		macgo.EntFilesDocuments,        // Access to Documents folder
	)

	// Hide from dock since this is a CLI tool
	macgo.AddPlistEntry("LSUIElement", true)

	// Start macgo
	macgo.Start()

	// Validate inputs
	if *inputPath == "" || *outputPath == "" {
		fmt.Fprintln(os.Stderr, "Usage: file-processor -input <path> -output <path> [-pattern <glob>] [-transform <type>]")
		fmt.Fprintln(os.Stderr, "\nTransforms: uppercase, lowercase, reverse")
		os.Exit(1)
	}

	// Process files
	if err := processFiles(*inputPath, *outputPath, *pattern, *transform); err != nil {
		log.Fatal(err)
	}

	fmt.Println("✓ File processing completed successfully")
}

func processFiles(inputPath, outputPath, pattern, transform string) error {
	// Check if input is a file or directory
	info, err := os.Stat(inputPath)
	if err != nil {
		return fmt.Errorf("cannot access input path: %w", err)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("cannot create output directory: %w", err)
	}

	var files []string
	if info.IsDir() {
		// Find matching files in directory
		matches, err := filepath.Glob(filepath.Join(inputPath, pattern))
		if err != nil {
			return fmt.Errorf("invalid pattern: %w", err)
		}
		files = matches
	} else {
		// Single file
		files = []string{inputPath}
	}

	if len(files) == 0 {
		return fmt.Errorf("no files found matching pattern %s", pattern)
	}

	fmt.Printf("Processing %d file(s)...\n", len(files))

	// Process each file
	for _, file := range files {
		if err := processFile(file, outputPath, transform); err != nil {
			log.Printf("Error processing %s: %v", file, err)
			continue
		}
		fmt.Printf("  ✓ Processed: %s\n", filepath.Base(file))
	}

	return nil
}

func processFile(inputFile, outputDir, transform string) error {
	// Open input file
	in, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("cannot open input file: %w", err)
	}
	defer in.Close()

	// Create output file with transformed suffix
	baseName := filepath.Base(inputFile)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)
	outputFile := filepath.Join(outputDir, fmt.Sprintf("%s_%s%s", nameWithoutExt, transform, ext))

	out, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("cannot create output file: %w", err)
	}
	defer out.Close()

	// Apply transformation
	scanner := bufio.NewScanner(in)
	writer := bufio.NewWriter(out)
	defer writer.Flush()

	for scanner.Scan() {
		line := scanner.Text()
		transformed := applyTransform(line, transform)
		if _, err := writer.WriteString(transformed + "\n"); err != nil {
			return fmt.Errorf("write error: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read error: %w", err)
	}

	return nil
}

func applyTransform(text, transform string) string {
	switch transform {
	case "uppercase":
		return strings.ToUpper(text)
	case "lowercase":
		return strings.ToLower(text)
	case "reverse":
		runes := []rune(text)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return string(runes)
	default:
		return text
	}
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}