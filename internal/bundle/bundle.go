// Package bundle provides macOS app bundle creation and management functionality.
// This package handles the creation, configuration, and signing of macOS app bundles
// for Go applications that need to access protected system resources.
package bundle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tmc/macgo/internal/plist"
	"github.com/tmc/macgo/internal/system"
)

// UIMode controls how the app appears in the macOS UI.
type UIMode string

const (
	// UIModeBackground sets LSBackgroundOnly=true. No UI at all.
	// Use for CLI tools, MCP servers, daemons. Prevents -1712 timeout.
	UIModeBackground UIMode = "background"

	// UIModeAccessory sets LSUIElement=true. Can show windows/menu bar but no Dock icon.
	// Use for menu bar apps, floating utilities.
	UIModeAccessory UIMode = "accessory"

	// UIModeRegular is a normal app with Dock icon and full UI.
	// Use for standard GUI applications.
	UIModeRegular UIMode = "regular"
)

// Bundle represents a macOS app bundle with its configuration and management methods.
type Bundle struct {
	// Path is the full path to the .app bundle directory
	Path string

	// Config contains the bundle configuration
	Config *Config

	// execPath is the original executable path
	execPath string

	// appName is the cleaned application name
	appName string

	// bundleID is the bundle identifier
	bundleID string

	// version is the application version
	version string
}

// Config holds configuration options for bundle creation and signing.
// This is a subset of the main macgo Config, containing only bundle-specific fields.
type Config struct {
	// AppName is the application name. Defaults to executable name.
	AppName string

	// BundleID is the bundle identifier. Defaults to inferred from module path or environment.
	BundleID string

	// Version is the application version. Defaults to "1.0.0".
	Version string

	// Permissions are the requested macOS permissions.
	Permissions []string

	// Custom allows specifying custom entitlements not covered by Permission constants.
	Custom []string

	// AppGroups specifies app group identifiers for sharing data between apps.
	AppGroups []string

	// Debug enables debug logging.
	Debug bool

	// KeepBundle prevents cleanup of temporary bundles.
	KeepBundle *bool

	// CodeSignIdentity is the signing identity to use for code signing.
	CodeSignIdentity string

	// AutoSign enables automatic detection of Developer ID certificates.
	AutoSign bool

	// AdHocSign enables ad-hoc code signing using the "-" identity.
	AdHocSign bool

	// CodeSigningIdentifier is the identifier to use for code signing.
	CodeSigningIdentifier string

	// Info allows specifying custom Info.plist keys.
	Info map[string]interface{}

	// UIMode controls how the app appears in the UI.
	// Default (empty or UIModeBackground): LSBackgroundOnly=true for CLI tools.
	UIMode UIMode
}

// shouldKeepBundle returns the effective KeepBundle value.
// Defaults to true to preserve bundles for inspection and reuse.
func (c *Config) shouldKeepBundle() bool {
	if c.KeepBundle == nil {
		return true // Default to keeping bundles
	}
	return *c.KeepBundle
}

// New creates a new Bundle instance for the given executable and configuration.
func New(execPath string, config *Config) (*Bundle, error) {
	if config == nil {
		config = &Config{}
	}

	// Determine app name
	appName := config.AppName
	if appName == "" {
		appName = filepath.Base(execPath)
		appName = strings.TrimSuffix(appName, filepath.Ext(appName))
	}

	// Clean and limit app name length
	appName = system.CleanAppName(appName)
	appName = system.LimitAppNameLength(appName, 251) // Reserve 4 chars for ".app"

	// Determine bundle ID
	bundleID := config.BundleID
	if bundleID == "" {
		bundleID = system.InferBundleID(appName)
	}

	// Determine version
	version := config.Version
	if version == "" {
		version = "1.0.0"
	}

	return &Bundle{
		Config:   config,
		execPath: execPath,
		appName:  appName,
		bundleID: bundleID,
		version:  version,
	}, nil
}

