//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func main() {
	if runtime.GOOS != "darwin" {
		fmt.Println("Benchmarks are designed for macOS")
		return
	}

	benchmarks := []struct {
		name     string
		pattern  string
		timeout  time.Duration
		expected bool // whether we expect this to work
	}{
		{"Bundle Creation", "BenchmarkBundleCreation", 2 * time.Minute, true},
		{"Bundle Creation with Entitlements", "BenchmarkBundleCreationWithEntitlements", 3 * time.Minute, true},
		{"Bundle Creation Memory", "BenchmarkBundleCreationMemory", 2 * time.Minute, true},
		{"Bundle Path Security", "BenchmarkPathSecurity", 1 * time.Minute, true},
		{"Bundle Plist Writing", "BenchmarkPlistWriting", 1 * time.Minute, true},
		{"Configuration Creation", "BenchmarkConfigCreation", 1 * time.Minute, true},
		{"Configuration Merging", "BenchmarkConfigMerging", 2 * time.Minute, true},
		{"Environment Variable Parsing", "BenchmarkEnvironmentVariableParsing", 1 * time.Minute, true},
		{"Plist Generation", "BenchmarkPlistGeneration", 2 * time.Minute, true},
		{"Security Path Validation", "BenchmarkPathValidation", 1 * time.Minute, true},
		{"Security XML Escaping", "BenchmarkXMLEscaping", 1 * time.Minute, true},
		{"Security Checksum Calculation", "BenchmarkChecksumCalculation", 2 * time.Minute, true},
		{"App Launch (may fail in CI)", "BenchmarkAppLaunch", 3 * time.Minute, false},
		{"Named Pipe Creation", "BenchmarkNamedPipeCreation", 1 * time.Minute, true},
		{"I/O Redirection", "BenchmarkIORedirection", 2 * time.Minute, true},
	}

	fmt.Println("Running macgo performance benchmarks...")
	fmt.Println("=" * 50)

	results := make(map[string]string)

	for _, bench := range benchmarks {
		fmt.Printf("Running %s...\n", bench.name)

		cmd := exec.Command("go", "test", "-bench", bench.pattern, "-benchmem", "-count=1", "-timeout", bench.timeout.String())
		cmd.Dir = "."

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		if err != nil {
			if bench.expected {
				fmt.Printf("  ❌ FAILED: %s\n", bench.name)
				fmt.Printf("  Error: %v\n", err)
				if len(outputStr) > 0 {
					fmt.Printf("  Output: %s\n", truncateOutput(outputStr))
				}
			} else {
				fmt.Printf("  ⚠️  EXPECTED FAILURE: %s\n", bench.name)
			}
			results[bench.name] = fmt.Sprintf("FAILED: %v", err)
		} else {
			fmt.Printf("  ✅ SUCCESS: %s\n", bench.name)
			results[bench.name] = extractBenchmarkResults(outputStr)
		}

		fmt.Println()
	}

	// Print summary
	fmt.Println("=" * 50)
	fmt.Println("BENCHMARK SUMMARY")
	fmt.Println("=" * 50)

	for _, bench := range benchmarks {
		result := results[bench.name]
		status := "✅"
		if strings.Contains(result, "FAILED") {
			if bench.expected {
				status = "❌"
			} else {
				status = "⚠️"
			}
		}
		fmt.Printf("%s %s: %s\n", status, bench.name, result)
	}
}

func truncateOutput(output string) string {
	lines := strings.Split(output, "\n")
	if len(lines) > 10 {
		return strings.Join(lines[:10], "\n") + "\n... (truncated)"
	}
	return output
}

func extractBenchmarkResults(output string) string {
	lines := strings.Split(output, "\n")
	var benchResults []string

	for _, line := range lines {
		if strings.Contains(line, "Benchmark") && strings.Contains(line, "ns/op") {
			// Extract just the benchmark name and performance
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				benchResults = append(benchResults, fmt.Sprintf("%s: %s %s", parts[0], parts[2], parts[3]))
			}
		}
	}

	if len(benchResults) == 0 {
		return "No benchmark results found"
	}

	return strings.Join(benchResults, "; ")
}
