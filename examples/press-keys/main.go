// press-keys: Simulate keyboard events to macOS windows using CGEvent APIs
//
// This tool focuses a target window and sends keyboard input to it.
// Requires Accessibility permission (TCC) to send synthetic keyboard events.
//
// Subcommands:
//
//	send-text  - Type text into a focused window
//	send-key   - Send individual keystrokes with optional modifiers
//
// Usage:
//
//	press-keys send-text [-window <app|title>] "text to type"
//	press-keys send-key [-window <app|title>] <key> [modifiers...]
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

// Focus an application by name using NSRunningApplication
bool focusApp(const char* appName) {
    @autoreleasepool {
        NSString* name = [NSString stringWithUTF8String:appName];
        NSArray<NSRunningApplication*>* apps = [[NSWorkspace sharedWorkspace] runningApplications];

        for (NSRunningApplication* app in apps) {
            if (app.localizedName && [app.localizedName localizedCaseInsensitiveContainsString:name]) {
                return [app activateWithOptions:NSApplicationActivateIgnoringOtherApps];
            }
        }
        return false;
    }
}

// Focus a window by title using CGWindowListCopyWindowInfo + AX API
bool focusWindowByTitle(const char* windowTitle) {
    @autoreleasepool {
        NSString* title = [NSString stringWithUTF8String:windowTitle];

        // Get window list
        CFArrayRef windowList = CGWindowListCopyWindowInfo(
            kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements,
            kCGNullWindowID);

        if (!windowList) return false;

        NSArray* windows = (__bridge_transfer NSArray*)windowList;

        for (NSDictionary* windowInfo in windows) {
            NSString* windowName = windowInfo[(NSString*)kCGWindowName];
            NSNumber* ownerPID = windowInfo[(NSString*)kCGWindowOwnerPID];

            if (windowName && [windowName localizedCaseInsensitiveContainsString:title]) {
                pid_t pid = [ownerPID intValue];

                // Activate the app that owns the window
                NSRunningApplication* app = [NSRunningApplication runningApplicationWithProcessIdentifier:pid];
                if (app) {
                    return [app activateWithOptions:NSApplicationActivateIgnoringOtherApps];
                }
            }
        }

        return false;
    }
}
*/
import "C"
import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/tmc/macgo"
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
	"space":     0x31,
	"return":    0x24,
	"enter":     0x4C,
	"tab":       0x30,
	"escape":    0x35,
	"esc":       0x35,
	"delete":    0x33,
	"backspace": 0x33,
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

func main() {
	if len(os.Args) < 2 {
		showUsage()
		os.Exit(1)
	}

	// Initialize macgo for Accessibility permission
	cfg := macgo.NewConfig().
		WithAppName("press-keys").
		WithCustom("com.apple.security.temporary-exception.apple-events").
		FromEnv()

	if err := macgo.Start(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "macgo: %v\n", err)
		os.Exit(1)
	}
	defer macgo.Cleanup()

	subcmd := os.Args[1]
	switch subcmd {
	case "send-text":
		runSendText(os.Args[2:])
	case "send-key":
		runSendKey(os.Args[2:])
	case "-h", "--help", "help":
		showUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", subcmd)
		showUsage()
		os.Exit(1)
	}
}

func showUsage() {
	fmt.Fprintf(os.Stderr, `press-keys - Send keyboard input to macOS windows

Usage:
  press-keys <subcommand> [options] <args>

Subcommands:
  send-text   Type text into a focused window
  send-key    Send individual keystrokes with optional modifiers

Examples:
  # Type text into the current focused window
  press-keys send-text "Hello, world!"

  # Focus Safari and type a URL
  press-keys send-text -window Safari "https://example.com"

  # Focus a window by title and type
  press-keys send-text -title "Untitled" "some text"

  # Send Enter key
  press-keys send-key enter

  # Send Cmd+S to save
  press-keys send-key -cmd s

  # Send Ctrl+C to Terminal window
  press-keys send-key -window Terminal -ctrl c

  # Send Cmd+Shift+Z (redo) to a specific app
  press-keys send-key -window "TextEdit" -cmd -shift z

Environment Variables:
  MACGO_DEBUG=1     Enable debug output

`)
}

