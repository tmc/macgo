package main

import (
	"fmt"
	"os/exec"
)

// Simple test functions to verify sdef execution works

func testMusicPlay() {
	fmt.Println("=== Testing Music.app play command ===")
	script := `tell application "Music" to play`
	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	fmt.Printf("Script: %s\n", script)
	if err != nil {
		fmt.Printf("Error: %v\n%s\n", err, output)
	} else {
		fmt.Printf("Success\n%s\n", output)
	}
}

func testMusicPause() {
	fmt.Println("\n=== Testing Music.app pause command ===")
	script := `tell application "Music" to pause`
	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	fmt.Printf("Script: %s\n", script)
	if err != nil {
		fmt.Printf("Error: %v\n%s\n", err, output)
	} else {
		fmt.Printf("Success\n%s\n", output)
	}
}

func testMusicStatus() {
	fmt.Println("\n=== Testing Music.app player state ===")
	script := `tell application "Music" to get player state`
	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	fmt.Printf("Script: %s\n", script)
	if err != nil {
		fmt.Printf("Error: %v\n%s\n", err, output)
	} else {
		fmt.Printf("Player state: %s\n", output)
	}
}

func testSafariURL() {
	fmt.Println("\n=== Testing Safari.app get URL ===")
	script := `tell application "Safari" to get URL of front document`
	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	fmt.Printf("Script: %s\n", script)
	if err != nil {
		fmt.Printf("Error: %v\n%s\n", err, output)
	} else {
		fmt.Printf("URL: %s\n", output)
	}
}

func testSafariOpenURL() {
	fmt.Println("\n=== Testing Safari.app open URL ===")
	script := `tell application "Safari"
	make new document
	set URL of front document to "https://golang.org"
end tell`
	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	fmt.Printf("Script: %s\n", script)
	if err != nil {
		fmt.Printf("Error: %v\n%s\n", err, output)
	} else {
		fmt.Printf("Success: Opened golang.org\n%s\n", output)
	}
}
