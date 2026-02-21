package launch

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// singleProcessSentinel is the environment variable that indicates
// the process has already been re-exec'd with new entitlements.
const singleProcessSentinel = "MACGO_SINGLE_PROCESS_ACTIVE"

// SingleProcessLauncher implements single-process mode via codesign + re-exec + setActivationPolicy.
//
// Instead of creating a .app bundle, this launcher:
//  1. Writes an entitlements plist to a temp file
//  2. Codesigns the current binary in-place with the requested entitlements
//  3. Re-execs itself so the kernel picks up the new code signature
//  4. After re-exec, calls NSApplication setActivationPolicy to become a foreground app
//
// IMPORTANT: syscall.Exec must happen BEFORE any NSApplication/GUI initialization.
// After GUI init, exec hangs due to stale Mach ports from the window server.
type SingleProcessLauncher struct {
	logger *Logger
}

// Launch implements the Launcher interface for single-process mode.
//
// The bundlePath argument is unused since no bundle is created.
// The execPath is the binary to codesign and re-exec.
func (t *SingleProcessLauncher) Launch(ctx context.Context, bundlePath, execPath string, cfg *Config) error {
	if t.logger == nil {
		t.logger = NewLogger()
	}

	// Check if we've already been re-exec'd with new entitlements.
	if os.Getenv(singleProcessSentinel) == "1" {
		t.logger.Debug("already re-exec'd, activating app")
		return t.activate(cfg)
	}

	// Phase 1: codesign the binary and re-exec.
	return t.codesignAndReexec(execPath, cfg)
}

// codesignAndReexec signs the current binary with entitlements and re-execs.
func (t *SingleProcessLauncher) codesignAndReexec(execPath string, cfg *Config) error {
	// Resolve the real executable path (follow symlinks)
	realExec, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}

	// Write entitlements to a temp file
	entPath, err := t.writeEntitlements(cfg)
	if err != nil {
		return fmt.Errorf("write entitlements: %w", err)
	}
	defer os.Remove(entPath)

	// Codesign the binary in-place with ad-hoc signature and entitlements
	if err := t.codesign(realExec, entPath, cfg); err != nil {
		return fmt.Errorf("codesign: %w", err)
	}

	// Set sentinel so the re-exec'd process knows it's been re-exec'd
	os.Setenv(singleProcessSentinel, "1")

	t.logger.Debug("re-exec'ing with new entitlements", "exec", realExec)

	// Re-exec before any GUI initialization.
	// Using syscall.Exec here is safe because we haven't touched NSApplication yet.
	return syscall.Exec(realExec, os.Args, os.Environ())
}

