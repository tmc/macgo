// Package macgo automatically creates and launches macOS app bundles to enable
// TCC (Transparency, Consent, and Control) permissions for command-line Go programs.
//
// On macOS, command-line binaries cannot access protected resources like camera,
// microphone, location, or user files because macOS requires applications to be
// properly bundled and signed to request these permissions through the TCC system.
// macgo bridges this gap by automatically wrapping your Go binary in a properly
// configured app bundle with the necessary entitlements.
//
// # Cross-Platform Compatibility
//
// macgo is designed to work gracefully on all platforms. On non-macOS platforms,
// all macgo functions are no-ops that do not affect program execution. This allows
// you to use macgo in cross-platform applications without platform-specific build
// constraints. When MACGO_DEBUG=1 is set, macgo will log informational messages
// about skipping operations on non-macOS platforms.
//
// # Quick Start
//
// The simplest way to use macgo is with auto-initialization:
//
//	package main
//
//	import _ "github.com/tmc/misc/macgo/auto/sandbox"
//
//	func main() {
//	    // Your code automatically runs with app sandbox enabled
//	}
//
// For more control, configure macgo explicitly:
//
//	package main
//
//	import "github.com/tmc/misc/macgo"
//
//	func init() {
//	    macgo.RequestEntitlements(
//	        macgo.EntAppSandbox,
//	        macgo.EntCamera,
//	        macgo.EntMicrophone,
//	    )
//	    macgo.Start()
//	}
//
//	func main() {
//	    // Your application code with requested permissions
//	}
//
// # Architecture
//
// macgo uses a clean, modular architecture following Go best practices:
//
//   - macgo: Main API and configuration
//   - bundle: App bundle creation and management
//   - security: Code signing and validation
//   - signal: Signal forwarding and process management
//   - entitlements: Entitlement definitions and constants
//   - process: Process launching and I/O handling
//
// # How It Works
//
//  1. Detection: Checks if already running in an app bundle
//  2. Bundle Creation: Creates .app bundle with entitlements and Info.plist
//  3. Code Signing: Automatically signs the bundle (ad-hoc by default)
//  4. Relaunching: Relaunches the process inside the app bundle
//  5. I/O Forwarding: Maintains stdin/stdout/stderr connectivity
//  6. Signal Forwarding: Forwards signals (including Ctrl+C) between processes
//
// # Available Entitlements
//
// macgo supports many macOS entitlements for accessing protected resources:
//
// TCC Permissions:
//   - EntCamera: Camera access
//   - EntMicrophone: Microphone access
//   - EntLocation: Location services
//   - EntAddressBook: Contacts access
//   - EntPhotos: Photos library access
//   - EntCalendars: Calendar access
//   - EntReminders: Reminders access
//
// App Sandbox:
//   - EntAppSandbox: Enable app sandbox
//   - EntUserSelectedReadOnly: Read access to user-selected files
//   - EntUserSelectedReadWrite: Read/write access to user-selected files
//   - EntNetworkClient: Outgoing network connections
//   - EntNetworkServer: Incoming network connections
//
// Hardware & Development:
//   - EntBluetooth: Bluetooth device access
//   - EntUSB: USB device access
//   - EntAllowJIT: JIT compilation
//   - EntDebugger: Debugger attachment
//
// # Environment Variables
//
// Configure macgo without code changes:
//
//	export MACGO_APP_NAME="MyApp"
//	export MACGO_BUNDLE_ID="com.example.myapp"
//	export MACGO_CAMERA=1
//	export MACGO_MIC=1
//	export MACGO_SANDBOX=1
//	export MACGO_DEBUG=1
//	./myapp
//
// # Auto-initialization Packages
//
// For convenience, macgo provides several auto-initialization packages:
//
//	// Basic - no sandbox
//	import _ "github.com/tmc/misc/macgo/auto"
//
//	// With app sandbox
//	import _ "github.com/tmc/misc/macgo/auto/sandbox"
//
//	// With sandbox + user file read access
//	import _ "github.com/tmc/misc/macgo/auto/sandbox/readonly"
//
//	// With improved signal handling (better Ctrl+C)
//	import _ "github.com/tmc/misc/macgo/auto/sandbox/signalhandler"
//
// # Signal Handling
//
// macgo provides robust signal handling to ensure Ctrl+C and other signals
// work correctly between the parent process and the app bundle:
//
//	// Enable improved signal handling (recommended)
//	macgo.EnableImprovedSignalHandling()
//
//	// Or disable if needed for compatibility
//	macgo.DisableSignals()
//
// # Code Signing
//
// macgo automatically performs ad-hoc code signing. For distribution:
//
//	// Use specific signing identity
//	macgo.EnableSigning("Developer ID Application: Your Name")
//
//	// Or just enable automatic signing
//	macgo.EnableSigning("")
//
// # Debugging
//
// Enable debug logging to see what macgo is doing:
//
//	macgo.EnableDebug()
//	// or set environment variable: MACGO_DEBUG=1
//
// This will show detailed logs about bundle creation, signing, and process management.
//
// # Requirements
//
//   - macOS 10.15+ (Catalina or later)
//   - Go 1.19+
//   - Xcode Command Line Tools (for code signing)
//
// # Security Considerations
//
//   - macgo creates temporary app bundles that are automatically cleaned up
//   - All entitlements must be explicitly requested
//   - Code signing is performed automatically for security
//   - Sandbox entitlements provide additional security boundaries
//   - File access is controlled through macOS entitlements system
package macgo