// Package signed provides automatic code signing for macgo v2.
// Simply import this package to enable automatic Developer ID detection.
package signed

import (
	"fmt"
	"os"

	macgo "github.com/tmc/misc/macgo"
)

func init() {
	// Enable auto-signing by default when this package is imported
	_ = os.Setenv("MACGO_AUTO_SIGN", "1")
}

// Request is a convenience function that creates a config with auto-signing and the given permissions.
func Request(perms ...macgo.Permission) error {
	cfg := &macgo.Config{
		Permissions: perms,
		AutoSign:    true,
	}
	if err := macgo.Start(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "macgo/auto/signed: failed to start with auto-signing: %v\n", err)
		return err
	}
	return nil
}

// Start is a convenience function that creates a config with auto-signing.
func Start(cfg *macgo.Config) error {
	if cfg == nil {
		cfg = &macgo.Config{}
	}
	cfg.AutoSign = true
	if err := macgo.Start(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "macgo/auto/signed: failed to start with auto-signing: %v\n", err)
		return err
	}
	return nil
}

// Auto loads configuration from environment, enables auto-signing, and starts macgo.
func Auto() error {
	cfg := new(macgo.Config).FromEnv()
	cfg.AutoSign = true
	if err := macgo.Start(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "macgo/auto/signed: failed to auto-start with signing: %v\n", err)
		return err
	}
	return nil
}
