package main

import (
	"fmt"
	"os/exec"
	"time"
)

// InspectSystemSettingsUI creates a diagnostic report of the System Settings UI structure
// This helps debug UI path issues and understand how the UI varies across macOS versions
func InspectSystemSettingsUI(service string) error {
	fmt.Println("Opening System Settings and inspecting UI structure...")
	fmt.Println("This requires Accessibility permission")
	fmt.Println()

	// Check accessibility permission first
	hasAccess, err := CheckAccessibilityPermission()
	if !hasAccess {
		fmt.Println("❌ tcc-helper does not have Accessibility permission")
		if err != nil {
			fmt.Printf("   Error: %v\n", err)
		}
		return fmt.Errorf("accessibility permission required for UI inspection")
	}

	fmt.Println("✓ Accessibility permission granted")
	fmt.Println()

	// Open System Settings
	svc, ok := tccServices[service]
	if !ok {
		return fmt.Errorf("unknown service: %s", service)
	}

	cmd := exec.Command("open", svc.Pane)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open System Settings: %w", err)
	}

	fmt.Printf("Opened System Settings to: %s\n", svc.Name)
	fmt.Println("Waiting for window to appear...")
	time.Sleep(3 * time.Second)

	// Run UI inspection script
	script := `
tell application "System Events"
	tell process "System Settings"
		-- Wait for window
		repeat 10 times
			if exists window 1 then exit repeat
			delay 0.5
		end repeat

		if not (exists window 1) then
			return "ERROR: No System Settings window found"
		end if

		set output to "=== System Settings UI Structure Inspection ===" & return & return

		-- Get window info
		try
			set windowName to name of window 1
			set output to output & "Window: " & windowName & return
		end try

		-- Count top-level UI elements
		try
			set groupCount to count of (every group of window 1)
			set output to output & "Groups in window: " & groupCount & return
		end try

		try
			set scrollAreaCount to count of (every scroll area of window 1)
			set output to output & "Scroll areas in window: " & scrollAreaCount & return
		end try

		set output to output & return & "=== Exploring Groups ===" & return

		-- Examine each group
		set allGroups to every group of window 1
		set groupNum to 1
		repeat with grp in allGroups
			try
				set output to output & return & "Group " & groupNum & ":" & return

				-- Count scroll areas in this group
				try
					set scrollAreas to every scroll area of grp
					set saCount to count of scrollAreas
					set output to output & "  Scroll areas: " & saCount & return

					if saCount > 0 then
						set saNum to 1
						repeat with sa in scrollAreas
							try
								-- Count tables in this scroll area
								set tables to every table of sa
								set tableCount to count of tables
								set output to output & "    Scroll area " & saNum & " has " & tableCount & " table(s)" & return

								if tableCount > 0 then
									-- Get info about first table
									set theTable to first table of sa
									try
										set rowCount to count of (every row of theTable)
										set output to output & "      Table has " & rowCount & " rows" & return
									end try
								end if
							on error errMsg
								set output to output & "    Error examining scroll area " & saNum & ": " & errMsg & return
							end try
							set saNum to saNum + 1
						end repeat
					end if
				on error errMsg
					set output to output & "  Error getting scroll areas: " & errMsg & return
				end try
			on error errMsg
				set output to output & "Error examining group " & groupNum & ": " & errMsg & return
			end try
			set groupNum to groupNum + 1
		end repeat

		set output to output & return & "=== Testing Common UI Paths ===" & return

		-- Test path 1: scroll area 1 of group 1
		try
			set testTable to first table of scroll area 1 of group 1 of window 1
			set rowCount to count of (every row of testTable)
			set output to output & "✓ Path 1 SUCCESS: scroll area 1 of group 1 (" & rowCount & " rows)" & return
		on error errMsg
			set output to output & "✗ Path 1 FAILED: scroll area 1 of group 1" & return
			set output to output & "  " & errMsg & return
		end try

		-- Test path 2: scroll area 1 of group 2
		try
			set testTable to first table of scroll area 1 of group 2 of window 1
			set rowCount to count of (every row of testTable)
			set output to output & "✓ Path 2 SUCCESS: scroll area 1 of group 2 (" & rowCount & " rows)" & return
		on error errMsg
			set output to output & "✗ Path 2 FAILED: scroll area 1 of group 2" & return
			set output to output & "  " & errMsg & return
		end try

		-- Test path 3: scroll area 1 of window
		try
			set testTable to first table of scroll area 1 of window 1
			set rowCount to count of (every row of testTable)
			set output to output & "✓ Path 3 SUCCESS: scroll area 1 of window 1 (" & rowCount & " rows)" & return
		on error errMsg
			set output to output & "✗ Path 3 FAILED: scroll area 1 of window 1" & return
			set output to output & "  " & errMsg & return
		end try

		set output to output & return & "=== Inspection Complete ===" & return

		return output
	end tell
end tell
`

	cmd = exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()

	fmt.Println()
	fmt.Println("=== INSPECTION RESULTS ===")
	fmt.Println(string(output))

	if err != nil {
		return fmt.Errorf("UI inspection failed: %w", err)
	}

	return nil
}
