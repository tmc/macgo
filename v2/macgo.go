// Package macgo provides simple macOS app bundle creation for TCC permissions.
//
// This is a simplified API designed following Russ Cox's principles:
// - Simple API surface
// - Explicit configuration
// - No global state
// - Focus on the common case
package macgo

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Permission represents a macOS permission type.
type Permission string

// Core permissions that 95% of users need.
const (
	Camera     Permission = "camera"
	Microphone Permission = "microphone"
	Location   Permission = "location"
	Files      Permission = "files"
	Network    Permission = "network"
)

// Config holds macgo configuration.
// Zero value is valid and uses sensible defaults.
type Config struct {
	// AppName is the application name. Defaults to executable name.
	AppName string

	// BundleID is the bundle identifier. Defaults to com.macgo.{appname}.
	BundleID string

	// Version is the application version. Defaults to "1.0.0".
	Version string

	// Permissions are the requested macOS permissions.
	Permissions []Permission

	// Custom allows specifying custom entitlements not covered by Permission constants.
	Custom []string

	// Debug enables debug logging.
	Debug bool

	// KeepBundle prevents cleanup of temporary bundles.
	KeepBundle bool

	// CodeSignIdentity is the signing identity to use for code signing.
	// If empty and AutoSign is false, the app bundle will not be signed.
	// Use "Developer ID Application" for automatic identity selection.
	CodeSignIdentity string

	// AutoSign enables automatic detection of Developer ID certificates.
	// If true and CodeSignIdentity is empty, macgo will try to find and use
	// a Developer ID Application certificate automatically.
	AutoSign bool

	// AdHocSign enables ad-hoc code signing using the "-" identity.
	// Ad-hoc signing provides basic code signing without requiring certificates.
	// This is useful for development and testing.
	AdHocSign bool

	// CodeSigningIdentifier is the identifier to use for code signing.
	// If empty, defaults to the bundle identifier.
	CodeSigningIdentifier string

	// ForceDirectExecution forces direct execution instead of LaunchServices.
	// This preserves terminal I/O (stdin/stdout/stderr) but may not trigger
	// proper TCC dialogs. Use this for CLI commands that need terminal output.
	ForceDirectExecution bool

	// ForceLaunchServices forces use of LaunchServices (open command).
	// This ensures proper TCC dialogs but breaks terminal I/O.
	// Use this for commands that need GUI interaction or browser automation.
	ForceLaunchServices bool
}

// FromEnv loads configuration from environment variables.
// This is explicit - no magic init() functions.
func (c *Config) FromEnv() *Config {
	if c == nil {
		c = &Config{}
	}

	if name := os.Getenv("MACGO_APP_NAME"); name != "" {
		c.AppName = name
	}

	if id := os.Getenv("MACGO_BUNDLE_ID"); id != "" {
		c.BundleID = id
	}

	if os.Getenv("MACGO_DEBUG") == "1" {
		c.Debug = true
	}

	if os.Getenv("MACGO_KEEP_BUNDLE") == "1" {
		c.KeepBundle = true
	}

	if identity := os.Getenv("MACGO_CODE_SIGN_IDENTITY"); identity != "" {
		c.CodeSignIdentity = identity
	}

	if os.Getenv("MACGO_AUTO_SIGN") == "1" {
		c.AutoSign = true
	}

	if os.Getenv("MACGO_AD_HOC_SIGN") == "1" {
		c.AdHocSign = true
	}

	// Parse permissions from environment
	if os.Getenv("MACGO_CAMERA") == "1" {
		c.Permissions = append(c.Permissions, Camera)
	}
	if os.Getenv("MACGO_MICROPHONE") == "1" {
		c.Permissions = append(c.Permissions, Microphone)
	}
	if os.Getenv("MACGO_LOCATION") == "1" {
		c.Permissions = append(c.Permissions, Location)
	}
	if os.Getenv("MACGO_FILES") == "1" {
		c.Permissions = append(c.Permissions, Files)
	}
	if os.Getenv("MACGO_NETWORK") == "1" {
		c.Permissions = append(c.Permissions, Network)
	}

	return c
}

