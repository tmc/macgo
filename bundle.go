package macgo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Cleanup manager for handling temporary resources safely
type cleanupManager struct {
	mu       sync.Mutex
	cleanups map[string]cleanupEntry
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

type cleanupEntry struct {
	path      string
	cleanupAt time.Time
	isDir     bool
}

var globalCleanupManager *cleanupManager
var cleanupOnce sync.Once

// initCleanupManager initializes the global cleanup manager
func initCleanupManager() {
	cleanupOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		globalCleanupManager = &cleanupManager{
			cleanups: make(map[string]cleanupEntry),
			ctx:      ctx,
			cancel:   cancel,
		}
		globalCleanupManager.start()
	})
}

// start begins the cleanup manager background process
func (cm *cleanupManager) start() {
	cm.wg.Add(1)
	go func() {
		defer cm.wg.Done()
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-cm.ctx.Done():
				return
			case <-ticker.C:
				cm.performCleanup()
			}
		}
	}()
}

// scheduleCleanup schedules a path for cleanup after the specified duration
func (cm *cleanupManager) scheduleCleanup(path string, delay time.Duration, isDir bool) {
	if cm == nil {
		return
	}

	cleanPath, err := securePath(path)
	if err != nil {
		debugf("Failed to validate cleanup path %s: %v", path, err)
		return
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.cleanups[cleanPath] = cleanupEntry{
		path:      cleanPath,
		cleanupAt: time.Now().Add(delay),
		isDir:     isDir,
	}
	debugf("Scheduled cleanup for %s in %v", cleanPath, delay)
}

// performCleanup removes expired temporary files and directories
func (cm *cleanupManager) performCleanup() {
	cm.mu.Lock()
	toCleanup := make([]cleanupEntry, 0)
	now := time.Now()

	for path, entry := range cm.cleanups {
		if now.After(entry.cleanupAt) {
			toCleanup = append(toCleanup, entry)
			delete(cm.cleanups, path)
		}
	}
	cm.mu.Unlock()

	// Perform cleanup outside of lock to avoid blocking
	for _, entry := range toCleanup {
		cm.safeRemove(entry)
	}
}

// safeRemove safely removes a file or directory with proper error handling
func (cm *cleanupManager) safeRemove(entry cleanupEntry) {
	// Validate path again before removal for extra safety
	if _, err := securePath(entry.path); err != nil {
		debugf("Cleanup cancelled for invalid path %s: %v", entry.path, err)
		return
	}

	// Check if the path still exists and is within expected boundaries
	info, err := os.Stat(entry.path)
	if os.IsNotExist(err) {
		debugf("Cleanup target %s already removed", entry.path)
		return
	}
	if err != nil {
		debugf("Failed to stat cleanup target %s: %v", entry.path, err)
		return
	}

	// Additional safety check: ensure we're removing what we expect
	if entry.isDir && !info.IsDir() {
		debugf("Cleanup safety check failed: expected directory but found file at %s", entry.path)
		return
	}
	if !entry.isDir && info.IsDir() {
		debugf("Cleanup safety check failed: expected file but found directory at %s", entry.path)
		return
	}

	// Perform the actual removal
	var removeErr error
	if entry.isDir {
		removeErr = os.RemoveAll(entry.path)
	} else {
		removeErr = os.Remove(entry.path)
	}

	if removeErr != nil {
		debugf("Failed to clean up %s: %v", entry.path, removeErr)
	} else {
		debugf("Successfully cleaned up %s", entry.path)
	}
}

// shutdown gracefully shuts down the cleanup manager
func (cm *cleanupManager) shutdown() {
	if cm == nil {
		return
	}
	cm.cancel()
	cm.wg.Wait()
}

// Security functions to prevent path traversal vulnerabilities

// sanitizePath cleans and validates a file path to prevent directory traversal attacks.
// It returns an error if the path contains dangerous elements.
func sanitizePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// Clean the path to resolve any .. or . elements
	cleaned := filepath.Clean(path)

	// Check for path traversal attempts
	if strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("path traversal detected in: %s", path)
	}

	// Check for absolute paths that could escape intended boundaries
	if filepath.IsAbs(cleaned) && !isAllowedAbsolutePath(cleaned) {
		return "", fmt.Errorf("absolute path not allowed: %s", path)
	}

	// Prevent null bytes and other dangerous characters
	if strings.ContainsAny(cleaned, "\x00\r\n") {
		return "", fmt.Errorf("invalid characters in path: %s", path)
	}

	return cleaned, nil
}

// isAllowedAbsolutePath checks if an absolute path is within allowed directories.
func isAllowedAbsolutePath(path string) bool {
	allowedPrefixes := []string{
		"/tmp/",
		"/var/folders/", // macOS temp directories
		os.TempDir(),
	}

	// Allow GOPATH and its subdirectories
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		allowedPrefixes = append(allowedPrefixes, gopath)
	}

	// Allow user home directory and its subdirectories
	if home, err := os.UserHomeDir(); err == nil {
		allowedPrefixes = append(allowedPrefixes, home)
	}

	// Allow standard development directories and system binaries
	allowedPrefixes = append(allowedPrefixes, "/usr/local/", "/opt/", "/usr/bin/", "/bin/", "/System/")

	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

