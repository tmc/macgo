// Package main provides a TCC (Transparency, Consent, and Control) helper tool
// that assists users in managing macOS privacy permissions through System Settings.
//
// This tool DOES NOT bypass TCC - it helps automate the UI navigation to make
// granting permissions easier during testing and development.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/tmc/macgo"
)

var (
	service     = flag.String("service", "", "TCC service (screen-recording, accessibility, automation, etc)")
	bundleID    = flag.String("bundle", "", "Bundle ID to grant permission to")
	action      = flag.String("action", "open", "Action: open (System Settings), check, prompt, automate, test-access, revoke, inspect-ui")
	appName     = flag.String("app", "", "Application name for automation (e.g., 'screen-capture')")
	appPath     = flag.String("path", "", "Full path to application bundle (e.g., '/Applications/MyApp.app')")
	list        = flag.Bool("list", false, "List common TCC services")
	interactive = flag.Bool("interactive", false, "Interactive mode with step-by-step guidance")
)

// Common TCC services and their System Settings paths
var tccServices = map[string]struct {
	Name        string
	Pane        string
	Description string
}{
	"screen-recording": {
		Name:        "Screen Recording",
		Pane:        "x-apple.systempreferences:com.apple.preference.security?Privacy_ScreenCapture",
		Description: "Allows apps to capture screen contents",
	},
	"accessibility": {
		Name:        "Accessibility",
		Pane:        "x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility",
		Description: "Allows apps to control your computer",
	},
	"automation": {
		Name:        "Automation",
		Pane:        "x-apple.systempreferences:com.apple.preference.security?Privacy_Automation",
		Description: "Allows apps to control other apps",
	},
	"camera": {
		Name:        "Camera",
		Pane:        "x-apple.systempreferences:com.apple.preference.security?Privacy_Camera",
		Description: "Allows apps to access camera",
	},
	"microphone": {
		Name:        "Microphone",
		Pane:        "x-apple.systempreferences:com.apple.preference.security?Privacy_Microphone",
		Description: "Allows apps to access microphone",
	},
	"full-disk-access": {
		Name:        "Full Disk Access",
		Pane:        "x-apple.systempreferences:com.apple.preference.security?Privacy_AllFiles",
		Description: "Allows apps to access all files",
	},
}

func init() {
	// We need automation permission to open System Settings
	// And accessibility permission for UI automation
	if os.Getenv("MACGO_SERVICES_VERSION") == "" {
		os.Setenv("MACGO_SERVICES_VERSION", "1") // Use stable V1
	}

	cfg := &macgo.Config{
		// Use a stable bundle ID so the app identity persists across builds
		BundleID: "com.github.tmc.macgo.tcc-helper",

		Permissions: []macgo.Permission{
			macgo.Files,
		},
		// Request Accessibility via custom entitlement for UI automation
		Custom: []string{
			"com.apple.security.automation.apple-events",
		},
		Debug: os.Getenv("MACGO_DEBUG") == "1",

		// Enable automatic signing with Developer ID if available
		// Falls back to stable ad-hoc signing if no Developer ID is found
		AutoSign: true,

		// Enable ad-hoc signing as fallback when no Developer ID is available
		// This provides a stable signature using the fixed identifier below
		AdHocSign: true,

		// Use stable identifier for code signing
		// This ensures the signature is consistent across rebuilds
		CodeSigningIdentifier: "com.github.tmc.macgo.tcc-helper",
	}

	if err := macgo.Start(cfg); err != nil {
		log.Fatalf("Failed to start macgo: %v", err)
	}
}

func main() {
	flag.Parse()

	if *list {
		listServices()
		return
	}

	// test-access doesn't require a service
	if *action == "test-access" {
		testAccessibilityPermission()
		return
	}

	if *service == "" {
		fmt.Fprintln(os.Stderr, "Usage: tcc-helper -service <service> [-bundle <bundle-id>] [-action open|check|prompt|automate]")
		fmt.Fprintln(os.Stderr, "       tcc-helper -action test-access")
		fmt.Fprintln(os.Stderr, "\nUse -list to see available services")
		os.Exit(1)
	}

	svc, ok := tccServices[*service]
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown service: %s\n", *service)
		fmt.Fprintln(os.Stderr, "Use -list to see available services")
		os.Exit(1)
	}

	switch *action {
	case "open":
		openSystemSettings(svc)
	case "check":
		checkPermission(svc, *bundleID)
	case "prompt":
		promptForPermission(svc, *bundleID)
	case "automate":
		// Use -path if provided, otherwise use -app
		appNameOrPath := *appPath
		if appNameOrPath == "" {
			appNameOrPath = *appName
		}
		automatePermissionGrant(svc, appNameOrPath)
	case "test-access":
		testAccessibilityPermission()
	case "revoke":
		revokePermission(svc, *appName)
	case "inspect-ui":
		if err := InspectSystemSettingsUI(*service); err != nil {
			fmt.Fprintf(os.Stderr, "UI inspection failed: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown action: %s\n", *action)
		os.Exit(1)
	}
}

