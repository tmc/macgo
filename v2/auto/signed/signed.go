// Package signed provides automatic code signing for macgo v2.
// Simply import this package to enable automatic Developer ID detection.
package signed

import (
	"os"

	macgo "github.com/tmc/misc/macgo/v2"
)

func init() {
	// Enable auto-signing by default when this package is imported
	os.Setenv("MACGO_AUTO_SIGN", "1")
}

// Request is a convenience function that creates a config with auto-signing and the given permissions.
func Request(perms ...macgo.Permission) error {
	cfg := &macgo.Config{
		Permissions: perms,
		AutoSign:    true,
	}
	return macgo.Start(cfg)
}

// Start is a convenience function that creates a config with auto-signing.
func Start(cfg *macgo.Config) error {
	if cfg == nil {
		cfg = &macgo.Config{}
	}
	cfg.AutoSign = true
	return macgo.Start(cfg)
}

// Auto loads configuration from environment, enables auto-signing, and starts macgo.
func Auto() error {
	cfg := new(macgo.Config).FromEnv()
	cfg.AutoSign = true
	return macgo.Start(cfg)
}