// securePath validates and secures a path for file operations.
// It combines path validation with additional security checks.
func securePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// First sanitize the path
	clean, err := sanitizePath(path)
	if err != nil {
		return "", fmt.Errorf("path sanitization failed: %w", err)
	}

	// Additional length check to prevent resource exhaustion
	if len(clean) > 4096 {
		return "", fmt.Errorf("path too long: %d characters", len(clean))
	}

	return clean, nil
}

// secureJoin safely joins path components while preventing traversal attacks.
func secureJoin(base string, elem ...string) (string, error) {
	// Validate base path
	cleanBase, err := securePath(base)
	if err != nil {
		return "", fmt.Errorf("invalid base path: %w", err)
	}

	// Validate and clean each element
	cleanElems := make([]string, len(elem))
	for i, e := range elem {
		// Don't allow absolute paths in elements
		if filepath.IsAbs(e) {
			return "", fmt.Errorf("absolute path not allowed in element: %s", e)
		}

		clean, err := sanitizePath(e)
		if err != nil {
			return "", fmt.Errorf("invalid path element %s: %w", e, err)
		}
		cleanElems[i] = clean
	}

	// Join all components
	result := filepath.Join(append([]string{cleanBase}, cleanElems...)...)

	// Final validation of the result
	final, err := securePath(result)
	if err != nil {
		return "", fmt.Errorf("final path validation failed: %w", err)
	}

	return final, nil
}

// validateExecutablePath validates that an executable path is safe to use.
func validateExecutablePath(execPath string) error {
	if execPath == "" {
		return fmt.Errorf("executable path cannot be empty")
	}

	// Clean and validate the path
	_, err := securePath(execPath)
	if err != nil {
		return fmt.Errorf("invalid executable path: %w", err)
	}

	// Check if the file exists and is executable
	info, err := os.Stat(execPath)
	if err != nil {
		return fmt.Errorf("executable not accessible: %w", err)
	}

	// Ensure it's a regular file
	if !info.Mode().IsRegular() {
		return fmt.Errorf("executable path is not a regular file: %s", execPath)
	}

	return nil
}

