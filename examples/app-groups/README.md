# App Groups Examples

This directory contains examples demonstrating macOS App Groups functionality with macgo. App Groups allow sandboxed applications to share data through secure shared containers.

## Overview

App Groups provide a secure way for multiple apps from the same developer to share data while maintaining sandbox security. In macOS 15 and later, app group containers are protected by System Integrity Protection, ensuring only authorized apps can access shared data.

## Examples

### 1. **app-groups-writer** - Data Writer
Demonstrates creating and writing shared data to an app group container.

**Key Features:**
- Automatic team ID detection and substitution
- Sandbox configuration with app groups
- Shared file creation and status tracking
- Graceful fallback for development environments

**Usage:**
```bash
cd app-groups-writer
go run main.go
```

### 2. **app-groups-reader** - Data Reader
Shows how to read shared data created by another app in the same app group.

**Key Features:**
- Bidirectional app group access
- Reading shared data from other apps
- File listing and status verification
- Cross-app communication demonstration

**Usage:**
```bash
cd app-groups-reader
go run main.go
```

### 3. **app-groups-verifier** - Configuration Verification
Comprehensive tool for testing and validating app groups setup.

**Key Features:**
- Certificate detection and validation
- Bundle creation verification
- Entitlements checking
- Container access testing
- Debug output and troubleshooting

**Usage:**
```bash
cd app-groups-verifier
go run main.go
```

### 4. **team-id-detector** - Automatic Team ID Detection
Demonstrates automatic detection and substitution of Apple Developer Team IDs.

**Key Features:**
- Certificate scanning via `security find-identity`
- Automatic `TEAMID` placeholder replacement
- Developer workflow automation
- Debug logging for troubleshooting

**Usage:**
```bash
cd team-id-detector
go run main.go
```

## Configuration

### App Group Format
macgo supports both iOS-style and macOS-style app group identifiers:

**iOS-Style (Recommended for 2025+):**
```go
AppGroups: []string{
    "group.com.example.shared-data",
}
```

**macOS-Style (Traditional):**
```go
AppGroups: []string{
    "TEAMID.shared-data", // TEAMID gets automatically substituted
}
```

### Required Configuration
All app group examples require:

```go
cfg := &macgo.Config{
    Permissions: []macgo.Permission{
        macgo.Sandbox, // Required for app groups
    },
    AppGroups: []string{
        "TEAMID.shared-data",
    },
    AutoSign: true, // Required for entitlements
}
```

## Production Requirements

For real app group functionality in production:

1. **Developer ID Certificates**: Install Apple Developer ID Application certificates
2. **Code Signing**: Use proper code signing (not ad-hoc)
3. **App Group Registration**: Register app groups in Apple Developer portal (for `group.` format)
4. **Team ID**: Replace `TEAMID` with actual team ID (automatic with certificates)
5. **Provisioning**: Include app group in provisioning profiles

## Development Testing

For development and testing without certificates:

- Apps use fallback directories (`~/macgo-app-groups-demo`)
- `TEAMID` placeholders remain unsubstituted
- Demonstrates file sharing concept
- Debug output shows certificate detection attempts

## Shared Container Location

**Production (with certificates):**
- `~/Library/Group Containers/<group-id>/`

**Development (fallback):**
- `~/macgo-app-groups-demo/`

## Troubleshooting

1. **No entitlements created**: Ensure `AutoSign: true` (not `AdHocSign`)
2. **Container access denied**: Check Developer ID certificates and code signing
3. **TEAMID not substituted**: Verify certificates with `security find-identity -v -p codesigning`
4. **Permission errors**: Enable sandbox: `Permissions: []macgo.Permission{macgo.Sandbox}`

## Related Documentation

- [Apple: Configuring app groups](https://developer.apple.com/documentation/xcode/configuring-app-groups)
- [Apple: App Sandbox Entitlement](https://developer.apple.com/documentation/bundleresources/entitlements/com.apple.security.app-sandbox)
- [Apple: Protecting local app data using containers](https://developer.apple.com/documentation/security/protecting-local-app-data-using-containers)

These examples demonstrate the complete workflow for implementing secure data sharing between macOS applications using App Groups.