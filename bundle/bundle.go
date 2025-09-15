// Package bundle provides macOS app bundle creation and management functionality.
// This package implements clean, focused bundle operations following the single
// responsibility principle.
package bundle

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tmc/misc/macgo"
)

// Creator implements the BundleCreator interface.
// It provides the core functionality for creating and managing macOS app bundles.
type Creator struct {
	pathValidator macgo.PathValidator
	signer        macgo.Signer
}

// NewCreator creates a new bundle Creator with the provided dependencies.
func NewCreator(pathValidator macgo.PathValidator, signer macgo.Signer) *Creator {
	return &Creator{
		pathValidator: pathValidator,
		signer:        signer,
	}
}

// Create creates an app bundle for the given executable with the provided configuration.
// This is the main orchestration function that coordinates all bundle creation steps.
func (c *Creator) Create(ctx context.Context, cfg *macgo.Config, execPath string) (string, error) {
	if cfg == nil {
		cfg = macgo.NewConfig()
	}

	// Basic configuration validation
	if cfg.Entitlements == nil {
		cfg.Entitlements = make(macgo.Entitlements)
	}
	if cfg.PlistEntries == nil {
		cfg.PlistEntries = make(map[string]any)
	}

	// Validate executable path
	if err := c.pathValidator.Validate(execPath); err != nil {
		return "", fmt.Errorf("bundle: invalid executable path: %w", err)
	}

	// Determine bundle path
	bundlePath, err := c.determineBundlePath(cfg, execPath)
	if err != nil {
		return "", fmt.Errorf("bundle: determine bundle path: %w", err)
	}

	// Check if bundle already exists and is up to date
	if exists, err := c.Exists(cfg, execPath); err != nil {
		return "", fmt.Errorf("bundle: check bundle existence: %w", err)
	} else if exists {
		if upToDate, err := c.IsUpToDate(cfg, execPath, bundlePath); err != nil {
			return "", fmt.Errorf("bundle: check if bundle is up to date: %w", err)
		} else if upToDate {
			return bundlePath, nil
		}
	}

	// Create bundle directory structure
	if err := c.createDirectoryStructure(bundlePath); err != nil {
		return "", fmt.Errorf("bundle: create directory structure: %w", err)
	}

	// Copy executable
	if err := c.copyExecutable(execPath, bundlePath, cfg); err != nil {
		return "", fmt.Errorf("bundle: copy executable: %w", err)
	}

	// Generate Info.plist
	if err := c.generateInfoPlist(bundlePath, cfg, execPath); err != nil {
		return "", fmt.Errorf("bundle: generate Info.plist: %w", err)
	}

	// Generate entitlements.plist if needed
	if len(cfg.Entitlements) > 0 {
		if err := c.generateEntitlementsPlist(bundlePath, cfg); err != nil {
			return "", fmt.Errorf("bundle: generate entitlements.plist: %w", err)
		}
	}

	// Sign the bundle if auto-signing is enabled
	if cfg.AutoSign && c.signer != nil {
		if err := c.signer.Sign(ctx, bundlePath, cfg.SigningIdentity); err != nil {
			return "", fmt.Errorf("bundle: sign bundle: %w", err)
		}
	}

	return bundlePath, nil
}

// Exists checks if a bundle already exists for the given configuration.
func (c *Creator) Exists(cfg *macgo.Config, execPath string) (bool, error) {
	bundlePath, err := c.determineBundlePath(cfg, execPath)
	if err != nil {
		return false, fmt.Errorf("bundle: determine bundle path: %w", err)
	}

	info, err := os.Stat(bundlePath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("bundle: stat bundle path: %w", err)
	}

	return info.IsDir(), nil
}

// IsUpToDate checks if an existing bundle is up to date with the current executable.
func (c *Creator) IsUpToDate(cfg *macgo.Config, execPath, bundlePath string) (bool, error) {
	// Get executable info
	execInfo, err := os.Stat(execPath)
	if err != nil {
		return false, fmt.Errorf("bundle: stat executable: %w", err)
	}

	// Get bundle executable path
	appName := c.getApplicationName(cfg, execPath)
	bundleExecPath := filepath.Join(bundlePath, "Contents", "MacOS", appName)

	// Check if bundle executable exists
	bundleExecInfo, err := os.Stat(bundleExecPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("bundle: stat bundle executable: %w", err)
	}

	// Compare modification times
	if execInfo.ModTime().After(bundleExecInfo.ModTime()) {
		return false, nil
	}

	// Compare checksums for more accurate comparison
	execChecksum, err := c.calculateChecksum(execPath)
	if err != nil {
		return false, fmt.Errorf("bundle: calculate executable checksum: %w", err)
	}

	bundleChecksum, err := c.calculateChecksum(bundleExecPath)
	if err != nil {
		return false, fmt.Errorf("bundle: calculate bundle executable checksum: %w", err)
	}

	return execChecksum == bundleChecksum, nil
}

// determineBundlePath determines where the bundle should be created.
func (c *Creator) determineBundlePath(cfg *macgo.Config, execPath string) (string, error) {
	if cfg.CustomDestinationAppPath != "" {
		return cfg.CustomDestinationAppPath, nil
	}

	appName := c.getApplicationName(cfg, execPath)
	bundleName := appName + ".app"

	// Try to use the same directory as the executable
	execDir := filepath.Dir(execPath)
	bundlePath := filepath.Join(execDir, bundleName)

	// Validate the path
	if err := c.pathValidator.Validate(bundlePath); err != nil {
		// Fall back to temp directory if validation fails
		tempDir := os.TempDir()
		bundlePath = filepath.Join(tempDir, bundleName)
	}

	return bundlePath, nil
}

// getApplicationName returns the application name from config or derives it from executable path.
func (c *Creator) getApplicationName(cfg *macgo.Config, execPath string) string {
	if cfg.ApplicationName != "" {
		return cfg.ApplicationName
	}
	return strings.TrimSuffix(filepath.Base(execPath), filepath.Ext(execPath))
}

// calculateChecksum calculates SHA256 checksum of a file.
func (c *Creator) calculateChecksum(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read file for checksum: %w", err)
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}
