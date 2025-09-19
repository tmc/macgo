// Package main provides a command-line tool for programmatic control of macOS system preferences.
// This tool demonstrates various methods to interact with system settings including:
// - Using the defaults command for direct preference manipulation
// - AppleScript automation for UI control
// - URL schemes for opening specific preference panes
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	macgo "github.com/tmc/misc/macgo"
)

var (
	open     = flag.String("open", "", "open a specific preference pane (e.g., privacy-automation, network, sound)")
	get      = flag.String("get", "", "get a preference value (domain key format: com.apple.finder ShowStatusBar)")
	set      = flag.String("set", "", "set a preference value (domain key value format: com.apple.finder ShowStatusBar true)")
	list     = flag.String("list", "", "list preferences for a domain (e.g., com.apple.finder)")
	detect   = flag.Bool("detect", false, "detect preference changes (run before and after making changes in UI)")
	grant    = flag.String("grant", "", "grant automation permission for an app (e.g., 'Brave Browser')")
	panes    = flag.Bool("panes", false, "list all available preference panes")
	defaults = flag.Bool("defaults", false, "show common defaults commands")
	help     = flag.Bool("help", false, "show help")
	verbose  = flag.Bool("v", false, "verbose output")
)

// PreferencePane represents a system preference pane
type PreferencePane struct {
	Name        string
	ID          string
	URLScheme   string
	Description string
}

var commonPanes = []PreferencePane{
	{"General", "com.apple.systempreferences.GeneralSettings", "x-apple.systempreferences:com.apple.systempreferences.GeneralSettings", "General system settings"},
	{"Accessibility", "com.apple.preference.universalaccess", "x-apple.systempreferences:com.apple.preference.universalaccess", "Accessibility options"},
	{"Privacy & Security", "com.apple.preference.security", "x-apple.systempreferences:com.apple.preference.security", "Privacy and security settings"},
	{"Privacy - Automation", "com.apple.preference.security", "x-apple.systempreferences:com.apple.preference.security?Privacy_Automation", "Automation permissions"},
	{"Privacy - Camera", "com.apple.preference.security", "x-apple.systempreferences:com.apple.preference.security?Privacy_Camera", "Camera access"},
	{"Privacy - Microphone", "com.apple.preference.security", "x-apple.systempreferences:com.apple.preference.security?Privacy_Microphone", "Microphone access"},
	{"Privacy - Screen Recording", "com.apple.preference.security", "x-apple.systempreferences:com.apple.preference.security?Privacy_ScreenCapture", "Screen recording permissions"},
	{"Privacy - Files and Folders", "com.apple.preference.security", "x-apple.systempreferences:com.apple.preference.security?Privacy_FilesAndFolders", "File system access"},
	{"Network", "com.apple.preference.network", "x-apple.systempreferences:com.apple.preference.network", "Network settings"},
	{"Sound", "com.apple.preference.sound", "x-apple.systempreferences:com.apple.preference.sound", "Sound settings"},
	{"Displays", "com.apple.preference.displays", "x-apple.systempreferences:com.apple.preference.displays", "Display settings"},
	{"Keyboard", "com.apple.preference.keyboard", "x-apple.systempreferences:com.apple.preference.keyboard", "Keyboard settings"},
	{"Mouse", "com.apple.preference.mouse", "x-apple.systempreferences:com.apple.preference.mouse", "Mouse settings"},
	{"Trackpad", "com.apple.preference.trackpad", "x-apple.systempreferences:com.apple.preference.trackpad", "Trackpad settings"},
	{"Dock", "com.apple.preference.dock", "x-apple.systempreferences:com.apple.preference.dock", "Dock settings"},
	{"Mission Control", "com.apple.preference.expose", "x-apple.systempreferences:com.apple.preference.expose", "Mission Control settings"},
}

