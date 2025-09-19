package plist

import (
	"fmt"
	"os"
	"strings"

	"github.com/tmc/misc/macgo/helpers/bundle"
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
// Uses the same inference logic as the helpers/bundle package.
func GenerateDefaultBundleID(appName string) string {
	// Use the bundle inference logic from helpers/bundle
	// Import is done at the top of the file
	return bundle.InferBundleID(appName)
}
