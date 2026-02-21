// Command lsopen is a pure Go replacement for /usr/bin/open on macOS.
//
// It uses purego to call _LSOpenURLsWithCompletionHandler, the same private
// LaunchServices SPI that /usr/bin/open uses internally. No cgo required.
//
// All flags from the original open(1) are supported.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	ls "github.com/tmc/macgo/internal/launchservices"
)

func main() {
	runtime.LockOSThread()

	opts, err := parseArgs(os.Args[1:])
	if err != nil {
		fatal(err)
	}

	if err := run(opts); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "lsopen: %v\n", err)
	os.Exit(1)
}

func run(opts *options) error {
	// -f: read stdin into temp file, open with text editor
	if opts.readStdin {
		path, err := readStdinToTemp()
		if err != nil {
			return err
		}
		opts.files = append(opts.files, path)
		opts.useTextEditor = true
	}

	// -h: search for headers
	if opts.searchHeaders {
		return openHeaders(opts)
	}

	// Categorize files into apps and documents
	var appPaths []string
	var docPaths []string
	var urlStrings []string
	for _, f := range opts.files {
		if opts.forceURL || looksLikeURL(f) {
			urlStrings = append(urlStrings, f)
		} else if isAppBundle(f) && opts.appName == "" && opts.bundleID == "" && !opts.useTextEdit && !opts.useTextEditor {
			appPaths = append(appPaths, f)
		} else {
			docPaths = append(docPaths, f)
		}
	}

	lsOpts := buildLSOpts(opts)
	var allPids []int

	// Launch app bundles directly
	for _, app := range appPaths {
		abs, err := filepath.Abs(app)
		if err != nil {
			return fmt.Errorf("unable to interpret '%s': %w", app, err)
		}
		if _, err := os.Stat(abs); err != nil {
			return fmt.Errorf("the file %s does not exist", app)
		}
		result, err := ls.LaunchApp(abs, nil, lsOpts)
		if err != nil {
			return err
		}
		if result.AlreadyRunning {
			warnAlreadyRunning(opts)
		}
		if result.PID > 0 {
			allPids = append(allPids, result.PID)
		}
	}

	// Open documents/files
	if len(docPaths) > 0 {
		appPath, err := resolveAppPath(opts)
		if err != nil {
			return err
		}

		// Verify files exist
		for _, f := range docPaths {
			abs, err := filepath.Abs(f)
			if err != nil {
				return fmt.Errorf("unable to interpret '%s': %w", f, err)
			}
			if _, err := os.Stat(abs); err != nil {
				return fmt.Errorf("the file %s does not exist", f)
			}
		}

		result, err := ls.OpenDocuments(docPaths, appPath, lsOpts)
		if err != nil {
			return err
		}
		if result.AlreadyRunning {
			warnAlreadyRunning(opts)
		}
		if result.PID > 0 {
			allPids = append(allPids, result.PID)
		}
	}

	// Open URL strings
	for _, u := range urlStrings {
		appPath, err := resolveAppPath(opts)
		if err != nil {
			return err
		}
		result, err := ls.OpenURL(u, appPath, lsOpts)
		if err != nil {
			return fmt.Errorf("unable to interpret '%s' as a URL: %w", u, err)
		}
		if result.PID > 0 {
			allPids = append(allPids, result.PID)
		}
	}

	// If no files/apps/urls at all, just launch the app (for -a/-b flags)
	if len(appPaths) == 0 && len(docPaths) == 0 && len(urlStrings) == 0 {
		appPath, err := resolveAppPath(opts)
		if err != nil {
			return err
		}
		if appPath == "" {
			fmt.Fprint(os.Stderr, usage+"\n")
			return nil
		}
		result, err := ls.LaunchApp(appPath, nil, lsOpts)
		if err != nil {
			return err
		}
		if result.AlreadyRunning {
			warnAlreadyRunning(opts)
		}
		if result.PID > 0 {
			allPids = append(allPids, result.PID)
		}
	}

	// -W: wait for apps to exit
	if opts.waitForExit && len(allPids) > 0 {
		return ls.WaitForExitMultiple(allPids)
	}

	return nil
}