var commonDefaults = map[string]string{
	"Show hidden files in Finder":           "defaults write com.apple.finder AppleShowAllFiles -bool true",
	"Show path bar in Finder":               "defaults write com.apple.finder ShowPathbar -bool true",
	"Show status bar in Finder":             "defaults write com.apple.finder ShowStatusBar -bool true",
	"Use list view in Finder by default":    "defaults write com.apple.finder FXPreferredViewStyle -string 'Nlsv'",
	"Disable natural scrolling":             "defaults write NSGlobalDomain com.apple.swipescrolldirection -bool false",
	"Enable key repeat":                     "defaults write NSGlobalDomain ApplePressAndHoldEnabled -bool false",
	"Set fast key repeat":                   "defaults write NSGlobalDomain KeyRepeat -int 2",
	"Set short delay until key repeat":      "defaults write NSGlobalDomain InitialKeyRepeat -int 15",
	"Show all file extensions":              "defaults write NSGlobalDomain AppleShowAllExtensions -bool true",
	"Disable auto-correct":                  "defaults write NSGlobalDomain NSAutomaticSpellingCorrectionEnabled -bool false",
	"Enable tap to click":                   "defaults write com.apple.driver.AppleBluetoothMultitouch.trackpad Clicking -bool true",
	"Show battery percentage":               "defaults write com.apple.menuextra.battery ShowPercent -bool true",
	"Disable Dashboard":                     "defaults write com.apple.dashboard mcx-disabled -bool true",
	"Don't show Dashboard as a Space":       "defaults write com.apple.dock dashboard-in-overlay -bool true",
	"Automatically hide and show the Dock":  "defaults write com.apple.dock autohide -bool true",
	"Make Dock icons of hidden apps translucent": "defaults write com.apple.dock showhidden -bool true",
}

func main() {
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	// Request necessary permissions
	cfg := &macgo.Config{
		AppName:     "sys-prefs",
		Permissions: []macgo.Permission{},
		Custom: []string{
			"com.apple.security.automation.apple-events",
		},
		Debug: os.Getenv("MACGO_DEBUG") == "1",
	}

	err := macgo.Start(cfg)
	if err != nil {
		log.Fatalf("Failed to request permissions: %v", err)
	}

	// Handle commands
	switch {
	case *panes:
		listPanes()
	case *defaults:
		showCommonDefaults()
	case *open != "":
		openPane(*open)
	case *get != "":
		getValue(*get)
	case *set != "":
		setValue(*set)
	case *list != "":
		listDomain(*list)
	case *detect:
		detectChanges()
	case *grant != "":
		grantAutomation(*grant)
	default:
		showHelp()
	}
}

func showHelp() {
	fmt.Println("sys-prefs - Programmatic control of macOS system preferences")
	fmt.Println("===========================================================")
	fmt.Println()
	fmt.Println("Usage:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  sys-prefs -open privacy-automation     # Open Privacy & Security > Automation")
	fmt.Println("  sys-prefs -get 'com.apple.finder ShowStatusBar'  # Get Finder status bar setting")
	fmt.Println("  sys-prefs -set 'com.apple.finder ShowStatusBar true'  # Enable Finder status bar")
	fmt.Println("  sys-prefs -list com.apple.finder       # List all Finder preferences")
	fmt.Println("  sys-prefs -detect                      # Detect preference changes")
	fmt.Println("  sys-prefs -grant 'Brave Browser'       # Grant automation permission")
	fmt.Println("  sys-prefs -panes                       # List all preference panes")
	fmt.Println("  sys-prefs -defaults                    # Show common defaults commands")
	fmt.Println()
	fmt.Println("Common Preference Panes:")
	fmt.Println("  privacy-automation    Privacy & Security > Automation")
	fmt.Println("  privacy-camera       Privacy & Security > Camera")
	fmt.Println("  privacy-microphone   Privacy & Security > Microphone")
	fmt.Println("  privacy-screen       Privacy & Security > Screen Recording")
	fmt.Println("  privacy-files        Privacy & Security > Files and Folders")
	fmt.Println("  network             Network settings")
	fmt.Println("  sound               Sound settings")
	fmt.Println("  displays            Display settings")
	fmt.Println("  keyboard            Keyboard settings")
}

func listPanes() {
	fmt.Println("Available System Preference Panes:")
	fmt.Println("==================================")
	for _, pane := range commonPanes {
		fmt.Printf("\n%s\n", pane.Name)
		fmt.Printf("  ID: %s\n", pane.ID)
		fmt.Printf("  URL: %s\n", pane.URLScheme)
		fmt.Printf("  Description: %s\n", pane.Description)
	}
}

func showCommonDefaults() {
	fmt.Println("Common defaults Commands:")
	fmt.Println("========================")
	for desc, cmd := range commonDefaults {
		fmt.Printf("\n%s:\n  %s\n", desc, cmd)
	}
	fmt.Println("\nNote: After changing defaults, you may need to restart affected applications.")
	fmt.Println("For Finder: killall Finder")
	fmt.Println("For Dock: killall Dock")
}

