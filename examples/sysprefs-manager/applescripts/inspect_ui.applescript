tell application "System Events"
	tell process "System Settings"
		set frontmost to true
		delay 0.5
		
		set uiTree to ""
		
		try
			set w to window 1
			set uiTree to uiTree & "Window 1: " & (name of w) & "\n"
			
			-- Dump groups
			repeat with g in UI elements of w
				set uiTree to uiTree & "  " & (class of g) & " " & (name of g) & "\n"
				-- One level deeper
				repeat with c in UI elements of g
					set uiTree to uiTree & "    " & (class of c) & " " & (name of c) & "\n"
				end repeat
			end repeat
		on error e
			return "Error inspecting UI: " & e
		end try
		
		return uiTree
	end tell
end tell
