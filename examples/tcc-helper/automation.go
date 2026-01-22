package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// GrantPermissionWithUI uses AppleScript UI automation to grant a permission
// This requires the tcc-helper itself to have Accessibility permission
//
// Parameters:
//   - service: TCC service name (e.g., "screen-recording")
//   - appNameOrPath: Either an app name (e.g., "screen-capture") or full path (e.g., "/Applications/MyApp.app")
func GrantPermissionWithUI(service, appNameOrPath string) error {
	fmt.Println("Attempting UI automation to grant permission...")
	fmt.Println("Note: This requires tcc-helper to have Accessibility permission")
	fmt.Println()

	// First, open System Settings to the correct pane
	svc, ok := tccServices[service]
	if !ok {
		return fmt.Errorf("unknown service: %s", service)
	}

	// Open System Settings
	cmd := exec.Command("open", svc.Pane)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open System Settings: %w", err)
	}

	fmt.Printf("Opened System Settings to: %s\n", svc.Name)
	fmt.Println("Waiting for window to appear...")
	time.Sleep(3 * time.Second)

	// Determine if we have a path or just a name
	appPath := appNameOrPath
	if !strings.HasPrefix(appNameOrPath, "/") {
		// It's a name, try to find it in common locations
		appPath = findAppPath(appNameOrPath)
		if appPath == "" {
			return fmt.Errorf("could not find app: %s. Try providing the full path with -path flag", appNameOrPath)
		}
		fmt.Printf("Found app at: %s\n", appPath)
	}

	// AppleScript to automate the full grant process
	script := fmt.Sprintf(`
tell application "System Events"
	tell process "System Settings"
		-- Wait for window
		set windowFound to false
		repeat 10 times
			if exists window 1 then
				set windowFound to true
				exit repeat
			end if
			delay 0.5
		end repeat

		if not windowFound then
			return "ERROR: System Settings window not found"
		end if

		set debugInfo to ""

		-- Try to unlock if locked
		try
			set lockButton to first button of window 1 whose description contains "lock"
			if exists lockButton then
				if description of lockButton contains "locked" then
					set debugInfo to debugInfo & "Clicking lock button (requires authentication)..." & return
					click lockButton
					delay 2
					set debugInfo to debugInfo & "NOTE: You may need to authenticate" & return
				end if
			end if
		on error errMsg
			set debugInfo to debugInfo & "Note: Could not find/click lock button: " & errMsg & return
		end try

		-- Try to find and click the '+' button
		try
			set addButton to first button of window 1 whose description is "add"
			if exists addButton then
				set debugInfo to debugInfo & "Clicking '+' button..." & return
				click addButton
				delay 2
			else
				return "ERROR: Could not find '+' button" & return & debugInfo
			end if
		on error errMsg
			return "ERROR: Failed to click '+' button: " & errMsg & return & debugInfo
		end try

		-- Now handle the file picker dialog
		set debugInfo to debugInfo & "Waiting for file picker dialog..." & return
		delay 1

		-- Try to navigate to the app using Cmd+Shift+G (Go to folder)
		try
			-- Focus on the file dialog
			set fileDialog to first window whose name contains "Choose"
			if exists fileDialog then
				set debugInfo to debugInfo & "Found file picker dialog" & return

				-- Press Cmd+Shift+G to open "Go to folder" sheet
				keystroke "g" using {command down, shift down}
				delay 1

				-- Type the full path
				set debugInfo to debugInfo & "Navigating to: %s" & return
				keystroke "%s"
				delay 0.5

				-- Press Return to navigate
				keystroke return
				delay 1

				-- Press Return again to select and open
				keystroke return
				delay 1

				set debugInfo to debugInfo & "SUCCESS: Granted permission to %s" & return
				return "SUCCESS: Permission granted" & return & return & "Debug info:" & return & debugInfo
			else
				return "ERROR: Could not find file picker dialog" & return & debugInfo
			end if
		on error errMsg
			return "ERROR: Failed to navigate file picker: " & errMsg & return & return & "Debug info:" & return & debugInfo
		end try
	end tell
end tell
`, appPath, appPath, appPath)

	cmd = exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()

	fmt.Println()
	fmt.Println("=== UI Automation Result ===")
	fmt.Println(string(output))

	if err != nil {
		return fmt.Errorf("UI automation failed: %w\nOutput: %s", err, string(output))
	}

	// Check if the output indicates success
	if strings.Contains(string(output), "SUCCESS") {
		fmt.Println()
		fmt.Println("✓ Permission grant automation completed")
		fmt.Println("Please verify the permission was granted in System Settings")
		return nil
	}

	return nil
}