func openPane(pane string) {
	var url string

	// Map common names to URLs
	switch strings.ToLower(pane) {
	case "privacy", "security", "privacy-security":
		url = "x-apple.systempreferences:com.apple.preference.security"
	case "privacy-automation", "automation":
		url = "x-apple.systempreferences:com.apple.preference.security?Privacy_Automation"
	case "privacy-camera", "camera":
		url = "x-apple.systempreferences:com.apple.preference.security?Privacy_Camera"
	case "privacy-microphone", "microphone":
		url = "x-apple.systempreferences:com.apple.preference.security?Privacy_Microphone"
	case "privacy-screen", "screen-recording":
		url = "x-apple.systempreferences:com.apple.preference.security?Privacy_ScreenCapture"
	case "privacy-files", "files":
		url = "x-apple.systempreferences:com.apple.preference.security?Privacy_FilesAndFolders"
	case "network":
		url = "x-apple.systempreferences:com.apple.preference.network"
	case "sound":
		url = "x-apple.systempreferences:com.apple.preference.sound"
	case "displays", "display":
		url = "x-apple.systempreferences:com.apple.preference.displays"
	case "keyboard":
		url = "x-apple.systempreferences:com.apple.preference.keyboard"
	case "mouse":
		url = "x-apple.systempreferences:com.apple.preference.mouse"
	case "trackpad":
		url = "x-apple.systempreferences:com.apple.preference.trackpad"
	case "dock":
		url = "x-apple.systempreferences:com.apple.preference.dock"
	default:
		// Try to use it as a direct pane ID
		url = fmt.Sprintf("x-apple.systempreferences:%s", pane)
	}

	if *verbose {
		fmt.Printf("Opening preference pane: %s\n", url)
	}

	cmd := exec.Command("open", url)
	err := cmd.Run()
	if err != nil {
		log.Fatalf("Failed to open preference pane: %v", err)
	}
	fmt.Printf("Opened preference pane: %s\n", pane)
}

func getValue(input string) {
	parts := strings.Fields(input)
	if len(parts) < 2 {
		log.Fatalf("Invalid format. Use: domain key (e.g., com.apple.finder ShowStatusBar)")
	}

	domain := parts[0]
	key := parts[1]

	cmd := exec.Command("defaults", "read", domain, key)
	output, err := cmd.Output()
	if err != nil {
		log.Fatalf("Failed to read preference: %v", err)
	}

	fmt.Printf("%s %s = %s", domain, key, string(output))
}

func setValue(input string) {
	parts := strings.Fields(input)
	if len(parts) < 3 {
		log.Fatalf("Invalid format. Use: domain key value (e.g., com.apple.finder ShowStatusBar true)")
	}

	domain := parts[0]
	key := parts[1]
	value := strings.Join(parts[2:], " ")

	// Determine value type and format appropriately
	var args []string
	switch strings.ToLower(value) {
	case "true", "false":
		args = []string{"write", domain, key, "-bool", value}
	case "yes", "no":
		v := "true"
		if strings.ToLower(value) == "no" {
			v = "false"
		}
		args = []string{"write", domain, key, "-bool", v}
	default:
		// Try to detect if it's a number
		if _, err := fmt.Sscanf(value, "%d", new(int)); err == nil {
			args = []string{"write", domain, key, "-int", value}
		} else if _, err := fmt.Sscanf(value, "%f", new(float64)); err == nil {
			args = []string{"write", domain, key, "-float", value}
		} else {
			args = []string{"write", domain, key, "-string", value}
		}
	}

	if *verbose {
		fmt.Printf("Running: defaults %s\n", strings.Join(args, " "))
	}

	cmd := exec.Command("defaults", args...)
	err := cmd.Run()
	if err != nil {
		log.Fatalf("Failed to set preference: %v", err)
	}

	fmt.Printf("Set %s %s to %s\n", domain, key, value)

	// Suggest restart if needed
	if strings.Contains(domain, "finder") {
		fmt.Println("Note: Run 'killall Finder' to apply Finder changes")
	} else if strings.Contains(domain, "dock") {
		fmt.Println("Note: Run 'killall Dock' to apply Dock changes")
	}
}

