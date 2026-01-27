package system

import (
	"testing"
)

func TestParseMacOSVersion(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantMajor   int
		wantMinor   int
		wantPatch   int
		wantErr     bool
		wantRelease string
	}{
		{
			name:        "Sequoia_15_0",
			input:       "15.0",
			wantMajor:   15,
			wantMinor:   0,
			wantPatch:   0,
			wantRelease: "Sequoia",
		},
		{
			name:        "Sequoia_15_0_1",
			input:       "15.0.1",
			wantMajor:   15,
			wantMinor:   0,
			wantPatch:   1,
			wantRelease: "Sequoia",
		},
		{
			name:        "Sonoma_14_2",
			input:       "14.2",
			wantMajor:   14,
			wantMinor:   2,
			wantPatch:   0,
			wantRelease: "Sonoma",
		},
		{
			name:        "Sonoma_14_2_1",
			input:       "14.2.1",
			wantMajor:   14,
			wantMinor:   2,
			wantPatch:   1,
			wantRelease: "Sonoma",
		},
		{
			name:        "Ventura_13_0",
			input:       "13.0",
			wantMajor:   13,
			wantMinor:   0,
			wantPatch:   0,
			wantRelease: "Ventura",
		},
		{
			name:        "Monterey_12_6",
			input:       "12.6",
			wantMajor:   12,
			wantMinor:   6,
			wantPatch:   0,
			wantRelease: "Monterey",
		},
		{
			name:        "BigSur_11_0",
			input:       "11.0",
			wantMajor:   11,
			wantMinor:   0,
			wantPatch:   0,
			wantRelease: "Big Sur",
		},
		{
			name:        "major_only",
			input:       "15",
			wantMajor:   15,
			wantMinor:   0,
			wantPatch:   0,
			wantRelease: "Sequoia",
		},
		{
			name:    "invalid_empty",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid_format",
			input:   "abc",
			wantErr: true,
		},
		{
			name:    "invalid_minor",
			input:   "15.abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMacOSVersion(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseMacOSVersion(%q) expected error, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseMacOSVersion(%q) unexpected error: %v", tt.input, err)
				return
			}

			if got.Major != tt.wantMajor {
				t.Errorf("Major = %d, want %d", got.Major, tt.wantMajor)
			}
			if got.Minor != tt.wantMinor {
				t.Errorf("Minor = %d, want %d", got.Minor, tt.wantMinor)
			}
			if got.Patch != tt.wantPatch {
				t.Errorf("Patch = %d, want %d", got.Patch, tt.wantPatch)
			}
			if got.Raw != tt.input {
				t.Errorf("Raw = %q, want %q", got.Raw, tt.input)
			}
			if tt.wantRelease != "" && got.ReleaseName() != tt.wantRelease {
				t.Errorf("ReleaseName() = %q, want %q", got.ReleaseName(), tt.wantRelease)
			}
		})
	}
}

