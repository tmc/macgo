set appName to "{{APP_NAME}}"

tell application "System Events"
	tell process "System Settings"
		set frontmost to true
		
		-- Defensive: Try multiple hierarchies commonly found in macOS 13/14
		set foundRow to missing value
		
		-- Try standard privacy list location (simplified search)
		try
			set targetList to group 1 of scroll area 1 of group 1 of group 2 of split group 1 of group 1 of window 1
		on error
			-- Fallback search strategy: try invalidation safe query
			try
				set targetList to list 1 of scroll area 1 of group 1 of group 2 of split group 1 of group 1 of window 1
			on error
				set targetList to outline 1 of scroll area 1 of group 1 of group 2 of split group 1 of group 1 of window 1
			end try
		end try
		
		-- Iterate rows
		repeat with r in rows of targetList
			try
				-- Check if row contains the app name text
				-- Try getting value of text field
				if (value of static text 1 of r) contains appName then
					set foundRow to r
					exit repeat
				end if
			on error
				try
					-- Try name of row itself
					if (name of r) contains appName then
						set foundRow to r
						exit repeat
					end if
				end try
			end try
		end repeat
		
		if foundRow is missing value then
			return "NOT_FOUND"
		end if
		
		-- Found row, get switch/checkbox value
		try
			set theSwitch to value indicator 1 of foundRow
		on error
			set theSwitch to checkbox 1 of foundRow
		end try
		
		set currentValue to value of theSwitch as boolean
		if currentValue then
			return "ENABLED"
		else
			return "DISABLED"
		end if
		
	end tell
end tell
