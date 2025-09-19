// Package main provides an osascript wrapper for managing named AppleScripts.
// This example demonstrates how to create, store, and execute named scripts
// for tasks like browser automation and system control.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	macgo "github.com/tmc/misc/macgo"
)

var (
	script     = flag.String("script", "", "name of the script to run")
	list       = flag.Bool("list", false, "list all available scripts")
	create     = flag.String("create", "", "create a new script with the given name")
	edit       = flag.String("edit", "", "edit an existing script")
	remove     = flag.String("remove", "", "remove a script")
	scriptsDir = flag.String("dir", "", "scripts directory (default: ~/.osascripts)")
	help       = flag.Bool("help", false, "show help")
	prefs      = flag.Bool("prefs", false, "open Privacy & Security preferences")
)

func main() {
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	// Determine app name based on what we're doing
	appName := getAppName()
	if *script != "" {
		// Use script name as app name when running a script
		appName = *script
	}

	// Request necessary permissions including automation
	cfg := &macgo.Config{
		AppName:     appName,
		Permissions: []macgo.Permission{macgo.Files},
		Custom: []string{
			"com.apple.security.automation.apple-events",
		},
		Debug: os.Getenv("MACGO_DEBUG") == "1",
	}

	err := macgo.Start(cfg)
	if err != nil {
		log.Fatalf("Failed to request permissions: %v", err)
	}

	// Determine scripts directory
	dir := getScriptsDir()
	if err := ensureScriptsDir(dir); err != nil {
		log.Fatalf("Failed to create scripts directory: %v", err)
	}

	// Get bundle resources directory for embedded scripts
	bundleDir := getBundleResourcesDir()
	if bundleDir != "" {
		if err := ensureBundleScriptsDir(bundleDir); err != nil {
			log.Printf("Warning: Failed to create bundle scripts directory: %v", err)
		}
	}

	// Handle commands
	switch {
	case *prefs:
		openPrivacyPrefs()
	case *list:
		listScripts(dir)
	case *create != "":
		createScript(dir, *create)
	case *edit != "":
		editScript(dir, *edit)
	case *remove != "":
		removeScript(dir, *remove)
	case *script != "":
		runScript(dir, *script)
	default:
		showHelp()
	}
}

func getAppName() string {
	if len(os.Args) > 0 {
		name := filepath.Base(os.Args[0])
		// Remove common Go build artifacts
		if strings.HasPrefix(name, "go-build") || strings.HasPrefix(name, "__debug_bin") {
			return "osascript-wrapper"
		}
		return name
	}
	return "_"
}

func getScriptsDir() string {
	if *scriptsDir != "" {
		return *scriptsDir
	}

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %v", err)
	}

	return filepath.Join(home, ".osascripts")
}

func ensureScriptsDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}

func listScripts(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Printf("No scripts directory found at %s\n", dir)
		return
	}

	var scripts []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".applescript") {
			name := strings.TrimSuffix(entry.Name(), ".applescript")
			scripts = append(scripts, name)
		}
	}

	if len(scripts) == 0 {
		fmt.Println("No scripts found")
		return
	}

	fmt.Printf("Available scripts in %s:\n", dir)
	for _, script := range scripts {
		fmt.Printf("  %s\n", script)
	}
}

func createScript(dir, name string) {
	scriptPath := filepath.Join(dir, name+".applescript")

	// Check if script already exists
	if _, err := os.Stat(scriptPath); err == nil {
		fmt.Printf("Script '%s' already exists. Use -edit to modify it.\n", name)
		return
	}

	// Get script template based on name
	template := getScriptTemplate(name)

	err := os.WriteFile(scriptPath, []byte(template), 0644)
	if err != nil {
		log.Fatalf("Failed to create script: %v", err)
	}

	fmt.Printf("Created script '%s' at %s\n", name, scriptPath)
	fmt.Printf("Edit it with: %s -edit %s\n", getAppName(), name)
}

func editScript(dir, name string) {
	scriptPath := filepath.Join(dir, name+".applescript")

	// Check if script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		fmt.Printf("Script '%s' not found. Use -create to create it.\n", name)
		return
	}

	// Open in default editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nano" // fallback to nano
	}

	cmd := exec.Command(editor, scriptPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		log.Fatalf("Failed to edit script: %v", err)
	}
}

func removeScript(dir, name string) {
	scriptPath := filepath.Join(dir, name+".applescript")

	err := os.Remove(scriptPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Script '%s' not found\n", name)
		} else {
			log.Fatalf("Failed to remove script: %v", err)
		}
		return
	}

	fmt.Printf("Removed script '%s'\n", name)
}

func runScript(dir, name string) {
	// Sync the specific script to bundle if we're in a bundle
	bundleDir := getBundleResourcesDir()
	if bundleDir != "" {
		if err := syncScriptToBundle(dir, bundleDir, name); err != nil {
			log.Printf("Warning: Failed to sync script to bundle: %v", err)
		}
	}

	// Try to find script in bundle first, then fall back to external directory
	scriptPath := findScript(dir, name)
	if scriptPath == "" {
		fmt.Printf("Script '%s' not found. Available scripts:\n", name)
		listScripts(dir)
		return
	}

	fmt.Printf("Running script '%s' from %s...\n", name, scriptPath)

	// Get any additional arguments to pass to the script
	scriptArgs := flag.Args()

	// Execute the script with arguments
	cmdArgs := []string{scriptPath}
	cmdArgs = append(cmdArgs, scriptArgs...)

	cmd := exec.Command("osascript", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		log.Fatalf("Failed to run script: %v", err)
	}
}

