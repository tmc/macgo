// Package launch provides application launching strategies for macOS app bundles.
package launch

import (
	"context"
	"fmt"
	"os"
)

// Strategy represents different ways to launch an application.
type Strategy int

const (
	// StrategyDirect executes the binary directly within the bundle.
	StrategyDirect Strategy = iota
	// StrategyServices uses LaunchServices via the 'open' command.
	StrategyServices
)

// Config contains the launch-specific configuration extracted from the main Config.
// This avoids importing the main package and keeps the launch package focused.
type Config struct {
	// AppName is the name of the application
	AppName string
	// BundleID is the bundle identifier
	BundleID string
	// Permissions are the requested macOS permissions (as strings)
	Permissions []string
	// Debug enables debug logging
	Debug bool
	// ForceLaunchServices forces use of LaunchServices
	ForceLaunchServices bool
	// ForceDirectExecution forces direct execution
	ForceDirectExecution bool
	// Background indicates the app should not steal focus (LSBackgroundOnly apps)
	Background bool
}

// Launcher defines the interface for launching applications.
type Launcher interface {
	// Launch executes the application using the appropriate strategy.
	Launch(ctx context.Context, bundlePath, execPath string, cfg *Config) error
}

// Manager coordinates different launch strategies.
type Manager struct {
	directLauncher   Launcher
	servicesLauncher Launcher
}

// New creates a new launch manager with default launchers.
// The services launcher version is selected via MACGO_LAUNCHER_VERSION or MACGO_SERVICES_VERSION env var.
// - Default: ServicesLauncher (v1, STABLE - uses config-file strategy with continuous polling)
// - Version "2" or "v2": ServicesLauncherV2 (EXPERIMENTAL - similar to v1 but with future enhancements)
func New() *Manager {
	// Determine which services launcher to use
	var servicesLauncher Launcher
	version := os.Getenv("MACGO_LAUNCHER_VERSION")
	if version == "" {
		version = os.Getenv("MACGO_SERVICES_VERSION")
	}

	if version == "2" || version == "v2" {
		servicesLauncher = &ServicesLauncherV2{}
		if os.Getenv("MACGO_DEBUG") == "1" {
			fmt.Fprintf(os.Stderr, "macgo: selected ServicesLauncherV2\n")
		}
	} else {
		servicesLauncher = &ServicesLauncher{}
		if os.Getenv("MACGO_DEBUG") == "1" && version != "" {
			fmt.Fprintf(os.Stderr, "macgo: selected ServicesLauncher (v1) - unknown version %q\n", version)
		}
	}

	return &Manager{
		directLauncher:   &DirectLauncher{},
		servicesLauncher: servicesLauncher,
	}
}

// NewWithLaunchers creates a new launch manager with custom launchers.
func NewWithLaunchers(direct, services Launcher) *Manager {
	return &Manager{
		directLauncher:   direct,
		servicesLauncher: services,
	}
}

// Launch determines the appropriate strategy and launches the application.
func (m *Manager) Launch(ctx context.Context, bundlePath, execPath string, cfg *Config) error {
	strategy := m.determineStrategy(cfg)

	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "macgo: selected launch strategy: %v\n", strategy)
	}

	switch strategy {
	case StrategyDirect:
		if cfg.Debug {
			fmt.Fprintf(os.Stderr, "macgo: using direct execution\n")
		}
		return m.directLauncher.Launch(ctx, bundlePath, execPath, cfg)
	case StrategyServices:
		if cfg.Debug {
			fmt.Fprintf(os.Stderr, "macgo: using LaunchServices\n")
		}
		return m.servicesLauncher.Launch(ctx, bundlePath, execPath, cfg)
	default:
		return fmt.Errorf("unknown launch strategy: %v", strategy)
	}
}

// determineStrategy selects the appropriate launch strategy based on configuration.
func (m *Manager) determineStrategy(cfg *Config) Strategy {
	// Check overrides for Direct Execution (Opt-out)
	if cfg.ForceDirectExecution {
		if cfg.Debug {
			fmt.Fprintf(os.Stderr, "macgo: forced direct execution via config\n")
		}
		return StrategyDirect
	}

	if os.Getenv("MACGO_FORCE_DIRECT") == "1" {
		if cfg.Debug {
			fmt.Fprintf(os.Stderr, "macgo: forced direct execution via environment\n")
		}
		return StrategyDirect
	}

	// Default to LaunchServices for TCC compatibility
	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "macgo: using default strategy (LaunchServices)\n")
	}
	return StrategyServices
}

// String returns a string representation of the strategy.
func (s Strategy) String() string {
	switch s {
	case StrategyDirect:
		return "direct"
	case StrategyServices:
		return "services"
	default:
		return "unknown"
	}
}
