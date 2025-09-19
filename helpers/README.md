# macgo/helpers

The `helpers` package provides utility functions for macgo that are useful for external users. This package exposes key functionality from macgo's internal packages, making it easy to work with macOS app bundle creation, code signing, team ID detection, permission management, and validation utilities.

## Installation

```bash
go get github.com/tmc/misc/macgo/helpers
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/tmc/misc/macgo/helpers"
)

func main() {
    // Detect team ID from certificates
    teamID, err := helpers.DetectTeamID()
    if err != nil {
        log.Printf("Team ID detection failed: %v", err)
    } else {
        fmt.Printf("Team ID: %s\n", teamID)
    }

    // Generate a bundle ID
    bundleID := helpers.InferBundleID("MyApp")
    fmt.Printf("Bundle ID: %s\n", bundleID)

    // Validate permissions
    perms := []helpers.Permission{helpers.Camera, helpers.Microphone}
    if err := helpers.ValidatePermissions(perms); err != nil {
        log.Fatal(err)
    }

    // Get entitlements for permissions
    entitlements := helpers.GetEntitlements(perms)
    fmt.Printf("Entitlements: %v\n", entitlements)
}
```

## Features

### Team ID Detection and Management

- `DetectTeamID()` - Automatically detect Apple Developer Team ID from certificates
- `IsValidTeamID(teamID)` - Validate team ID format
- `SubstituteTeamIDInGroups(groups, teamID)` - Replace TEAMID placeholders in app groups
- `AutoSubstituteTeamIDInGroups(groups)` - Detect and substitute team ID automatically

### Bundle ID and App Name Utilities

- `InferBundleID(appName)` - Generate bundle ID from Go module information
- `ValidateBundleID(bundleID)` - Validate bundle ID format and conventions
- `CleanAppName(name)` - Remove problematic characters from app names
- `ExtractAppNameFromPath(execPath)` - Extract app name from executable path
- `SanitizeBundleID(bundleID)` - Clean and normalize bundle ID format

### Code Signing Utilities

- `FindDeveloperID()` - Find Developer ID Application certificate
- `ListAvailableIdentities()` - List all available code signing identities
- `ValidateCodeSignIdentity(identity)` - Validate code signing identity
- `VerifySignature(bundlePath)` - Verify app bundle code signature
- `GetSignatureInfo(bundlePath)` - Get detailed signature information
- `HasDeveloperIDCertificate()` - Check if Developer ID certificate is available

### Permission Management

- `ValidatePermissions(perms)` - Validate permission combinations
- `GetEntitlements(perms)` - Get entitlements for permissions
- `RequiresTCC(perms)` - Check if permissions require TCC dialogs
- `GetTCCServices(perms)` - Get TCC service names for permission reset
- `AllPermissions()` - Get list of all available permissions
- `PermissionDescription(perm)` - Get human-readable permission description

### App Groups Validation

- `ValidateAppGroups(groups, perms)` - Validate app group configuration

## Available Permissions

- `helpers.Camera` - Camera access
- `helpers.Microphone` - Microphone access
- `helpers.Location` - Location services
- `helpers.Files` - File system access with user selection
- `helpers.Network` - Network client access
- `helpers.Sandbox` - App sandbox with restricted file access

## Example Usage

See the [helpers-demo example](../examples/helpers-demo/) for a comprehensive demonstration of all functionality.

## Integration with macgo

The helpers package is designed to work alongside the main macgo package. You can use helpers for advanced configuration and validation, then use the results with macgo:

```go
// Use helpers to prepare configuration
teamID, _, err := helpers.AutoSubstituteTeamIDInGroups(appGroups)
if err != nil {
    log.Fatal(err)
}

bundleID := helpers.InferBundleID("MyApp")
if err := helpers.ValidateBundleID(bundleID); err != nil {
    log.Fatal(err)
}

// Use with macgo
cfg := &macgo.Config{
    AppName:   "MyApp",
    BundleID:  bundleID,
    AppGroups: appGroups,
    Permissions: []macgo.Permission{macgo.Camera, macgo.Microphone},
}

err = macgo.Start(cfg)
```

## Documentation

For detailed API documentation, run:

```bash
go doc github.com/tmc/misc/macgo/helpers
```

Or view online at [pkg.go.dev](https://pkg.go.dev/github.com/tmc/misc/macgo/helpers).