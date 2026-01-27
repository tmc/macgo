package osa

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework Cocoa -framework ApplicationServices

#import <Foundation/Foundation.h>
#import <Cocoa/Cocoa.h>
#import <ApplicationServices/ApplicationServices.h>

// ExecuteAppleScript runs an AppleScript string and returns the result.
char* execute_applescript(const char* script, char** outResult) {
    @autoreleasepool {
        NSString* scriptStr = [NSString stringWithUTF8String:script];
        NSDictionary* errorInfo = nil;

        NSAppleScript* appleScript = [[NSAppleScript alloc] initWithSource:scriptStr];
        if (!appleScript) {
            return strdup("failed to create NSAppleScript");
        }

        NSAppleEventDescriptor* result = [appleScript executeAndReturnError:&errorInfo];

        if (errorInfo) {
            NSNumber* errorNum = [errorInfo objectForKey:NSAppleScriptErrorNumber];
            NSString* errorMsg = [errorInfo objectForKey:NSAppleScriptErrorMessage];

            NSString* fullError;
            if (errorNum && errorMsg) {
                fullError = [NSString stringWithFormat:@"AppleScript error %@: %@",
                    errorNum, errorMsg];
            } else if (errorMsg) {
                fullError = errorMsg;
            } else {
                fullError = @"unknown AppleScript error";
            }

            return strdup([fullError UTF8String]);
        }

        if (result && outResult) {
            NSString* resultStr = [result stringValue];
            if (resultStr) {
                *outResult = strdup([resultStr UTF8String]);
            } else {
                *outResult = NULL;
            }
        }

        return NULL; // Success
    }
}
*/
import "C"
import (
	"embed"
	"fmt"
	"os/exec"
	"strings"
	"unsafe"
)

var scriptsFS embed.FS

// SetScriptsFS sets the file system for loading embedded scripts
func SetScriptsFS(fs embed.FS) {
	scriptsFS = fs
}

// RunScript loads an AppleScript from the embedded FS and executes it.
// It performs text replacements based on the provided map.
func RunScript(name string, replacements map[string]string) (string, error) {
	data, err := scriptsFS.ReadFile("applescripts/" + name)
	if err != nil {
		return "", fmt.Errorf("failed to read script %s: %w", name, err)
	}
	scriptContent := string(data)

	// Perform replacements
	for key, value := range replacements {
		scriptContent = strings.ReplaceAll(scriptContent, key, value)
	}

	// Try in-process execution first
	res, err := Execute(scriptContent)
	if err == nil {
		return res, nil
	}

	// Debug error string
	errMsg := err.Error()
	// fmt.Printf("DEBUG: In-process error: %q\n", errMsg)

	// If in-process fails with authorization error, fallback to osascript CLI
	// -1743: Not authorized to send Apple events
	// -2741: Expected end of line but found class name
	if strings.Contains(errMsg, "-1743") || strings.Contains(errMsg, "-2741") {
		fmt.Printf("macgo: Falling back to osascript CLI due to error: %v\n", err)
		return ExecuteCLI(scriptContent)
	}

	return "", err
}

// ExecuteCLI runs AppleScript via the osascript command line tool.
// This is useful as a fallback when in-process execution is blocked by TCC.
func ExecuteCLI(script string) (string, error) {
	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("osascript CLI failed: %v\nOutput: %s", err, out)
	}
	return strings.TrimSpace(string(out)), nil
}

// Execute runs an AppleScript string in-process.
func Execute(script string) (string, error) {
	cScript := C.CString(script)
	defer C.free(unsafe.Pointer(cScript))

	var cResult *C.char

	cErr := C.execute_applescript(cScript, &cResult)
	if cErr != nil {
		errStr := C.GoString(cErr)
		C.free(unsafe.Pointer(cErr))
		return "", fmt.Errorf("%s", errStr)
	}

	var result string
	if cResult != nil {
		result = C.GoString(cResult)
		C.free(unsafe.Pointer(cResult))
	}

	return result, nil
}
