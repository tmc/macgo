set paneURL to "{{PANE_URL}}"

tell application "System Settings" to activate
delay 0.5

-- Use 'open location' which is robust
tell application "System Events"
	open location paneURL
end tell

-- Wait for window
delay 1
return "Opened " & paneURL