// createBundle creates an app bundle for an executable.
// It returns the path to the created or existing app bundle.
// If an error occurs during creation, it returns the error.
func createBundle(execPath string) (string, error) {
	// Validate executable path for security
	if err := validateExecutablePath(execPath); err != nil {
		return "", fmt.Errorf("security validation failed: %w", err)
	}

	// Get executable name and determine app name
	name := filepath.Base(execPath)
	appName := name
	if DefaultConfig.ApplicationName != "" {
		// Sanitize application name to prevent injection
		cleanAppName, err := sanitizePath(DefaultConfig.ApplicationName)
		if err != nil {
			return "", fmt.Errorf("invalid application name: %w", err)
		}
		appName = cleanAppName
	}

	// Check if using go run (temporary binary)
	isTemp := strings.Contains(execPath, "go-build")

	// Determine bundle location
	var dir, appPath string
	var fileHash string

	// Use custom path if specified
	if DefaultConfig.CustomDestinationAppPath != "" {
		// Validate and sanitize custom path
		cleanCustomPath, err := securePath(DefaultConfig.CustomDestinationAppPath)
		if err != nil {
			return "", fmt.Errorf("invalid custom destination path: %w", err)
		}
		appPath = cleanCustomPath
		dir = filepath.Dir(appPath)
	} else if isTemp {
		// For temporary binaries, use a system temp directory
		tmp, err := os.MkdirTemp("", "macgo-*")
		if err != nil {
			return "", fmt.Errorf("create temp dir for app bundle: %w", err)
		}

		// Create unique name with hash
		fileHash, err = checksum(execPath)
		if err != nil {
			debugf("Failed to calculate executable checksum: %v", err)
			// Fallback to timestamp if checksum fails
			fileHash = fmt.Sprintf("%d", time.Now().UnixNano())
		}
		shortHash := fileHash[:8]

		// Unique app name for temporary bundles
		appName = fmt.Sprintf("%s-%s", appName, shortHash)
		// Use secure path joining for app path
		appPath, err = secureJoin(tmp, appName+".app")
		if err != nil {
			return "", fmt.Errorf("failed to create secure app path: %w", err)
		}
		dir = tmp
	} else {
		// For regular binaries, use GOPATH/bin
		gopath := os.Getenv("GOPATH")
		if gopath == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("get home directory for app bundle: %w", err)
			}
			var joinErr error
			gopath, joinErr = secureJoin(home, "go")
			if joinErr != nil {
				return "", fmt.Errorf("failed to create secure GOPATH: %w", joinErr)
			}
		} else {
			// Validate GOPATH from environment
			var pathErr error
			gopath, pathErr = securePath(gopath)
			if pathErr != nil {
				return "", fmt.Errorf("invalid GOPATH from environment: %w", pathErr)
			}
		}

		var err error
		dir, err = secureJoin(gopath, "bin")
		if err != nil {
			return "", fmt.Errorf("failed to create secure bin directory path: %w", err)
		}
		appPath, err = secureJoin(dir, appName+".app")
		if err != nil {
			return "", fmt.Errorf("failed to create secure app path: %w", err)
		}

		// Check for existing bundle that's up to date
		if existing := checkExisting(appPath, execPath); existing {
			debugf("Using existing app bundle at: %s", appPath)
			return appPath, nil
		}
	}

	// Check developer environment for potential issues
	checkDeveloperEnvironment()

	// Create app bundle structure using secure path operations
	contentsPath, err := secureJoin(appPath, "Contents")
	if err != nil {
		return "", fmt.Errorf("failed to create secure Contents path: %w", err)
	}
	macosPath, err := secureJoin(contentsPath, "MacOS")
	if err != nil {
		return "", fmt.Errorf("failed to create secure MacOS path: %w", err)
	}

	if err := os.MkdirAll(macosPath, 0755); err != nil {
		return "", fmt.Errorf("create bundle directory structure: %w", err)
	}

	// Generate bundle ID
	bundleID := DefaultConfig.BundleID
	if bundleID == "" {
		// TODO: infer from go binary runtime package
		bundleID = fmt.Sprintf("com.macgo.%s", appName)
		if isTemp && len(fileHash) >= 8 {
			bundleID = fmt.Sprintf("com.macgo.%s.%s", appName, fileHash[:8])
		}
	}

	// Create Info.plist entries
	plist := map[string]any{
		"CFBundleExecutable":      name,
		"CFBundleIdentifier":      bundleID,
		"CFBundleName":            appName,
		"CFBundleIconFile":        "ExecutableBinaryIcon",
		"CFBundlePackageType":     "APPL",
		"CFBundleVersion":         "1.0",
		"NSHighResolutionCapable": true,
		// Set LSUIElement based on whether app should be visible in dock
		// If LSUIElement=true, app runs in background (no dock icon or menu)
		// If false, app appears in dock
		"LSUIElement": !DefaultConfig.Relaunch, // If relaunch is true, we want to be visible
	}

	// Add user-defined entries
	for k, v := range DefaultConfig.PlistEntries {
		plist[k] = v
	}

	// Write Info.plist using secure path
	infoPlistPath, err := secureJoin(contentsPath, "Info.plist")
	if err != nil {
		return "", fmt.Errorf("failed to create secure Info.plist path: %w", err)
	}
	if err := writePlist(infoPlistPath, plist); err != nil {
		return "", fmt.Errorf("write Info.plist file: %w", err)
	}

	// Write entitlements if any are enabled
	hasEnabledEntitlements := false
	entitlements := make(map[string]any)
	for k, v := range DefaultConfig.Entitlements {
		if v {
			entitlements[string(k)] = v
			hasEnabledEntitlements = true
		}
	}

	if hasEnabledEntitlements {
		entPath, err := secureJoin(contentsPath, "entitlements.plist")
		if err != nil {
			return "", fmt.Errorf("failed to create secure entitlements.plist path: %w", err)
		}
		if err := writePlist(entPath, entitlements); err != nil {
			return "", fmt.Errorf("write entitlements.plist file: %w", err)
		}
	}

	// Copy the executable using secure path
	bundleExecPath, err := secureJoin(macosPath, name)
	if err != nil {
		return "", fmt.Errorf("failed to create secure executable path: %w", err)
	}
	if err := copyFile(execPath, bundleExecPath); err != nil {
		return "", fmt.Errorf("copy executable to app bundle: %w", err)
	}

	// Attempt to copy in "ExecutableBinaryIcon.icns" if it exists:
	defaultIconPath := "/System/Library/CoreServices/CoreTypes.bundle/Contents/Resources/ExecutableBinaryIcon.icns"
	// Validate system icon path for security
	if cleanIconPath, iconErr := securePath(defaultIconPath); iconErr == nil {
		if _, err := os.Stat(cleanIconPath); err == nil {
			resourcesPath, pathErr := secureJoin(contentsPath, "Resources")
			if pathErr != nil {
				debugf("Failed to create secure Resources path: %v", pathErr)
			} else {
				iconPath, iconPathErr := secureJoin(resourcesPath, "ExecutableBinaryIcon.icns")
				if iconPathErr != nil {
					debugf("Failed to create secure icon path: %v", iconPathErr)
				} else {
					if err := os.MkdirAll(resourcesPath, 0755); err != nil {
						debugf("Failed to create Resources directory: %v", err)
					}
					if err := copyFile(cleanIconPath, iconPath); err != nil {
						debugf("Failed to copy default icon: %v", err)
					}
				}
			}
		}
	}

	// Make executable
	if err := os.Chmod(bundleExecPath, 0755); err != nil {
		return "", fmt.Errorf("set executable permissions: %w", err)
	}

	// Set cleanup for temporary bundles using secure cleanup manager
	if isTemp && !DefaultConfig.KeepTemp {
		debugf("Created temporary app bundle at: %s", appPath)
		initCleanupManager()
		globalCleanupManager.scheduleCleanup(dir, 30*time.Second, true)
	} else {
		debugf("Created app bundle at: %s", appPath)
	}

	// Auto-sign the bundle if requested
	if DefaultConfig.AutoSign {
		if err := signBundle(appPath); err != nil {
			// Log the error but continue - signing is optional for functionality
			debugf("Warning: Error signing bundle: %v", err)
		}
	}

	return appPath, nil
}

