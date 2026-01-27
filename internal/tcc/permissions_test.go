package tcc

import (
	"testing"
)

func TestValidatePermissions(t *testing.T) {
	tests := []struct {
		name        string
		permissions []Permission
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid permissions",
			permissions: []Permission{Camera, Microphone, Location},
			wantErr:     false,
		},
		{
			name:        "empty permissions",
			permissions: []Permission{},
			wantErr:     false,
		},
		{
			name:        "duplicate permissions",
			permissions: []Permission{Camera, Camera, Microphone},
			wantErr:     false, // Duplicates are allowed and ignored
		},
		{
			name:        "unknown permission",
			permissions: []Permission{"unknown"},
			wantErr:     true,
			errContains: "unknown permission",
		},
		{
			name:        "mixed valid and invalid",
			permissions: []Permission{Camera, "invalid", Microphone},
			wantErr:     true,
			errContains: "unknown permission",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePermissions(tt.permissions)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidatePermissions() expected error but got none")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("ValidatePermissions() error = %v, want to contain %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("ValidatePermissions() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestValidateAppGroups(t *testing.T) {
	tests := []struct {
		name        string
		groups      []string
		permissions []Permission
		wantErr     bool
		errContains string
	}{
		{
			name:        "no app groups",
			groups:      []string{},
			permissions: []Permission{Camera},
			wantErr:     false,
		},
		{
			name:        "valid app groups with sandbox",
			groups:      []string{"group.com.example.shared"},
			permissions: []Permission{Sandbox},
			wantErr:     false,
		},
		{
			name:        "app groups without sandbox",
			groups:      []string{"group.com.example.shared"},
			permissions: []Permission{Camera},
			wantErr:     true,
			errContains: "app groups require sandbox permission",
		},
		{
			name:        "invalid group format",
			groups:      []string{"com.example.shared"},
			permissions: []Permission{Sandbox},
			wantErr:     true,
			errContains: "must start with 'group.'",
		},
		{
			name:        "too short group ID",
			groups:      []string{"group."},
			permissions: []Permission{Sandbox},
			wantErr:     true,
			errContains: "too short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAppGroups(tt.groups, tt.permissions)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateAppGroups() expected error but got none")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateAppGroups() error = %v, want to contain %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateAppGroups() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestGetEntitlements(t *testing.T) {
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
			name:        "single permission",
			permissions: []Permission{Camera},
			want:        []string{"com.apple.security.device.camera"},
		},
		{
			name:        "multiple permissions",
			permissions: []Permission{Camera, Microphone, Location},
			want: []string{
				"com.apple.security.device.camera",
				"com.apple.security.device.microphone",
				"com.apple.security.personal-information.location",
			},
		},
		{
			name:        "duplicate permissions",
			permissions: []Permission{Camera, Camera},
			want:        []string{"com.apple.security.device.camera"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetEntitlements(tt.permissions)
			if len(got) != len(tt.want) {
				t.Errorf("GetEntitlements() = %v, want %v", got, tt.want)
				return
			}
			for _, entitlement := range tt.want {
				if !containsString(got, entitlement) {
					t.Errorf("GetEntitlements() missing entitlement %q", entitlement)
				}
			}
		})
	}
}

func TestRequiresTCC(t *testing.T) {
	tests := []struct {
		name        string
		permissions []Permission
		want        bool
	}{
		{
			name:        "no permissions",
			permissions: []Permission{},
			want:        false,
		},
		{
			name:        "non-TCC permissions",
			permissions: []Permission{Network, Sandbox},
			want:        false,
		},
		{
			name:        "TCC permissions",
			permissions: []Permission{Camera, Microphone},
			want:        true,
		},
		{
			name:        "screen recording permission",
			permissions: []Permission{ScreenRecording},
			want:        true,
		},
		{
			name:        "mixed permissions",
			permissions: []Permission{Network, Camera, Sandbox},
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RequiresTCC(tt.permissions)
			if got != tt.want {
				t.Errorf("RequiresTCC() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetTCCServices(t *testing.T) {
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
			name:        "non-TCC permissions",
			permissions: []Permission{Network, Sandbox},
			want:        []string{},
		},
		{
			name:        "TCC permissions",
			permissions: []Permission{Camera, Microphone, Location},
			want:        []string{"Camera", "Microphone", "Location"},
		},
		{
			name:        "screen recording permission",
			permissions: []Permission{ScreenRecording},
			want:        []string{"ScreenCapture"},
		},
		{
			name:        "duplicate permissions",
			permissions: []Permission{Camera, Camera},
			want:        []string{"Camera"},
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

// Helper functions
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}()))
}

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
