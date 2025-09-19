// Package adhoc provides ad-hoc code signing for macgo v2.
// Simply import this package to enable ad-hoc signing automatically.
package adhoc

import (
	"fmt"
	"os"

	macgo "github.com/tmc/misc/macgo"
)

func init() {
	// Enable ad-hoc signing by default when this package is imported
	os.Setenv("MACGO_AD_HOC_SIGN", "1")
}

// Request is a convenience function that creates a config with ad-hoc signing and the given permissions.
func Request(perms ...macgo.Permission) error {
	cfg := &macgo.Config{
		Permissions: perms,
		AdHocSign:   true,
	}
	if err := macgo.Start(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "macgo/auto/adhoc: failed to start with ad-hoc signing: %v\n", err)
		return err
	}
	return nil
}

// Start is a convenience function that creates a config with ad-hoc signing.
func Start(cfg *macgo.Config) error {
	if cfg == nil {
		cfg = &macgo.Config{}
	}
	cfg.AdHocSign = true
	if err := macgo.Start(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "macgo/auto/adhoc: failed to start with ad-hoc signing: %v\n", err)
		return err
	}
	return nil
}

// Auto loads configuration from environment, enables ad-hoc signing, and starts macgo.
func Auto() error {
	cfg := new(macgo.Config).FromEnv()
	cfg.AdHocSign = true
	if err := macgo.Start(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "macgo/auto/adhoc: failed to auto-start with ad-hoc signing: %v\n", err)
		return err
	}
	return nil
}
