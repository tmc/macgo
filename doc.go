// Package macgo provides seamless macOS application bundle creation with permissions and entitlements.
//
// macgo enables Go programs to integrate with macOS security features by automatically creating
// app bundles with proper structure, Info.plist, entitlements, and code signing. This allows
// Go applications to request system permissions (camera, microphone, files, etc.) just like
// native macOS applications.
//
// # Quick Start
//
// The simplest way to use macgo is with the Request function:
//
//	err := macgo.Request(macgo.Camera)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// Camera permission granted, proceed with camera access
//
// Request multiple permissions at once:
//
//	err := macgo.Request(macgo.Camera, macgo.Microphone, macgo.Files)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Configuration
//
// For more control, use the Config struct with builder methods:
//
//	cfg := macgo.NewConfig().
//	    WithAppName("MyApp").
//	    WithBundleID("com.example.myapp").
//	    WithPermissions(macgo.Camera, macgo.Microphone).
//	    WithAdHocSign().
//	    WithDebug()
//
//	err := macgo.Start(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Available Permissions
//
//   - Camera: access to camera
//   - Microphone: access to microphone
//   - Location: access to location services
//   - ScreenRecording: access to screen recording
//   - Accessibility: access to Accessibility features (e.g. input simulation)
//   - Files: access to user-selected files
//   - Network: network client/server access
//   - Sandbox: enable app sandbox with restricted file access
//
// # Launch Modes
//
// macgo supports three launch strategies:
//
// Bundle mode (default): creates a .app bundle, relaunches via LaunchServices, and
// forwards I/O. Required for TCC-dialog permissions (Camera, Microphone, Location).
//
// Single-process mode: codesigns the binary in-place, re-execs for the kernel to
// pick up new entitlements, then calls setActivationPolicy. No bundle, no child
// process, no I/O forwarding. Only works for entitlement-only permissions
// (Accessibility, Virtualization, Network). Enable via [Config.WithSingleProcess]
// or MACGO_SINGLE_PROCESS=1.
//
// The internal/launchservices package provides a standalone pure Go replacement
// for /usr/bin/open (available as cmd/lsopen), but is not used by the default
// bundle launch path due to run loop constraints in library contexts.
//
// # Code Signing
//
// macgo supports multiple signing approaches:
//
// Ad-hoc signing for development:
//
//	cfg := macgo.NewConfig().WithAdHocSign()
//
// Automatic Developer ID detection:
//
//	cfg := macgo.NewConfig().WithAutoSign()
//
// Specific signing identity:
//
//	cfg := macgo.NewConfig().WithCodeSigning("Developer ID Application: Your Name")
//
// # Environment Variables
//
// All configuration can be driven by environment variables, which are read by
// [Config.FromEnv]. Variables are grouped by category below.
//
// Application identity:
//
//	MACGO_APP_NAME            Application display name
//	MACGO_APP_NAME_PREFIX     Prefix prepended to app names (e.g. "Dev-")
//	MACGO_BUNDLE_ID           Bundle identifier (e.g. "com.example.myapp")
//	MACGO_BUNDLE_ID_PREFIX    Prefix prepended to bundle IDs (e.g. "test.")
//
// Permissions (set to "1" to enable):
//
//	MACGO_CAMERA              Request camera access
//	MACGO_MICROPHONE          Request microphone access
//	MACGO_LOCATION            Request location services access
//	MACGO_SCREEN_RECORDING    Request screen recording access
//	MACGO_FILES               Request user-selected file access
//	MACGO_NETWORK             Request network client access
//	MACGO_SANDBOX             Enable app sandbox
//
// Code signing:
//
//	MACGO_AD_HOC_SIGN         Ad-hoc sign the bundle (set to "1")
//	MACGO_AUTO_SIGN           Auto-detect Developer ID certificate (set to "1")
//	MACGO_CODE_SIGN_IDENTITY  Specific signing identity string
//
// Launch behavior (set to "1" unless noted):
//
//	MACGO_SINGLE_PROCESS      Single-process mode: codesign + re-exec, no bundle
//	MACGO_FORCE_DIRECT        Force direct binary execution (skip LaunchServices)
//	MACGO_FORCE_LAUNCH_SERVICES  Force LaunchServices even when not needed
//	MACGO_NO_RELAUNCH         Disable automatic relaunch entirely
//	MACGO_OPEN_NEW_INSTANCE   Set to "0" to disable -n flag for open command
//
// Bundle and debugging:
//
//	MACGO_DEBUG               Enable debug logging to stderr (set to "1")
//	MACGO_KEEP_BUNDLE         Preserve temporary bundle after execution (set to "1")
//	MACGO_ICON                Path to .icns file for the app icon
//	MACGO_PROVISIONING_PROFILE  Path to provisioning profile to embed
//	MACGO_RESET_PERMISSIONS   Reset TCC permissions before requesting (set to "1")
//
// Development:
//
//	MACGO_DEV_MODE            Dev mode: signed wrapper exec's the source binary (set to "1")
//	MACGO_TTY_PASSTHROUGH     Pass TTY device to child process (experimental, set to "1")
//
// Internal (set by macgo, not typically set by users):
//
//	MACGO_STDIN_PIPE          Path to stdin named pipe (child reads from parent)
//	MACGO_STDOUT_PIPE         Path to stdout named pipe (child writes to parent)
//	MACGO_STDERR_PIPE         Path to stderr named pipe (child writes to parent)
//	MACGO_DONE_FILE           Path to sentinel file signaling child exit
//	MACGO_CWD                 Original working directory to restore in child
//	MACGO_BUNDLE_PATH         Path to the .app bundle (for config file matching)
//	MACGO_ORIGINAL_EXECUTABLE Path to the original binary before bundle copy
//	MACGO_SINGLE_PROCESS_ACTIVE  Sentinel: set to "1" after single-process re-exec
//
// # Bundle Structure
//
// macgo creates a standard macOS app bundle:
//
//	MyApp.app/
//	├── Contents/
//	│   ├── Info.plist
//	│   ├── MacOS/
//	│   │   └── MyApp
//	│   ├── Resources/
//	│   └── _CodeSignature/
//
// # Bundle ID Generation
//
// macgo generates bundle IDs from your Go module path:
//
//	github.com/user/project     → com.github.user.project.appname
//	gitlab.com/org/tool         → com.gitlab.org.tool.appname
//	example.com/app             → com.example.app.appname
//	Private/custom modules      → io.username.appname
//
// # Auto Packages
//
// Import auto packages for pre-configured setups:
//
//	import (
//	    _ "github.com/tmc/macgo/auto/media"   // Camera + Microphone
//	    _ "github.com/tmc/macgo/auto/files"   // File access
//	    _ "github.com/tmc/macgo/auto/adhoc"   // Ad-hoc signing
//	    "github.com/tmc/macgo"
//	)
//
//	func main() {
//	    macgo.Request()
//	}
//
// # Cleanup
//
// When using macgo with I/O forwarding (the default), call Cleanup on exit:
//
//	func main() {
//	    defer macgo.Cleanup()
//	    err := macgo.Request(macgo.Camera)
//	    // ...
//	}
//
// Signal handlers (SIGINT, SIGTERM) call cleanup automatically, but an explicit
// defer ensures proper behavior on normal exit.
//
// # TCC Integration
//
// macgo integrates with macOS TCC (Transparency, Consent, and Control) to:
//   - Generate proper entitlements for requested permissions
//   - Handle permission prompts automatically
//   - Reset permissions for testing (when requested)
//   - Support both sandboxed and non-sandboxed applications
package macgo
