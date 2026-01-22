//go:build cgo

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreGraphics -framework Foundation

#include <CoreGraphics/CoreGraphics.h>
#include <Foundation/Foundation.h>

// WindowInfoC represents a window's information in C
typedef struct {
    int32_t windowID;
    int32_t ownerPID;
    uint32_t displayID;
    char ownerName[256];
    CGRect bounds;
} WindowInfoC;

// GetDisplayForWindow returns the display ID that contains the window's center point
uint32_t GetDisplayForWindow(CGRect bounds) {
    CGPoint center = CGPointMake(bounds.origin.x + bounds.size.width / 2,
                                  bounds.origin.y + bounds.size.height / 2);

    uint32_t displayCount;
    CGDirectDisplayID displays[32];
    CGGetDisplaysWithPoint(center, 32, displays, &displayCount);

    if (displayCount > 0) {
        return displays[0];
    }

    // Fallback to main display if point isn't on any display
    return CGMainDisplayID();
}

// GetWindowList retrieves all windows and returns them as an array
// includeOffscreen: 0 for on-screen only, 1 for all windows including minimized/hidden
WindowInfoC* GetWindowList(int* count, int includeOffscreen) {
    CGWindowListOption options = kCGWindowListExcludeDesktopElements;
    if (!includeOffscreen) {
        options |= kCGWindowListOptionOnScreenOnly;
    }
    CFArrayRef windowList = CGWindowListCopyWindowInfo(options, kCGNullWindowID);

    if (windowList == NULL) {
        *count = 0;
        return NULL;
    }

    CFIndex windowCount = CFArrayGetCount(windowList);
    WindowInfoC* windows = (WindowInfoC*)malloc(sizeof(WindowInfoC) * windowCount);

    int validCount = 0;
    for (CFIndex i = 0; i < windowCount; i++) {
        CFDictionaryRef window = (CFDictionaryRef)CFArrayGetValueAtIndex(windowList, i);

        // Get window ID
        CFNumberRef windowIDRef = (CFNumberRef)CFDictionaryGetValue(window, kCGWindowNumber);
        int32_t windowID = 0;
        if (windowIDRef) {
            CFNumberGetValue(windowIDRef, kCFNumberSInt32Type, &windowID);
        }

        // Get owner PID
        CFNumberRef ownerPIDRef = (CFNumberRef)CFDictionaryGetValue(window, kCGWindowOwnerPID);
        int32_t ownerPID = 0;
        if (ownerPIDRef) {
            CFNumberGetValue(ownerPIDRef, kCFNumberSInt32Type, &ownerPID);
        }

        // Get owner name
        CFStringRef ownerNameRef = (CFStringRef)CFDictionaryGetValue(window, kCGWindowOwnerName);
        if (ownerNameRef) {
            CFStringGetCString(ownerNameRef, windows[validCount].ownerName, 256, kCFStringEncodingUTF8);
        } else {
            windows[validCount].ownerName[0] = '\0';
        }

        // Get bounds
        CFDictionaryRef boundsRef = (CFDictionaryRef)CFDictionaryGetValue(window, kCGWindowBounds);
        CGRect bounds = CGRectZero;
        if (boundsRef) {
            CGRectMakeWithDictionaryRepresentation(boundsRef, &bounds);
        }

        windows[validCount].windowID = windowID;
        windows[validCount].ownerPID = ownerPID;
        windows[validCount].displayID = GetDisplayForWindow(bounds);
        windows[validCount].bounds = bounds;
        validCount++;
    }

    CFRelease(windowList);
    *count = validCount;
    return windows;
}

void FreeWindowList(WindowInfoC* windows) {
    if (windows) {
        free(windows);
    }
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

func getWindowList(includeOffscreen bool) ([]WindowInfo, error) {
	var count C.int
	var includeOffscreenFlag C.int
	if includeOffscreen {
		includeOffscreenFlag = 1
	}
	windowList := C.GetWindowList(&count, includeOffscreenFlag)
	if windowList == nil {
		return nil, fmt.Errorf("failed to get window list from Core Graphics")
	}
	

	// Convert C array to Go slice
	windows := make([]WindowInfo, int(count))
	for i := 0; i < int(count); i++ {
		cWindow := (*C.WindowInfoC)(unsafe.Pointer(uintptr(unsafe.Pointer(windowList)) + uintptr(i)*unsafe.Sizeof(*windowList)))
		windows[i] = WindowInfo{
			WindowID:  int32(cWindow.windowID),
			OwnerPID:  int32(cWindow.ownerPID),
			OwnerName: C.GoString(&cWindow.ownerName[0]),
			DisplayID: uint32(cWindow.displayID),
			X:         float64(cWindow.bounds.origin.x),
			Y:         float64(cWindow.bounds.origin.y),
			Width:     float64(cWindow.bounds.size.width),
			Height:    float64(cWindow.bounds.size.height),
		}
	}

	return windows, nil
}