// writeEntitlements creates a temporary entitlements plist file.
func (t *SingleProcessLauncher) writeEntitlements(cfg *Config) (string, error) {
	var entries []string
	for _, ent := range cfg.Entitlements {
		entries = append(entries, fmt.Sprintf("\t\t<key>%s</key>\n\t\t<true/>", ent))
	}

	// Also map standard permissions to entitlement keys
	for _, perm := range cfg.Permissions {
		if ent := permissionToEntitlement(perm); ent != "" {
			entries = append(entries, fmt.Sprintf("\t\t<key>%s</key>\n\t\t<true/>", ent))
		}
	}

	if len(entries) == 0 {
		// Even with no entitlements, we need a valid plist for codesign
		entries = append(entries, "\t\t<key>com.apple.security.get-task-allow</key>\n\t\t<true/>")
	}

	content := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
	<dict>
%s
	</dict>
</plist>
`, strings.Join(entries, "\n"))

	f, err := os.CreateTemp("", "macgo-entitlements-*.plist")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", fmt.Errorf("write entitlements: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", fmt.Errorf("close entitlements: %w", err)
	}

	if cfg.Debug {
		t.logger.Debug("wrote entitlements", "path", f.Name(), "count", len(entries))
	}
	return f.Name(), nil
}

// codesign runs codesign on the binary with the given entitlements.
func (t *SingleProcessLauncher) codesign(binaryPath, entitlementsPath string, cfg *Config) error {
	args := []string{
		"--sign", "-", // ad-hoc
		"--force",
		"--entitlements", entitlementsPath,
		binaryPath,
	}

	if cfg.Debug {
		t.logger.Debug("codesigning", "cmd", "codesign "+strings.Join(args, " "))
	}

	cmd := exec.Command("codesign", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("codesign failed: %w\noutput: %s", err, string(output))
	}

	if cfg.Debug && len(output) > 0 {
		t.logger.Debug("codesign output", "output", string(output))
	}
	return nil
}

// activate calls NSApplication setActivationPolicy and activateIgnoringOtherApps
// to make the process appear as a foreground app with menu bar.
func (t *SingleProcessLauncher) activate(cfg *Config) error {
	uiMode := cfg.UIMode
	if uiMode == "" {
		uiMode = "regular"
	}

	// Determine activation policy
	// 0 = NSApplicationActivationPolicyRegular
	// 1 = NSApplicationActivationPolicyAccessory
	// 2 = NSApplicationActivationPolicyProhibited
	var policy int
	switch uiMode {
	case "regular":
		policy = 0
	case "accessory":
		policy = 1
	case "background":
		// No GUI activation needed for background mode
		t.logger.Debug("single-process mode: background, skipping activation")
		return nil
	default:
		policy = 0
	}

	// Load AppKit framework
	_, err := purego.Dlopen("/System/Library/Frameworks/AppKit.framework/AppKit", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		return fmt.Errorf("load AppKit: %w", err)
	}

	clsNSApp := objc.GetClass("NSApplication")
	if clsNSApp == 0 {
		return fmt.Errorf("failed to get NSApplication class")
	}

	selSharedApp := objc.RegisterName("sharedApplication")
	selSetPolicy := objc.RegisterName("setActivationPolicy:")
	selActivate := objc.RegisterName("activateIgnoringOtherApps:")

	// Get shared application instance
	app := objc.ID(clsNSApp).Send(selSharedApp)
	if app == 0 {
		return fmt.Errorf("failed to get shared NSApplication")
	}

	// Set activation policy
	ok := app.Send(selSetPolicy, policy)
	if ok == 0 {
		t.logger.Warn("setActivationPolicy returned NO", "policy", policy)
	} else {
		t.logger.Debug("set activation policy", "policy", policy, "mode", uiMode)
	}

	// Activate the app (bring to foreground, show in menu bar)
	if policy == 0 {
		app.Send(selActivate, true)
		t.logger.Debug("activated app (ignoring other apps)")
	}

	// Set Dock icon if provided and in regular mode
	if cfg.IconPath != "" && policy == 0 {
		if err := t.setDockIcon(app, cfg.IconPath); err != nil {
			t.logger.Warn("failed to set dock icon", "error", err, "path", cfg.IconPath)
		}
	}

	return nil
}

// setDockIcon loads an image file and sets it as the application's Dock icon.
func (t *SingleProcessLauncher) setDockIcon(app objc.ID, iconPath string) error {
	clsNSImage := objc.GetClass("NSImage")
	if clsNSImage == 0 {
		return fmt.Errorf("failed to get NSImage class")
	}

	clsNSString := objc.GetClass("NSString")
	if clsNSString == 0 {
		return fmt.Errorf("failed to get NSString class")
	}

	selStringWithUTF8 := objc.RegisterName("stringWithUTF8String:")
	selInitWithContentsOfFile := objc.RegisterName("initWithContentsOfFile:")
	selAlloc := objc.RegisterName("alloc")
	selSetAppIcon := objc.RegisterName("setApplicationIconImage:")

	// Create NSString path
	pathStr := objc.ID(clsNSString).Send(selStringWithUTF8, iconPath)
	if pathStr == 0 {
		return fmt.Errorf("failed to create NSString for path")
	}

	// Create NSImage from file
	img := objc.ID(clsNSImage).Send(selAlloc)
	img = img.Send(selInitWithContentsOfFile, pathStr)
	if img == 0 {
		return fmt.Errorf("failed to load image from %s", iconPath)
	}

	// Set as app icon
	app.Send(selSetAppIcon, img)
	t.logger.Debug("set dock icon", "path", iconPath)
	return nil
}

// permissionToEntitlement maps a permission string to its entitlement key.
func permissionToEntitlement(perm string) string {
	switch perm {
	case "camera":
		return "com.apple.security.device.camera"
	case "microphone":
		return "com.apple.security.device.microphone"
	case "location":
		return "com.apple.security.personal-information.location"
	case "sandbox":
		return "com.apple.security.app-sandbox"
	case "files":
		return "com.apple.security.files.user-selected.read-only"
	case "network":
		return "com.apple.security.network.client"
	case "screen-recording":
		return "com.apple.security.screen-capture"
	case "accessibility":
		return "com.apple.security.accessibility"
	default:
		return ""
	}
}
