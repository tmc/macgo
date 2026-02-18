package bundle

import "testing"

func TestExtractProfileEntitlements(t *testing.T) {
	tests := []struct {
		name    string
		xml     string
		wantApp string
		wantTeam string
	}{
		{
			name: "both entitlements present",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Entitlements</key>
	<dict>
		<key>com.apple.application-identifier</key>
		<string>ABC123DEF4.com.example.myapp</string>
		<key>com.apple.developer.team-identifier</key>
		<string>ABC123DEF4</string>
		<key>get-task-allow</key>
		<true/>
	</dict>
	<key>ExpirationDate</key>
	<date>2025-12-31T00:00:00Z</date>
</dict>
</plist>`,
			wantApp:  "ABC123DEF4.com.example.myapp",
			wantTeam: "ABC123DEF4",
		},
		{
			name: "missing specific keys",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
	<key>Entitlements</key>
	<dict>
		<key>get-task-allow</key>
		<true/>
		<key>keychain-access-groups</key>
		<array>
			<string>ABC123DEF4.*</string>
		</array>
	</dict>
</dict>
</plist>`,
			wantApp:  "",
			wantTeam: "",
		},
		{
			name: "no entitlements dict",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
	<key>TeamName</key>
	<string>My Team</string>
	<key>ExpirationDate</key>
	<date>2025-12-31T00:00:00Z</date>
</dict>
</plist>`,
			wantApp:  "",
			wantTeam: "",
		},
		{
			name: "nested dicts with booleans skipped",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
	<key>Entitlements</key>
	<dict>
		<key>com.apple.application-identifier</key>
		<string>XYZ789.com.test.nested</string>
		<key>com.apple.security.app-sandbox</key>
		<true/>
		<key>com.apple.security.network.client</key>
		<true/>
		<key>com.apple.developer.team-identifier</key>
		<string>XYZ789</string>
		<key>com.apple.developer.icloud-container-identifiers</key>
		<dict>
			<key>com.apple.application-identifier</key>
			<string>SHOULD_NOT_MATCH</string>
		</dict>
	</dict>
</dict>
</plist>`,
			wantApp:  "XYZ789.com.test.nested",
			wantTeam: "XYZ789",
		},
		{
			name:     "empty input",
			xml:      "",
			wantApp:  "",
			wantTeam: "",
		},
		{
			name: "only app identifier",
			xml: `<plist version="1.0">
<dict>
	<key>Entitlements</key>
	<dict>
		<key>com.apple.application-identifier</key>
		<string>TEAM.com.example.app</string>
	</dict>
</dict>
</plist>`,
			wantApp:  "TEAM.com.example.app",
			wantTeam: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractProfileEntitlements([]byte(tt.xml))
			if got.ApplicationIdentifier != tt.wantApp {
				t.Errorf("ApplicationIdentifier = %q, want %q", got.ApplicationIdentifier, tt.wantApp)
			}
			if got.TeamIdentifier != tt.wantTeam {
				t.Errorf("TeamIdentifier = %q, want %q", got.TeamIdentifier, tt.wantTeam)
			}
		})
	}
}
