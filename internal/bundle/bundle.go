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

	// reused indicates the bundle was reused from a previous run (no signing needed)
	reused bool
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

	// CustomStrings allows specifying custom entitlements with string values.
	CustomStrings map[string]string

	// CustomArrays allows specifying custom entitlements with array values.
	CustomArrays map[string][]string

	// AppGroups specifies app group identifiers for sharing data between apps.
	AppGroups []string

	// Debug enables debug logging.
	Debug bool

	// CleanupBundle enables cleanup of the app bundle after execution.
	CleanupBundle bool

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

	// DevMode creates a stable wrapper bundle that exec's the original binary.
	// This preserves TCC permissions across rebuilds since only the wrapper is signed,
	// not the development binary. Enable via MACGO_DEV_MODE=1 for development workflows.
	DevMode bool

	// ProvisioningProfile is the path to a provisioning profile to embed in the bundle.
	ProvisioningProfile string

	// IconPath is the path to an .icns file to use as the app icon.
	IconPath string
}

// shouldCleanupBundle returns true if the bundle should be removed.
// Defaults to false (bundle is kept for reuse).
func (c *Config) shouldCleanupBundle() bool {
	return c.CleanupBundle
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

	// Check if bundle already exists and should be kept (not cleaned up)
	if !b.Config.shouldCleanupBundle() {
		if _, err := os.Stat(bundleDir); err == nil {
			// Check if the original executable has changed by comparing SHA256
			if b.isBundleUpToDate() {
				if b.Config.Debug {
					fmt.Fprintf(os.Stderr, "macgo: reusing existing bundle at %s (binary unchanged)\n", bundleDir)
				}
				b.reused = true
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

	execName := filepath.Base(b.appName)
	destExec := filepath.Join(macosDir, execName)

	if b.Config.DevMode {
		// DevMode: Copy the binary and store the dev target path.
		// At runtime, the bundled binary will exec the dev target.
		// This preserves TCC permissions since the bundle signature stays stable.
		if err := system.CopyFile(b.execPath, destExec); err != nil {
			return fmt.Errorf("failed to copy executable: %w", err)
		}
		if err := os.Chmod(destExec, 0755); err != nil {
			return fmt.Errorf("failed to set executable permissions: %w", err)
		}
		// Store the target binary path - runtime will exec this
		if err := b.storeDevModeTarget(contentsDir); err != nil {
			return fmt.Errorf("failed to store dev target: %w", err)
		}
		if b.Config.Debug {
			fmt.Fprintf(os.Stderr, "macgo: dev mode enabled - bundle will exec %s at runtime\n", b.execPath)
		}
	} else {
		// Normal mode: Copy the executable directly
		if err := system.CopyFile(b.execPath, destExec); err != nil {
			return fmt.Errorf("failed to copy executable: %w", err)
		}
		if err := os.Chmod(destExec, 0755); err != nil {
			return fmt.Errorf("failed to set executable permissions: %w", err)
		}

		// Store the original binary's hash for future up-to-date checks.
		// This is done BEFORE code signing since signing modifies the binary.
		if err := b.storeSourceHash(contentsDir); err != nil {
			if b.Config.Debug {
				fmt.Fprintf(os.Stderr, "macgo: warning: failed to store source hash: %v\n", err)
			}
			// Non-fatal - bundle will just be recreated on next run
		}
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

	// Set app icon if provided
	if b.Config.IconPath != "" {
		infoCfg.CustomKeys["CFBundleIconFile"] = filepath.Base(b.Config.IconPath)
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

	// Auto-derive string entitlements from provisioning profile or signing identity.
	derived := b.deriveStringEntitlements()
	if len(derived) > 0 {
		if b.Config.CustomStrings == nil {
			b.Config.CustomStrings = make(map[string]string)
		}
		for k, v := range derived {
			if _, set := b.Config.CustomStrings[k]; !set {
				b.Config.CustomStrings[k] = v
				if b.Config.Debug {
					fmt.Fprintf(os.Stderr, "macgo: auto-derived entitlement %s=%s\n", k, v)
				}
			}
		}
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
			Permissions:   plistPermissions,
			Custom:        customEntitlements,
			CustomStrings: b.Config.CustomStrings,
			CustomArrays:  b.Config.CustomArrays,
			AppGroups:     b.Config.AppGroups,
		}
		if err := plist.WriteEntitlements(entPath, entCfg); err != nil {
			return fmt.Errorf("failed to write entitlements: %w", err)
		}
	}

	// Copy provisioning profile if specified
	if b.Config.ProvisioningProfile != "" {
		profileDest := filepath.Join(contentsDir, "embedded.provisionprofile")
		if err := system.CopyFile(b.Config.ProvisioningProfile, profileDest); err != nil {
			return fmt.Errorf("copying provisioning profile: %w", err)
		}
		if b.Config.Debug {
			fmt.Fprintf(os.Stderr, "macgo: embedded provisioning profile from %s\n", b.Config.ProvisioningProfile)
		}
	}

	// Copy icon file if specified
	if b.Config.IconPath != "" {
		resourcesDir := filepath.Join(contentsDir, "Resources")
		if err := os.MkdirAll(resourcesDir, 0755); err != nil {
			return fmt.Errorf("failed to create Resources directory: %w", err)
		}
		iconName := filepath.Base(b.Config.IconPath)
		iconDest := filepath.Join(resourcesDir, iconName)
		if err := system.CopyFile(b.Config.IconPath, iconDest); err != nil {
			return fmt.Errorf("failed to copy icon: %w", err)
		}
		if b.Config.Debug {
			fmt.Fprintf(os.Stderr, "macgo: copied icon %s to bundle\n", iconName)
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
// If the bundle was reused (not recreated), signing is skipped since the
// existing signature is still valid.
func (b *Bundle) Sign() error {
	if b.Path == "" {
		return fmt.Errorf("bundle not created - call Create() first")
	}

	// Skip signing if bundle was reused - existing signature is still valid
	if b.reused {
		if b.Config.Debug {
			fmt.Fprintf(os.Stderr, "macgo: skipping code signing (bundle reused)\n")
		}
		return nil
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
	custom []string, customStrings map[string]string, customArrays map[string][]string, appGroups []string, debug bool, cleanupBundle bool,
	codeSignIdentity, codeSigningIdentifier string, autoSign, adHocSign bool,
	info map[string]interface{}, uiMode UIMode, devMode bool, provisioningProfile string, iconPath string) (*Bundle, error) {

	config := &Config{
		AppName:               appName,
		BundleID:              bundleID,
		Version:               version,
		Permissions:           permissions,
		Custom:                custom,
		CustomStrings:         customStrings,
		CustomArrays:          customArrays,
		AppGroups:             appGroups,
		Debug:                 debug,
		CleanupBundle:         cleanupBundle,
		CodeSignIdentity:      codeSignIdentity,
		CodeSigningIdentifier: codeSigningIdentifier,
		AutoSign:              autoSign,
		AdHocSign:             adHocSign,
		Info:                  info,
		UIMode:                uiMode,
		DevMode:               devMode,
		ProvisioningProfile:   provisioningProfile,
		IconPath:              iconPath,
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

// sourceHashFile is the name of the file storing the original binary's SHA256 hash.
// This hash is captured BEFORE code signing so we can accurately detect source changes.
const sourceHashFile = ".source_hash"

// isBundleUpToDate checks if the bundle was created from the current source binary.
// For normal bundles, it compares the stored hash against the current source binary's hash.
// For dev mode bundles, it only checks that the target path is the same (binary can change).
func (b *Bundle) isBundleUpToDate() bool {
	if b.Path == "" {
		return false
	}

	// Check if this is a dev mode bundle first
	targetPath := filepath.Join(b.Path, "Contents", devModeTargetFile)
	if _, err := os.Stat(targetPath); err == nil {
		// This is a dev mode bundle - use dev mode check
		return b.isDevModeBundleUpToDate()
	}

	// Normal bundle - check source hash
	hashPath := filepath.Join(b.Path, "Contents", sourceHashFile)
	storedHashBytes, err := os.ReadFile(hashPath)
	if err != nil {
		if b.Config.Debug {
			fmt.Fprintf(os.Stderr, "macgo: no stored hash found: %v\n", err)
		}
		return false
	}
	storedHash := strings.TrimSpace(string(storedHashBytes))

	// Calculate hash of current source executable
	currentHash, err := system.CalculateFileSHA256(b.execPath)
	if err != nil {
		if b.Config.Debug {
			fmt.Fprintf(os.Stderr, "macgo: failed to calculate source binary hash: %v\n", err)
		}
		return false
	}

	if b.Config.Debug {
		fmt.Fprintf(os.Stderr, "macgo: comparing hashes - stored=%s current=%s\n", storedHash[:16]+"...", currentHash[:16]+"...")
	}

	return storedHash == currentHash
}

// storeSourceHash saves the source binary's SHA256 hash to a metadata file.
// This must be called BEFORE code signing since signing modifies the binary.
func (b *Bundle) storeSourceHash(contentsDir string) error {
	hash, err := system.CalculateFileSHA256(b.execPath)
	if err != nil {
		return fmt.Errorf("calculate hash: %w", err)
	}

	hashPath := filepath.Join(contentsDir, sourceHashFile)
	if err := os.WriteFile(hashPath, []byte(hash+"\n"), 0644); err != nil {
		return fmt.Errorf("write hash file: %w", err)
	}

	return nil
}

// devModeTargetFile is the name of the file storing the dev mode target binary path.
const devModeTargetFile = ".dev_target"

// storeDevModeTarget saves the target binary path for dev mode bundles.
// This is used to detect if the target path has changed (requiring bundle recreation).
func (b *Bundle) storeDevModeTarget(contentsDir string) error {
	targetPath := filepath.Join(contentsDir, devModeTargetFile)
	if err := os.WriteFile(targetPath, []byte(b.execPath+"\n"), 0644); err != nil {
		return fmt.Errorf("write target file: %w", err)
	}
	return nil
}

// isDevModeBundleUpToDate checks if a dev mode bundle is still valid.
// It verifies the target binary path hasn't changed (the binary itself can change).
func (b *Bundle) isDevModeBundleUpToDate() bool {
	if b.Path == "" {
		return false
	}

	// Check if this is a dev mode bundle by looking for the target file
	targetPath := filepath.Join(b.Path, "Contents", devModeTargetFile)
	storedTargetBytes, err := os.ReadFile(targetPath)
	if err != nil {
		// Not a dev mode bundle or file missing
		return false
	}
	storedTarget := strings.TrimSpace(string(storedTargetBytes))

	// Check if the target path is the same
	if storedTarget != b.execPath {
		if b.Config.Debug {
			fmt.Fprintf(os.Stderr, "macgo: dev mode target changed - stored=%s current=%s\n", storedTarget, b.execPath)
		}
		return false
	}

	// Verify the target binary still exists
	if _, err := os.Stat(b.execPath); err != nil {
		if b.Config.Debug {
			fmt.Fprintf(os.Stderr, "macgo: dev mode target binary not found: %v\n", err)
		}
		return false
	}

	if b.Config.Debug {
		fmt.Fprintf(os.Stderr, "macgo: dev mode bundle up-to-date (target: %s)\n", b.execPath)
	}
	return true
}
