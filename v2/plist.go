package macgo

import (
	"fmt"
	"os"
	"strings"
)

// writeInfoPlist creates a minimal Info.plist file.
func writeInfoPlist(path, appName, bundleID, execName, version string) error {
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleDisplayName</key>
	<string>%s</string>
	<key>CFBundleExecutable</key>
	<string>%s</string>
	<key>CFBundleIdentifier</key>
	<string>%s</string>
	<key>CFBundleName</key>
	<string>%s</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleVersion</key>
	<string>%s</string>
	<key>CFBundleShortVersionString</key>
	<string>%s</string>
	<key>LSUIElement</key>
	<true/>
	<key>NSHighResolutionCapable</key>
	<true/>
</dict>
</plist>`, escapeXML(appName), escapeXML(execName), escapeXML(bundleID), escapeXML(appName), escapeXML(version), escapeXML(version))

	return os.WriteFile(path, []byte(plist), 0644)
}

// writeEntitlements creates an entitlements.plist file.
func writeEntitlements(path string, cfg *Config) error {
	var entries []string

	// Add standard permissions
	for _, perm := range cfg.Permissions {
		switch perm {
		case Camera:
			entries = append(entries, `	<key>com.apple.security.device.camera</key>
	<true/>`)
		case Microphone:
			entries = append(entries, `	<key>com.apple.security.device.microphone</key>
	<true/>`)
		case Location:
			entries = append(entries, `	<key>com.apple.security.personal-information.location</key>
	<true/>`)
		case Sandbox:
			entries = append(entries, `	<key>com.apple.security.app-sandbox</key>
	<true/>`)
		case Files:
			entries = append(entries, `	<key>com.apple.security.files.user-selected.read-only</key>
	<true/>`)
		case Network:
			entries = append(entries, `	<key>com.apple.security.network.client</key>
	<true/>`)
		}
	}

	// Add custom entitlements
	for _, custom := range cfg.Custom {
		entries = append(entries, fmt.Sprintf(`	<key>%s</key>
	<true/>`, escapeXML(custom)))
	}

	if len(entries) == 0 {
		return nil
	}

	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
%s
</dict>
</plist>`, strings.Join(entries, "\n"))

	return os.WriteFile(path, []byte(plist), 0644)
}

// escapeXML escapes special characters for XML content.
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}
