# lsregister-tool

A simple wrapper around macOS's `lsregister` utility (`/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister`) for managing the Launch Services database.

## Usage

```bash
# Build the tool
go build

# Dump the Launch Services database
./lsregister-tool dump

# Search for an application (by name, ID, or path)
./lsregister-tool search "TextEdit"

# List all registered applications
./lsregister-tool list

# List all registered applications (as JSON)
./lsregister-tool list -json

# Show detailed info for an application
./lsregister-tool info "/System/Applications/TextEdit.app"

# Register an application (force update)
# Options: -r (recursive), -lazy <seconds>
./lsregister-tool register -r /Applications/MyApp.app

# Unregister an application
./lsregister-tool unregister /Applications/MyApp.app

# Check for plist errors
./lsregister-tool lint

# Garbage collect the database
./lsregister-tool gc

# Rescan default locations (seed)
./lsregister-tool seed

# Reset/Delete the database (requires confirmation and reboot)
./lsregister-tool reset
```

## Options

These options apply to `register`, `unregister`, and often `dump`:

*   `-v`: Verbose output
*   `-lazy N`: Sleep for N seconds before processing
*   `-r`: Recursive directory scan (descends into packages and invisible directories)

### Domains
Target specific domains (default is usually user+local+system depending on context):
*   `-user`: User domain (`~/Applications`)
*   `-local`: Local domain (`/Applications`)
*   `-system`: System domain (`/System/Applications`)
*   `-network`: Network domain

### Types
Filter by item type (default is `-apps`):
*   `-apps`: Application bundles
*   `-libs`: Libraries
*   `-all`: All types

## Examples

**Rebuild the database (safest method):**
```bash
./lsregister-tool seed -r -domain local -domain system -domain user
```
*Note: Using `reset` (which uses `-delete`) is drastic and requires a reboot.*

**Register all apps in /Applications recursively:**
```bash
./lsregister-tool register -r -local -system /Applications
```