// checkExisting checks if an existing app bundle is up to date.
// Returns true if the bundle exists and is up to date.
// Returns false if the bundle doesn't exist or if the binary has changed, so a new bundle should be created.
func checkExisting(appPath, execPath string) bool {
	name := filepath.Base(execPath)
	bundleExecPath := filepath.Join(appPath, "Contents", "MacOS", name)

	// Check if the app bundle exists
	if _, err := os.Stat(appPath); err != nil {
		debugf("App bundle does not exist at: %s", appPath)
		return false
	}

	// Check if the executable exists in the bundle
	if _, err := os.Stat(bundleExecPath); err != nil {
		debugf("Executable does not exist in app bundle: %s", bundleExecPath)
		return false
	}

	// Compare checksums
	srcHash, err := checksum(execPath)
	if err != nil {
		debugf("Error calculating source checksum: %v", err)
		return false
	}

	bundleHash, err := checksum(bundleExecPath)
	if err != nil {
		debugf("Error calculating bundle checksum: %v", err)
		return false
	}

	if srcHash == bundleHash {
		debugf("App bundle is up to date")
		return true
	}

	debugf("Binary changed - will create new app bundle with potentially updated entitlements")
	// Remove the old bundle entirely to ensure all contents are updated
	// Use secure removal with proper validation
	if cleanAppPath, pathErr := securePath(appPath); pathErr != nil {
		debugf("Invalid app path for removal: %v", pathErr)
	} else {
		// Verify this is actually an app bundle before removing
		if strings.HasSuffix(cleanAppPath, ".app") {
			if err := os.RemoveAll(cleanAppPath); err != nil {
				debugf("Error removing old app bundle: %v", err)
			} else {
				debugf("Successfully removed old app bundle: %s", cleanAppPath)
			}
		} else {
			debugf("Skipping removal of non-app bundle path: %s", cleanAppPath)
		}
	}

	return false
}

// relaunch restarts the application through the app bundle.
func relaunch(appPath, execPath string) {
	// Create pipes for IO redirection
	pipes := make([]string, 3)
	initCleanupManager() // Ensure cleanup manager is initialized

	for i, name := range []string{"stdin", "stdout", "stderr"} {
		pipe, err := createPipe("macgo-" + name)
		if err != nil {
			debugf("error creating %s pipe: %v", name, err)
			return
		}
		pipes[i] = pipe

		// Schedule secure cleanup for pipe and its parent directory
		globalCleanupManager.scheduleCleanup(pipe, 5*time.Minute, false)
		pipeDir := filepath.Dir(pipe)
		globalCleanupManager.scheduleCleanup(pipeDir, 6*time.Minute, true)
	}

	// Prepare open command arguments
	args := []string{
		"-a", appPath,
		"--wait-apps",
		"--stdin", pipes[0],
		"--stdout", pipes[1],
		"--stderr", pipes[2],
	}

	// Set environment to prevent relaunching again
	os.Setenv("MACGO_NO_RELAUNCH", "1")

	// Pass original arguments
	if len(os.Args) > 1 {
		args = append(args, "--args")
		args = append(args, os.Args[1:]...)
	}

	// Launch app bundle
	cmd := exec.Command("open", args...)

	// Set process group ID to match the parent process
	// This ensures proper signal handling between parent and child
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0, // Use the parent's process group
	}

	if err := cmd.Start(); err != nil {
		debugf("error starting app bundle: %v", err)
		return
	}

	// Set up signal forwarding from parent to child process group
	forwardSignals(cmd.Process.Pid)

	// Create debug log files for stdout/stderr if debug is enabled
	var stdoutTee, stderrTee io.Writer = os.Stdout, os.Stderr
	debugf("Setting up IO redirection (debug enabled: %t)", isDebugEnabled())
	if isDebugEnabled() {
		if stdoutFile, err := createDebugLogFile("stdout"); err == nil {
			stdoutTee = io.MultiWriter(os.Stdout, stdoutFile)
			defer stdoutFile.Close()
		} else {
			debugf("Failed to create stdout debug log: %v", err)
		}
		if stderrFile, err := createDebugLogFile("stderr"); err == nil {
			stderrTee = io.MultiWriter(os.Stderr, stderrFile)
			defer stderrFile.Close()
		} else {
			debugf("Failed to create stderr debug log: %v", err)
		}
	}

	// Handle stdin
	go pipeIO(pipes[0], os.Stdin, nil)

	// Handle stdout
	go pipeIO(pipes[1], nil, stdoutTee)

	// Handle stderr
	go pipeIO(pipes[2], nil, stderrTee)

	// Wait for process to finish and exit with its status code
	err := cmd.Wait()
	if err != nil {
		debugf("error waiting for app bundle: %v", err)
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}

	os.Exit(0)
}

