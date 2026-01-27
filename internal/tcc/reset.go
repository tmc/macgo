package tcc

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tmc/macgo/bundle"
)

// ResolutionConfig holds configuration for resolving bundle IDs.
type ResolutionConfig struct {
	BundleID string
	AppName  string
	Debug    bool
}

// Reset resets TCC permissions for the given bundle ID.
func Reset(bundleID string, debug bool) error {
	if bundleID == "" {
		return fmt.Errorf("bundle ID cannot be empty")
	}

	if debug {
		fmt.Fprintf(os.Stderr, "tcc: resetting TCC permissions for bundle ID: %s\n", bundleID)
	}

	// Execute tccutil reset All command
	cmd := exec.Command("tccutil", "reset", "All", bundleID)
	output, err := cmd.CombinedOutput()

	if err != nil {
		if debug {
			fmt.Fprintf(os.Stderr, "tcc: tccutil output: %s\n", string(output))
		}
		return fmt.Errorf("tccutil reset failed: %w", err)
	}

	if debug {
		fmt.Fprintf(os.Stderr, "tcc: successfully reset TCC permissions for %s\n", bundleID)
		if len(output) > 0 {
			fmt.Fprintf(os.Stderr, "tcc: tccutil output: %s\n", string(output))
		}
	}

	return nil
}

// ResetWithConfig resets TCC permissions using the provided configuration.
// It will resolve the bundle ID if not provided in the config.
func ResetWithConfig(cfg ResolutionConfig) error {
	bundleID, err := ResolveBundleID(cfg)
	if err != nil {
		return fmt.Errorf("failed to resolve bundle ID: %w", err)
	}

	return Reset(bundleID, cfg.Debug)
}

// ResolveBundleID resolves the bundle ID from the configuration.
// If BundleID is provided in config, it uses that.
// Otherwise, it infers one from the AppName or executable name.
func ResolveBundleID(cfg ResolutionConfig) (string, error) {
	// Use explicitly provided bundle ID
	if cfg.BundleID != "" {
		return cfg.BundleID, nil
	}

	// Determine app name
	appName := cfg.AppName
	if appName == "" {
		execPath, err := os.Executable()
		if err != nil {
			return "", fmt.Errorf("failed to get executable path: %w", err)
		}
		appName = strings.TrimSuffix(filepath.Base(execPath), filepath.Ext(execPath))
	}

	if appName == "" {
		return "", fmt.Errorf("could not determine app name")
	}

	// Create a bundle ID using the inference logic
	bundleID := bundle.InferBundleID(appName)

	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "tcc: inferred bundle ID: %s (from app name: %s)\n", bundleID, appName)
	}

	return bundleID, nil
}

// ResetSpecificServices resets only specific TCC services for a bundle ID.
func ResetSpecificServices(bundleID string, services []string, debug bool) error {
	if bundleID == "" {
		return fmt.Errorf("bundle ID cannot be empty")
	}

	if len(services) == 0 {
		return nil // Nothing to reset
	}

	for _, service := range services {
		if debug {
			fmt.Fprintf(os.Stderr, "tcc: resetting %s permission for bundle ID: %s\n", service, bundleID)
		}

		cmd := exec.Command("tccutil", "reset", service, bundleID)
		output, err := cmd.CombinedOutput()

		if err != nil {
			if debug {
				fmt.Fprintf(os.Stderr, "tcc: tccutil output for %s: %s\n", service, string(output))
			}
			// Log the error but continue with other services
			if debug {
				fmt.Fprintf(os.Stderr, "tcc: failed to reset %s permission: %v\n", service, err)
			}
			continue
		}

		if debug {
			fmt.Fprintf(os.Stderr, "tcc: successfully reset %s permission for %s\n", service, bundleID)
			if len(output) > 0 {
				fmt.Fprintf(os.Stderr, "tcc: tccutil output for %s: %s\n", service, string(output))
			}
		}
	}

	return nil
}

// ResetForPermissions resets TCC permissions for the specific permissions provided.
func ResetForPermissions(bundleID string, perms []Permission, debug bool) error {
	services := GetTCCServices(perms)
	if len(services) == 0 {
		if debug {
			fmt.Fprintf(os.Stderr, "tcc: no TCC services to reset for provided permissions\n")
		}
		return nil
	}

	return ResetSpecificServices(bundleID, services, debug)
}
