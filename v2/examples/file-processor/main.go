// Package main demonstrates a CLI file processing tool using the v2 macgo API.
// This example shows the cleaner, more explicit configuration approach of v2.
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

	macgo "github.com/tmc/misc/macgo/v2"
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

	// Configure macgo v2 - much cleaner than v1!
	cfg := &macgo.Config{
		AppName:  "FileProcessor",
		BundleID: "com.example.fileprocessor",
		Permissions: []macgo.Permission{
			macgo.Files,    // File read/write access
			macgo.Network,  // Optional: for downloading files
		},
		// Hide from dock since this is a CLI tool
		LSUIElement: true,
		Debug:       os.Getenv("MACGO_DEBUG") == "1",
	}

	// Start macgo with the configuration
	if err := macgo.Start(cfg); err != nil {
		log.Fatalf("Failed to start macgo: %v", err)
	}

	// Validate inputs
	if *inputPath == "" || *outputPath == "" {
		fmt.Fprintln(os.Stderr, "Usage: file-processor -input <path> -output <path> [-pattern <glob>] [-transform <type>]")
		fmt.Fprintln(os.Stderr, "\nTransforms: uppercase, lowercase, reverse")
		fmt.Fprintln(os.Stderr, "\nExample: MACGO_DEBUG=1 go run main.go -input ~/Documents -output ~/processed -pattern '*.txt' -transform uppercase")
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

	lineCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		transformed := applyTransform(line, transform)
		if _, err := writer.WriteString(transformed + "\n"); err != nil {
			return fmt.Errorf("write error: %w", err)
		}
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read error: %w", err)
	}

	fmt.Printf("    Lines processed: %d\n", lineCount)
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
	case "wordcount":
		words := strings.Fields(text)
		return fmt.Sprintf("%s [%d words]", text, len(words))
	case "title":
		return strings.Title(strings.ToLower(text))
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