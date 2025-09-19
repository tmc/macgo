// Package macgo provides simple macOS app bundle creation and TCC permission management.
//
// macgo enables Go applications to request macOS system permissions (camera, microphone,
// files, etc.) by automatically creating app bundles with proper entitlements and handling
// the relaunch process when necessary.
//
// # Basic Usage
//
// The simplest way to use macgo is with the Request function:
//
//	err := macgo.Request(macgo.Camera, macgo.Microphone)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Advanced Configuration
//
// For more control, use the Config struct:
//
//	cfg := macgo.NewConfig().
//	    WithAppName("MyApp").
//	    WithPermissions(macgo.Camera, macgo.Files).
//	    WithDebug()
//	err := macgo.Start(cfg)
//
// # TCC Database Access
//
// macgo now provides comprehensive TCC (Transparency, Consent, and Control) database
// access for querying and managing permissions:
//
//	// Check if we have Full Disk Access
//	if macgo.CheckFullDiskAccess() {
//	    // Open TCC database for reading
//	    db, err := macgo.OpenTCCDatabase()
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    defer db.Close()
//
//	    // List all permissions
//	    entries, err := db.ListAllPermissions()
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//
//	    // Format and display
//	    output, _ := macgo.FormatTCCEntries(entries, "table")
//	    fmt.Println(output)
//	}
//
// # Permission Queries
//
// Simple permission queries without SQL:
//
//	pq, err := macgo.NewPermissionQuery()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	if pq.HasCameraAccess() {
//	    fmt.Println("Camera access granted")
//	}
//
//	if pq.HasMicrophoneAccess() {
//	    fmt.Println("Microphone access granted")
//	}
//
//	// Get all permissions at once
//	status, err := macgo.GetAllPermissions()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Permissions: Camera=%v, Mic=%v, FDA=%v\n",
//	    status.Camera, status.Microphone, status.FullDiskAccess)
//
// # Waiting for Permissions
//
// You can wait for permissions to be granted:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	// Wait for Full Disk Access
//	err := macgo.WaitForFullDiskAccess(ctx, 1*time.Second)
//	if err != nil {
//	    log.Fatal("FDA not granted:", err)
//	}
//
//	// Wait for a specific permission
//	err = macgo.WaitForPermission(macgo.Camera, 30*time.Second)
//	if err != nil {
//	    log.Fatal("Camera permission not granted:", err)
//	}
//
// # Resetting Permissions
//
// Reset TCC permissions for testing:
//
//	// Reset all permissions for current app
//	err := macgo.ResetPermissions()
//
//	// Reset specific service
//	err := macgo.ResetServicePermission("camera")
//
// # Available Permissions
//
// Core permissions:
//   - Camera: Camera access
//   - Microphone: Microphone access
//   - Location: Location services
//   - Files: File system access
//   - Network: Network access
//   - Sandbox: App sandbox
//
// # Environment Variables
//
// macgo can be configured via environment variables:
//   - MACGO_APP_NAME: Application name
//   - MACGO_BUNDLE_ID: Bundle identifier
//   - MACGO_DEBUG=1: Enable debug logging
//   - MACGO_KEEP_BUNDLE=1: Preserve bundle after execution
//   - MACGO_AUTO_SIGN=1: Enable automatic code signing
//   - MACGO_CAMERA=1: Request camera permission
//   - MACGO_MICROPHONE=1: Request microphone permission
//   - MACGO_FILES=1: Request file access permission
//
// # Bundle ID Generation
//
// macgo automatically generates meaningful bundle IDs from your Go module:
//   - github.com/user/repo → com.github.user.repo.appname
//   - gitlab.com/company/tool → com.gitlab.company.tool.appname
//   - example.com/service → com.example.service.appname
//
// This provides unique, meaningful identifiers instead of generic "com.macgo" prefixes.
//
// # TCC Services
//
// Common TCC service identifiers:
//   - kTCCServiceCamera: Camera access
//   - kTCCServiceMicrophone: Microphone access
//   - kTCCServiceScreenCapture: Screen recording
//   - kTCCServiceSystemPolicyAllFiles: Full Disk Access
//   - kTCCServiceAddressBook: Contacts access
//   - kTCCServiceCalendar: Calendar access
//   - kTCCServicePhotos: Photos library access
//
// Use macgo.KnownTCCServices() to get a full list with descriptions.
package macgo