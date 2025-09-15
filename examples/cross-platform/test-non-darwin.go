// +build ignore

// This file simulates how macgo behaves on non-Darwin platforms.
// Run with: go run test-non-darwin.go
package main

import (
	"fmt"
)

// Mock runtime.GOOS for testing
const mockGOOS = "linux"

func mockWarnIfNotDarwin(operation string) {
	if mockGOOS != "darwin" {
		fmt.Printf("[DEBUG] macgo: %s has no effect on non-macOS platforms (current: %s)\n", operation, mockGOOS)
	}
}

func mockIsInAppBundle() bool {
	if mockGOOS != "darwin" {
		return false
	}
	return false // Would normally check for actual bundle
}

func main() {
	fmt.Printf("Simulating macgo behavior on %s\n", mockGOOS)

	// Simulate macgo API calls on non-Darwin
	mockWarnIfNotDarwin("requesting entitlements")
	mockWarnIfNotDarwin("setting app name")
	mockWarnIfNotDarwin("enabling code signing")

	fmt.Printf("IsInAppBundle(): %v\n", mockIsInAppBundle())

	fmt.Println("Application runs normally on non-macOS platforms!")
}