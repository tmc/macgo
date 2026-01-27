# ScreenCaptureKit Purego Example

This example demonstrates how to use macOS ScreenCaptureKit framework via purego for direct Objective-C runtime access, without using cgo or higher-level bindings like darwinkit.

## Overview

This example replicates the functionality of the [darwinkit screencapturekit example](https://github.com/progrium/darwinkit/tree/main/macos/_examples/screencapturekit) but uses purego for direct access to the Objective-C runtime and ScreenCaptureKit framework.

## Features

- Direct Objective-C runtime access via purego
- Creates a native macOS GUI application
- Demonstrates ScreenCaptureKit framework loading
- Shows basic UI construction with AppKit classes
- Pure Go implementation without cgo

## Building

```bash
go build
```

## Running

```bash
./screencapture-purego
```

## Implementation Notes

### Purego vs Darwinkit

This example uses purego's lower-level Objective-C runtime bindings directly, whereas the darwinkit example uses higher-level generated bindings. The tradeoffs are:

**Purego approach (this example):**
- More verbose code
- Direct control over Objective-C message sends
- No code generation required
- Smaller dependency footprint
- Requires manual selector registration

**Darwinkit approach:**
- More Go-idiomatic APIs
- Type-safe method calls
- Automatic memory management helpers
- Generated from framework headers
- Easier to use for complex scenarios

### Limitations

This is a simplified demonstration that shows:
- Framework loading via purego
- Basic AppKit UI creation
- Objective-C class and selector usage

For a full implementation with working screen capture, completion handlers, and delegates, see the darwinkit example.

### Key Differences from Darwinkit Example

1. **Framework Loading**: Direct `purego.Dlopen()` instead of implicit loading
2. **Class Access**: `objc.GetClass()` instead of pre-generated class wrappers
3. **Method Calls**: `Send()` with registered selectors instead of Go methods
4. **Memory Management**: Manual selector registration and NSRect handling
5. **Type Safety**: Runtime checks instead of compile-time type safety

## Requirements

- macOS 12.3+ (ScreenCaptureKit was introduced in macOS 12.3)
- Go 1.23+
- Screen Recording permission (macOS will prompt when needed)

## References

- [ScreenCaptureKit Documentation](https://developer.apple.com/documentation/screencapturekit)
- [Purego](https://github.com/ebitengine/purego)
- [Darwinkit ScreenCaptureKit Example](https://github.com/progrium/darwinkit/tree/main/macos/_examples/screencapturekit)