// pipeIO copies data between a pipe and stdin/stdout/stderr.
func pipeIO(pipe string, in io.Reader, out io.Writer) {
	pipeIOContext(context.Background(), pipe, in, out)
}

// pipeIOContext copies data between a pipe and stdin/stdout/stderr with context support.
// The context allows for cancellation of long-running I/O operations.
func pipeIOContext(ctx context.Context, pipe string, in io.Reader, out io.Writer) {
	mode := os.O_RDONLY
	if in != nil {
		mode = os.O_WRONLY
	}

	f, err := os.OpenFile(pipe, mode, 0)
	if err != nil {
		debugf("error opening pipe: %v", err)
		return
	}
	defer f.Close()

	// Create a channel to signal completion
	done := make(chan struct{})

	go func() {
		if in != nil {
			io.Copy(f, in)
		} else {
			io.Copy(out, f)
		}
		close(done)
	}()

	// Wait for either completion or context cancellation
	select {
	case <-done:
		// Normal completion
	case <-ctx.Done():
		debugf("pipeIO cancelled due to context: %v", ctx.Err())
		// Close the file to interrupt the copy operation
		f.Close()
	}
}

// createPipe creates a named pipe securely to prevent race conditions.
func createPipe(prefix string) (string, error) {
	// Validate prefix to prevent injection
	cleanPrefix, err := sanitizePath(prefix)
	if err != nil {
		return "", fmt.Errorf("invalid pipe prefix: %w", err)
	}

	// Create a secure temporary directory first
	tmpDir, err := os.MkdirTemp("", "macgo-pipes-*")
	if err != nil {
		return "", fmt.Errorf("create secure pipe directory: %w", err)
	}

	// Set restrictive permissions on the pipe directory
	if err := os.Chmod(tmpDir, 0700); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("set pipe directory permissions: %w", err)
	}

	// Generate a unique pipe name within the secure directory
	pipeName := fmt.Sprintf("%s-%d-%d", cleanPrefix, os.Getpid(), time.Now().UnixNano())
	pipePath, err := secureJoin(tmpDir, pipeName)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("create secure pipe path: %w", err)
	}

	// Validate mkfifo binary path for security
	mkfifoPath, err := exec.LookPath("mkfifo")
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("mkfifo not found: %w", err)
	}

	cleanMkfifoPath, err := securePath(mkfifoPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("mkfifo path validation failed: %w", err)
	}

	// Create the named pipe atomically
	cmd := exec.Command(cleanMkfifoPath, pipePath)
	cmd.Env = []string{"PATH=/usr/bin:/bin"} // Restricted environment

	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("create named pipe: %w", err)
	}

	// Set restrictive permissions on the pipe
	if err := os.Chmod(pipePath, 0600); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("set pipe permissions: %w", err)
	}

	return pipePath, nil
}

// checksum calculates the SHA-256 hash of a file.
func checksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// createDebugLogFile creates a debug log file for capturing IO
func createDebugLogFile(streamName string) (*os.File, error) {
	logPath := fmt.Sprintf("/tmp/macgo-debug-%s-%d.txt", streamName, os.Getpid())
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	debugf("Created %s debug log: %s", streamName, logPath)
	return file, nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0755)
}

// escapeXML escapes XML special characters to prevent XML injection vulnerabilities.
// It replaces the following characters with their XML entity equivalents:
// & -> &amp;
// < -> &lt;
// > -> &gt;
// " -> &quot;
// ' -> &apos;
func escapeXML(s string) string {
	if s == "" {
		return s
	}
	
	// Use strings.Replacer for efficient multiple replacements
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&apos;",
	)
	
	return replacer.Replace(s)
}

