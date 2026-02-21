// Package launchservices provides pure Go bindings for the macOS LaunchServices
// private SPI _LSOpenURLsWithCompletionHandler, the same function that
// /usr/bin/open uses internally. No cgo required.
//
// It exposes a minimal API for launching applications and opening files/URLs
// via LaunchServices, with support for environment variables, I/O redirection,
// architecture selection, and process wait.
package launchservices

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/ebitengine/purego/objc"
)

// Options controls how an application is launched.
type Options struct {
	// Activate brings the application to the foreground. Default true.
	Activate bool
	// Hide launches the application hidden.
	Hide bool
	// NewInstance launches a new instance even if one is already running.
	NewInstance bool
	// Fresh launches without restoring windows from previous session.
	Fresh bool
	// WaitForCheckIn tells LaunchServices to wait for the app to check in.
	WaitForCheckIn bool
	// Arguments are passed to the application as command-line arguments.
	Arguments []string
	// Environment is a map of environment variables to set for the launched app.
	Environment map[string]string
	// StdinPath redirects the launched app's stdin to this file path.
	StdinPath string
	// StdoutPath redirects the launched app's stdout to this file path.
	StdoutPath string
	// StderrPath redirects the launched app's stderr to this file path.
	StderrPath string
	// Arch specifies the CPU architecture to launch under (e.g., "arm64", "x86_64").
	Arch string
	// AddToRecents controls whether the opened item appears in Recents. Default true.
	AddToRecents bool
}

// DefaultOptions returns Options with sensible defaults.
func DefaultOptions() Options {
	return Options{
		Activate:     true,
		AddToRecents: true,
	}
}

// Result contains information about a launched application.
type Result struct {
	// PID is the process ID of the launched application, or 0 if unavailable.
	PID int
	// AlreadyRunning is true if the application was already running.
	AlreadyRunning bool
}

// LaunchApp launches an application by its bundle file path (e.g., "/Applications/Safari.app").
// Pass nil documents to launch the app without opening any files.
func LaunchApp(appPath string, documents []string, opts Options) (Result, error) {
	initObjC()

	abs, err := filepath.Abs(appPath)
	if err != nil {
		return Result{}, fmt.Errorf("resolve app path: %w", err)
	}
	appURL := nsURLFileURLWithPath(abs)
	if appURL == 0 {
		return Result{}, fmt.Errorf("invalid app path: %s", appPath)
	}

	urls, err := buildDocumentURLs(documents)
	if err != nil {
		return Result{}, err
	}

	optDict := buildOptionDict(opts)
	return callLSOpen(urls, appURL, optDict)
}

// LaunchBundleID launches an application by its bundle identifier
// (e.g., "com.apple.Safari").
func LaunchBundleID(bundleID string, documents []string, opts Options) (Result, error) {
	initObjC()

	appURL, err := AppURLFromBundleID(bundleID)
	if err != nil {
		return Result{}, err
	}

	urls, err := buildDocumentURLs(documents)
	if err != nil {
		return Result{}, err
	}

	optDict := buildOptionDict(opts)
	return callLSOpen(urls, appURL, optDict)
}

// OpenDocuments opens files with the default application for each file type.
// An explicit appPath may be provided to override the default; pass "" to auto-resolve.
func OpenDocuments(files []string, appPath string, opts Options) (Result, error) {
	initObjC()

	if len(files) == 0 {
		return Result{}, fmt.Errorf("no files specified")
	}

	var appURL objc.ID
	if appPath != "" {
		abs, err := filepath.Abs(appPath)
		if err != nil {
			return Result{}, fmt.Errorf("resolve app path: %w", err)
		}
		appURL = nsURLFileURLWithPath(abs)
	} else {
		var err error
		appURL, err = defaultAppForFile(files[0])
		if err != nil {
			return Result{}, fmt.Errorf("no application knows how to open %s", files[0])
		}
	}

	urls, err := buildDocumentURLs(files)
	if err != nil {
		return Result{}, err
	}

	optDict := buildOptionDict(opts)
	return callLSOpen(urls, appURL, optDict)
}

