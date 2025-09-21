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
//	    WithAdHocSign().  // Sign for development
//	    WithDebug()       // Enable debug output
//
//	err := macgo.Start(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Available Permissions
//
// Core permissions that cover most use cases:
//
//	Camera     - Camera access (AVCaptureDevice)
//	Microphone - Microphone access (AVAudioSession)
//	Location   - Location services (CoreLocation)
//	Files      - File system access with user selection
//	Network    - Network client and server connections
//	Sandbox    - App sandbox with restricted file access
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
//	cfg := macgo.NewConfig().WithSigningIdentity("Developer ID Application: Your Name")
//
// # Environment Variables
//
// Configure macgo via environment variables:
//
//	MACGO_APP_NAME=MyApp              # Application name
//	MACGO_APP_NAME_PREFIX=Dev-        # Prefix for app names (e.g., "Dev-MyApp")
//	MACGO_BUNDLE_ID=com.example.app   # Bundle identifier
//	MACGO_BUNDLE_ID_PREFIX=test.      # Prefix for bundle IDs
//	MACGO_DEBUG=1                      # Enable debug output
//	MACGO_KEEP_BUNDLE=1                # Keep bundle after execution
//	MACGO_NO_RELAUNCH=1                # Disable automatic relaunch
//	MACGO_AD_HOC_SIGN=1                # Enable ad-hoc signing
//	MACGO_AUTO_SIGN=1                  # Auto-detect signing identity
//	MACGO_RESET_PERMISSIONS=1          # Reset permissions before requesting
//
// Permission-specific variables:
//
//	MACGO_CAMERA=1                     # Request camera permission
//	MACGO_MICROPHONE=1                 # Request microphone permission
//	MACGO_LOCATION=1                   # Request location permission
//	MACGO_FILES=1                      # Request file access permission
//	MACGO_NETWORK=1                    # Request network permission
//	MACGO_SANDBOX=1                    # Enable app sandbox
//
// # Bundle Structure
//
// macgo creates a standard macOS app bundle:
//
//	MyApp.app/
//	├── Contents/
//	│   ├── Info.plist          # App metadata and configuration
//	│   ├── MacOS/
//	│   │   └── MyApp           # Executable (your Go binary)
//	│   ├── Resources/          # Icons and other resources
//	│   └── _CodeSignature/     # Code signing data
//
// # Bundle ID Generation
//
// macgo intelligently generates bundle IDs from your Go module path:
//
//	github.com/user/project     → com.github.user.project.appname
//	gitlab.com/org/tool         → com.gitlab.org.tool.appname
//	example.com/app             → com.example.app.appname
//	Private/custom modules      → io.username.appname (using system username)
//
// This ensures unique, meaningful identifiers for your applications.
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
//	    // Permissions and signing are pre-configured
//	    macgo.Request()
//	}
//
// # TCC Integration
//
// macgo integrates with macOS TCC (Transparency, Consent, and Control) to:
//   - Generate proper entitlements for requested permissions
//   - Handle permission prompts automatically
//   - Reset permissions for testing (when requested)
//   - Support both sandboxed and non-sandboxed applications
//
// # Examples
//
// See the examples directory for complete applications demonstrating:
//   - Basic permission requests
//   - Camera and microphone access
//   - File system access
//   - Code signing workflows
//   - Sandboxed applications
//   - Custom bundle configuration
package macgo