func getScriptTemplate(name string) string {
	switch {
	case strings.Contains(strings.ToLower(name), "brave"):
		return braveTemplate
	case strings.Contains(strings.ToLower(name), "chrome"):
		return chromeTemplate
	case strings.Contains(strings.ToLower(name), "safari"):
		return safariTemplate
	case strings.Contains(strings.ToLower(name), "finder"):
		return finderTemplate
	default:
		return defaultTemplate
	}
}

const defaultTemplate = `-- Default AppleScript template
-- Edit this script to perform your desired actions

tell application "System Events"
	display dialog "Hello from osascript wrapper!"
end tell
`

const braveTemplate = `-- Brave Browser Automation Script
-- Automate Brave browser tasks

tell application "Brave Browser"
	activate

	-- Create new tab and navigate
	tell window 1
		set newTab to make new tab with properties {URL:"https://example.com"}
		set active tab index to (count of tabs)
	end tell

	-- Wait for page to load
	delay 2

	-- Example: Execute JavaScript
	tell active tab of window 1
		set pageTitle to execute javascript "document.title"
	end tell

	display dialog "Page title: " & pageTitle
end tell
`

const chromeTemplate = `-- Chrome Browser Automation Script
-- Automate Chrome browser tasks

tell application "Google Chrome"
	activate

	-- Create new tab and navigate
	tell window 1
		set newTab to make new tab with properties {URL:"https://example.com"}
		set active tab index to (count of tabs)
	end tell

	-- Wait for page to load
	delay 2

	-- Example: Get page title
	tell active tab of window 1
		set pageTitle to execute javascript "document.title"
	end tell

	display dialog "Page title: " & pageTitle
end tell
`

const safariTemplate = `-- Safari Browser Automation Script
-- Automate Safari browser tasks

tell application "Safari"
	activate

	-- Create new tab and navigate
	tell window 1
		set current tab to (make new tab with properties {URL:"https://example.com"})
	end tell

	-- Wait for page to load
	delay 2

	-- Example: Get page title
	tell current tab of window 1
		set pageTitle to name
	end tell

	display dialog "Page title: " & pageTitle
end tell
`

const finderTemplate = `-- Finder Automation Script
-- Automate Finder tasks

tell application "Finder"
	activate

	-- Open a new window to the home folder
	make new Finder window to home folder

	-- Example: List files in current directory
	set fileList to name of every file in the desktop

	display dialog "Files on desktop: " & (count of fileList)
end tell
`

func openPrivacyPrefs() {
	fmt.Println("Opening Privacy & Security preferences...")

	// Try System Settings first (macOS 13+), then fall back to System Preferences
	scripts := []string{
		`tell application "System Settings"
	activate
	reveal anchor "Privacy_Automation" of pane id "com.apple.preference.security"
end tell`,
		`tell application "System Preferences"
	activate
	set the current pane to pane id "com.apple.preference.security"
	reveal anchor "Privacy_Automation"
end tell`,
	}

	var lastErr error
	for _, script := range scripts {
		cmd := exec.Command("osascript", "-e", script)
		err := cmd.Run()
		if err == nil {
			fmt.Println("Navigate to Automation in the left sidebar and grant your terminal permission to control Brave Browser.")
			return
		}
		lastErr = err
	}

	fmt.Printf("Failed to open Privacy preferences: %v\n", lastErr)
	fmt.Println("Please manually open:")
	fmt.Println("- macOS 13+: System Settings > Privacy & Security > Automation")
	fmt.Println("- macOS 12-: System Preferences > Security & Privacy > Automation")
	fmt.Println("Then grant your terminal permission to control Brave Browser.")
}

