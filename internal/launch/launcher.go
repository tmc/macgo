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
func New() *Manager {
	return &Manager{
		directLauncher:   &DirectLauncher{},
		servicesLauncher: &ServicesLauncher{},
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
	// Check explicit configuration overrides first
	if cfg.ForceLaunchServices {
		if cfg.Debug {
			fmt.Fprintf(os.Stderr, "macgo: forced LaunchServices via config\n")
		}
		return StrategyServices
	}

	if cfg.ForceDirectExecution {
		if cfg.Debug {
			fmt.Fprintf(os.Stderr, "macgo: forced direct execution via config\n")
		}
		return StrategyDirect
	}

	// Check environment variable overrides
	if os.Getenv("MACGO_FORCE_LAUNCH_SERVICES") == "1" {
		if cfg.Debug {
			fmt.Fprintf(os.Stderr, "macgo: forced LaunchServices via environment\n")
		}
		return StrategyServices
	}

	if os.Getenv("MACGO_FORCE_DIRECT") == "1" {
		if cfg.Debug {
			fmt.Fprintf(os.Stderr, "macgo: forced direct execution via environment\n")
		}
		return StrategyDirect
	}

	// Determine based on required permissions
	needsLaunchServices := m.requiresLaunchServices(cfg.Permissions)

	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "macgo: needsLaunchServices: %v (permissions: %v)\n", needsLaunchServices, cfg.Permissions)
	}

	if needsLaunchServices {
		return StrategyServices
	}

	return StrategyDirect
}

// requiresLaunchServices determines if the requested permissions require LaunchServices.
func (m *Manager) requiresLaunchServices(permissions []string) bool {
	for _, perm := range permissions {
		switch perm {
		case "files", "camera", "microphone", "location":
			return true
		}
	}
	return false
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
