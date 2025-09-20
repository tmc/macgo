package system

import (
	"os"
	"strconv"
	"strings"
)

// Environment variable constants for macgo
const (
	// Core configuration
	EnvAppName    = "MACGO_APP_NAME"
	EnvBundleID   = "MACGO_BUNDLE_ID"
	EnvDebug      = "MACGO_DEBUG"
	EnvKeepBundle = "MACGO_KEEP_BUNDLE"
	EnvVersion    = "MACGO_VERSION"

	// Code signing
	EnvCodeSignIdentity = "MACGO_CODE_SIGN_IDENTITY"
	EnvAutoSign         = "MACGO_AUTO_SIGN"
	EnvAdHocSign        = "MACGO_AD_HOC_SIGN"

	// Launch behavior
	EnvNoRelaunch           = "MACGO_NO_RELAUNCH"
	EnvForceLaunchServices  = "MACGO_FORCE_LAUNCH_SERVICES"
	EnvForceDirectExecution = "MACGO_FORCE_DIRECT"

	// Permission flags
	EnvCamera     = "MACGO_CAMERA"
	EnvMicrophone = "MACGO_MICROPHONE"
	EnvLocation   = "MACGO_LOCATION"
	EnvFiles      = "MACGO_FILES"
	EnvNetwork    = "MACGO_NETWORK"
	EnvSandbox    = "MACGO_SANDBOX"

	// TCC and permissions
	EnvResetPermissions = "MACGO_RESET_PERMISSIONS"

	// Testing and development
	EnvTestIntegration = "MACGO_TEST_INTEGRATION"
)

// GetBool returns the boolean value of an environment variable.
// Returns true if the variable is set to "1", "true", "yes", or "on" (case-insensitive).
// Returns false otherwise.
func GetBool(key string) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

// GetString returns the string value of an environment variable.
// Returns the defaultValue if the variable is not set or empty.
func GetString(key, defaultValue string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	return value
}

// GetInt returns the integer value of an environment variable.
// Returns the defaultValue if the variable is not set, empty, or cannot be parsed.
func GetInt(key string, defaultValue int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return intValue
}

// GetStringSlice returns a slice of strings from an environment variable.
// The value should be comma-separated. Empty values are filtered out.
func GetStringSlice(key string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	var result []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}

	return result
}

// SetBool sets an environment variable to a boolean value.
// Sets to "1" for true, "0" for false.
func SetBool(key string, value bool) error {
	if value {
		return os.Setenv(key, "1")
	}
	return os.Setenv(key, "0")
}

// IsDebugEnabled checks if debug mode is enabled via environment variable.
func IsDebugEnabled() bool {
	return GetBool(EnvDebug)
}

// IsRelaunchDisabled checks if app bundle relaunch is disabled.
func IsRelaunchDisabled() bool {
	return GetBool(EnvNoRelaunch)
}

// IsResetPermissionsEnabled checks if TCC permission reset is enabled.
func IsResetPermissionsEnabled() bool {
	return GetBool(EnvResetPermissions)
}

// GetLaunchMode determines the launch mode based on environment variables.
// Returns "launch_services", "direct", or "auto" (default).
func GetLaunchMode() string {
	if GetBool(EnvForceLaunchServices) {
		return "launch_services"
	}
	if GetBool(EnvForceDirectExecution) {
		return "direct"
	}
	return "auto"
}

// GetPermissionFlags returns a map of permission names to their enabled status.
func GetPermissionFlags() map[string]bool {
	return map[string]bool{
		"camera":     GetBool(EnvCamera),
		"microphone": GetBool(EnvMicrophone),
		"location":   GetBool(EnvLocation),
		"files":      GetBool(EnvFiles),
		"network":    GetBool(EnvNetwork),
		"sandbox":    GetBool(EnvSandbox),
	}
}

// HasAnyPermissionFlags returns true if any permission environment variables are set.
func HasAnyPermissionFlags() bool {
	flags := GetPermissionFlags()
	for _, enabled := range flags {
		if enabled {
			return true
		}
	}
	return false
}

// GetEnabledPermissions returns a slice of permission names that are enabled via environment variables.
func GetEnabledPermissions() []string {
	var enabled []string
	flags := GetPermissionFlags()

	for permission, isEnabled := range flags {
		if isEnabled {
			enabled = append(enabled, permission)
		}
	}

	return enabled
}

// SaveEnv saves current environment variable values for later restoration.
// Returns a map that can be passed to RestoreEnv.
func SaveEnv(keys []string) map[string]string {
	saved := make(map[string]string)
	for _, key := range keys {
		saved[key] = os.Getenv(key)
	}
	return saved
}

// RestoreEnv restores environment variables from a saved state.
// Use with SaveEnv for temporary environment modifications.
func RestoreEnv(saved map[string]string) {
	for key, value := range saved {
		if value == "" {
			_ = os.Unsetenv(key)
		} else {
			_ = os.Setenv(key, value)
		}
	}
}

// AllMacgoEnvVars returns a list of all known macgo environment variables.
func AllMacgoEnvVars() []string {
	return []string{
		EnvAppName,
		EnvBundleID,
		EnvDebug,
		EnvKeepBundle,
		EnvVersion,
		EnvCodeSignIdentity,
		EnvAutoSign,
		EnvAdHocSign,
		EnvNoRelaunch,
		EnvForceLaunchServices,
		EnvForceDirectExecution,
		EnvCamera,
		EnvMicrophone,
		EnvLocation,
		EnvFiles,
		EnvNetwork,
		EnvSandbox,
		EnvResetPermissions,
		EnvTestIntegration,
	}
}

// ClearAllMacgoEnv removes all macgo environment variables.
// Useful for testing to ensure clean state.
func ClearAllMacgoEnv() {
	for _, key := range AllMacgoEnvVars() {
		_ = os.Unsetenv(key)
	}
}

// GetMacgoEnvSnapshot returns a snapshot of all current macgo environment variables.
func GetMacgoEnvSnapshot() map[string]string {
	return SaveEnv(AllMacgoEnvVars())
}