// buildLSOpts converts CLI options to launchservices.Options.
func buildLSOpts(opts *options) ls.Options {
	lsOpts := ls.DefaultOptions()
	lsOpts.Activate = !opts.background
	lsOpts.Hide = opts.hidden
	lsOpts.NewInstance = opts.newInstance
	lsOpts.Fresh = opts.fresh
	lsOpts.WaitForCheckIn = opts.waitForExit
	lsOpts.Arguments = opts.args
	lsOpts.StdinPath = opts.stdinPath
	lsOpts.StdoutPath = opts.stdoutPath
	lsOpts.StderrPath = opts.stderrPath
	lsOpts.Arch = opts.arch
	lsOpts.AddToRecents = !opts.reveal

	if len(opts.env) > 0 {
		lsOpts.Environment = make(map[string]string, len(opts.env))
		for _, e := range opts.env {
			k, v := parseEnvVar(e)
			lsOpts.Environment[k] = v
		}
	}

	return lsOpts
}

// resolveAppPath returns the filesystem path to the app, or "" if none specified.
func resolveAppPath(opts *options) (string, error) {
	switch {
	case opts.useTextEdit:
		return appPathFromBundleID("com.apple.TextEdit")
	case opts.useTextEditor:
		url, err := ls.DefaultTextEditorURL()
		if err != nil {
			return appPathFromBundleID("com.apple.TextEdit")
		}
		return ls.URLPath(url), nil
	case opts.appName != "":
		url, err := ls.AppURLFromName(opts.appName)
		if err != nil {
			return "", err
		}
		return ls.URLPath(url), nil
	case opts.bundleID != "":
		return appPathFromBundleID(opts.bundleID)
	case opts.reveal:
		return appPathFromBundleID("com.apple.finder")
	default:
		return "", nil
	}
}

// appPathFromBundleID resolves a bundle ID to a filesystem path.
func appPathFromBundleID(bundleID string) (string, error) {
	url, err := ls.AppURLFromBundleID(bundleID)
	if err != nil {
		return "", err
	}
	return ls.URLPath(url), nil
}

func readStdinToTemp() (string, error) {
	f, err := os.CreateTemp("", "open_*.txt")
	if err != nil {
		return "", fmt.Errorf("unable to open temporary file: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, os.Stdin); err != nil {
		os.Remove(f.Name())
		return "", fmt.Errorf("error reading from stdin: %w", err)
	}
	return f.Name(), nil
}

func isAppBundle(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	if !strings.HasSuffix(abs, ".app") {
		return false
	}
	info, err := os.Stat(abs)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func parseEnvVar(s string) (string, string) {
	if k, v, ok := strings.Cut(s, "="); ok {
		return k, v
	}
	return s, ""
}

func warnAlreadyRunning(opts *options) {
	hasEnv := len(opts.env) > 0
	hasIO := opts.stdinPath != "" || opts.stdoutPath != "" || opts.stderrPath != ""
	if !hasEnv && !hasIO {
		return
	}
	msg := "Application was already running"
	switch {
	case hasEnv && hasIO:
		msg += " and so the additional environment variables and redirected stdin/stdout/stderr provided could not be set."
	case hasEnv:
		msg += " and so the additional environment variables could not be set."
	case hasIO:
		msg += " and so the redirected stdin/stdout/stderr provided could not be set."
	}
	fmt.Fprintln(os.Stderr, msg)
}

func looksLikeURL(s string) bool {
	if strings.HasPrefix(s, "/") || strings.HasPrefix(s, "./") || strings.HasPrefix(s, "../") || strings.HasPrefix(s, "~") {
		return false
	}
	if strings.Contains(s, "://") {
		return true
	}
	lower := strings.ToLower(s)
	for _, suffix := range []string{".com", ".org", ".net"} {
		if strings.HasSuffix(lower, suffix) || strings.Contains(lower, suffix+"/") {
			return true
		}
	}
	return false
}