func showHelp() {
	fmt.Println("OSAScript Wrapper - Manage and run named AppleScripts")
	fmt.Println("==================================================")
	fmt.Println()
	fmt.Println("Usage:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	appName := getAppName()
	fmt.Printf("  %s -create brave-automation     # Create a new Brave automation script\n", appName)
	fmt.Printf("  %s -script brave-automation     # Run the brave-automation script\n", appName)
	fmt.Printf("  %s -list                        # List all available scripts\n", appName)
	fmt.Printf("  %s -edit brave-automation       # Edit the script in $EDITOR\n", appName)
	fmt.Printf("  %s -remove old-script           # Remove a script\n", appName)
	fmt.Printf("  %s -prefs                       # Open Privacy & Security preferences\n", appName)
	fmt.Println()
	fmt.Println("Script Templates:")
	fmt.Println("  brave-*    : Creates Brave browser automation template")
	fmt.Println("  chrome-*   : Creates Chrome browser automation template")
	fmt.Println("  safari-*   : Creates Safari browser automation template")
	fmt.Println("  finder-*   : Creates Finder automation template")
	fmt.Println("  other      : Creates generic template")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  EDITOR     : Text editor to use for editing scripts (default: nano)")
	fmt.Println()
	fmt.Println("Scripts are stored in ~/.osascripts/ by default")
	fmt.Println("When running in an app bundle, scripts are embedded in the bundle and auto-updated")
}

// getBundleResourcesDir returns the Resources directory of the current app bundle, if we're running in one
func getBundleResourcesDir() string {
	executable, err := os.Executable()
	if err != nil {
		return ""
	}

	// Check if we're in an app bundle (path should contain .app/Contents/MacOS/)
	if strings.Contains(executable, ".app/Contents/MacOS/") {
		bundlePath := executable
		// Find the .app directory
		for !strings.HasSuffix(bundlePath, ".app") && bundlePath != "/" {
			bundlePath = filepath.Dir(bundlePath)
		}
		if strings.HasSuffix(bundlePath, ".app") {
			return filepath.Join(bundlePath, "Contents", "Resources")
		}
	}
	return ""
}

// ensureBundleScriptsDir creates the scripts directory in the bundle if it doesn't exist
func ensureBundleScriptsDir(bundleDir string) error {
	scriptsDir := filepath.Join(bundleDir, "scripts")
	return os.MkdirAll(scriptsDir, 0755)
}

// syncScriptToBundle copies a specific script from external directory to bundle if it's newer
func syncScriptToBundle(externalDir, bundleDir, scriptName string) error {
	scriptsDir := filepath.Join(bundleDir, "scripts")
	scriptFileName := scriptName + ".applescript"

	srcPath := filepath.Join(externalDir, scriptFileName)
	dstPath := filepath.Join(scriptsDir, scriptFileName)

	// Check if source script exists
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return nil // Nothing to sync
	}

	needsUpdate, err := scriptNeedsUpdate(srcPath, dstPath)
	if err != nil {
		return fmt.Errorf("failed to check if script %s needs update: %v", scriptFileName, err)
	}

	if needsUpdate {
		if err := copyScript(srcPath, dstPath); err != nil {
			return fmt.Errorf("failed to copy script %s to bundle: %v", scriptFileName, err)
		}
		fmt.Printf("Updated embedded script: %s\n", scriptFileName)
	}

	return nil
}

// syncScriptsToBundle copies scripts from external directory to bundle when they're newer
func syncScriptsToBundle(externalDir, bundleDir string) error {
	scriptsDir := filepath.Join(bundleDir, "scripts")

	// Read scripts from external directory
	entries, err := os.ReadDir(externalDir)
	if err != nil {
		// External directory doesn't exist, nothing to sync
		return nil
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".applescript") {
			continue
		}

		srcPath := filepath.Join(externalDir, entry.Name())
		dstPath := filepath.Join(scriptsDir, entry.Name())

		needsUpdate, err := scriptNeedsUpdate(srcPath, dstPath)
		if err != nil {
			log.Printf("Warning: Failed to check if script %s needs update: %v", entry.Name(), err)
			continue
		}

		if needsUpdate {
			if err := copyScript(srcPath, dstPath); err != nil {
				log.Printf("Warning: Failed to copy script %s to bundle: %v", entry.Name(), err)
				continue
			}
			fmt.Printf("Updated embedded script: %s\n", entry.Name())
		}
	}

	return nil
}

// scriptNeedsUpdate checks if a script needs to be updated in the bundle
func scriptNeedsUpdate(srcPath, dstPath string) (bool, error) {
	// If destination doesn't exist, needs update
	dstInfo, err := os.Stat(dstPath)
	if os.IsNotExist(err) {
		return true, nil
	}
	if err != nil {
		return false, err
	}

	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return false, err
	}

	// Check modification time
	if srcInfo.ModTime().After(dstInfo.ModTime()) {
		return true, nil
	}

	// Check content hash to be sure
	srcHash, err := getFileHash(srcPath)
	if err != nil {
		return false, err
	}

	dstHash, err := getFileHash(dstPath)
	if err != nil {
		return false, err
	}

	return srcHash != dstHash, nil
}

// copyScript copies a script file from source to destination
func copyScript(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return err
	}

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		return err
	}

	// Copy file permissions
	srcInfo, err := src.Stat()
	if err != nil {
		return err
	}

	return dst.Chmod(srcInfo.Mode())
}

// getFileHash calculates SHA256 hash of a file
func getFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// findScript looks for a script in bundle first, then external directory
func findScript(externalDir, name string) string {
	scriptName := name + ".applescript"

	// First, try bundle directory if we're in a bundle
	bundleDir := getBundleResourcesDir()
	if bundleDir != "" {
		bundlePath := filepath.Join(bundleDir, "scripts", scriptName)
		if _, err := os.Stat(bundlePath); err == nil {
			return bundlePath
		}
	}

	// Fall back to external directory
	externalPath := filepath.Join(externalDir, scriptName)
	if _, err := os.Stat(externalPath); err == nil {
		return externalPath
	}

	return ""
}
