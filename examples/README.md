# macgo Examples

This directory contains functional examples demonstrating various macgo features and use cases. All examples have been tested and verified to build successfully.

## Basic Examples

### getting-started
**Purpose**: Simple introduction to macgo with basic permission requests
**Features**: Camera and microphone permissions, basic setup

### hello
**Purpose**: Minimal "Hello World" example
**Features**: Basic macgo initialization

### hello-files
**Purpose**: File access permissions example
**Features**: User-selected file access, basic file operations

### camera-mic
**Purpose**: Camera and microphone access demonstration
**Features**: TCC permission requests, hardware access testing

## Code Signing Examples

### auto-signed
**Purpose**: Automatic code signing with Developer ID
**Features**: Auto-detection of signing certificates, production signing

### code-signing
**Purpose**: Manual code signing configuration
**Features**: Custom signing identity, entitlements configuration

## Advanced Features

### app-groups
Contains multiple sub-examples for App Groups functionality:
- **app-groups-reader**: Reading shared data from app groups
- **app-groups-verifier**: Verifying app group access
- **app-groups-writer**: Writing data to shared app groups

### app-sandbox
Contains multiple sub-examples for App Sandbox functionality:
- **sandbox-container**: Working with sandbox containers
- **security-scoped-bookmarks**: Using security-scoped bookmarks for persistent file access
- **user-selected-files**: User-selected file access patterns

### sandboxed-file-exec
**Purpose**: File execution within app sandbox
**Features**: Sandboxed execution, file permissions

## System Integration

### desktop-list
**Purpose**: Desktop and window management
**Features**: System information access, desktop interaction

### osascript-wrapper
**Purpose**: AppleScript integration and script management
**Features**: Running AppleScript from Go, script storage and execution

## Building and Running Examples

Each example can be built and run independently:

```bash
# Navigate to any example directory
cd examples/getting-started

# Build the example
go build .

# Run the example
./getting-started
```

For examples with subdirectories (app-groups, app-sandbox), navigate to the specific sub-example:

```bash
cd examples/app-groups/app-groups-writer
go build .
./app-groups-writer
```

## Example Categories

- **Beginner**: getting-started, hello, hello-files
- **Hardware Access**: camera-mic
- **Code Signing**: auto-signed, code-signing
- **Advanced Permissions**: app-groups/*, app-sandbox/*, sandboxed-file-exec
- **System Integration**: desktop-list, osascript-wrapper

## Requirements

- macOS 10.15 or later
- Go 1.21 or later
- Xcode Command Line Tools (for code signing)

## Notes

- Some examples require specific macOS permissions that will be requested at runtime
- Code signing examples may require valid Developer ID certificates
- App Groups examples require proper app group configuration
- AppleScript examples may require Automation permissions for target applications