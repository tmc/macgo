package plist

import (
	"fmt"
	"os"
	"strings"
)

// InfoPlistConfig holds configuration for generating Info.plist files.
type InfoPlistConfig struct {
	AppName    string
	BundleID   string
	ExecName   string
	Version    string
	CustomKeys map[string]interface{}
}

// WriteInfoPlist creates a minimal Info.plist file at the specified path.
// It generates a standard macOS app bundle Info.plist with required keys.
func WriteInfoPlist(path string, cfg InfoPlistConfig) error {
	if err := validateInfoPlistConfig(cfg); err != nil {
		return fmt.Errorf("invalid info plist config: %w", err)
	}

	content := generateInfoPlistContent(cfg)
	return os.WriteFile(path, []byte(content), 0644)
}

// validateInfoPlistConfig validates the configuration for Info.plist generation.
func validateInfoPlistConfig(cfg InfoPlistConfig) error {
	if cfg.AppName == "" {
		return fmt.Errorf("app name is required")
	}
	if cfg.BundleID == "" {
		return fmt.Errorf("bundle ID is required")
	}
	if cfg.ExecName == "" {
		return fmt.Errorf("executable name is required")
	}
	if cfg.Version == "" {
		return fmt.Errorf("version is required")
	}
	return nil
}

// generateInfoPlistContent generates the XML content for an Info.plist file.
func generateInfoPlistContent(cfg InfoPlistConfig) string {
	var entries []string

	// Standard required entries
	entries = append(entries, xmlKeyValue("CFBundleDisplayName", cfg.AppName))
	entries = append(entries, xmlKeyValue("CFBundleExecutable", cfg.ExecName))
	entries = append(entries, xmlKeyValue("CFBundleIdentifier", cfg.BundleID))
	entries = append(entries, xmlKeyValue("CFBundleName", cfg.AppName))
	entries = append(entries, xmlKeyValue("CFBundlePackageType", "APPL"))
	entries = append(entries, xmlKeyValue("CFBundleVersion", cfg.Version))
	entries = append(entries, xmlKeyValue("CFBundleShortVersionString", cfg.Version))

	// Default behavior: no dock icon, high resolution capable
	entries = append(entries, xmlKeyBool("LSUIElement", true))
	entries = append(entries, xmlKeyBool("NSHighResolutionCapable", true))

	// Add custom keys if provided
	for key, value := range cfg.CustomKeys {
		switch v := value.(type) {
		case string:
			entries = append(entries, xmlKeyValue(key, v))
		case bool:
			entries = append(entries, xmlKeyBool(key, v))
		case []string:
			entries = append(entries, xmlKeyArray(key, v))
		default:
			// For unsupported types, convert to string
			entries = append(entries, xmlKeyValue(key, fmt.Sprintf("%v", v)))
		}
	}

	dictContent := strings.Join(entries, "\n")
	return wrapPlist(wrapDict(dictContent))
}

// GenerateDefaultBundleID creates a default bundle ID based on the app name.
// Format: com.macgo.{normalized-app-name}
func GenerateDefaultBundleID(appName string) string {
	// Normalize app name for bundle ID
	normalized := strings.ToLower(appName)
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.ReplaceAll(normalized, "-", "")
	normalized = strings.ReplaceAll(normalized, "_", "")

	// Remove any non-alphanumeric characters
	var result strings.Builder
	for _, r := range normalized {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result.WriteRune(r)
		}
	}

	if result.Len() == 0 {
		return "com.macgo.app"
	}

	return "com.macgo." + result.String()
}