func listDomain(domain string) {
	cmd := exec.Command("defaults", "read", domain)
	output, err := cmd.Output()
	if err != nil {
		log.Fatalf("Failed to read domain: %v", err)
	}

	fmt.Printf("Preferences for %s:\n", domain)
	fmt.Println(string(output))
}

func detectChanges() {
	stateFile := "/tmp/sys-prefs-defaults.txt"

	// Check if state file exists
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		// First run - capture initial state
		fmt.Println("Capturing initial preference state...")
		cmd := exec.Command("defaults", "read")
		output, err := cmd.Output()
		if err != nil {
			log.Fatalf("Failed to read defaults: %v", err)
		}

		err = os.WriteFile(stateFile, output, 0644)
		if err != nil {
			log.Fatalf("Failed to write state file: %v", err)
		}

		fmt.Println("Initial state captured.")
		fmt.Println("Now make your changes in System Settings, then run this command again to see what changed.")
	} else {
		// Second run - compare with previous state
		fmt.Println("Detecting changes...")

		// Read old state
		oldState, err := os.ReadFile(stateFile)
		if err != nil {
			log.Fatalf("Failed to read state file: %v", err)
		}

		// Get current state
		cmd := exec.Command("defaults", "read")
		newState, err := cmd.Output()
		if err != nil {
			log.Fatalf("Failed to read defaults: %v", err)
		}

		// Save new state for next comparison
		err = os.WriteFile(stateFile, newState, 0644)
		if err != nil {
			log.Fatalf("Failed to update state file: %v", err)
		}

		// Compare states
		diff := compareStates(string(oldState), string(newState))
		if len(diff) == 0 {
			fmt.Println("No changes detected.")
		} else {
			fmt.Println("Changes detected:")
			fmt.Println("================")
			for _, change := range diff {
				fmt.Println(change)
			}
		}
	}
}

func compareStates(old, new string) []string {
	oldLines := make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(old))
	for scanner.Scan() {
		oldLines[scanner.Text()] = true
	}

	var changes []string
	scanner = bufio.NewScanner(strings.NewReader(new))
	for scanner.Scan() {
		line := scanner.Text()
		if !oldLines[line] && strings.TrimSpace(line) != "" {
			changes = append(changes, line)
		}
	}

	return changes
}

func grantAutomation(appName string) {
	fmt.Printf("Attempting to grant automation permission for '%s'...\n", appName)
	fmt.Println("Note: This requires manual approval in System Settings")

	// Open the automation preference pane
	openPane("privacy-automation")

	// Try to use AppleScript to help locate the app
	script := fmt.Sprintf(`
tell application "System Events"
	tell application process "System Settings"
		-- Wait for the window to load
		delay 1

		-- Try to find and select the app in the list
		try
			tell window 1
				-- This is approximate and may not work on all macOS versions
				display dialog "Please find '%s' in the list and check the box to grant permission."
			end tell
		end try
	end tell
end tell
`, appName)

	cmd := exec.Command("osascript", "-e", script)
	err := cmd.Run()
	if err != nil {
		// Fallback instructions
		fmt.Printf("\nPlease manually:\n")
		fmt.Printf("1. Find your terminal application in the left list\n")
		fmt.Printf("2. Check the box next to '%s' to grant permission\n", appName)
		fmt.Printf("3. You may need to restart the terminal for changes to take effect\n")
	}
}

// Helper function to execute AppleScript
func runAppleScript(script string) error {
	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil && *verbose {
		fmt.Printf("AppleScript error: %s\n", string(output))
	}
	return err
}

// Helper function to parse JSON output from defaults
func parseDefaultsJSON(data []byte) (map[string]interface{}, error) {
	var result map[string]interface{}

	// Try to parse as JSON first
	err := json.Unmarshal(data, &result)
	if err == nil {
		return result, nil
	}

	// If not JSON, try to parse the plist format
	// This is a simplified parser - in production you'd want to use a proper plist library
	result = make(map[string]interface{})
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var currentKey string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasSuffix(line, "= (") || strings.HasSuffix(line, "= {") {
			parts := strings.Split(line, " = ")
			if len(parts) >= 1 {
				currentKey = strings.Trim(parts[0], "\" ")
				result[currentKey] = []string{}
			}
		} else if strings.Contains(line, " = ") {
			parts := strings.Split(line, " = ")
			if len(parts) == 2 {
				key := strings.Trim(parts[0], "\" ")
				value := strings.Trim(parts[1], "\";")
				result[key] = value
			}
		}
	}

	return result, nil
}