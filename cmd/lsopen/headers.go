package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// openHeaders implements the -h (--header) flag: searches for header files
// matching the given filenames in SDK framework locations and opens them.
func openHeaders(opts *options) error {
	if len(opts.files) == 0 {
		return fmt.Errorf("no header filenames specified")
	}

	// Find Xcode developer dir
	devDir := os.Getenv("DEVELOPER_DIR")
	if devDir == "" {
		out, err := exec.Command("/usr/bin/xcode-select", "-p").Output()
		if err != nil {
			devDir = "/Applications/Xcode.app/Contents/Developer"
		} else {
			devDir = strings.TrimSpace(string(out))
		}
	}

	// Collect search paths
	searchPaths := headerSearchPaths(devDir, opts.sdk)

	var found []string
	for _, name := range opts.files {
		matches := findHeaders(searchPaths, name)
		if len(matches) == 0 {
			fmt.Fprintf(os.Stderr, "Unable to find header file matching %s\n", name)
			continue
		}
		found = append(found, matches...)
	}

	if len(found) == 0 {
		return fmt.Errorf("unable to find header files matching %s", strings.Join(opts.files, ", "))
	}

	// Open found headers with the resolved app (TextEdit or default)
	opts.files = found
	opts.searchHeaders = false
	return run(opts)
}

// headerSearchPaths returns directories to search for headers.
func headerSearchPaths(devDir, sdkFilter string) []string {
	var paths []string

	// System framework headers
	frameworkDirs := []string{
		"/System/Library/Frameworks",
		"/System/Library/PrivateFrameworks",
	}

	// User/local framework headers
	home, _ := os.UserHomeDir()
	if home != "" {
		frameworkDirs = append(frameworkDirs,
			filepath.Join(home, "Library/Frameworks"),
		)
	}
	frameworkDirs = append(frameworkDirs,
		"/Library/Frameworks",
		"/usr/include",
		"/usr/local/include",
	)

	// SDK paths from Xcode
	platformsDir := filepath.Join(devDir, "Platforms")
	if entries, err := os.ReadDir(platformsDir); err == nil {
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".platform") {
				continue
			}
			sdkDir := filepath.Join(platformsDir, e.Name(), "Developer", "SDKs")
			if sdkEntries, err := os.ReadDir(sdkDir); err == nil {
				for _, sdk := range sdkEntries {
					if sdkFilter != "" && !strings.Contains(strings.ToLower(sdk.Name()), strings.ToLower(sdkFilter)) {
						continue
					}
					sdkPath := filepath.Join(sdkDir, sdk.Name())
					paths = append(paths,
						filepath.Join(sdkPath, "System/Library/Frameworks"),
						filepath.Join(sdkPath, "usr/include"),
					)
				}
			}
		}
	}

	paths = append(paths, frameworkDirs...)

	return paths
}

// findHeaders searches for header files matching a name in the given paths.
func findHeaders(searchPaths []string, name string) []string {
	var results []string

	// If name has .h extension already, search for exact match
	// Otherwise search for both name and name.h
	patterns := []string{name}
	if !strings.HasSuffix(name, ".h") {
		patterns = append(patterns, name+".h")
	}

	for _, base := range searchPaths {
		for _, pattern := range patterns {
			// Search in framework Headers/PrivateHeaders directories
			matches, _ := filepath.Glob(filepath.Join(base, "*.framework", "Headers", pattern))
			results = append(results, matches...)
			matches, _ = filepath.Glob(filepath.Join(base, "*.framework", "PrivateHeaders", pattern))
			results = append(results, matches...)

			// Also search directly in include directories
			full := filepath.Join(base, pattern)
			if _, err := os.Stat(full); err == nil {
				results = append(results, full)
			}
		}
	}

	return dedupStrings(results)
}

func dedupStrings(ss []string) []string {
	seen := make(map[string]bool, len(ss))
	var out []string
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
