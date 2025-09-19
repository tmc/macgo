// Package macgo provides simple macOS app bundle creation and TCC permission management.
//
// macgo enables Go applications to request macOS system permissions (camera, microphone,
// files, etc.) by automatically creating app bundles with proper entitlements and handling
// the relaunch process when necessary.
//
// This is a simplified API designed following Go's design principles:
// - Simple API surface with sensible defaults
// - Explicit configuration over magic behavior
// - No global state or init() side effects
// - Focus on the common case (95% of users)
//
// Basic usage:
//
//	err := macgo.Request(macgo.Camera, macgo.Microphone)
//	if err != nil {
//		log.Fatal(err)
//	}
//
// Advanced configuration:
//
//	cfg := macgo.NewConfig().
//		WithAppName("MyApp").
//		WithPermissions(macgo.Camera, macgo.Files).
//		WithDebug()
//	err := macgo.Start(cfg)
package macgo

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/tmc/misc/macgo/internal/tcc"
	"github.com/tmc/misc/macgo/internal/system"
)

// Permission represents a macOS system permission that can be requested.
// These correspond to TCC (Transparency, Consent, Control) permission types.
type Permission string

// Core permissions covering 95% of use cases.
const (
	Camera     Permission = "camera"     // Camera access (com.apple.security.device.camera)
	Microphone Permission = "microphone" // Microphone access (com.apple.security.device.audio-input)
	Location   Permission = "location"   // Location services (com.apple.security.personal-information.location)
	Files      Permission = "files"      // File system access with user selection
	Network    Permission = "network"    // Network client/server access
	Sandbox    Permission = "sandbox"    // App sandbox with restricted file access
)

// NewConfig creates a new Config with sensible defaults.
// The zero value is valid, so this is equivalent to &Config{}.
func NewConfig() *Config {
	return &Config{}
}

// Config holds macgo configuration options.
// The zero value is valid and uses sensible defaults.
// Use NewConfig() and builder methods for fluent configuration.
type Config struct {
	// AppName is the application name. Defaults to executable name.
	AppName string

	// BundleID is the bundle identifier. Defaults to module-based ID (e.g., com.github.user.repo.appname).
	BundleID string

	// Version is the application version. Defaults to "1.0.0".
	Version string

	// Permissions are the requested macOS permissions.
	Permissions []Permission

	// Custom allows specifying custom entitlements not covered by Permission constants.
	Custom []string

	// AppGroups specifies app group identifiers for sharing data between apps.
	// Requires sandbox permission and com.apple.security.application-groups entitlement.
	AppGroups []string

	// Debug enables debug logging.
	Debug bool

	// KeepBundle prevents cleanup of temporary bundles. Use a pointer to distinguish
	// between explicitly set false and default (which is true).
	KeepBundle *bool

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
// This provides explicit configuration without magic init() functions.
//
// Supported environment variables:
//   MACGO_APP_NAME          - Application name
//   MACGO_BUNDLE_ID         - Bundle identifier
//   MACGO_DEBUG=1           - Enable debug logging
//   MACGO_KEEP_BUNDLE=1     - Preserve bundle after execution
//   MACGO_CODE_SIGN_IDENTITY - Code signing identity
//   MACGO_AUTO_SIGN=1       - Enable automatic code signing
//   MACGO_AD_HOC_SIGN=1     - Enable ad-hoc code signing
//   MACGO_CAMERA=1          - Request camera permission
//   MACGO_MICROPHONE=1      - Request microphone permission
//   MACGO_LOCATION=1        - Request location permission
//   MACGO_FILES=1           - Request file access permission
//   MACGO_NETWORK=1         - Request network permission
//   MACGO_SANDBOX=1         - Enable app sandbox
//   MACGO_FORCE_LAUNCH_SERVICES=1 - Force use of LaunchServices
//   MACGO_FORCE_DIRECT=1    - Force direct execution
func (c *Config) FromEnv() *Config {
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
		keepBundle := true
		c.KeepBundle = &keepBundle
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
	if os.Getenv("MACGO_SANDBOX") == "1" {
		c.Permissions = append(c.Permissions, Sandbox)
	}

	// Parse launch preferences from environment
	if os.Getenv("MACGO_FORCE_LAUNCH_SERVICES") == "1" {
		c.ForceLaunchServices = true
	}
	if os.Getenv("MACGO_FORCE_DIRECT") == "1" {
		c.ForceDirectExecution = true
	}

	return c
}

// WithAppName sets the application name for the bundle.
// If empty, defaults to the executable name.
func (c *Config) WithAppName(name string) *Config {
	c.AppName = name
	return c
}

// WithPermissions adds macOS system permissions to request.
// Permissions are additive - multiple calls append to the list.
func (c *Config) WithPermissions(perms ...Permission) *Config {
	c.Permissions = append(c.Permissions, perms...)
	return c
}

// WithCustom adds custom entitlements not covered by Permission constants.
// Use full entitlement identifiers (e.g. "com.apple.security.device.capture").
func (c *Config) WithCustom(entitlements ...string) *Config {
	c.Custom = append(c.Custom, entitlements...)
	return c
}

// WithAppGroups adds app group identifiers for sharing data between sandboxed apps.
// Requires Sandbox permission. Use reverse-DNS format (e.g. "group.com.example.shared").
func (c *Config) WithAppGroups(groups ...string) *Config {
	c.AppGroups = append(c.AppGroups, groups...)
	return c
}

// WithDebug enables debug logging.
func (c *Config) WithDebug() *Config {
	c.Debug = true
	return c
}

// WithCodeSigning enables code signing with the specified identity.
// Use "Developer ID Application" for automatic identity selection.
func (c *Config) WithCodeSigning(identity string) *Config {
	c.CodeSignIdentity = identity
	return c
}

// WithAutoSign enables automatic detection and use of Developer ID certificates.
// macgo will search for and use an available Developer ID Application certificate.
func (c *Config) WithAutoSign() *Config {
	c.AutoSign = true
	return c
}

// WithAdHocSign enables ad-hoc code signing using the "-" identity.
// Ad-hoc signing provides basic code integrity without requiring certificates.
// Useful for development and testing.
func (c *Config) WithAdHocSign() *Config {
	c.AdHocSign = true
	return c
}

// shouldKeepBundle returns the effective KeepBundle value.
// Defaults to true to preserve bundles for inspection and reuse.
func (c *Config) shouldKeepBundle() bool {
	if c.KeepBundle == nil {
		return true // Default to keeping bundles
	}
	return *c.KeepBundle
}

// Validate checks the configuration for common issues and dependency requirements.
// Returns an error if the configuration is invalid.
func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}

	// Validate permissions and their dependencies
	var tccPerms []tcc.Permission
	for _, perm := range c.Permissions {
		tccPerms = append(tccPerms, tcc.Permission(perm))
	}
	if err := tcc.ValidatePermissions(tccPerms); err != nil {
		return fmt.Errorf("invalid permissions: %w", err)
	}

	// Validate app groups dependencies
	if err := tcc.ValidateAppGroups(c.AppGroups, tccPerms); err != nil {
		return fmt.Errorf("invalid app groups: %w", err)
	}

	// Validate bundle ID format if specified
	if c.BundleID != "" {
		if err := system.ValidateBundleID(c.BundleID); err != nil {
			return fmt.Errorf("invalid bundle ID: %w", err)
		}
	}

	// Validate app name constraints
	if c.AppName != "" {
		if err := system.ValidateAppName(c.AppName); err != nil {
			return fmt.Errorf("invalid app name: %w", err)
		}
	}

	return nil
}