// writePlist writes a map to a plist file.
func writePlist[K ~string](path string, data map[K]any) error {
	var sb strings.Builder

	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
`)

	for k, v := range data {
		// Escape the key to prevent XML injection
		sb.WriteString(fmt.Sprintf("\t<key>%s</key>\n", escapeXML(string(k))))

		switch val := v.(type) {
		case bool:
			if val {
				sb.WriteString("\t<true/>\n")
			} else {
				sb.WriteString("\t<false/>\n")
			}
		case string:
			// Escape the string value to prevent XML injection
			sb.WriteString(fmt.Sprintf("\t<string>%s</string>\n", escapeXML(val)))
		case int, int32, int64:
			sb.WriteString(fmt.Sprintf("\t<integer>%v</integer>\n", val))
		case float32, float64:
			sb.WriteString(fmt.Sprintf("\t<real>%v</real>\n", val))
		default:
			// Escape the stringified value to prevent XML injection
			sb.WriteString(fmt.Sprintf("\t<string>%s</string>\n", escapeXML(fmt.Sprintf("%v", val))))
		}
	}

	sb.WriteString("</dict>\n</plist>")
	return os.WriteFile(path, []byte(sb.String()), 0644)
}

// Environment variable detection for entitlements
func init() {
	// Check environment variables for permissions and entitlements
	envVars := map[string]string{
		// Basic TCC permissions (legacy)
		"MACGO_CAMERA":   string(EntCamera),
		"MACGO_MIC":      string(EntMicrophone),
		"MACGO_LOCATION": string(EntLocation),
		"MACGO_CONTACTS": string(EntAddressBook),
		"MACGO_PHOTOS":   string(EntPhotos),
		"MACGO_CALENDAR": string(EntCalendars),

		// App Sandbox entitlements
		"MACGO_APP_SANDBOX":    string(EntAppSandbox),
		"MACGO_NETWORK_CLIENT": string(EntNetworkClient),
		"MACGO_NETWORK_SERVER": string(EntNetworkServer),

		// Device entitlements
		"MACGO_BLUETOOTH":   string(EntBluetooth),
		"MACGO_USB":         string(EntUSB),
		"MACGO_AUDIO_INPUT": string(EntAudioInput),
		"MACGO_PRINT":       string(EntPrint),

		// File entitlements
		"MACGO_USER_FILES_READ":  string(EntUserSelectedReadOnly),
		"MACGO_USER_FILES_WRITE": string(EntUserSelectedReadWrite),
		"MACGO_DOWNLOADS_READ":   string(EntDownloadsReadOnly),
		"MACGO_DOWNLOADS_WRITE":  string(EntDownloadsReadWrite),
		"MACGO_PICTURES_READ":    string(EntPicturesReadOnly),
		"MACGO_PICTURES_WRITE":   string(EntPicturesReadWrite),
		"MACGO_MUSIC_READ":       string(EntMusicReadOnly),
		"MACGO_MUSIC_WRITE":      string(EntMusicReadWrite),
		"MACGO_MOVIES_READ":      string(EntMoviesReadOnly),
		"MACGO_MOVIES_WRITE":     string(EntMoviesReadWrite),

		// Hardened Runtime entitlements
		"MACGO_ALLOW_JIT":                    string(EntAllowJIT),
		"MACGO_ALLOW_UNSIGNED_MEMORY":        string(EntAllowUnsignedExecutableMemory),
		"MACGO_ALLOW_DYLD_ENV":               string(EntAllowDyldEnvVars),
		"MACGO_DISABLE_LIBRARY_VALIDATION":   string(EntDisableLibraryValidation),
		"MACGO_DISABLE_EXEC_PAGE_PROTECTION": string(EntDisableExecutablePageProtection),
		"MACGO_DEBUGGER":                     string(EntDebugger),
	}

	for env, entitlement := range envVars {
		if os.Getenv(env) == "1" {
			DefaultConfig.AddEntitlement(Entitlement(entitlement))
		}
	}
}

// createFromTemplate creates an app bundle from an embedded template
func createFromTemplate(template fs.FS, appPath, execPath, appName string) (string, error) {
	// Validate all input paths for security
	if err := validateExecutablePath(execPath); err != nil {
		return "", fmt.Errorf("invalid executable path: %w", err)
	}

	cleanAppPath, err := securePath(appPath)
	if err != nil {
		return "", fmt.Errorf("invalid app path: %w", err)
	}
	appPath = cleanAppPath

	cleanAppName, err := sanitizePath(appName)
	if err != nil {
		return "", fmt.Errorf("invalid app name: %w", err)
	}
	appName = cleanAppName

	// Create the app bundle directory
	if err := os.MkdirAll(appPath, 0755); err != nil {
		return "", fmt.Errorf("create app bundle directory: %w", err)
	}

	// Walk the template and copy all files to the app bundle
	err = fs.WalkDir(template, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory
		if path == "." {
			return nil
		}

		// Validate template path for security
		cleanPath, pathErr := sanitizePath(path)
		if pathErr != nil {
			return fmt.Errorf("invalid template path %s: %w", path, pathErr)
		}

		// Full path in the target app bundle using secure join
		targetPath, pathErr := secureJoin(appPath, cleanPath)
		if pathErr != nil {
			return fmt.Errorf("failed to create secure target path for %s: %w", path, pathErr)
		}

		// Create directories
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		// Special handling for executables - replace with the actual executable
		if strings.Contains(cleanPath, "Contents/MacOS/") && strings.HasSuffix(cleanPath, ".placeholder") {
			// Extract the executable name without the .placeholder suffix
			dirPath := filepath.Dir(targetPath)
			execName := filepath.Base(execPath)

			// Ensure the directory exists
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				return fmt.Errorf("create executable directory: %w", err)
			}

			// Copy the executable to the bundle using secure path
			bundleExecPath, pathErr := secureJoin(dirPath, execName)
			if pathErr != nil {
				return fmt.Errorf("failed to create secure executable path: %w", pathErr)
			}
			if err := copyFile(execPath, bundleExecPath); err != nil {
				return fmt.Errorf("copy executable: %w", err)
			}

			// Make it executable
			return os.Chmod(bundleExecPath, 0755)
		}

		// Special handling for Info.plist - process templated values
		if strings.HasSuffix(path, "Info.plist") {
			// Read the template plist
			data, err := fs.ReadFile(template, path)
			if err != nil {
				return fmt.Errorf("read template Info.plist: %w", err)
			}

			// Replace placeholder values with properly escaped values
			content := string(data)
			content = strings.ReplaceAll(content, "{{BundleName}}", escapeXML(appName))
			content = strings.ReplaceAll(content, "{{BundleExecutable}}", escapeXML(filepath.Base(execPath)))

			bundleID := DefaultConfig.BundleID
			if bundleID == "" {
				bundleID = fmt.Sprintf("com.macgo.%s", appName)
			}
			content = strings.ReplaceAll(content, "{{BundleIdentifier}}", escapeXML(bundleID))

			// Add user-defined plist entries
			// This is a simple approach - for more complex needs, use a proper plist library
			for k, v := range DefaultConfig.PlistEntries {
				// Escape key to prevent XML injection
				key := fmt.Sprintf("<key>%s</key>", escapeXML(k))
				var valueTag string
				switch val := v.(type) {
				case bool:
					if val {
						valueTag = "<true/>"
					} else {
						valueTag = "<false/>"
					}
				case string:
					// Escape string value to prevent XML injection
					valueTag = fmt.Sprintf("<string>%s</string>", escapeXML(val))
				case int, int32, int64:
					valueTag = fmt.Sprintf("<integer>%v</integer>", val)
				case float32, float64:
					valueTag = fmt.Sprintf("<real>%v</real>", val)
				default:
					// Escape stringified value to prevent XML injection
					valueTag = fmt.Sprintf("<string>%s</string>", escapeXML(fmt.Sprintf("%v", val)))
				}

				// Insert before closing dict
				closingDict := "</dict>"
				insertPos := strings.LastIndex(content, closingDict)
				if insertPos != -1 {
					content = content[:insertPos] + "\t" + key + "\n\t" + valueTag + "\n" + content[insertPos:]
				}
			}

			// Write the processed plist
			return os.WriteFile(targetPath, []byte(content), 0644)
		}

		// Special handling for entitlements.plist
		if strings.HasSuffix(path, "entitlements.plist") {
			// Create a map for enabled entitlements only
			entitlements := make(map[string]any)
			hasEnabledEntitlements := false
			for k, v := range DefaultConfig.Entitlements {
				if v {
					entitlements[string(k)] = v
					hasEnabledEntitlements = true
				}
			}

			// Only write the entitlements plist if there are enabled entitlements
			if hasEnabledEntitlements {
				return writePlist(targetPath, entitlements)
			}
			// Skip writing the file if no entitlements are enabled
			return nil
		}

		// For normal files, just copy them
		data, err := fs.ReadFile(template, path)
		if err != nil {
			return fmt.Errorf("read template file %s: %w", path, err)
		}

		return os.WriteFile(targetPath, data, 0644)
	})

	if err != nil {
		return "", fmt.Errorf("process template: %w", err)
	}

	// Auto-sign the bundle if requested
	if DefaultConfig.AutoSign {
		if err := signBundle(appPath); err != nil {
			debugf("Error signing bundle: %v", err)
			// Continue even if signing fails
		}
	}

	return appPath, nil
}

// validateSigningIdentity validates and sanitizes a code signing identity.
func validateSigningIdentity(identity string) error {
	if identity == "" {
		return nil // Empty identity is valid (uses ad-hoc signing)
	}

	// Check for dangerous characters that could be used for command injection
	if strings.ContainsAny(identity, "\x00\r\n;|&`$(){}[]<>\"'\\") {
		return fmt.Errorf("signing identity contains invalid characters")
	}

	// Check for obvious command injection attempts
	dangerousPatterns := []string{
		"--", "rm ", "mv ", "cp ", "cat ", "echo ", "sh ", "bash ", "/bin/", "/usr/bin/",
		"sudo", "su ", "chmod", "chown", "> ", "< ", "| ", "& ", "; ", "$(", "`",
	}

	lowerIdentity := strings.ToLower(identity)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerIdentity, pattern) {
			return fmt.Errorf("signing identity contains potentially dangerous pattern: %s", pattern)
		}
	}

	// Length check to prevent resource exhaustion
	if len(identity) > 256 {
		return fmt.Errorf("signing identity too long: %d characters", len(identity))
	}

	// Valid signing identities should match expected patterns
	validPatterns := []string{
		"Developer ID Application:",
		"Mac Developer:",
		"Apple Development:",
		"Apple Distribution:",
		"iPhone Developer:",
		"iPhone Distribution:",
		"3rd Party Mac Developer Application:",
		"3rd Party Mac Developer Installer:",
	}

	// Special case: ad-hoc signing
	if identity == "-" {
		return nil
	}

	// Check if it matches any valid signing identity pattern
	for _, pattern := range validPatterns {
		if strings.HasPrefix(identity, pattern) {
			return nil
		}
	}

	// If it doesn't match known patterns, it might be a certificate hash
	// Valid certificate hashes are 40-character hex strings
	if len(identity) == 40 {
		for _, c := range identity {
			if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F') || (c >= 'a' && c <= 'f')) {
				return fmt.Errorf("invalid certificate hash format")
			}
		}
		return nil
	}

	return fmt.Errorf("signing identity does not match known valid patterns")
}

