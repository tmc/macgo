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
	"runtime"
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

	// Permissions are the requested macOS permissions.
	Permissions []Permission

	// Custom allows specifying custom entitlements not covered by Permission constants.
	Custom []string

	// Debug enables debug logging.
	Debug bool

	// KeepBundle prevents cleanup of temporary bundles.
	KeepBundle bool
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