// findAppPath attempts to locate an app by name in common locations
func findAppPath(appName string) string {
	// Add .app extension if not present
	if !strings.HasSuffix(appName, ".app") {
		appName = appName + ".app"
	}

	// Common locations to search
	searchPaths := []string{
		"/Applications/" + appName,
		"/System/Applications/" + appName,
		"/Applications/Utilities/" + appName,
		"/System/Applications/Utilities/" + appName,
		// User's Applications folder
		// Note: We could expand ~ but for now we'll keep it simple
	}

	// Also try the current working directory and its build output
	if cwd, err := exec.Command("pwd").Output(); err == nil {
		cwdStr := strings.TrimSpace(string(cwd))
		searchPaths = append(searchPaths, cwdStr+"/"+appName)
		searchPaths = append(searchPaths, cwdStr+"/build/"+appName)
	}

	for _, path := range searchPaths {
		if _, err := exec.Command("test", "-d", path).Output(); err == nil {
			return path
		}
	}

	return ""
}

// RevokePermissionWithUI uses AppleScript UI automation to revoke a permission
// This requires the tcc-helper itself to have Accessibility permission
func RevokePermissionWithUI(service, appName string) error {
	fmt.Println("Attempting UI automation to revoke permission...")
	fmt.Println("Note: This requires tcc-helper to have Accessibility permission")
	fmt.Println()

	// First, open System Settings to the correct pane
	svc, ok := tccServices[service]
	if !ok {
		return fmt.Errorf("unknown service: %s", service)
	}

	// Open System Settings
	cmd := exec.Command("open", svc.Pane)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open System Settings: %w", err)
	}

	// Wait for System Settings to open
	time.Sleep(2 * time.Second)

	// AppleScript to automate removing the app
	// This uses robust UI path detection that tries multiple approaches
	script := fmt.Sprintf(`
tell application "System Events"
	tell process "System Settings"
		-- Wait for window
		repeat 10 times
			if exists window 1 then exit repeat
			delay 0.5
		end repeat

		-- Try to unlock if locked
		try
			set lockButton to first button of window 1 whose description contains "lock"
			if exists lockButton then
				if description of lockButton contains "locked" then
					click lockButton
					delay 2
					-- TODO: Handle authentication dialog
				end if
			end if
		end try

		-- Try to find the app in the list and select it
		try
			-- System Settings UI can vary, try multiple approaches
			set foundApp to false
			set appTable to missing value
			set debugInfo to ""

			-- Strategy: Try multiple UI paths from most to least specific
			-- Path 1: table of scroll area 1 of group 1
			if appTable is missing value then
				try
					set appTable to first table of scroll area 1 of group 1 of window 1
					set debugInfo to debugInfo & "Found table at: scroll area 1 of group 1" & return
				on error errMsg
					set debugInfo to debugInfo & "Path 1 failed (scroll area 1 of group 1): " & errMsg & return
				end try
			end if

			-- Path 2: table of scroll area 1 of group 2
			if appTable is missing value then
				try
					set appTable to first table of scroll area 1 of group 2 of window 1
					set debugInfo to debugInfo & "Found table at: scroll area 1 of group 2" & return
				on error errMsg
					set debugInfo to debugInfo & "Path 2 failed (scroll area 1 of group 2): " & errMsg & return
				end try
			end if

			-- Path 3: table of scroll area 1 directly in window
			if appTable is missing value then
				try
					set appTable to first table of scroll area 1 of window 1
					set debugInfo to debugInfo & "Found table at: scroll area 1 of window 1" & return
				on error errMsg
					set debugInfo to debugInfo & "Path 3 failed (scroll area 1 of window 1): " & errMsg & return
				end try
			end if

			-- Path 4: Try any table in any scroll area in any group
			if appTable is missing value then
				try
					set allGroups to every group of window 1
					repeat with grp in allGroups
						try
							set scrollAreas to every scroll area of grp
							repeat with sa in scrollAreas
								try
									set appTable to first table of sa
									set debugInfo to debugInfo & "Found table in a group's scroll area" & return
									exit repeat
								end try
							end repeat
							if appTable is not missing value then exit repeat
						end try
					end repeat
				on error errMsg
					set debugInfo to debugInfo & "Path 4 failed (search all groups): " & errMsg & return
				end try
			end if

			if appTable is missing value then
				return "ERROR: Could not find app list table" & return & return & "Debug info:" & return & debugInfo
			end if

			set appRows to rows of appTable
			set debugInfo to debugInfo & "Found " & (count of appRows) & " rows in table" & return

			repeat with appRow in appRows
				try
					-- Try to get the text from the row - try multiple methods
					set rowText to ""

					-- Method 1: static text value
					try
						set rowText to value of static text 1 of appRow
					end try

					-- Method 2: name property
					if rowText = "" then
						try
							set rowText to name of appRow
						end try
					end if

					-- Method 3: title property
					if rowText = "" then
						try
							set rowText to title of appRow
						end try
					end if

					-- Method 4: description property
					if rowText = "" then
						try
							set rowText to description of appRow
						end try
					end if

					if rowText contains "%s" then
						select appRow
						set foundApp to true
						set debugInfo to debugInfo & "Found app: " & rowText & return
						delay 0.5

						-- Click the '-' button - try multiple approaches
						set removedSuccessfully to false

						-- Try 1: button with description "remove"
						try
							set removeButton to first button of window 1 whose description is "remove"
							click removeButton
							delay 0.5
							set removedSuccessfully to true
						end try

						-- Try 2: button with name "-"
						if not removedSuccessfully then
							try
								set removeButton to first button of window 1 whose name is "-"
								click removeButton
								delay 0.5
								set removedSuccessfully to true
							end try
						end if

						-- Try 3: button with role description containing "remove"
						if not removedSuccessfully then
							try
								set removeButton to first button of window 1 whose role description contains "remove"
								click removeButton
								delay 0.5
								set removedSuccessfully to true
							end try
						end if

						if removedSuccessfully then
							return "SUCCESS: Removed %s" & return & return & "Debug info:" & return & debugInfo
						else
							return "ERROR: Found app but could not find remove button" & return & return & "Debug info:" & return & debugInfo
						end if
					end if
				end try
			end repeat

			if foundApp is false then
				set debugInfo to debugInfo & "Searched all rows but did not find '%s'" & return
				return "ERROR: Could not find %s in the list" & return & return & "Debug info:" & return & debugInfo
			end if
		on error errMsg
			return "ERROR: " & errMsg & return & return & "Debug info:" & return & debugInfo
		end try
	end tell
end tell
`, appName, appName, appName, appName)

	cmd = exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()

	fmt.Println("UI Automation result:")
	fmt.Println(string(output))

	if err != nil {
		return fmt.Errorf("UI automation failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// CheckAccessibilityPermission checks if osascript has the necessary permissions
// for UI automation to work. This requires Accessibility permission to access UI elements.
func CheckAccessibilityPermission() (bool, error) {
	// Test actual UI element access which requires Accessibility permission (-25211)
	// Just getting process name can succeed with only Automation permission, giving false positive
	// We need to test accessing UI elements like the real automation does
	script := `tell application "System Events" to tell process "System Settings" to get windows`

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()

	outputStr := strings.TrimSpace(string(output))

	// Check for permission errors
	if err != nil {
		// Check for Accessibility permission error (-25211) - this is what we expect if permission is missing
		if strings.Contains(outputStr, "not allowed assistive access") ||
			strings.Contains(outputStr, "-25211") {
			return false, fmt.Errorf("osascript is not allowed assistive access (-25211)\n\nThe Accessibility permission is required for UI automation. To fix:\n1. Open System Settings > Privacy & Security > Accessibility\n2. Ensure tcc-helper or your terminal is in the list and checked\n3. You may need to remove and re-add the app")
		}

		// Check for Automation permission error (-1743)
		if strings.Contains(outputStr, "Not authorized to send Apple events") ||
			strings.Contains(outputStr, "-1743") {
			return false, fmt.Errorf("Not authorized to send Apple events to System Events (-1743)\n\nThe Automation permission is required. To fix:\n1. Open System Settings > Privacy & Security > Automation\n2. Find iTerm (or your terminal) in the list\n3. Check the box next to 'System Events'")
		}

		return false, fmt.Errorf("permission check failed: %w\nOutput: %s", err, outputStr)
	}

	// If we got here with no error, we successfully accessed UI elements - permissions are good!
	return true, nil
}