func listServices() {
	fmt.Println("Available TCC Services:")
	fmt.Println()
	for key, svc := range tccServices {
		fmt.Printf("  %-20s %s\n", key, svc.Name)
		fmt.Printf("  %-20s %s\n", "", svc.Description)
		fmt.Println()
	}
	fmt.Println("Usage:")
	fmt.Println("  tcc-helper -service screen-recording -action open")
	fmt.Println("  tcc-helper -service accessibility -bundle com.example.app -action prompt")
	fmt.Println("  tcc-helper -service screen-recording -action automate -app screen-capture")
	fmt.Println("  tcc-helper -service screen-recording -action automate -path /Applications/MyApp.app")
	fmt.Println("  tcc-helper -service screen-recording -action revoke -app ScreenCaptureKit-Example")
	fmt.Println("  tcc-helper -service screen-recording -action inspect-ui")
}

func openSystemSettings(svc struct {
	Name        string
	Pane        string
	Description string
}) {
	fmt.Printf("Opening System Settings: %s\n", svc.Name)
	fmt.Printf("Description: %s\n\n", svc.Description)

	// Use 'open' command to open System Settings to specific pane
	cmd := exec.Command("open", svc.Pane)
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to open System Settings: %v", err)
	}

	fmt.Println("System Settings opened!")
	fmt.Println()
	fmt.Println("To grant permission:")
	fmt.Println("  1. Click the lock icon (🔒) to unlock")
	fmt.Println("  2. Click the '+' button to add an application")
	fmt.Println("  3. Navigate to and select your application")
	fmt.Println("  4. Click 'Open' to grant permission")
	fmt.Println("  5. Lock the settings again (optional)")
}

func checkPermission(svc struct {
	Name        string
	Pane        string
	Description string
}, bundleID string) {
	if bundleID == "" {
		fmt.Fprintln(os.Stderr, "Error: -bundle required for check action")
		os.Exit(1)
	}

	fmt.Printf("Checking %s permission for: %s\n", svc.Name, bundleID)
	fmt.Println()
	fmt.Println("Note: There is no official API to check TCC permissions.")
	fmt.Println("The app must attempt to use the permission and handle denial.")
	fmt.Println()
	fmt.Println("You can check manually in System Settings:")
	openSystemSettings(svc)
}

func promptForPermission(svc struct {
	Name        string
	Pane        string
	Description string
}, bundleID string) {
	fmt.Printf("Permission Required: %s\n", svc.Name)
	fmt.Printf("Description: %s\n", svc.Description)
	if bundleID != "" {
		fmt.Printf("For bundle: %s\n", bundleID)
	}
	fmt.Println()
	fmt.Println("This tool will open System Settings to help you grant the permission.")
	fmt.Println("Press Enter to continue, or Ctrl+C to cancel...")

	// Wait for user confirmation
	fmt.Scanln()

	openSystemSettings(svc)

	fmt.Println()
	fmt.Println("Waiting 5 seconds for you to grant permission...")
	time.Sleep(5 * time.Second)

	fmt.Println()
	fmt.Println("Please verify the permission was granted in System Settings.")
}

func testAccessibilityPermission() {
	fmt.Println("Testing tcc-helper accessibility permission...")
	fmt.Println()

	hasAccess, err := CheckAccessibilityPermission()

	if hasAccess {
		fmt.Println("✓ tcc-helper HAS Accessibility permission")
		fmt.Println("  UI automation should work")
		fmt.Println()
		fmt.Println("You can now use:")
		fmt.Println("  - tcc-helper -service <service> -action automate -app <app-name>")
		fmt.Println("  - tcc-helper -service <service> -action revoke -app <app-name>")
	} else {
		fmt.Println("❌ tcc-helper does NOT have Accessibility permission")
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
		}
		fmt.Println()
		fmt.Println("To grant permission:")
		fmt.Println("  1. Run: tcc-helper -service accessibility -action open")
		fmt.Println("  2. Look for 'tcc-helper' in the list")
		fmt.Println("  3. Check the box next to tcc-helper")
		fmt.Println()
		fmt.Println("Opening Accessibility settings now...")
		openSystemSettings(tccServices["accessibility"])
		os.Exit(1)
	}
}