// signBundle codesigns the app bundle using the system's codesign tool.
// It returns an error if codesigning fails, which can happen if:
// - The codesign tool is not available
// - No valid signing identity is present
// - The app bundle is malformed
// This is considered a non-critical error and macgo will still work without signed bundles,
// but signed bundles are required for certain entitlements to function properly.
func signBundle(appPath string) error {
	// Validate and sanitize app path to prevent command injection
	cleanAppPath, err := securePath(appPath)
	if err != nil {
		return fmt.Errorf("invalid app path for signing: %w", err)
	}
	appPath = cleanAppPath

	// Validate and sanitize signing identity
	identity := DefaultConfig.SigningIdentity
	if err := validateSigningIdentity(identity); err != nil {
		return fmt.Errorf("invalid signing identity: %w", err)
	}

	// Check if codesign is available
	codesignPath, err := exec.LookPath("codesign")
	if err != nil {
		return fmt.Errorf("codesign tool not found: %w", err)
	}

	// Validate codesign binary path for additional security
	if cleanCodesignPath, pathErr := securePath(codesignPath); pathErr != nil {
		return fmt.Errorf("codesign binary path validation failed: %w", pathErr)
	} else {
		codesignPath = cleanCodesignPath
	}

	// Build the codesign command with validated arguments
	args := []string{"--force", "--deep"}

	// Add entitlements if available (using secure path operations)
	contentsPath, pathErr := secureJoin(appPath, "Contents")
	if pathErr != nil {
		return fmt.Errorf("failed to create secure Contents path: %w", pathErr)
	}
	entitlementsPath, pathErr := secureJoin(contentsPath, "entitlements.plist")
	if pathErr != nil {
		return fmt.Errorf("failed to create secure entitlements path: %w", pathErr)
	}

	if _, err := os.Stat(entitlementsPath); err == nil {
		debugf("Using entitlements file: %s", entitlementsPath)
		args = append(args, "--entitlements", entitlementsPath)
	} else {
		debugf("No entitlements file found at: %s", entitlementsPath)
	}

	// Add signing identity (already validated)
	if identity != "" {
		debugf("Using specified signing identity: %s", identity)
		args = append(args, "--sign", identity)
	} else {
		// Use ad-hoc signing with "-s -"
		debugf("Using ad-hoc signing with -s -")
		args = append(args, "--sign", "-")
	}

	// Add the app path (already validated)
	args = append(args, appPath)

	// Execute codesign with validated binary and arguments
	cmd := exec.Command(codesignPath, args...)

	// Set up secure execution environment
	cmd.Env = []string{
		"PATH=/usr/bin:/bin", // Restricted PATH to prevent binary hijacking
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("codesign failed: %w, output: %s", err, output)
	}

	debugf("Successfully codesigned app bundle: %s", appPath)
	return nil
}