// OpenURL opens a URL string with the default or specified application.
func OpenURL(urlStr string, appPath string, opts Options) (Result, error) {
	initObjC()

	nsurl := nsURLFromString(urlStr)
	if nsurl == 0 {
		return Result{}, fmt.Errorf("invalid URL: %s", urlStr)
	}

	arr := objc.ID(clsNSArray).Send(objc.RegisterName("arrayWithObject:"), nsurl)

	var appURL objc.ID
	if appPath != "" {
		abs, err := filepath.Abs(appPath)
		if err != nil {
			return Result{}, fmt.Errorf("resolve app path: %w", err)
		}
		appURL = nsURLFileURLWithPath(abs)
	} else {
		appURL = defaultAppForURL(nsurl)
	}

	optDict := buildOptionDict(opts)
	return callLSOpen(arr, appURL, optDict)
}

// AppURLFromBundleID finds the file URL of an application by bundle identifier.
func AppURLFromBundleID(bundleID string) (objc.ID, error) {
	initObjC()
	cfBundleID := nsString(bundleID)
	urls := fnLSCopyAppURLsForBundleID(cfBundleID, 0)
	if urls == 0 {
		return 0, fmt.Errorf("unable to find application with bundle identifier %s", bundleID)
	}
	first := urls.Send(selFirstObject)
	if first == 0 {
		return 0, fmt.Errorf("unable to find application with bundle identifier %s", bundleID)
	}
	return first, nil
}

// AppURLFromName finds the file URL of an application by display name using NSWorkspace.
func AppURLFromName(name string) (objc.ID, error) {
	initObjC()
	return appURLFromName(name)
}

// DefaultAppForFile returns the default application URL for a given file path.
func DefaultAppForFile(path string) (objc.ID, error) {
	initObjC()
	return defaultAppForFile(path)
}

// DefaultTextEditorURL returns the URL of the default text editor.
func DefaultTextEditorURL() (objc.ID, error) {
	initObjC()
	contentType := nsString("public.plain-text")
	url := fnLSCopyDefaultAppURLForContentType(contentType, 0x00000002 /* kLSRolesEditor */, 0)
	if url == 0 {
		return 0, fmt.Errorf("unable to determine default text editor")
	}
	return url, nil
}

// buildDocumentURLs creates an NSArray of file URLs from paths.
// Returns an empty NSArray if paths is nil.
func buildDocumentURLs(paths []string) (objc.ID, error) {
	if len(paths) == 0 {
		return objc.ID(clsNSArray).Send(objc.RegisterName("array")), nil
	}

	arr := objc.ID(clsNSMutableArray).Send(selAlloc)
	arr = arr.Send(selInitWithCapacity, uintptr(len(paths)))
	for _, f := range paths {
		abs, err := filepath.Abs(f)
		if err != nil {
			return 0, fmt.Errorf("resolve path %q: %w", f, err)
		}
		url := nsURLFileURLWithPath(abs)
		arr.Send(selAddObject, url)
	}
	return arr, nil
}

// buildOptionDict creates an NSDictionary from Options.
func buildOptionDict(opts Options) objc.ID {
	dict := objc.ID(clsNSMutableDictionary).Send(selAlloc)
	dict = dict.Send(selInit)

	dictSetBool(dict, symActivateKey, opts.Activate)

	if opts.Hide {
		dictSetBool(dict, symHideKey, true)
	}

	if !opts.NewInstance {
		dictSetBool(dict, symPreferRunningInstanceKey, true)
	}

	if opts.Fresh && symLaunchWithoutRestoringStateKey != 0 {
		dictSetBool(dict, symLaunchWithoutRestoringStateKey, true)
	}

	if opts.WaitForCheckIn {
		dictSetBool(dict, symWaitForCheckInKey, true)
	}

	if len(opts.Arguments) > 0 {
		arr := objc.ID(clsNSMutableArray).Send(selAlloc)
		arr = arr.Send(selInitWithCapacity, uintptr(len(opts.Arguments)))
		for _, a := range opts.Arguments {
			arr.Send(selAddObject, nsString(a))
		}
		dictSet(dict, symArgumentsKey, arr)
	}

	if len(opts.Environment) > 0 {
		envDict := objc.ID(clsNSMutableDictionary).Send(selAlloc)
		envDict = envDict.Send(selInit)
		for k, v := range opts.Environment {
			dictSet(envDict, nsString(k), nsString(v))
		}
		dictSet(dict, symEnvironmentVariablesKey, envDict)
	}

	if opts.StdinPath != "" {
		dictSet(dict, symStdInPathKey, nsString(opts.StdinPath))
	}
	if opts.StdoutPath != "" {
		dictSet(dict, symStdOutPathKey, nsString(opts.StdoutPath))
	}
	if opts.StderrPath != "" {
		dictSet(dict, symStdErrPathKey, nsString(opts.StderrPath))
	}

	if opts.Arch != "" {
		cpuType, cpuSubtype := parseArch(opts.Arch)
		if cpuType != 0 {
			dictSetInt(dict, symArchitectureKey, cpuType)
			if cpuSubtype != 0 {
				dictSetInt(dict, symArchitectureSubtypeKey, cpuSubtype)
			}
		}
	}

	if !opts.AddToRecents {
		dictSetBool(dict, symAddToRecentsKey, false)
	}

	return dict
}