// Start initializes macgo with the given configuration.
// Creates an app bundle if needed and handles permission requests.
// On non-macOS platforms, this is a no-op that returns nil.
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
// The context can be used to cancel the bundle creation and launch process.
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

// Request is a convenience function for requesting specific permissions.
// Creates a minimal config and starts macgo immediately.
// Equivalent to Start(&Config{Permissions: perms}).
func Request(perms ...Permission) error {
	return Start(&Config{Permissions: perms})
}

// Auto loads configuration from environment variables and starts macgo.
// Useful for external configuration without code changes.
// Equivalent to Start(NewConfig().FromEnv()).
func Auto() error {
	return Start(new(Config).FromEnv())
}

// OpenSystemPreferences attempts to open macOS Privacy & Security settings.
// Tries to open Full Disk Access directly, falls back to general Privacy pane.
// Useful when your app needs Full Disk Access or other manual permission grants.
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

// copyToClipboard copies text to the system clipboard using pbcopy.
func copyToClipboard(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// LaunchAppBundle launches an app bundle using the open command.
// This ensures proper registration with TCC for permission dialogs.
func LaunchAppBundle(bundlePath string) error {
	if !strings.HasSuffix(bundlePath, ".app") {
		return fmt.Errorf("not an app bundle: %s", bundlePath)
	}

	cmd := exec.Command("open", bundlePath, "--args")
	return cmd.Run()
}

// ShowFullDiskAccessInstructions provides instructions for granting Full Disk Access.
// Optionally opens System Preferences if openSettings is true.
// The programPath parameter is provided for future use in displaying specific instructions.
func ShowFullDiskAccessInstructions(programPath string, openSettings bool) {
	if openSettings {
		// Open System Settings
		OpenSystemPreferences()
	}

	// The programPath parameter is available for future enhancements
	// to provide program-specific instructions
	_ = programPath // Acknowledge the parameter is intentionally unused for now
}