// Create creates the app bundle with the configured settings.
// This method implements the functionality from createSimpleBundle.
func (b *Bundle) Create() error {
	// Determine bundle location - prefer ~/go/bin/ if it exists
	bundleBaseDir := os.TempDir()
	if goPath := os.Getenv("GOPATH"); goPath != "" {
		bundleBaseDir = filepath.Join(goPath, "bin")
	} else if homeDir, err := os.UserHomeDir(); err == nil {
		goBinDir := filepath.Join(homeDir, "go", "bin")
		if _, err := os.Stat(goBinDir); err == nil {
			bundleBaseDir = goBinDir
		}
	}

	// Create bundle directory
	bundleDir := filepath.Join(bundleBaseDir, b.appName+".app")
	b.Path = bundleDir

	// Check if bundle already exists and should be kept
	if b.Config.shouldKeepBundle() {
		if _, err := os.Stat(bundleDir); err == nil {
			// Check if the original executable has changed by comparing SHA256
			if b.isBundleUpToDate() {
				if b.Config.Debug {
					fmt.Fprintf(os.Stderr, "macgo: reusing existing bundle at %s (binary unchanged)\n", bundleDir)
				}
				return nil
			} else {
				if b.Config.Debug {
					fmt.Fprintf(os.Stderr, "macgo: binary changed, recreating bundle at %s\n", bundleDir)
				}
				// Remove the outdated bundle
				if err := os.RemoveAll(bundleDir); err != nil && !os.IsNotExist(err) {
					if os.IsPermission(err) {
						fmt.Fprintf(os.Stderr, "macgo: warning: failed to remove outdated bundle at %s (permission denied), attempting to overwrite: %v\n", bundleDir, err)
					} else {
						return fmt.Errorf("failed to remove outdated bundle: %w", err)
					}
				}
			}
		}
	} else {
		// Remove old bundle if not keeping it
		if err := os.RemoveAll(bundleDir); err != nil && !os.IsNotExist(err) {
			if os.IsPermission(err) {
				fmt.Fprintf(os.Stderr, "macgo: warning: failed to remove old bundle at %s (permission denied), attempting to overwrite: %v\n", bundleDir, err)
			} else {
				return fmt.Errorf("failed to remove old bundle: %w", err)
			}
		}
	}

	// Create directory structure
	contentsDir := filepath.Join(bundleDir, "Contents")
	macosDir := filepath.Join(contentsDir, "MacOS")
	if err := os.MkdirAll(macosDir, 0755); err != nil {
		return fmt.Errorf("failed to create bundle directories: %w", err)
	}

	// Copy the executable directly
	execName := filepath.Base(b.appName)
	destExec := filepath.Join(macosDir, execName)
	if err := system.CopyFile(b.execPath, destExec); err != nil {
		return fmt.Errorf("failed to copy executable: %w", err)
	}
	if err := os.Chmod(destExec, 0755); err != nil {
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}

	// Create Info.plist path
	plistPath := filepath.Join(contentsDir, "Info.plist")

	// Prepare Info.plist config
	infoCfg := plist.InfoPlistConfig{
		AppName:    b.appName,
		BundleID:   b.bundleID,
		ExecName:   execName,
		Version:    b.version,
		CustomKeys: make(map[string]interface{}),
	}

	// Set UI mode based on config (default: background for CLI tools)
	switch b.Config.UIMode {
	case UIModeAccessory:
		// LSUIElement=true: menu bar apps, no Dock icon but can show UI
		infoCfg.CustomKeys["LSUIElement"] = true
	case UIModeRegular:
		// Normal app: appears in Dock, full UI
		// Don't set LSBackgroundOnly or LSUIElement
	default:
		// UIModeBackground or empty: LSBackgroundOnly=true for CLI tools
		// Prevents -1712 AppleEvent timeout for pure Go binaries
		infoCfg.BackgroundOnly = true
	}

	// Copy custom Info keys
	for k, v := range b.Config.Info {
		infoCfg.CustomKeys[k] = v
	}

	// Helper: Auto-inject Usage Descriptions for known TCC permissions if missing
	for _, perm := range b.Config.Permissions {
		if strings.Contains(strings.ToLower(perm), "accessibility") { // matches "accessibility"
			const key = "NSAccessibilityUsageDescription"
			if _, exists := infoCfg.CustomKeys[key]; !exists {
				infoCfg.CustomKeys[key] = "This application requires accessibility permissions to function properly."
				if b.Config.Debug {
					fmt.Fprintf(os.Stderr, "macgo: auto-injected %s\n", key)
				}
			}
		}
		// Add others (Camera, Mic) as needed in future
	}

	if err := plist.WriteInfoPlist(plistPath, infoCfg); err != nil {
		return fmt.Errorf("failed to write Info.plist: %w", err)
	}

	// Create entitlements if needed
	// We now include entitlements for ad-hoc signing to support Apple Silicon (get-task-allow).
	needsEntitlements := (len(b.Config.Permissions) > 0 || len(b.Config.Custom) > 0) ||
		(b.Config.CodeSignIdentity != "-" && !b.Config.AdHocSign) ||
		b.Config.AdHocSign || b.Config.CodeSignIdentity == "-"

	if needsEntitlements {
		entPath := filepath.Join(contentsDir, "entitlements.plist")

		// Convert string permissions to plist.Permission
		var plistPermissions []plist.Permission
		for _, p := range b.Config.Permissions {
			plistPermissions = append(plistPermissions, plist.Permission(p))
		}

		customEntitlements := b.Config.Custom
		// Inject get-task-allow for ad-hoc signing if not already present
		if b.Config.AdHocSign || b.Config.CodeSignIdentity == "-" {
			hasGetTaskAllow := false
			for _, c := range customEntitlements {
				if c == "com.apple.security.get-task-allow" {
					hasGetTaskAllow = true
					break
				}
			}
			if !hasGetTaskAllow {
				customEntitlements = append(customEntitlements, "com.apple.security.get-task-allow")
			}
		}

		entCfg := plist.EntitlementsConfig{
			Permissions: plistPermissions,
			Custom:      customEntitlements,
			AppGroups:   b.Config.AppGroups,
		}
		if err := plist.WriteEntitlements(entPath, entCfg); err != nil {
			return fmt.Errorf("failed to write entitlements: %w", err)
		}
	}

	// Recursively fix permissions if running under sudo
	if err := b.fixOwner(bundleDir); err != nil {
		if b.Config.Debug {
			fmt.Fprintf(os.Stderr, "macgo: warning: failed to fix bundle ownership: %v\n", err)
		}
	}

	return nil
}