func TestMacOSVersion_String(t *testing.T) {
	tests := []struct {
		name  string
		major int
		minor int
		patch int
		want  string
	}{
		{
			name:  "full_version",
			major: 14,
			minor: 2,
			patch: 1,
			want:  "14.2.1",
		},
		{
			name:  "no_patch",
			major: 14,
			minor: 2,
			patch: 0,
			want:  "14.2",
		},
		{
			name:  "major_only",
			major: 15,
			minor: 0,
			patch: 0,
			want:  "15",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := MacOSVersion{
				Major: tt.major,
				Minor: tt.minor,
				Patch: tt.patch,
			}
			got := v.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMacOSVersion_IsAtLeast(t *testing.T) {
	tests := []struct {
		name       string
		version    MacOSVersion
		checkMajor int
		checkMinor int
		checkPatch int
		want       bool
	}{
		{
			name:       "exact_match",
			version:    MacOSVersion{Major: 14, Minor: 2, Patch: 1},
			checkMajor: 14,
			checkMinor: 2,
			checkPatch: 1,
			want:       true,
		},
		{
			name:       "newer_major",
			version:    MacOSVersion{Major: 15, Minor: 0, Patch: 0},
			checkMajor: 14,
			checkMinor: 0,
			checkPatch: 0,
			want:       true,
		},
		{
			name:       "newer_minor",
			version:    MacOSVersion{Major: 14, Minor: 3, Patch: 0},
			checkMajor: 14,
			checkMinor: 2,
			checkPatch: 0,
			want:       true,
		},
		{
			name:       "newer_patch",
			version:    MacOSVersion{Major: 14, Minor: 2, Patch: 2},
			checkMajor: 14,
			checkMinor: 2,
			checkPatch: 1,
			want:       true,
		},
		{
			name:       "older_major",
			version:    MacOSVersion{Major: 13, Minor: 0, Patch: 0},
			checkMajor: 14,
			checkMinor: 0,
			checkPatch: 0,
			want:       false,
		},
		{
			name:       "older_minor",
			version:    MacOSVersion{Major: 14, Minor: 1, Patch: 0},
			checkMajor: 14,
			checkMinor: 2,
			checkPatch: 0,
			want:       false,
		},
		{
			name:       "older_patch",
			version:    MacOSVersion{Major: 14, Minor: 2, Patch: 0},
			checkMajor: 14,
			checkMinor: 2,
			checkPatch: 1,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.version.IsAtLeast(tt.checkMajor, tt.checkMinor, tt.checkPatch)
			if got != tt.want {
				t.Errorf("IsAtLeast(%d, %d, %d) = %v, want %v",
					tt.checkMajor, tt.checkMinor, tt.checkPatch, got, tt.want)
			}
		})
	}
}

func TestMacOSVersion_IsVenturaOrLater(t *testing.T) {
	tests := []struct {
		name    string
		version MacOSVersion
		want    bool
	}{
		{
			name:    "Sequoia",
			version: MacOSVersion{Major: 15, Minor: 0},
			want:    true,
		},
		{
			name:    "Sonoma",
			version: MacOSVersion{Major: 14, Minor: 0},
			want:    true,
		},
		{
			name:    "Ventura",
			version: MacOSVersion{Major: 13, Minor: 0},
			want:    true,
		},
		{
			name:    "Monterey",
			version: MacOSVersion{Major: 12, Minor: 0},
			want:    false,
		},
		{
			name:    "Big_Sur",
			version: MacOSVersion{Major: 11, Minor: 0},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.version.IsVenturaOrLater()
			if got != tt.want {
				t.Errorf("IsVenturaOrLater() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMacOSVersion_UseSystemSettings(t *testing.T) {
	tests := []struct {
		name    string
		version MacOSVersion
		want    bool
	}{
		{
			name:    "Ventura_uses_system_settings",
			version: MacOSVersion{Major: 13, Minor: 0},
			want:    true,
		},
		{
			name:    "Sonoma_uses_system_settings",
			version: MacOSVersion{Major: 14, Minor: 0},
			want:    true,
		},
		{
			name:    "Monterey_uses_system_preferences",
			version: MacOSVersion{Major: 12, Minor: 0},
			want:    false,
		},
		{
			name:    "Big_Sur_uses_system_preferences",
			version: MacOSVersion{Major: 11, Minor: 0},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.version.UseSystemSettings()
			if got != tt.want {
				t.Errorf("UseSystemSettings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMacOSVersion_ReleaseName(t *testing.T) {
	tests := []struct {
		major int
		minor int
		want  string
	}{
		{major: 15, minor: 0, want: "Sequoia"},
		{major: 14, minor: 0, want: "Sonoma"},
		{major: 13, minor: 0, want: "Ventura"},
		{major: 12, minor: 0, want: "Monterey"},
		{major: 11, minor: 0, want: "Big Sur"},
		{major: 10, minor: 15, want: "Catalina"},
		{major: 10, minor: 14, want: "Mojave"},
		{major: 10, minor: 13, want: "High Sierra"},
		{major: 10, minor: 12, want: "Sierra or earlier"},
		{major: 16, minor: 0, want: "Future macOS"},
		{major: 9, minor: 0, want: "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			v := MacOSVersion{Major: tt.major, Minor: tt.minor}
			got := v.ReleaseName()
			if got != tt.want {
				t.Errorf("ReleaseName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetMacOSVersion(t *testing.T) {
	// This is an integration test that requires a real macOS system
	version, err := GetMacOSVersion()
	if err != nil {
		t.Skipf("Skipping integration test: %v", err)
		return
	}

	// Basic sanity checks
	if version.Major < 11 {
		t.Errorf("Expected macOS 11+, got %d", version.Major)
	}

	releaseName := version.ReleaseName()
	if releaseName == "Unknown" {
		t.Errorf("Expected known release name, got %q for version %s", releaseName, version.String())
	}

	t.Logf("Detected macOS %s (%s)", version.String(), releaseName)
}

func TestMacOSVersion_IsSonomaOrLater(t *testing.T) {
	tests := []struct {
		name    string
		version MacOSVersion
		want    bool
	}{
		{
			name:    "Sequoia",
			version: MacOSVersion{Major: 15},
			want:    true,
		},
		{
			name:    "Sonoma",
			version: MacOSVersion{Major: 14},
			want:    true,
		},
		{
			name:    "Ventura",
			version: MacOSVersion{Major: 13},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.version.IsSonomaOrLater()
			if got != tt.want {
				t.Errorf("IsSonomaOrLater() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMacOSVersion_IsSequoiaOrLater(t *testing.T) {
	tests := []struct {
		name    string
		version MacOSVersion
		want    bool
	}{
		{
			name:    "Sequoia",
			version: MacOSVersion{Major: 15},
			want:    true,
		},
		{
			name:    "Sonoma",
			version: MacOSVersion{Major: 14},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.version.IsSequoiaOrLater()
			if got != tt.want {
				t.Errorf("IsSequoiaOrLater() = %v, want %v", got, tt.want)
			}
		})
	}
}

// BenchmarkGetMacOSVersion benchmarks version detection
func BenchmarkGetMacOSVersion(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = GetMacOSVersion()
	}
}

// BenchmarkParseMacOSVersion benchmarks version parsing
func BenchmarkParseMacOSVersion(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = ParseMacOSVersion("14.2.1")
	}
}
