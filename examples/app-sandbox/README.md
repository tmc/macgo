# App Sandbox Examples

This directory contains examples demonstrating macOS App Sandbox functionality with macgo. App Sandbox provides security isolation for applications while controlling access to system resources and user data.

## Overview

App Sandbox is a security technology that restricts an application's access to system resources. It provides fine-grained control over what an app can access, helping protect user data and system integrity while allowing apps to function properly.

## Examples

### 1. **sandbox-container** - Container Access Demonstration
Shows how sandboxed apps operate within their assigned container directories.

**Key Features:**
- Container directory access (`~/Library/Containers/<bundle-id>/`)
- File operations within sandbox boundaries
- Demonstration of access restrictions
- Directory listing and path resolution

**Usage:**
```bash
cd sandbox-container
go run main.go
```

### 2. **user-selected-files** - User File Selection
Demonstrates how sandboxed apps can access user-selected files with proper permissions.

**Key Features:**
- User-selected files permission (`com.apple.security.files.user-selected.read-write`)
- File access patterns for user-chosen content
- Directory access testing
- Path resolution in sandbox context

**Usage:**
```bash
cd user-selected-files
go run main.go
```

### 3. **security-scoped-bookmarks** - Persistent File Access
Shows the concept of security-scoped bookmarks for retaining file access across app launches.

**Key Features:**
- Security-scoped bookmark concepts
- Persistent file access simulation
- Access lifecycle demonstration
- Bookmark storage patterns

**Usage:**
```bash
cd security-scoped-bookmarks
go run main.go
```

## Configuration

### Basic Sandbox Configuration
All sandbox examples require the sandbox permission:

```go
cfg := &macgo.Config{
    Permissions: []macgo.Permission{
        macgo.Sandbox, // Required for sandbox operation
    },
    AutoSign: true, // Required for entitlements
}
```

### Available Sandbox Permissions

**File Access:**
- `macgo.UserSelectedFiles` - Access to user-selected files
- `macgo.Downloads` - Read/write access to Downloads folder
- `macgo.Pictures` - Read/write access to Pictures folder
- `macgo.Music` - Read/write access to Music folder
- `macgo.Movies` - Read/write access to Movies folder

**Network Access:**
- `macgo.NetworkIncoming` - Incoming network connections
- `macgo.NetworkOutgoing` - Outgoing network connections

**Device Access:**
- `macgo.Camera` - Camera access
- `macgo.Microphone` - Microphone access
- `macgo.Location` - Location services
- `macgo.Bluetooth` - Bluetooth functionality

**System Integration:**
- `macgo.Printing` - Printing capabilities
- `macgo.AddressBook` - Contacts access
- `macgo.Calendar` - Calendar access

## Sandbox Container Structure

When an app runs in sandbox mode, macOS creates a container structure:

```
~/Library/Containers/<bundle-id>/
├── Data/
│   ├── Desktop/          # App's view of Desktop
│   ├── Documents/        # App's view of Documents
│   ├── Downloads/        # App's view of Downloads
│   ├── Library/          # App's private library
│   │   ├── Preferences/  # App preferences
│   │   ├── Caches/      # App caches
│   │   └── ...
│   └── tmp/             # Temporary files
└── .../
```

## File Access Patterns

### 1. Container Access (Always Available)
```go
// App can always access its own container
homeDir, _ := os.UserHomeDir() // Points to container
file := filepath.Join(homeDir, "Documents", "myfile.txt")
os.WriteFile(file, data, 0644) // Always works
```

### 2. User-Selected Files
```go
// Requires user interaction (NSOpenPanel, drag-drop, etc.)
// In practice, file paths come from user selection
// Go code can then access these specific files
selectedFile := "/Users/username/Documents/user-chosen.txt"
data, err := os.ReadFile(selectedFile) // Works if user selected this file
```

### 3. Security-Scoped Bookmarks
```go
// Persistent access to previously user-selected files
// Requires native Cocoa integration for real implementation
// This example shows the conceptual workflow
```

## Permission Requirements

### Development Testing
For development without certificates:
- Apps run with basic sandbox restrictions
- File operations demonstrate access patterns
- Debug output shows permission attempts

### Production Deployment
For production apps with proper code signing:
- Full sandbox enforcement
- Entitlements properly validated
- System dialogs for permission requests
- Proper container isolation

## Common Sandbox Scenarios

### 1. Document-Based Apps
```go
cfg := &macgo.Config{
    Permissions: []macgo.Permission{
        macgo.Sandbox,
        macgo.UserSelectedFiles, // For opening/saving documents
    },
}
```

### 2. Media Apps
```go
cfg := &macgo.Config{
    Permissions: []macgo.Permission{
        macgo.Sandbox,
        macgo.Pictures,  // Access to Photos library
        macgo.Music,     // Access to Music library
        macgo.Movies,    // Access to Movies
    },
}
```

### 3. Network Apps
```go
cfg := &macgo.Config{
    Permissions: []macgo.Permission{
        macgo.Sandbox,
        macgo.NetworkOutgoing, // Make network requests
        macgo.NetworkIncoming, // Accept connections
    },
}
```

## Security Considerations

1. **Principle of Least Privilege**: Only request permissions your app actually needs
2. **User Transparency**: Clearly explain why permissions are needed
3. **Graceful Degradation**: Handle permission denial gracefully
4. **Data Minimization**: Only access the minimum necessary user data
5. **Secure Storage**: Store sensitive data appropriately within container

## Troubleshooting

### Common Issues

1. **Permission Denied Errors**
   - Ensure `macgo.Sandbox` permission is included
   - Check that `AutoSign: true` is set (not `AdHocSign`)
   - Verify required permissions are listed in config

2. **File Access Failures**
   - User-selected files require actual user interaction in production
   - Container paths differ from regular filesystem paths
   - Some directories require specific permissions

3. **Entitlements Not Applied**
   - Must use `AutoSign: true` for entitlements
   - Ad-hoc signing doesn't include entitlements
   - Verify app bundle creation succeeded

### Debug Output
Enable debug mode to see sandbox operations:
```go
cfg := &macgo.Config{
    Debug: true, // Shows sandbox setup and permissions
}
```

## Related Documentation

- [Apple: App Sandbox](https://developer.apple.com/documentation/security/app_sandbox)
- [Apple: Entitlements](https://developer.apple.com/documentation/bundleresources/entitlements)
- [Apple: Security-Scoped Bookmarks](https://developer.apple.com/library/archive/documentation/Security/Conceptual/AppSandboxDesignGuide/AppSandboxInDepth/AppSandboxInDepth.html#//apple_ref/doc/uid/TP40011183-CH3-SW16)
- [Apple: File System Access](https://developer.apple.com/documentation/security/app_sandbox/accessing_files_from_the_macos_app_sandbox)

These examples demonstrate secure file access patterns and sandbox integration for macOS applications using macgo.