// fixOwner changes ownership of the bundle to the SUDO_USER if running as root.
func (b *Bundle) fixOwner(path string) error {
	// Only proceed if running as root
	if os.Geteuid() != 0 {
		return nil
	}

	// Check for SUDO_UID and SUDO_GID
	sudoUID := os.Getenv("SUDO_UID")
	sudoGID := os.Getenv("SUDO_GID")

	if sudoUID == "" || sudoGID == "" {
		return nil
	}

	// Parse UID/GID
	uid := 0
	gid := 0
	if _, err := fmt.Sscanf(sudoUID, "%d", &uid); err != nil {
		return nil
	}
	if _, err := fmt.Sscanf(sudoGID, "%d", &gid); err != nil {
		return nil
	}

	// Walk and chmod
	return filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return os.Chown(p, uid, gid)
	})
}

// Sign performs code signing on the bundle.
// This method coordinates the signing process and delegates to signing.go.
func (b *Bundle) Sign() error {
	if b.Path == "" {
		return fmt.Errorf("bundle not created - call Create() first")
	}

	// Code sign the bundle if identity is provided, auto-detect, or ad-hoc
	if b.Config.CodeSignIdentity != "" {
		if err := codeSignBundle(b.Path, b.Config); err != nil {
			return fmt.Errorf("code signing failed: %w", err)
		}
		if b.Config.Debug {
			fmt.Fprintf(os.Stderr, "macgo: code signed with identity: %s\n", b.Config.CodeSignIdentity)
		}
	} else if b.Config.AdHocSign {
		b.Config.CodeSignIdentity = "-"
		if err := codeSignBundle(b.Path, b.Config); err != nil {
			return fmt.Errorf("ad-hoc signing failed: %w", err)
		}
		if b.Config.Debug {
			fmt.Fprintf(os.Stderr, "macgo: ad-hoc signed\n")
		}
	} else if b.Config.AutoSign {
		if identity := findDeveloperID(b.Config.Debug); identity != "" {
			b.Config.CodeSignIdentity = identity
			if err := codeSignBundle(b.Path, b.Config); err != nil {
				if b.Config.Debug {
					fmt.Fprintf(os.Stderr, "macgo: auto-signing failed, continuing unsigned: %v\n", err)
				}
			} else {
				if b.Config.Debug {
					fmt.Fprintf(os.Stderr, "macgo: auto-signed with identity: %s\n", identity)
				}
			}
		} else if b.Config.Debug {
			fmt.Fprintf(os.Stderr, "macgo: no Developer ID found, creating unsigned bundle\n")
		}
	}

	return nil
}

