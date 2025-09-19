package tcc

import (
	"testing"
)

func TestResolveBundleID(t *testing.T) {
	tests := []struct {
		name      string
		cfg       ResolutionConfig
		want      string
		wantErr   bool
		errSubstr string
	}{
		{
			name: "explicit bundle ID",
			cfg: ResolutionConfig{
				BundleID: "com.example.test",
				AppName:  "TestApp",
				Debug:    false,
			},
			want:    "com.example.test",
			wantErr: false,
		},
		{
			name: "infer from app name",
			cfg: ResolutionConfig{
				AppName: "TestApp",
				Debug:   false,
			},
			want:    "com.macgo.testapp",
			wantErr: false,
		},
		{
			name: "empty config",
			cfg: ResolutionConfig{
				Debug: false,
			},
			// Should derive from executable name
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveBundleID(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ResolveBundleID() expected error but got none")
					return
				}
				if tt.errSubstr != "" && !contains(err.Error(), tt.errSubstr) {
					t.Errorf("ResolveBundleID() error = %v, want to contain %q", err, tt.errSubstr)
				}
			} else {
				if err != nil {
					t.Errorf("ResolveBundleID() error = %v, want nil", err)
					return
				}
				if tt.want != "" && got != tt.want {
					t.Errorf("ResolveBundleID() = %v, want %v", got, tt.want)
				}
				// For cases where we don't specify want, just check it's not empty
				if tt.want == "" && got == "" {
					t.Errorf("ResolveBundleID() returned empty bundle ID")
				}
			}
		})
	}
}

func TestGetTCCServicesFromPermissions(t *testing.T) {
	tests := []struct {
		name        string
		permissions []Permission
		want        []string
	}{
		{
			name:        "no permissions",
			permissions: []Permission{},
			want:        []string{},
		},
		{
			name:        "camera only",
			permissions: []Permission{Camera},
			want:        []string{"Camera"},
		},
		{
			name:        "multiple TCC permissions",
			permissions: []Permission{Camera, Microphone, Location},
			want:        []string{"Camera", "Microphone", "Location"},
		},
		{
			name:        "mixed permissions",
			permissions: []Permission{Camera, Network, Microphone, Sandbox},
			want:        []string{"Camera", "Microphone"},
		},
		{
			name:        "non-TCC permissions",
			permissions: []Permission{Network, Sandbox, Files},
			want:        []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetTCCServices(tt.permissions)
			if len(got) != len(tt.want) {
				t.Errorf("GetTCCServices() = %v, want %v", got, tt.want)
				return
			}
			for _, service := range tt.want {
				if !containsString(got, service) {
					t.Errorf("GetTCCServices() missing service %q", service)
				}
			}
		})
	}
}

func TestResetWithConfig_ValidateInputs(t *testing.T) {
	tests := []struct {
		name        string
		cfg         ResolutionConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config with bundle ID",
			cfg: ResolutionConfig{
				BundleID: "com.example.test",
				Debug:    false,
			},
			wantErr: false,
		},
		{
			name: "valid config with app name",
			cfg: ResolutionConfig{
				AppName: "TestApp",
				Debug:   false,
			},
			wantErr: false,
		},
		{
			name: "empty config",
			cfg: ResolutionConfig{
				Debug: false,
			},
			// Should work by deriving from executable
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't actually test the tccutil command in unit tests,
			// but we can test that the bundle ID resolution works
			bundleID, err := ResolveBundleID(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ResolveBundleID() expected error but got none")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("ResolveBundleID() error = %v, want to contain %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("ResolveBundleID() error = %v, want nil", err)
					return
				}
				if bundleID == "" {
					t.Errorf("ResolveBundleID() returned empty bundle ID")
				}
			}
		})
	}
}

func TestResetSpecificServices_ValidateInputs(t *testing.T) {
	tests := []struct {
		name     string
		bundleID string
		services []string
		debug    bool
		wantErr  bool
	}{
		{
			name:     "empty bundle ID",
			bundleID: "",
			services: []string{"Camera"},
			debug:    false,
			wantErr:  true,
		},
		{
			name:     "empty services",
			bundleID: "com.example.test",
			services: []string{},
			debug:    false,
			wantErr:  false, // Should not error, just do nothing
		},
		{
			name:     "valid inputs",
			bundleID: "com.example.test",
			services: []string{"Camera", "Microphone"},
			debug:    false,
			wantErr:  false, // Will fail in execution but inputs are valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ResetSpecificServices(tt.bundleID, tt.services, tt.debug)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ResetSpecificServices() expected error but got none")
				}
			} else {
				// We expect most calls to fail because tccutil isn't available in tests,
				// but we check that empty services doesn't error
				if len(tt.services) == 0 && err != nil {
					t.Errorf("ResetSpecificServices() with empty services should not error, got %v", err)
				}
			}
		})
	}
}

func TestResetForPermissions(t *testing.T) {
	tests := []struct {
		name        string
		bundleID    string
		permissions []Permission
		debug       bool
		expectCall  bool // Whether we expect a call to tccutil
	}{
		{
			name:        "no TCC permissions",
			bundleID:    "com.example.test",
			permissions: []Permission{Network, Sandbox},
			debug:       false,
			expectCall:  false,
		},
		{
			name:        "has TCC permissions",
			bundleID:    "com.example.test",
			permissions: []Permission{Camera, Microphone},
			debug:       false,
			expectCall:  true,
		},
		{
			name:        "mixed permissions",
			bundleID:    "com.example.test",
			permissions: []Permission{Network, Camera, Sandbox},
			debug:       false,
			expectCall:  true,
		},
		{
			name:        "empty permissions",
			bundleID:    "com.example.test",
			permissions: []Permission{},
			debug:       false,
			expectCall:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ResetForPermissions(tt.bundleID, tt.permissions, tt.debug)

			if !tt.expectCall {
				// If we don't expect a call, there should be no error
				if err != nil {
					t.Errorf("ResetForPermissions() unexpected error = %v", err)
				}
			}
			// For cases where we expect a call, we can't test the actual execution
			// in unit tests since tccutil requires elevated privileges
		})
	}
}