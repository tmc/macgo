package plist

import (
	"fmt"
	"os"
	"strings"

	"github.com/tmc/macgo/bundle"
)

// InfoPlistConfig holds configuration for generating Info.plist files.
type InfoPlistConfig struct {
	AppName        string
	BundleID       string
	ExecName       string
	Version        string
	BackgroundOnly bool
	CustomKeys     map[string]interface{}
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

	// Helper to check if a key is overridden by CustomKeys
	isOverridden := func(key string) bool {
		_, ok := cfg.CustomKeys[key]
		return ok
	}

	// Standard required entries
	if !isOverridden("CFBundleDisplayName") {
		entries = append(entries, xmlKeyValue("CFBundleDisplayName", cfg.AppName))
	}
	if !isOverridden("CFBundleExecutable") {
		entries = append(entries, xmlKeyValue("CFBundleExecutable", cfg.ExecName))
	}
	if !isOverridden("CFBundleIdentifier") {
		entries = append(entries, xmlKeyValue("CFBundleIdentifier", cfg.BundleID))
	}
	if !isOverridden("CFBundleName") {
		entries = append(entries, xmlKeyValue("CFBundleName", cfg.AppName))
	}
	if !isOverridden("CFBundlePackageType") {
		entries = append(entries, xmlKeyValue("CFBundlePackageType", "APPL"))
	}
	if !isOverridden("CFBundleVersion") {
		entries = append(entries, xmlKeyValue("CFBundleVersion", cfg.Version))
	}
	if !isOverridden("CFBundleShortVersionString") {
		entries = append(entries, xmlKeyValue("CFBundleShortVersionString", cfg.Version))
	}

	// Default behavior: no dock icon (unless disabled for FDA registration), high resolution capable
	// LSUIElement=true makes app background (no dock icon) but may prevent FDA panel registration
	showInDock := os.Getenv("MACGO_SHOW_IN_DOCK") == "1"

	// Only apply default background/UI element logic if neither is overridden
	if !isOverridden("LSBackgroundOnly") && !isOverridden("LSUIElement") {
		if cfg.BackgroundOnly {
			entries = append(entries, xmlKeyBool("LSBackgroundOnly", true))
		} else {
			entries = append(entries, xmlKeyBool("LSUIElement", !showInDock))
		}
	}

	if !isOverridden("NSHighResolutionCapable") {
		entries = append(entries, xmlKeyBool("NSHighResolutionCapable", true))
	}

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
