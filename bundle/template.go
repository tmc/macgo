package bundle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tmc/misc/macgo"
)

// generateInfoPlist creates the Info.plist file for the app bundle.
func (c *Creator) generateInfoPlist(bundlePath string, cfg *macgo.Config, execPath string) error {
	appName := c.getApplicationName(cfg, execPath)
	bundleID := c.getBundleID(cfg, appName)

	plistPath := filepath.Join(bundlePath, "Contents", "Info.plist")

	content := c.buildInfoPlistContent(appName, bundleID, cfg)

	if err := os.WriteFile(plistPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write Info.plist: %w", err)
	}

	return nil
}

// generateEntitlementsPlist creates the entitlements.plist file for the app bundle.
func (c *Creator) generateEntitlementsPlist(bundlePath string, cfg *macgo.Config) error {
	if len(cfg.Entitlements) == 0 {
		return nil // No entitlements to write
	}

	plistPath := filepath.Join(bundlePath, "Contents", "entitlements.plist")

	content := c.buildEntitlementsPlistContent(cfg.Entitlements)

	if err := os.WriteFile(plistPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write entitlements.plist: %w", err)
	}

	return nil
}

// getBundleID returns the bundle ID from config or generates a default one.
func (c *Creator) getBundleID(cfg *macgo.Config, appName string) string {
	if cfg.BundleID != "" {
		return cfg.BundleID
	}
	return fmt.Sprintf("com.macgo.%s", strings.ToLower(appName))
}

// buildInfoPlistContent constructs the Info.plist XML content.
func (c *Creator) buildInfoPlistContent(appName, bundleID string, cfg *macgo.Config) string {
	var builder strings.Builder

	builder.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleExecutable</key>
	<string>` + appName + `</string>
	<key>CFBundleIdentifier</key>
	<string>` + bundleID + `</string>
	<key>CFBundleName</key>
	<string>` + appName + `</string>
	<key>CFBundleDisplayName</key>
	<string>` + appName + `</string>
	<key>CFBundleVersion</key>
	<string>1.0</string>
	<key>CFBundleShortVersionString</key>
	<string>1.0</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleSignature</key>
	<string>????</string>
	<key>CFBundleInfoDictionaryVersion</key>
	<string>6.0</string>
`)

	// Add LSUIElement if set (controls dock icon visibility)
	if lsuiElement, exists := cfg.PlistEntries["LSUIElement"]; exists {
		if hide, ok := lsuiElement.(bool); ok {
			builder.WriteString("\t<key>LSUIElement</key>\n")
			if hide {
				builder.WriteString("\t<true/>\n")
			} else {
				builder.WriteString("\t<false/>\n")
			}
		}
	}

	// Add custom plist entries
	for key, value := range cfg.PlistEntries {
		if key == "LSUIElement" {
			continue // Already handled above
		}
		c.addPlistEntry(&builder, key, value)
	}

	builder.WriteString("</dict>\n</plist>\n")

	return builder.String()
}

// buildEntitlementsPlistContent constructs the entitlements.plist XML content.
func (c *Creator) buildEntitlementsPlistContent(entitlements macgo.Entitlements) string {
	var builder strings.Builder

	builder.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
`)

	for entitlement, value := range entitlements {
		builder.WriteString("\t<key>" + string(entitlement) + "</key>\n")
		if value {
			builder.WriteString("\t<true/>\n")
		} else {
			builder.WriteString("\t<false/>\n")
		}
	}

	builder.WriteString("</dict>\n</plist>\n")

	return builder.String()
}

// addPlistEntry adds a single plist entry to the builder.
func (c *Creator) addPlistEntry(builder *strings.Builder, key string, value any) {
	builder.WriteString("\t<key>" + escapeXML(key) + "</key>\n")

	switch v := value.(type) {
	case string:
		builder.WriteString("\t<string>" + escapeXML(v) + "</string>\n")
	case bool:
		if v {
			builder.WriteString("\t<true/>\n")
		} else {
			builder.WriteString("\t<false/>\n")
		}
	case int:
		builder.WriteString(fmt.Sprintf("\t<integer>%d</integer>\n", v))
	case float64:
		builder.WriteString(fmt.Sprintf("\t<real>%f</real>\n", v))
	default:
		// Fall back to string representation with XML escaping
		builder.WriteString("\t<string>" + escapeXML(fmt.Sprintf("%v", v)) + "</string>\n")
	}
}

// escapeXML escapes XML special characters to prevent XML injection vulnerabilities.
// This function is extracted from the original bundle.go.
func escapeXML(s string) string {
	if s == "" {
		return s
	}

	// Use strings.Replacer for efficient multiple replacements
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&apos;",
	)

	return replacer.Replace(s)
}

// writePlistGeneric writes a generic map to a plist file.
// This function is extracted from the original bundle.go writePlist function.
func writePlistGeneric[K ~string](path string, data map[K]any) error {
	var sb strings.Builder

	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
`)

	for k, v := range data {
		// Escape the key to prevent XML injection
		sb.WriteString(fmt.Sprintf("\t<key>%s</key>\n", escapeXML(string(k))))

		switch val := v.(type) {
		case bool:
			if val {
				sb.WriteString("\t<true/>\n")
			} else {
				sb.WriteString("\t<false/>\n")
			}
		case string:
			// Escape the string value to prevent XML injection
			sb.WriteString(fmt.Sprintf("\t<string>%s</string>\n", escapeXML(val)))
		case int, int32, int64:
			sb.WriteString(fmt.Sprintf("\t<integer>%v</integer>\n", val))
		case float32, float64:
			sb.WriteString(fmt.Sprintf("\t<real>%v</real>\n", val))
		default:
			// Escape the stringified value to prevent XML injection
			sb.WriteString(fmt.Sprintf("\t<string>%s</string>\n", escapeXML(fmt.Sprintf("%v", val))))
		}
	}

	sb.WriteString("</dict>\n</plist>")
	return os.WriteFile(path, []byte(sb.String()), 0644)
}
