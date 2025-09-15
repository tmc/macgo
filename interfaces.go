package macgo

import (
	"context"
	"os"
)

// BundleCreator defines the interface for creating macOS app bundles.
// This interface separates bundle creation logic from the main package,
// enabling better testing and different implementation strategies.
type BundleCreator interface {
	// Create creates an app bundle for the given executable with the provided configuration.
	// Returns the path to the created bundle or an error if creation fails.
	Create(ctx context.Context, cfg *Config, execPath string) (bundlePath string, err error)

	// Exists checks if a bundle already exists for the given configuration.
	Exists(cfg *Config, execPath string) (bool, error)

	// IsUpToDate checks if an existing bundle is up to date with the current executable.
	IsUpToDate(cfg *Config, execPath, bundlePath string) (bool, error)
}

// PathValidator defines the interface for validating and sanitizing file paths.
// This interface encapsulates security-critical path operations.
type PathValidator interface {
	// Validate checks if a path is safe to use.
	Validate(path string) error

	// Sanitize cleans and validates a path, returning the sanitized version.
	Sanitize(path string) (string, error)

	// SecureJoin joins path elements in a secure way, preventing directory traversal.
	SecureJoin(base string, elements ...string) (string, error)
}

// ProcessLauncher defines the interface for launching processes within app bundles.
// This interface abstracts process management and I/O redirection.
type ProcessLauncher interface {
	// Launch starts a process within an app bundle with the given arguments.
	Launch(ctx context.Context, bundlePath string, args []string) error

	// Relaunch relaunches the current process within an app bundle.
	Relaunch(ctx context.Context, bundlePath, execPath string, args []string) error
}

// Signer defines the interface for code signing operations.
// This interface encapsulates all code signing functionality.
type Signer interface {
	// Sign signs an app bundle with the given identity.
	// If identity is empty, ad-hoc signing is used.
	Sign(ctx context.Context, bundlePath, identity string) error

	// Verify verifies the signature of an app bundle.
	Verify(bundlePath string) error

	// ValidateIdentity checks if a signing identity is valid.
	ValidateIdentity(identity string) error
}

// SignalForwarder defines the interface for forwarding signals between processes.
// This interface handles signal propagation from parent to child processes.
type SignalForwarder interface {
	// Forward sets up signal forwarding from the current process to the target process.
	Forward(ctx context.Context, target *os.Process) error

	// Stop stops signal forwarding.
	Stop() error
}

// EntitlementRegistry defines the interface for managing entitlements.
// This interface provides a clean way to register and query entitlements.
type EntitlementRegistry interface {
	// Register registers an entitlement with the given value.
	Register(entitlement Entitlement, value bool)

	// Get returns the value of an entitlement, or false if not set.
	Get(entitlement Entitlement) bool

	// GetAll returns all registered entitlements.
	GetAll() Entitlements

	// Clear removes all registered entitlements.
	Clear()
}