// debugf prints debug messages to stderr if MACGO_DEBUG=1 is set in the environment.
// It prefixes all messages with "macgo:" and adds a timestamp to help with troubleshooting.
func debugf(format string, args ...any) {
	if os.Getenv("MACGO_DEBUG") == "1" {
		timestamp := time.Now().Format("15:04:05.000")
		prefix := fmt.Sprintf("[macgo:%s] ", timestamp)
		fmt.Fprintf(os.Stderr, prefix+format+"\n", args...)
	}
}

// checkDeveloperEnvironment checks for common macOS developer environment issues
func checkDeveloperEnvironment() {
	if !isDebugEnabled() {
		return
	}

	// Check Xcode developer directory
	devDir := os.Getenv("DEVELOPER_DIR")
	if devDir == "" {
		// Get default from xcode-select
		cmd := exec.Command("xcode-select", "--print-path")
		output, err := cmd.Output()
		if err != nil {
			debugf("Warning: Could not get Xcode developer directory: %v", err)
			return
		}
		devDir = strings.TrimSpace(string(output))
	}

	debugf("Xcode developer directory: %s", devDir)

	// Check if Platforms directory exists (required for 'open' command)
	platformsDir := filepath.Join(devDir, "Platforms")
	if _, err := os.Stat(platformsDir); err != nil {
		debugf("WARNING: Platforms directory missing at %s", platformsDir)
		debugf("This may cause 'open' command to fail when launching app bundles")
		debugf("SOLUTIONS (try in order):")
		debugf("  1. sudo xcode-select --reset")
		debugf("  2. sudo xcode-select --switch /Library/Developer/CommandLineTools")
		debugf("  3. xcode-select --install (to reinstall Command Line Tools)")
		debugf("Note: Full Xcode is NOT required - this is a Command Line Tools config issue")

		// Try to auto-detect if we can suggest a specific fix
		if _, err := os.Stat("/Applications/Xcode.app"); err == nil {
			debugf("  Alternative: Use Xcode path: sudo xcode-select --switch /Applications/Xcode.app/Contents/Developer")
		}

		// Check if Command Line Tools are properly installed
		if _, err := os.Stat("/Library/Developer/CommandLineTools/usr/bin"); err != nil {
			debugf("  Command Line Tools may not be properly installed")
		}
	} else {
		debugf("Platforms directory found at %s", platformsDir)
	}
}

func isDebugEnabled() bool {
	return os.Getenv("MACGO_DEBUG") == "1"
}
