package macgo

import (
	"context"
	"fmt"
	"os"

	"github.com/tmc/misc/macgo/internal/bundle"
	"github.com/tmc/misc/macgo/internal/launch"
	"github.com/tmc/misc/macgo/internal/system"
	"github.com/tmc/misc/macgo/internal/tcc"
	"github.com/tmc/misc/macgo/teamid"
)

// startDarwin implements the macOS-specific logic.
func startDarwin(ctx context.Context, cfg *Config) error {
	// Auto-detect and substitute team ID in app groups if needed
	if err := substituteTeamID(cfg); err != nil && cfg.Debug {
		fmt.Fprintf(os.Stderr, "macgo: failed to substitute team ID: %v\n", err)
	}
	// Check for permission reset flag
	if system.IsResetPermissionsEnabled() {
		resolutionCfg := tcc.ResolutionConfig{
			BundleID: cfg.BundleID,
			AppName:  cfg.AppName,
			Debug:    cfg.Debug,
		}
		if err := tcc.ResetWithConfig(resolutionCfg); err != nil {
			if cfg.Debug {
				fmt.Fprintf(os.Stderr, "macgo: failed to reset permissions: %v\n", err)
			}
		}
	}

	// Skip if already in app bundle
	if system.IsInAppBundle() {
		if cfg.Debug {
			fmt.Fprintf(os.Stderr, "macgo: already in app bundle\n")
		}
		return nil
	}

	// Skip if disabled
	if system.IsRelaunchDisabled() {
		if cfg.Debug {
			fmt.Fprintf(os.Stderr, "macgo: relaunch disabled\n")
		}
		return nil
	}

	// Get current executable
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("macgo: get executable: %w", err)
	}

	// Create or reuse bundle
	bundleObj, err := createSimpleBundle(execPath, cfg)
	if err != nil {
		return fmt.Errorf("macgo: bundle operation: %w", err)
	}
	bundlePath := bundleObj.Path

	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "macgo: using bundle at %s\n", bundlePath)
		fmt.Fprintf(os.Stderr, "macgo: permissions requested: %v\n", cfg.Permissions)
	}

	// Relaunch in bundle
	return relaunchInBundle(ctx, bundlePath, execPath, cfg)
}

// createSimpleBundle creates a minimal app bundle with the given configuration.
func createSimpleBundle(execPath string, cfg *Config) (*bundle.Bundle, error) {
	// Convert permissions to strings
	var permissions []string
	for _, perm := range cfg.Permissions {
		permissions = append(permissions, string(perm))
	}

	// Use the bundle package to create the bundle
	return bundle.Create(
		execPath,
		cfg.AppName,
		cfg.BundleID,
		cfg.Version,
		permissions,
		cfg.Custom,
		cfg.AppGroups,
		cfg.Debug,
		cfg.KeepBundle,
		cfg.CodeSignIdentity,
		cfg.CodeSigningIdentifier,
		cfg.AutoSign,
		cfg.AdHocSign,
	)
}

// convertPermissions converts Permission values to strings for the launch package.
func convertPermissions(permissions []Permission) []string {
	var result []string
	for _, perm := range permissions {
		result = append(result, string(perm))
	}
	return result
}

// relaunchInBundle launches the app bundle using the launch package.
func relaunchInBundle(ctx context.Context, bundlePath, execPath string, cfg *Config) error {
	// Convert main config to launch config
	launchCfg := &launch.Config{
		AppName:              cfg.AppName,
		BundleID:             cfg.BundleID,
		Permissions:          convertPermissions(cfg.Permissions),
		Debug:                cfg.Debug,
		ForceLaunchServices:  cfg.ForceLaunchServices,
		ForceDirectExecution: cfg.ForceDirectExecution,
	}

	// Create launch manager and execute
	manager := launch.New()
	return manager.Launch(ctx, bundlePath, execPath, launchCfg)
}

// substituteTeamID automatically detects team ID and substitutes "TEAMID" placeholders in app groups
func substituteTeamID(cfg *Config) error {
	if len(cfg.AppGroups) == 0 {
		return nil
	}

	// Use the helpers package for team ID detection and substitution
	teamID, substitutions, err := teamid.AutoSubstituteTeamIDInGroups(cfg.AppGroups)
	if err != nil {
		return fmt.Errorf("team ID detection failed: %w", err)
	}

	if cfg.Debug && substitutions > 0 {
		fmt.Printf("macgo: detected team ID %s, updated app groups: %v\n", teamID, cfg.AppGroups)
	}

	return nil
}
