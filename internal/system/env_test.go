package system

import (
	"os"
	"testing"
)

func TestGetBool(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
		want  bool
	}{
		{
			name:  "true value 1",
			key:   "TEST_BOOL_1",
			value: "1",
			want:  true,
		},
		{
			name:  "true value true",
			key:   "TEST_BOOL_TRUE",
			value: "true",
			want:  true,
		},
		{
			name:  "true value yes",
			key:   "TEST_BOOL_YES",
			value: "yes",
			want:  true,
		},
		{
			name:  "true value on",
			key:   "TEST_BOOL_ON",
			value: "on",
			want:  true,
		},
		{
			name:  "true value uppercase",
			key:   "TEST_BOOL_UPPER",
			value: "TRUE",
			want:  true,
		},
		{
			name:  "false value 0",
			key:   "TEST_BOOL_0",
			value: "0",
			want:  false,
		},
		{
			name:  "false value false",
			key:   "TEST_BOOL_FALSE",
			value: "false",
			want:  false,
		},
		{
			name:  "false value empty",
			key:   "TEST_BOOL_EMPTY",
			value: "",
			want:  false,
		},
		{
			name:  "false value random",
			key:   "TEST_BOOL_RANDOM",
			value: "random",
			want:  false,
		},
		{
			name:  "unset variable",
			key:   "TEST_BOOL_UNSET",
			value: "", // Will not be set
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up first
			os.Unsetenv(tt.key)

			// Set the environment variable if value is not empty or if we want to test empty
			if tt.value != "" || tt.name == "false value empty" {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}

			got := GetBool(tt.key)
			if got != tt.want {
				t.Errorf("GetBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetString(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue string
		want         string
	}{
		{
			name:         "existing value",
			key:          "TEST_STRING_EXIST",
			value:        "test_value",
			defaultValue: "default",
			want:         "test_value",
		},
		{
			name:         "empty value returns default",
			key:          "TEST_STRING_EMPTY",
			value:        "",
			defaultValue: "default",
			want:         "default",
		},
		{
			name:         "unset value returns default",
			key:          "TEST_STRING_UNSET",
			value:        "", // Will not be set
			defaultValue: "default",
			want:         "default",
		},
		{
			name:         "whitespace value",
			key:          "TEST_STRING_SPACE",
			value:        "  test  ",
			defaultValue: "default",
			want:         "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up first
			os.Unsetenv(tt.key)

			// Set the environment variable if we want to test non-unset cases
			if tt.name != "unset value returns default" {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}

			got := GetString(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("GetString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue int
		want         int
	}{
		{
			name:         "valid integer",
			key:          "TEST_INT_VALID",
			value:        "42",
			defaultValue: 0,
			want:         42,
		},
		{
			name:         "invalid integer returns default",
			key:          "TEST_INT_INVALID",
			value:        "not_a_number",
			defaultValue: 100,
			want:         100,
		},
		{
			name:         "empty value returns default",
			key:          "TEST_INT_EMPTY",
			value:        "",
			defaultValue: 200,
			want:         200,
		},
		{
			name:         "unset value returns default",
			key:          "TEST_INT_UNSET",
			value:        "", // Will not be set
			defaultValue: 300,
			want:         300,
		},
		{
			name:         "negative integer",
			key:          "TEST_INT_NEGATIVE",
			value:        "-10",
			defaultValue: 0,
			want:         -10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up first
			os.Unsetenv(tt.key)

			// Set the environment variable if we want to test non-unset cases
			if tt.name != "unset value returns default" {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}

			got := GetInt(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("GetInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetStringSlice(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
		want  []string
	}{
		{
			name:  "single value",
			key:   "TEST_SLICE_SINGLE",
			value: "value1",
			want:  []string{"value1"},
		},
		{
			name:  "multiple values",
			key:   "TEST_SLICE_MULTI",
			value: "value1,value2,value3",
			want:  []string{"value1", "value2", "value3"},
		},
		{
			name:  "values with spaces",
			key:   "TEST_SLICE_SPACES",
			value: " value1 , value2 , value3 ",
			want:  []string{"value1", "value2", "value3"},
		},
		{
			name:  "empty value",
			key:   "TEST_SLICE_EMPTY",
			value: "",
			want:  nil,
		},
		{
			name:  "empty components filtered",
			key:   "TEST_SLICE_FILTER",
			value: "value1,,value2,,",
			want:  []string{"value1", "value2"},
		},
		{
			name:  "unset variable",
			key:   "TEST_SLICE_UNSET",
			value: "", // Will not be set
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up first
			os.Unsetenv(tt.key)

			// Set the environment variable if we want to test non-unset cases
			if tt.name != "unset variable" {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}

			got := GetStringSlice(tt.key)
			if !equalStringSlices(got, tt.want) {
				t.Errorf("GetStringSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetBool(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value bool
		want  string
	}{
		{
			name:  "set true",
			key:   "TEST_SET_TRUE",
			value: true,
			want:  "1",
		},
		{
			name:  "set false",
			key:   "TEST_SET_FALSE",
			value: false,
			want:  "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer os.Unsetenv(tt.key)

			err := SetBool(tt.key, tt.value)
			if err != nil {
				t.Errorf("SetBool() error = %v", err)
				return
			}

			got := os.Getenv(tt.key)
			if got != tt.want {
				t.Errorf("SetBool() set %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsDebugEnabled(t *testing.T) {
	defer os.Unsetenv(EnvDebug)

	// Test enabled
	os.Setenv(EnvDebug, "1")
	if !IsDebugEnabled() {
		t.Errorf("IsDebugEnabled() = false, want true")
	}

	// Test disabled
	os.Setenv(EnvDebug, "0")
	if IsDebugEnabled() {
		t.Errorf("IsDebugEnabled() = true, want false")
	}

	// Test unset
	os.Unsetenv(EnvDebug)
	if IsDebugEnabled() {
		t.Errorf("IsDebugEnabled() = true, want false when unset")
	}
}

func TestIsRelaunchDisabled(t *testing.T) {
	defer os.Unsetenv(EnvNoRelaunch)

	// Test enabled (disabled relaunch)
	os.Setenv(EnvNoRelaunch, "1")
	if !IsRelaunchDisabled() {
		t.Errorf("IsRelaunchDisabled() = false, want true")
	}

	// Test disabled (enabled relaunch)
	os.Setenv(EnvNoRelaunch, "0")
	if IsRelaunchDisabled() {
		t.Errorf("IsRelaunchDisabled() = true, want false")
	}

	// Test unset
	os.Unsetenv(EnvNoRelaunch)
	if IsRelaunchDisabled() {
		t.Errorf("IsRelaunchDisabled() = true, want false when unset")
	}
}

func TestGetLaunchMode(t *testing.T) {
	// Clean up first
	os.Unsetenv(EnvForceLaunchServices)
	os.Unsetenv(EnvForceDirectExecution)
	defer func() {
		os.Unsetenv(EnvForceLaunchServices)
		os.Unsetenv(EnvForceDirectExecution)
	}()

	tests := []struct {
		name            string
		launchServices  string
		directExecution string
		want            string
	}{
		{
			name:           "default mode",
			launchServices: "",
			directExecution: "",
			want:           "auto",
		},
		{
			name:           "force launch services",
			launchServices: "1",
			directExecution: "",
			want:           "launch_services",
		},
		{
			name:           "force direct execution",
			launchServices: "",
			directExecution: "1",
			want:           "direct",
		},
		{
			name:           "both set - launch services wins",
			launchServices: "1",
			directExecution: "1",
			want:           "launch_services",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up
			os.Unsetenv(EnvForceLaunchServices)
			os.Unsetenv(EnvForceDirectExecution)

			// Set environment variables
			if tt.launchServices != "" {
				os.Setenv(EnvForceLaunchServices, tt.launchServices)
			}
			if tt.directExecution != "" {
				os.Setenv(EnvForceDirectExecution, tt.directExecution)
			}

			got := GetLaunchMode()
			if got != tt.want {
				t.Errorf("GetLaunchMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetPermissionFlags(t *testing.T) {
	// Clean up all permission environment variables
	permissionEnvs := []string{
		EnvCamera, EnvMicrophone, EnvLocation,
		EnvFiles, EnvNetwork, EnvSandbox,
	}
	for _, env := range permissionEnvs {
		os.Unsetenv(env)
	}
	defer func() {
		for _, env := range permissionEnvs {
			os.Unsetenv(env)
		}
	}()

	// Set some permissions
	os.Setenv(EnvCamera, "1")
	os.Setenv(EnvMicrophone, "1")
	os.Setenv(EnvNetwork, "0") // Explicitly disabled
	// Leave others unset

	flags := GetPermissionFlags()

	expectedFlags := map[string]bool{
		"camera":     true,
		"microphone": true,
		"location":   false,
		"files":      false,
		"network":    false,
		"sandbox":    false,
	}

	for permission, expected := range expectedFlags {
		if flags[permission] != expected {
			t.Errorf("GetPermissionFlags()[%s] = %v, want %v", permission, flags[permission], expected)
		}
	}
}

func TestGetEnabledPermissions(t *testing.T) {
	// Clean up all permission environment variables
	permissionEnvs := []string{
		EnvCamera, EnvMicrophone, EnvLocation,
		EnvFiles, EnvNetwork, EnvSandbox,
	}
	for _, env := range permissionEnvs {
		os.Unsetenv(env)
	}
	defer func() {
		for _, env := range permissionEnvs {
			os.Unsetenv(env)
		}
	}()

	// Set some permissions
	os.Setenv(EnvCamera, "1")
	os.Setenv(EnvFiles, "1")

	enabled := GetEnabledPermissions()
	expected := []string{"camera", "files"}

	if len(enabled) != len(expected) {
		t.Errorf("GetEnabledPermissions() length = %v, want %v", len(enabled), len(expected))
		return
	}

	for _, exp := range expected {
		if !containsString(enabled, exp) {
			t.Errorf("GetEnabledPermissions() missing %s", exp)
		}
	}
}

func TestSaveAndRestoreEnv(t *testing.T) {
	keys := []string{"TEST_ENV_1", "TEST_ENV_2", "TEST_ENV_3"}

	// Set initial values
	os.Setenv("TEST_ENV_1", "initial1")
	os.Setenv("TEST_ENV_2", "initial2")
	// Leave TEST_ENV_3 unset

	// Save current state
	saved := SaveEnv(keys)

	// Modify environment
	os.Setenv("TEST_ENV_1", "modified1")
	os.Setenv("TEST_ENV_2", "modified2")
	os.Setenv("TEST_ENV_3", "modified3")

	// Verify changes
	if os.Getenv("TEST_ENV_1") != "modified1" {
		t.Errorf("Expected TEST_ENV_1 to be modified")
	}

	// Restore environment
	RestoreEnv(saved)

	// Verify restoration
	if os.Getenv("TEST_ENV_1") != "initial1" {
		t.Errorf("TEST_ENV_1 not restored correctly: got %q, want %q", os.Getenv("TEST_ENV_1"), "initial1")
	}
	if os.Getenv("TEST_ENV_2") != "initial2" {
		t.Errorf("TEST_ENV_2 not restored correctly: got %q, want %q", os.Getenv("TEST_ENV_2"), "initial2")
	}
	if os.Getenv("TEST_ENV_3") != "" {
		t.Errorf("TEST_ENV_3 should be empty after restore: got %q", os.Getenv("TEST_ENV_3"))
	}

	// Clean up
	for _, key := range keys {
		os.Unsetenv(key)
	}
}

func TestClearAllMacgoEnv(t *testing.T) {
	// Set some macgo environment variables
	os.Setenv(EnvDebug, "1")
	os.Setenv(EnvCamera, "1")
	os.Setenv(EnvAppName, "TestApp")

	// Clear all
	ClearAllMacgoEnv()

	// Verify they're cleared
	if os.Getenv(EnvDebug) != "" {
		t.Errorf("EnvDebug not cleared")
	}
	if os.Getenv(EnvCamera) != "" {
		t.Errorf("EnvCamera not cleared")
	}
	if os.Getenv(EnvAppName) != "" {
		t.Errorf("EnvAppName not cleared")
	}
}

func TestGetMacgoEnvSnapshot(t *testing.T) {
	// Set some macgo environment variables
	os.Setenv(EnvDebug, "1")
	os.Setenv(EnvCamera, "1")
	defer func() {
		os.Unsetenv(EnvDebug)
		os.Unsetenv(EnvCamera)
	}()

	snapshot := GetMacgoEnvSnapshot()

	// Verify snapshot contains our variables
	if snapshot[EnvDebug] != "1" {
		t.Errorf("Snapshot missing EnvDebug value")
	}
	if snapshot[EnvCamera] != "1" {
		t.Errorf("Snapshot missing EnvCamera value")
	}

	// Verify snapshot contains all expected keys
	allKeys := AllMacgoEnvVars()
	for _, key := range allKeys {
		if _, exists := snapshot[key]; !exists {
			t.Errorf("Snapshot missing key: %s", key)
		}
	}
}

// Helper functions
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	if (a == nil) != (b == nil) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}