// WithPermissions adds permissions to the config.
func (c *Config) WithPermissions(perms ...Permission) *Config {
	if c == nil {
		c = &Config{}
	}
	c.Permissions = append(c.Permissions, perms...)
	return c
}

// WithCustom adds custom entitlements to the config.
func (c *Config) WithCustom(entitlements ...string) *Config {
	if c == nil {
		c = &Config{}
	}
	c.Custom = append(c.Custom, entitlements...)
	return c
}

// WithDebug enables debug logging.
func (c *Config) WithDebug() *Config {
	if c == nil {
		c = &Config{}
	}
	c.Debug = true
	return c
}

// WithCodeSigning enables code signing with the specified identity.
func (c *Config) WithCodeSigning(identity string) *Config {
	if c == nil {
		c = &Config{}
	}
	c.CodeSignIdentity = identity
	return c
}

// WithAutoSign enables automatic detection and use of Developer ID certificates.
func (c *Config) WithAutoSign() *Config {
	if c == nil {
		c = &Config{}
	}
	c.AutoSign = true
	return c
}

// WithAdHocSign enables ad-hoc code signing.
func (c *Config) WithAdHocSign() *Config {
	if c == nil {
		c = &Config{}
	}
	c.AdHocSign = true
	return c
}

// Start initializes macgo with the given configuration.
// On non-macOS platforms, this is a no-op.
func Start(cfg *Config) error {
	if runtime.GOOS != "darwin" {
		if cfg != nil && cfg.Debug {
			fmt.Fprintf(os.Stderr, "macgo: skipping on %s\n", runtime.GOOS)
		}
		return nil
	}

	if cfg == nil {
		cfg = &Config{}
	}

	return startDarwin(context.Background(), cfg)
}

// StartContext is like Start but accepts a context for cancellation.
func StartContext(ctx context.Context, cfg *Config) error {
	if runtime.GOOS != "darwin" {
		if cfg != nil && cfg.Debug {
			fmt.Fprintf(os.Stderr, "macgo: skipping on %s\n", runtime.GOOS)
		}
		return nil
	}

	if cfg == nil {
		cfg = &Config{}
	}

	return startDarwin(ctx, cfg)
}

// Request is a convenience function that creates a config with the given permissions and starts macgo.
// Equivalent to Start(&Config{Permissions: perms}).
func Request(perms ...Permission) error {
	return Start(&Config{Permissions: perms})
}

// Auto loads configuration from environment and starts macgo.
// Equivalent to Start(new(Config).FromEnv()).
func Auto() error {
	return Start(new(Config).FromEnv())
}

// OpenSystemPreferences attempts to open the Privacy & Security settings.
// This is useful when your app needs Full Disk Access or other system permissions.
func OpenSystemPreferences() error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("system preferences only available on macOS")
	}

	// Try opening the Full Disk Access pane directly
	cmd := exec.Command("open", "x-apple.systempreferences:com.apple.preference.security?Privacy_AllFiles")
	if err := cmd.Run(); err != nil {
		// Fallback to general Privacy & Security
		cmd = exec.Command("open", "x-apple.systempreferences:com.apple.preference.security")
		return cmd.Run()
	}
	return nil
}

// copyToClipboard attempts to copy text to the system clipboard
func copyToClipboard(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// LaunchAppBundle uses the open command to launch an app bundle, which properly registers it with TCC
func LaunchAppBundle(bundlePath string) error {
	if !strings.HasSuffix(bundlePath, ".app") {
		return fmt.Errorf("not an app bundle: %s", bundlePath)
	}

	cmd := exec.Command("open", bundlePath, "--args")
	return cmd.Run()
}

// ShowFullDiskAccessInstructions provides simple instructions for granting Full Disk Access.
func ShowFullDiskAccessInstructions(programPath string, openSettings bool) {
	if openSettings {
		// Open System Settings
		OpenSystemPreferences()
	}
}