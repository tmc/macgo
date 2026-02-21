package launchservices

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"github.com/ebitengine/purego/objc"
)

// nsString creates an NSString from a Go string.
func nsString(s string) objc.ID {
	b := append([]byte(s), 0)
	return objc.ID(clsNSString).Send(selStringWithUTF8, uintptr(unsafe.Pointer(&b[0])))
}

// goString extracts a Go string from an NSString (or CFStringRef).
func goString(id objc.ID) string {
	if id == 0 {
		return ""
	}
	ptr := objc.Send[*byte](id, selUTF8String)
	if ptr == nil {
		return ""
	}
	data := unsafe.Slice(ptr, 1<<30)
	for i, b := range data {
		if b == 0 {
			return string(data[:i])
		}
	}
	return ""
}

// nsURLFromString creates NSURL from a URL string.
func nsURLFromString(s string) objc.ID {
	return objc.ID(clsNSURL).Send(selURLWithString, nsString(s))
}

// nsURLFileURLWithPath creates a file:// NSURL from a filesystem path.
func nsURLFileURLWithPath(path string) objc.ID {
	return objc.ID(clsNSURL).Send(selFileURLWithPath, nsString(path))
}

// dictSet sets a key-value pair in an NSMutableDictionary.
func dictSet(dict, key, value objc.ID) {
	dict.Send(selSetObjectForKey, value, key)
}

// dictSetBool sets a bool value for a key in an NSMutableDictionary.
func dictSetBool(dict, key objc.ID, val bool) {
	var bval uintptr
	if val {
		bval = 1
	}
	num := objc.ID(clsNSNumber).Send(selNumberWithBool, bval)
	dictSet(dict, key, num)
}

// dictSetInt sets an integer value for a key in an NSMutableDictionary.
func dictSetInt(dict, key objc.ID, val int) {
	num := objc.ID(clsNSNumber).Send(selNumberWithInteger, uintptr(val))
	dictSet(dict, key, num)
}

// appURLFromName finds the application URL by name using NSWorkspace.
func appURLFromName(name string) (objc.ID, error) {
	if strings.Contains(name, "/") || strings.HasSuffix(name, ".app") {
		abs, err := filepath.Abs(name)
		if err == nil {
			if _, err := os.Stat(abs); err == nil {
				return nsURLFileURLWithPath(abs), nil
			}
		}
		if filepath.IsAbs(name) {
			if _, err := os.Stat(name); err == nil {
				return nsURLFileURLWithPath(name), nil
			}
		}
	}

	ws := objc.ID(clsNSWorkspace).Send(selSharedWorkspace)
	path := ws.Send(selFullPathForApp, nsString(name))
	if path == 0 {
		return 0, fmt.Errorf("unable to find application named '%s'", name)
	}
	pathStr := goString(path)
	if pathStr == "" {
		return 0, fmt.Errorf("unable to find application named '%s'", name)
	}
	return nsURLFileURLWithPath(pathStr), nil
}

// defaultAppForFile resolves the default application for a file path.
func defaultAppForFile(path string) (objc.ID, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return 0, err
	}
	url := nsURLFileURLWithPath(abs)
	app := defaultAppForURL(url)
	if app == 0 {
		return 0, fmt.Errorf("no default application for %s", path)
	}
	return app, nil
}

// defaultAppForURL resolves the default application for a URL.
func defaultAppForURL(url objc.ID) objc.ID {
	if url == 0 {
		return 0
	}
	const kLSRolesAll = 0xFFFFFFFF
	return fnLSCopyDefaultAppURLForURL(url, kLSRolesAll, 0)
}

// cfRetain retains a CoreFoundation object.
func cfRetain(cf uintptr) uintptr {
	if cf == 0 {
		return 0
	}
	return fnCFRetain(cf)
}

// cfErrorGetDomain gets the domain string from a CFError.
func cfErrorGetDomain(err uintptr) string {
	if err == 0 {
		return ""
	}
	domain := fnCFErrorGetDomain(err)
	return goString(domain)
}

// cfErrorGetCode gets the error code from a CFError.
func cfErrorGetCode(err uintptr) int64 {
	if err == 0 {
		return 0
	}
	return fnCFErrorGetCode(err)
}

// cfErrorDescription gets a description from a CFError via NSError bridging.
func cfErrorDescription(err uintptr) string {
	if err == 0 {
		return ""
	}
	desc := objc.ID(err).Send(selLocalizedDesc)
	return goString(desc)
}

// lsASNExtractParts extracts high and low parts from an LSASN.
func lsASNExtractParts(asn uintptr) (uint32, uint32) {
	var high, low uint32
	fnLSASNExtractHighAndLowParts(asn, &high, &low)
	return high, low
}

// getProcessPID converts an ASN (high/low) to a PID via GetProcessPID.
func getProcessPID(high, low uint32) int32 {
	type psn struct {
		high uint32
		low  uint32
	}
	p := psn{high: high, low: low}
	var pid int32
	ret := fnGetProcessPID(uintptr(unsafe.Pointer(&p)), &pid)
	if ret != 0 {
		return 0
	}
	return pid
}

// pumpRunLoop runs the CF run loop for the given duration.
func pumpRunLoop(d time.Duration) {
	fnCFRunLoopRunInMode(kCFRunLoopDefaultMode, d.Seconds(), false)
}

// URLPath extracts the filesystem path from an NSURL (calls -[NSURL path]).
func URLPath(url objc.ID) string {
	if url == 0 {
		return ""
	}
	path := url.Send(objc.RegisterName("path"))
	return goString(path)
}
