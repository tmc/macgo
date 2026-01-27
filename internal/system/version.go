package system

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// MacOSVersion represents a parsed macOS version
type MacOSVersion struct {
	Major int
	Minor int
	Patch int
	Raw   string
}

// String returns the version as a string
func (v MacOSVersion) String() string {
	if v.Patch > 0 {
		return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	}
	if v.Minor > 0 {
		return fmt.Sprintf("%d.%d", v.Major, v.Minor)
	}
	return fmt.Sprintf("%d", v.Major)
}

// ReleaseName returns the marketing name for the macOS version
func (v MacOSVersion) ReleaseName() string {
	switch v.Major {
	case 15:
		return "Sequoia"
	case 14:
		return "Sonoma"
	case 13:
		return "Ventura"
	case 12:
		return "Monterey"
	case 11:
		return "Big Sur"
	case 10:
		if v.Minor >= 15 {
			return "Catalina"
		} else if v.Minor >= 14 {
			return "Mojave"
		} else if v.Minor >= 13 {
			return "High Sierra"
		}
		return "Sierra or earlier"
	default:
		if v.Major > 15 {
			return "Future macOS"
		}
		return "Unknown"
	}
}

// IsAtLeast checks if this version is at least the specified version
func (v MacOSVersion) IsAtLeast(major, minor, patch int) bool {
	if v.Major != major {
		return v.Major > major
	}
	if v.Minor != minor {
		return v.Minor > minor
	}
	return v.Patch >= patch
}

// IsVenturaOrLater returns true if running macOS 13 (Ventura) or later
func (v MacOSVersion) IsVenturaOrLater() bool {
	return v.IsAtLeast(13, 0, 0)
}

// IsSonomaOrLater returns true if running macOS 14 (Sonoma) or later
func (v MacOSVersion) IsSonomaOrLater() bool {
	return v.IsAtLeast(14, 0, 0)
}

// IsSequoiaOrLater returns true if running macOS 15 (Sequoia) or later
func (v MacOSVersion) IsSequoiaOrLater() bool {
	return v.IsAtLeast(15, 0, 0)
}

// UseSystemSettings returns true if this version uses "System Settings"
// instead of "System Preferences" (Ventura and later)
func (v MacOSVersion) UseSystemSettings() bool {
	return v.IsVenturaOrLater()
}

// GetMacOSVersion retrieves the current macOS version
func GetMacOSVersion() (MacOSVersion, error) {
	cmd := exec.Command("sw_vers", "-productVersion")
	output, err := cmd.Output()
	if err != nil {
		return MacOSVersion{}, fmt.Errorf("failed to run sw_vers: %w", err)
	}

	versionStr := strings.TrimSpace(string(output))
	return ParseMacOSVersion(versionStr)
}

// ParseMacOSVersion parses a version string like "14.2.1" or "15.0"
func ParseMacOSVersion(version string) (MacOSVersion, error) {
	result := MacOSVersion{Raw: version}

	parts := strings.Split(version, ".")
	if len(parts) < 1 {
		return result, fmt.Errorf("invalid version format: %s", version)
	}

	// Parse major version
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return result, fmt.Errorf("invalid major version: %w", err)
	}
	result.Major = major

	// Parse minor version (optional)
	if len(parts) > 1 {
		minor, err := strconv.Atoi(parts[1])
		if err != nil {
			return result, fmt.Errorf("invalid minor version: %w", err)
		}
		result.Minor = minor
	}

	// Parse patch version (optional)
	if len(parts) > 2 {
		patch, err := strconv.Atoi(parts[2])
		if err != nil {
			return result, fmt.Errorf("invalid patch version: %w", err)
		}
		result.Patch = patch
	}

	return result, nil
}

// MustGetMacOSVersion retrieves the macOS version and panics on error
// This is useful for initialization code where the version must be available
func MustGetMacOSVersion() MacOSVersion {
	version, err := GetMacOSVersion()
	if err != nil {
		panic(fmt.Sprintf("failed to get macOS version: %v", err))
	}
	return version
}
