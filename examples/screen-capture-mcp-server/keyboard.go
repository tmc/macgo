//go:build cgo

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreGraphics -framework Carbon -framework AppKit -framework Foundation

#include <CoreGraphics/CoreGraphics.h>
#include <Carbon/Carbon.h>
#include <AppKit/AppKit.h>
#include <unistd.h>

void pressKey(CGKeyCode keyCode) {
    CGEventRef keyDown = CGEventCreateKeyboardEvent(NULL, keyCode, true);
    CGEventPost(kCGHIDEventTap, keyDown);
    CFRelease(keyDown);

    CGEventRef keyUp = CGEventCreateKeyboardEvent(NULL, keyCode, false);
    CGEventPost(kCGHIDEventTap, keyUp);
    CFRelease(keyUp);
}

void pressKeyWithModifiers(CGKeyCode keyCode, bool command, bool shift, bool opt, bool ctrl) {
    CGEventRef keyDown = CGEventCreateKeyboardEvent(NULL, keyCode, true);

    CGEventFlags flags = 0;
    if (command) flags |= kCGEventFlagMaskCommand;
    if (shift) flags |= kCGEventFlagMaskShift;
    if (opt) flags |= kCGEventFlagMaskAlternate;
    if (ctrl) flags |= kCGEventFlagMaskControl;

    CGEventSetFlags(keyDown, flags);
    CGEventPost(kCGHIDEventTap, keyDown);
    CFRelease(keyDown);

    CGEventRef keyUp = CGEventCreateKeyboardEvent(NULL, keyCode, false);
    CGEventSetFlags(keyUp, flags);
    CGEventPost(kCGHIDEventTap, keyUp);
    CFRelease(keyUp);
}
*/
import "C"
import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Key codes from Carbon/HIToolbox/Events.h
var keyMap = map[string]C.CGKeyCode{
	"a": 0x00, "s": 0x01, "d": 0x02, "f": 0x03, "h": 0x04, "g": 0x05, "z": 0x06, "x": 0x07,
	"c": 0x08, "v": 0x09, "b": 0x0B, "q": 0x0C, "w": 0x0D, "e": 0x0E, "r": 0x0F,
	"y": 0x10, "t": 0x11, "1": 0x12, "2": 0x13, "3": 0x14, "4": 0x15, "6": 0x16, "5": 0x17,
	"=": 0x18, "9": 0x19, "7": 0x1A, "-": 0x1B, "8": 0x1C, "0": 0x1D, "]": 0x1E, "o": 0x1F,
	"u": 0x20, "[": 0x21, "i": 0x22, "p": 0x23, "l": 0x25, "j": 0x26, "'": 0x27,
	"k": 0x28, ";": 0x29, "\\": 0x2A, ",": 0x2B, "/": 0x2C, "n": 0x2D, "m": 0x2E, ".": 0x2F,
	"`": 0x32,

	// Special keys
	"space":          0x31,
	"return":         0x24,
	"enter":          0x4C,
	"tab":            0x30,
	"escape":         0x35,
	"esc":            0x35,
	"delete":         0x33,
	"backspace":      0x33,
	"forward-delete": 0x75,

	// Arrow keys
	"left":  0x7B,
	"right": 0x7C,
	"down":  0x7D,
	"up":    0x7E,

	// Navigation
	"home":     0x73,
	"end":      0x77,
	"pageup":   0x74,
	"pagedown": 0x79,

	// Function keys
	"f1":  0x7A,
	"f2":  0x78,
	"f3":  0x63,
	"f4":  0x76,
	"f5":  0x60,
	"f6":  0x61,
	"f7":  0x62,
	"f8":  0x64,
	"f9":  0x65,
	"f10": 0x6D,
	"f11": 0x67,
	"f12": 0x6F,
}

func typeString(s string) error {
	// Default delay resembling fast typing
	delay := 10 * time.Millisecond

	for _, r := range s {
		char := string(r)

		useShift := false
		if r >= 'A' && r <= 'Z' {
			useShift = true
			char = strings.ToLower(char)
		}

		// Map symbols that need shift (US layout)
		if strings.ContainsRune("!@#$%^&*()_+{}|:\"<>?~", r) {
			useShift = true
			switch r {
			case '!':
				char = "1"
			case '@':
				char = "2"
			case '#':
				char = "3"
			case '$':
				char = "4"
			case '%':
				char = "5"
			case '^':
				char = "6"
			case '&':
				char = "7"
			case '*':
				char = "8"
			case '(':
				char = "9"
			case ')':
				char = "0"
			case '_':
				char = "-"
			case '+':
				char = "="
			case '{':
				char = "["
			case '}':
				char = "]"
			case '|':
				char = "\\"
			case ':':
				char = ";"
			case '"':
				char = "'"
			case '<':
				char = ","
			case '>':
				char = "."
			case '?':
				char = "/"
			case '~':
				char = "`"
			}
		}

		// Handle space
		if r == ' ' {
			char = "space"
		}

		code, ok := keyMap[char]
		if !ok {
			fmt.Fprintf(os.Stderr, "Warning: skipping unknown char '%c'\n", r)
			continue
		}

		C.pressKeyWithModifiers(code, C.bool(false), C.bool(useShift), C.bool(false), C.bool(false))
		time.Sleep(delay)
	}
	return nil
}

func pressKey(name string, cmd, shift, opt, ctrl bool) error {
	code, ok := keyMap[strings.ToLower(name)]
	if !ok {
		return fmt.Errorf("unknown key '%s'", name)
	}
	C.pressKeyWithModifiers(code, C.bool(cmd), C.bool(shift), C.bool(opt), C.bool(ctrl))
	return nil
}
