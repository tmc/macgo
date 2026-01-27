package launch

import (
	"context"
	"testing"
)

func TestStrategy_String(t *testing.T) {
	tests := []struct {
		strategy Strategy
		want     string
	}{
		{StrategyDirect, "direct"},
		{StrategyServices, "services"},
		{Strategy(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.strategy.String(); got != tt.want {
				t.Errorf("Strategy.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManager_determineStrategy(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   Strategy
	}{
		{
			name: "forced launch services via config",
			config: &Config{
				ForceLaunchServices: true,
			},
			want: StrategyServices,
		},
		{
			name: "forced direct execution via config",
			config: &Config{
				ForceDirectExecution: true,
			},
			want: StrategyDirect,
		},
		{
			name: "camera permission requires services",
			config: &Config{
				Permissions: []string{"camera"},
			},
			want: StrategyServices,
		},
		{
			name: "network permission uses default (services)",
			config: &Config{
				Permissions: []string{"network"},
			},
			want: StrategyServices,
		},
		{
			name: "sandbox permission uses default (services)",
			config: &Config{
				Permissions: []string{"sandbox"},
			},
			want: StrategyServices,
		},
		{
			name: "no permissions uses default (services)",
			config: &Config{
				Permissions: []string{},
			},
			want: StrategyServices,
		},
	}

	manager := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := manager.determineStrategy(tt.config)
			if got != tt.want {
				t.Errorf("Manager.determineStrategy() = %v, want %v", got, tt.want)
			}
		})
	}
}

// MockLauncher implements the Launcher interface for testing.
type MockLauncher struct {
	LaunchFunc func(ctx context.Context, bundlePath, execPath string, cfg *Config) error
	CallCount  int
	LastConfig *Config
}

func (m *MockLauncher) Launch(ctx context.Context, bundlePath, execPath string, cfg *Config) error {
	m.CallCount++
	m.LastConfig = cfg
	if m.LaunchFunc != nil {
		return m.LaunchFunc(ctx, bundlePath, execPath, cfg)
	}
	return nil
}

func TestManager_Launch_DirectStrategy(t *testing.T) {
	mockDirect := &MockLauncher{}
	mockServices := &MockLauncher{}

	manager := NewWithLaunchers(mockDirect, mockServices)

	cfg := &Config{
		Permissions:          []string{"network"},
		Debug:                true,
		ForceDirectExecution: true,
	}

	ctx := context.Background()
	err := manager.Launch(ctx, "/path/to/bundle.app", "/path/to/exec", cfg)

	if err != nil {
		t.Errorf("Manager.Launch() returned error: %v", err)
	}

	if mockDirect.CallCount != 1 {
		t.Errorf("Direct launcher called %d times, want 1", mockDirect.CallCount)
	}

	if mockServices.CallCount != 0 {
		t.Errorf("Services launcher called %d times, want 0", mockServices.CallCount)
	}

	if mockDirect.LastConfig != cfg {
		t.Errorf("Direct launcher received wrong config")
	}
}

func TestManager_Launch_ServicesStrategy(t *testing.T) {
	mockDirect := &MockLauncher{}
	mockServices := &MockLauncher{}

	manager := NewWithLaunchers(mockDirect, mockServices)

	cfg := &Config{
		Permissions: []string{"camera"},
		Debug:       true,
	}

	ctx := context.Background()
	err := manager.Launch(ctx, "/path/to/bundle.app", "/path/to/exec", cfg)

	if err != nil {
		t.Errorf("Manager.Launch() returned error: %v", err)
	}

	if mockDirect.CallCount != 0 {
		t.Errorf("Direct launcher called %d times, want 0", mockDirect.CallCount)
	}

	if mockServices.CallCount != 1 {
		t.Errorf("Services launcher called %d times, want 1", mockServices.CallCount)
	}

	if mockServices.LastConfig != cfg {
		t.Errorf("Services launcher received wrong config")
	}
}

func TestManager_Launch_ForcedStrategy(t *testing.T) {
	tests := []struct {
		name                 string
		forceLaunchServices  bool
		forceDirectExecution bool
		permissions          []string
		wantDirectCalls      int
		wantServicesCalls    int
	}{
		{
			name:                "force launch services overrides permissions",
			forceLaunchServices: true,
			permissions:         []string{"network"}, // Would normally use direct
			wantDirectCalls:     0,
			wantServicesCalls:   1,
		},
		{
			name:                 "force direct execution overrides permissions",
			forceDirectExecution: true,
			permissions:          []string{"camera"}, // Would normally use services
			wantDirectCalls:      1,
			wantServicesCalls:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDirect := &MockLauncher{}
			mockServices := &MockLauncher{}

			manager := NewWithLaunchers(mockDirect, mockServices)

			cfg := &Config{
				Permissions:          tt.permissions,
				ForceLaunchServices:  tt.forceLaunchServices,
				ForceDirectExecution: tt.forceDirectExecution,
			}

			ctx := context.Background()
			err := manager.Launch(ctx, "/path/to/bundle.app", "/path/to/exec", cfg)

			if err != nil {
				t.Errorf("Manager.Launch() returned error: %v", err)
			}

			if mockDirect.CallCount != tt.wantDirectCalls {
				t.Errorf("Direct launcher called %d times, want %d", mockDirect.CallCount, tt.wantDirectCalls)
			}

			if mockServices.CallCount != tt.wantServicesCalls {
				t.Errorf("Services launcher called %d times, want %d", mockServices.CallCount, tt.wantServicesCalls)
			}
		})
	}
}

func TestNew(t *testing.T) {
	manager := New()
	if manager == nil {
		t.Fatal("New() returned nil")
	}

	if manager.directLauncher == nil {
		t.Error("Direct launcher is nil")
	}

	if manager.servicesLauncher == nil {
		t.Error("Services launcher is nil")
	}
}

func TestNewWithLaunchers(t *testing.T) {
	mockDirect := &MockLauncher{}
	mockServices := &MockLauncher{}

	manager := NewWithLaunchers(mockDirect, mockServices)

	if manager == nil {
		t.Fatal("NewWithLaunchers() returned nil")
	}

	if manager.directLauncher != mockDirect {
		t.Error("Direct launcher not set correctly")
	}

	if manager.servicesLauncher != mockServices {
		t.Error("Services launcher not set correctly")
	}
}