// Validate checks if the bundle is properly formed and signed.
func (b *Bundle) Validate() error {
	if b.Path == "" {
		return fmt.Errorf("bundle path not set")
	}

	// Check if bundle directory exists
	if _, err := os.Stat(b.Path); err != nil {
		return fmt.Errorf("bundle does not exist: %w", err)
	}

	// Check required structure
	contentsDir := filepath.Join(b.Path, "Contents")
	macosDir := filepath.Join(contentsDir, "MacOS")
	plistPath := filepath.Join(contentsDir, "Info.plist")

	for _, path := range []string{contentsDir, macosDir, plistPath} {
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("required bundle component missing: %s", path)
		}
	}

	return nil
}

// CleanName returns the cleaned application name.
func (b *Bundle) CleanName() string {
	return b.appName
}

// BundleID returns the bundle identifier.
func (b *Bundle) BundleID() string {
	return b.bundleID
}

// Version returns the application version.
func (b *Bundle) Version() string {
	return b.version
}

// ExecutablePath returns the path to the executable inside the bundle.
func (b *Bundle) ExecutablePath() string {
	if b.Path == "" {
		return ""
	}
	execName := filepath.Base(b.appName)
	return filepath.Join(b.Path, "Contents", "MacOS", execName)
}

// Create is a convenience function that creates a bundle from execPath and config fields.
// This avoids the need for complex config conversion.
func Create(execPath string, appName, bundleID, version string, permissions []string,
	custom []string, appGroups []string, debug bool, keepBundle *bool,
	codeSignIdentity, codeSigningIdentifier string, autoSign, adHocSign bool,
	info map[string]interface{}, uiMode UIMode) (*Bundle, error) {

	config := &Config{
		AppName:               appName,
		BundleID:              bundleID,
		Version:               version,
		Permissions:           permissions,
		Custom:                custom,
		AppGroups:             appGroups,
		Debug:                 debug,
		KeepBundle:            keepBundle,
		CodeSignIdentity:      codeSignIdentity,
		CodeSigningIdentifier: codeSigningIdentifier,
		AutoSign:              autoSign,
		AdHocSign:             adHocSign,
		Info:                  info,
		UIMode:                uiMode,
	}

	bundle, err := New(execPath, config)
	if err != nil {
		return nil, err
	}

	if err := bundle.Create(); err != nil {
		return nil, err
	}

	if err := bundle.Sign(); err != nil {
		return nil, err
	}

	return bundle, nil
}

// isBundleUpToDate checks if the bundle contains the same binary as the original executable
// by comparing SHA256 hashes. Returns true if the bundle is up to date, false otherwise.
func (b *Bundle) isBundleUpToDate() bool {
	if b.Path == "" {
		return false
	}

	// Get the executable inside the bundle
	execName := filepath.Base(b.appName)
	bundleExecPath := filepath.Join(b.Path, "Contents", "MacOS", execName)

	// Check if bundle executable exists
	if _, err := os.Stat(bundleExecPath); err != nil {
		if b.Config.Debug {
			fmt.Fprintf(os.Stderr, "macgo: bundle executable not found: %v\n", err)
		}
		return false
	}

	// Calculate hash of original executable
	originalHash, err := system.CalculateFileSHA256(b.execPath)
	if err != nil {
		if b.Config.Debug {
			fmt.Fprintf(os.Stderr, "macgo: failed to calculate original binary hash: %v\n", err)
		}
		return false
	}

	// Calculate hash of bundle executable
	bundleHash, err := system.CalculateFileSHA256(bundleExecPath)
	if err != nil {
		if b.Config.Debug {
			fmt.Fprintf(os.Stderr, "macgo: failed to calculate bundle binary hash: %v\n", err)
		}
		return false
	}

	return originalHash == bundleHash
}