func runSendText(args []string) {
	fs := flag.NewFlagSet("send-text", flag.ExitOnError)
	window := fs.String("window", "", "Focus window by application name")
	title := fs.String("title", "", "Focus window by window title")
	delay := fs.Duration("delay", 30*time.Millisecond, "Delay between keystrokes")
	skipNewline := fs.Bool("skip-newline", false, "Don't send Enter after text")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `send-text - Type text into a focused window

Usage:
  press-keys send-text [options] <text>

Options:
`)
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Examples:
  press-keys send-text "Hello, world!"
  press-keys send-text -window Safari "https://example.com"
  press-keys send-text -title "Document" -skip-newline "partial input"
`)
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Error: text argument required")
		fs.Usage()
		os.Exit(1)
	}

	text := fs.Arg(0)

	// Focus window if specified
	if *window != "" {
		if !focusApp(*window) {
			fmt.Fprintf(os.Stderr, "Warning: could not focus app '%s'\n", *window)
		}
		time.Sleep(100 * time.Millisecond) // Wait for focus
	} else if *title != "" {
		if !focusWindowByTitle(*title) {
			fmt.Fprintf(os.Stderr, "Warning: could not focus window with title '%s'\n", *title)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Type the text
	typeString(text, *delay)

	// Send Enter unless skipped
	if !*skipNewline {
		code := keyMap["return"]
		C.pressKey(code)
	}
}

func runSendKey(args []string) {
	fs := flag.NewFlagSet("send-key", flag.ExitOnError)
	window := fs.String("window", "", "Focus window by application name")
	title := fs.String("title", "", "Focus window by window title")
	cmd := fs.Bool("cmd", false, "Hold Command key")
	shift := fs.Bool("shift", false, "Hold Shift key")
	opt := fs.Bool("opt", false, "Hold Option/Alt key")
	ctrl := fs.Bool("ctrl", false, "Hold Control key")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `send-key - Send individual keystrokes with modifiers

Usage:
  press-keys send-key [options] <key>

Options:
`)
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Supported Keys:
  Letters:     a-z
  Numbers:     0-9
  Special:     enter, return, tab, escape, esc, space, delete, backspace
  Arrows:      up, down, left, right
  Navigation:  home, end, pageup, pagedown
  Function:    f1-f12
  Symbols:     - = [ ] \ ; ' , . / `+"`"+`

Examples:
  press-keys send-key enter
  press-keys send-key -cmd s
  press-keys send-key -window Terminal -ctrl c
  press-keys send-key -cmd -shift z
`)
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Error: key argument required")
		fs.Usage()
		os.Exit(1)
	}

	keyName := fs.Arg(0)

	// Focus window if specified
	if *window != "" {
		if !focusApp(*window) {
			fmt.Fprintf(os.Stderr, "Warning: could not focus app '%s'\n", *window)
		}
		time.Sleep(100 * time.Millisecond)
	} else if *title != "" {
		if !focusWindowByTitle(*title) {
			fmt.Fprintf(os.Stderr, "Warning: could not focus window with title '%s'\n", *title)
		}
		time.Sleep(100 * time.Millisecond)
	}

	pressKey(keyName, *cmd, *shift, *opt, *ctrl)
}

func focusApp(appName string) bool {
	cstr := C.CString(appName)
	defer C.free(C.malloc(0)) // Placeholder - C.free not directly available
	return bool(C.focusApp(cstr))
}

func focusWindowByTitle(title string) bool {
	cstr := C.CString(title)
	return bool(C.focusWindowByTitle(cstr))
}

func typeString(s string, delay time.Duration) {
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
			if os.Getenv("MACGO_DEBUG") == "1" {
				fmt.Fprintf(os.Stderr, "Warning: skipping unknown char '%c'\n", r)
			}
			continue
		}

		C.pressKeyWithModifiers(code, C.bool(false), C.bool(useShift), C.bool(false), C.bool(false))
		time.Sleep(delay)
	}
}

func pressKey(name string, cmd, shift, opt, ctrl bool) {
	code, ok := keyMap[strings.ToLower(name)]
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: unknown key '%s'\n", name)
		os.Exit(1)
	}
	C.pressKeyWithModifiers(code, C.bool(cmd), C.bool(shift), C.bool(opt), C.bool(ctrl))
}