func automatePermissionGrant(svc struct {
	Name        string
	Pane        string
	Description string
}, appNameOrPath string) {
	if appNameOrPath == "" {
		fmt.Fprintln(os.Stderr, "Error: -app or -path required for automate action")
		fmt.Fprintln(os.Stderr, "Examples:")
		fmt.Fprintln(os.Stderr, "  tcc-helper -service screen-recording -action automate -app screen-capture")
		fmt.Fprintln(os.Stderr, "  tcc-helper -service screen-recording -action automate -path /Applications/MyApp.app")
		os.Exit(1)
	}

	fmt.Printf("Attempting to automate permission grant for: %s\n", appNameOrPath)
	fmt.Printf("Service: %s\n", svc.Name)
	fmt.Println()

	// Check if we have accessibility permission
	fmt.Println("Checking if tcc-helper has Accessibility permission...")
	hasAccess, err := CheckAccessibilityPermission()
	if !hasAccess {
		fmt.Println()
		fmt.Println("❌ tcc-helper does not have Accessibility permission")
		if err != nil {
			fmt.Printf("   Error: %v\n", err)
		}
		fmt.Println()
		fmt.Println("To use UI automation, you must grant Accessibility permission:")
		fmt.Println("  1. System Settings should open automatically to Accessibility")
		fmt.Println("  2. Click the lock icon to unlock (requires authentication)")
		fmt.Println("  3. Click the '+' button to add an application")
		fmt.Println("  4. Press Cmd+Shift+G and type: /usr/bin/osascript")
		fmt.Println("  5. Click 'Open' and ensure the checkbox is checked")
		fmt.Println("  6. Re-run this command")
		fmt.Println()
		fmt.Println("Opening Accessibility settings now...")
		openSystemSettings(tccServices["accessibility"])
		os.Exit(1)
	}

	fmt.Println("✓ tcc-helper has Accessibility permission")
	fmt.Println()

	// Attempt UI automation
	if err := GrantPermissionWithUI(*service, appNameOrPath); err != nil {
		fmt.Fprintf(os.Stderr, "Automation failed: %v\n", err)
		fmt.Println()
		fmt.Println("Falling back to manual instructions...")
		openSystemSettings(svc)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("✓ UI automation completed")
	fmt.Println("Please verify the permission was granted in System Settings.")
}

func revokePermission(svc struct {
	Name        string
	Pane        string
	Description string
}, appName string) {
	if appName == "" {
		fmt.Fprintln(os.Stderr, "Error: -app required for revoke action")
		fmt.Fprintln(os.Stderr, "Example: tcc-helper -service screen-recording -action revoke -app ScreenCaptureKit-Example")
		os.Exit(1)
	}

	fmt.Printf("Attempting to revoke %s permission for: %s\n", svc.Name, appName)
	fmt.Println()

	// Check if we have accessibility permission
	fmt.Println("Checking if tcc-helper has Accessibility permission...")
	hasAccess, err := CheckAccessibilityPermission()
	if !hasAccess {
		fmt.Println()
		fmt.Println("❌ tcc-helper does not have Accessibility permission")
		if err != nil {
			fmt.Printf("   Error: %v\n", err)
		}
		fmt.Println()
		fmt.Println("To use UI automation, you must grant Accessibility permission:")
		fmt.Println("  1. System Settings should open automatically to Accessibility")
		fmt.Println("  2. Click the lock icon to unlock (requires authentication)")
		fmt.Println("  3. Click the '+' button to add an application")
		fmt.Println("  4. Press Cmd+Shift+G and type: /usr/bin/osascript")
		fmt.Println("  5. Click 'Open' and ensure the checkbox is checked")
		fmt.Println("  6. Re-run this command")
		fmt.Println()
		fmt.Println("Opening Accessibility settings now...")
		openSystemSettings(tccServices["accessibility"])
		os.Exit(1)
	}

	fmt.Println("✓ tcc-helper has Accessibility permission")
	fmt.Println()

	// Attempt UI automation to remove the app
	if err := RevokePermissionWithUI(*service, appName); err != nil{
		fmt.Fprintf(os.Stderr, "Automation failed: %v\n", err)
		fmt.Println()
		fmt.Println("Falling back to manual instructions...")
		openSystemSettings(svc)
		fmt.Println()
		fmt.Println("To revoke permission manually:")
		fmt.Println("  1. Click the lock icon (🔒) to unlock")
		fmt.Println("  2. Find and select the application in the list")
		fmt.Println("  3. Click the '-' button to remove")
		fmt.Println("  4. Confirm removal")
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("✓ Permission revocation completed")
	fmt.Println("Please verify the application was removed from System Settings.")
}
