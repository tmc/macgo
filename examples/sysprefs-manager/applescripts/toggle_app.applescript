set appName to "{{APP_NAME}}"
set targetState to {{TARGET_STATE}} -- boolean literal true/false

tell application "System Events"
	tell process "System Settings"
		set frontmost to true
		
		-- Defensive: Try multiple hierarchies commonly found in macOS 13/14
		set foundRow to missing value
		
		-- Try standard privacy list location (simplified search)
		try
			set targetList to group 1 of scroll area 1 of group 1 of group 2 of split group 1 of group 1 of window 1
			set allUI to entire contents of targetList
		on error
			-- Fallback search strategy: find the first list/outline
			set targetList to UI element 1 of scroll area 1 of group 1 of group 2 of split group 1 of group 1 of window 1
		end try
		
		-- Iterate rows
		repeat with r in rows of targetList
			try
				-- Check if row contains the app name text
				if (name of UI element 1 of r) contains appName then
					set foundRow to r
					exit repeat
				end if
			end try
		end repeat
		
		if foundRow is missing value then
			return "App '" & appName & "' not found in list"
		end if
		
		-- Found row, look for the toggle/switch
		-- In macOS 13+, it's often a switch (checkbox)
		try
			set theSwitch to value indicator 1 of foundRow
		on error
			set theSwitch to checkbox 1 of foundRow
		end try
		
		set currentValue to value of theSwitch as boolean
		set desiredValue to targetState
		
		if currentValue is not desiredValue then
			click theSwitch
			delay 0.5
			
			-- Handle potential auth prompt (TCC modification often requires password)
			-- This script cannot enter password securely, so we just acknowledge we clicked.
			return "Clicked toggle for " & appName & " (Password might be required)"
		else
			return "App '" & appName & "' already in desired state (" & targetState & ")"
		end if
		
	end tell
end tell