// callLSOpen invokes _LSOpenURLsWithCompletionHandler and blocks until the
// completion handler fires or a timeout is reached.
//
// The caller must be on a thread with an active run loop (typically the main
// thread, locked via runtime.LockOSThread). The completion handler is delivered
// via a run loop source that requires pumping.
func callLSOpen(urls, appURL, optDict objc.ID) (Result, error) {
	resultCh := make(chan lsOpenResult, 1)

	block := objc.NewBlock(func(_ objc.Block, asn uintptr, alreadyRunning bool, cfErr uintptr) {
		var res lsOpenResult
		res.alreadyRunning = alreadyRunning
		res.err = cfErr
		if cfErr != 0 {
			cfRetain(cfErr)
		}
		if asn != 0 {
			high, low := lsASNExtractParts(asn)
			pid := getProcessPID(high, low)
			if pid > 0 {
				res.pid = int(pid)
			}
		}
		resultCh <- res
	})
	defer block.Release()

	fnLSOpenURLsWithCompletionHandler(urls, appURL, optDict, block)

	timeout := time.After(30 * time.Second)
	for {
		select {
		case res := <-resultCh:
			if res.err != 0 {
				return Result{}, formatLSError(res)
			}
			return Result{
				PID:            res.pid,
				AlreadyRunning: res.alreadyRunning,
			}, nil
		case <-timeout:
			return Result{}, fmt.Errorf("launch timed out after 30s")
		default:
			pumpRunLoop(50 * time.Millisecond)
		}
	}
}

type lsOpenResult struct {
	err            uintptr
	alreadyRunning bool
	pid            int
}

func formatLSError(res lsOpenResult) error {
	if res.err == 0 {
		return nil
	}
	domain := cfErrorGetDomain(res.err)
	code := cfErrorGetCode(res.err)
	desc := cfErrorDescription(res.err)
	if desc != "" {
		return fmt.Errorf("%s (domain=%s, code=%d)", desc, domain, code)
	}
	return fmt.Errorf("launch failed (domain=%s, code=%d)", domain, code)
}

// parseArch converts an architecture name to CPU type/subtype constants.
func parseArch(arch string) (int, int) {
	const (
		cpuTypeI386   = 7
		cpuTypeX86_64 = 0x01000007
		cpuTypeARM    = 12
		cpuTypeARM64  = 0x0100000C
	)
	const (
		cpuSubtypeARM64All  = 0
		cpuSubtypeARM64E    = 2
		cpuSubtypeX86_64All = 3
		cpuSubtypeX86_64H   = 8
		cpuSubtypeARM64_32  = 1
	)

	switch strings.ToLower(arch) {
	case "arm64":
		return cpuTypeARM64, cpuSubtypeARM64All
	case "arm64e":
		return cpuTypeARM64, cpuSubtypeARM64E
	case "arm64_32":
		return cpuTypeARM64, cpuSubtypeARM64_32
	case "x86_64":
		return cpuTypeX86_64, cpuSubtypeX86_64All
	case "x86_64h":
		return cpuTypeX86_64, cpuSubtypeX86_64H
	case "i386":
		return cpuTypeI386, 0
	case "arm":
		return cpuTypeARM, 0
	case "any":
		return 0, 0
	default:
		var t, s int
		if n, _ := fmt.Sscanf(arch, "%d/%d", &t, &s); n >= 1 {
			return t, s
		}
		return 0, 0
	}
}
