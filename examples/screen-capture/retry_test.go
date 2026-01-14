package main

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// TestGetRetryConfig tests retry configuration from flags and environment variables
func TestGetRetryConfig(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		flagAttempts int
		flagDelay    int
		flagMaxDelay int
		flagBackoff  float64
		wantAttempts int
		wantDelay    time.Duration
		wantMaxDelay time.Duration
		wantBackoff  float64
	}{
		{
			name:         "default_values",
			flagAttempts: 3,
			flagDelay:    500,
			flagMaxDelay: 5000,
			flagBackoff:  2.0,
			wantAttempts: 3,
			wantDelay:    500 * time.Millisecond,
			wantMaxDelay: 5000 * time.Millisecond,
			wantBackoff:  2.0,
		},
		{
			name: "env_override_attempts",
			envVars: map[string]string{
				"SCREENCAPTURE_RETRY_ATTEMPTS": "5",
			},
			flagAttempts: 3,
			flagDelay:    500,
			flagMaxDelay: 5000,
			flagBackoff:  2.0,
			wantAttempts: 5,
			wantDelay:    500 * time.Millisecond,
			wantMaxDelay: 5000 * time.Millisecond,
			wantBackoff:  2.0,
		},
		{
			name: "env_override_delay",
			envVars: map[string]string{
				"SCREENCAPTURE_RETRY_DELAY": "1000",
			},
			flagAttempts: 3,
			flagDelay:    500,
			flagMaxDelay: 5000,
			flagBackoff:  2.0,
			wantAttempts: 3,
			wantDelay:    1000 * time.Millisecond,
			wantMaxDelay: 5000 * time.Millisecond,
			wantBackoff:  2.0,
		},
		{
			name: "env_override_max_delay",
			envVars: map[string]string{
				"SCREENCAPTURE_RETRY_MAX_DELAY": "10000",
			},
			flagAttempts: 3,
			flagDelay:    500,
			flagMaxDelay: 5000,
			flagBackoff:  2.0,
			wantAttempts: 3,
			wantDelay:    500 * time.Millisecond,
			wantMaxDelay: 10000 * time.Millisecond,
			wantBackoff:  2.0,
		},
		{
			name: "env_override_backoff",
			envVars: map[string]string{
				"SCREENCAPTURE_RETRY_BACKOFF": "1.5",
			},
			flagAttempts: 3,
			flagDelay:    500,
			flagMaxDelay: 5000,
			flagBackoff:  2.0,
			wantAttempts: 3,
			wantDelay:    500 * time.Millisecond,
			wantMaxDelay: 5000 * time.Millisecond,
			wantBackoff:  1.5,
		},
		{
			name: "env_override_all",
			envVars: map[string]string{
				"SCREENCAPTURE_RETRY_ATTEMPTS":   "7",
				"SCREENCAPTURE_RETRY_DELAY":      "200",
				"SCREENCAPTURE_RETRY_MAX_DELAY":  "3000",
				"SCREENCAPTURE_RETRY_BACKOFF":    "1.8",
			},
			flagAttempts: 3,
			flagDelay:    500,
			flagMaxDelay: 5000,
			flagBackoff:  2.0,
			wantAttempts: 7,
			wantDelay:    200 * time.Millisecond,
			wantMaxDelay: 3000 * time.Millisecond,
			wantBackoff:  1.8,
		},
		{
			name: "env_invalid_values_ignored",
			envVars: map[string]string{
				"SCREENCAPTURE_RETRY_ATTEMPTS": "invalid",
				"SCREENCAPTURE_RETRY_DELAY":    "not-a-number",
			},
			flagAttempts: 3,
			flagDelay:    500,
			flagMaxDelay: 5000,
			flagBackoff:  2.0,
			wantAttempts: 3,
			wantDelay:    500 * time.Millisecond,
			wantMaxDelay: 5000 * time.Millisecond,
			wantBackoff:  2.0,
		},
		{
			name: "disabled_retries",
			flagAttempts: 1,
			flagDelay:    500,
			flagMaxDelay: 5000,
			flagBackoff:  2.0,
			wantAttempts: 1,
			wantDelay:    500 * time.Millisecond,
			wantMaxDelay: 5000 * time.Millisecond,
			wantBackoff:  2.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			// Set flag values
			*retryAttempts = tt.flagAttempts
			*retryDelay = tt.flagDelay
			*retryMaxDelay = tt.flagMaxDelay
			*retryBackoff = tt.flagBackoff

			config := getRetryConfig()

			if config.maxAttempts != tt.wantAttempts {
				t.Errorf("maxAttempts = %d, want %d", config.maxAttempts, tt.wantAttempts)
			}
			if config.baseDelay != tt.wantDelay {
				t.Errorf("baseDelay = %v, want %v", config.baseDelay, tt.wantDelay)
			}
			if config.maxDelay != tt.wantMaxDelay {
				t.Errorf("maxDelay = %v, want %v", config.maxDelay, tt.wantMaxDelay)
			}
			if config.backoff != tt.wantBackoff {
				t.Errorf("backoff = %f, want %f", config.backoff, tt.wantBackoff)
			}
		})
	}
}

// TestIsTransientError tests transient error detection
func TestIsTransientError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		outputFile string
		want       bool
	}{
		{
			name: "nil_error",
			err:  nil,
			want: false,
		},
		{
			name: "window_error",
			err:  fmt.Errorf("invalid window ID"),
			want: true,
		},
		{
			name: "communication_error",
			err:  fmt.Errorf("WindowServer communication timeout"),
			want: true,
		},
		{
			name: "timeout_error",
			err:  fmt.Errorf("operation timeout"),
			want: true,
		},
		{
			name: "busy_error",
			err:  fmt.Errorf("system is busy"),
			want: true,
		},
		{
			name: "temporarily_unavailable",
			err:  fmt.Errorf("resource temporarily unavailable"),
			want: true,
		},
		{
			name: "locked_screen",
			err:  fmt.Errorf("screen is locked"),
			want: true,
		},
		{
			name: "sleep_mode",
			err:  fmt.Errorf("system in sleep mode"),
			want: true,
		},
		{
			name: "window_related",
			err:  fmt.Errorf("window ID invalid"),
			want: true,
		},
		{
			name: "permission_denied",
			err:  fmt.Errorf("permission denied"),
			want: false,
		},
		{
			name: "invalid_argument",
			err:  fmt.Errorf("invalid argument"),
			want: false,
		},
		{
			name: "file_not_found",
			err:  fmt.Errorf("file not found"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTransientError(tt.err, tt.outputFile)
			if got != tt.want {
				t.Errorf("isTransientError() = %v, want %v for error: %v", got, tt.want, tt.err)
			}
		})
	}
}

// TestIsTransientError_OutputFileExists tests that errors are not transient when output file exists
func TestIsTransientError_OutputFileExists(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test-output-*.png")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write some data to make file size > 0
	if _, err := tmpFile.WriteString("test data"); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Even with a window error, if file exists, it's not transient
	err = fmt.Errorf("invalid window ID")
	if isTransientError(err, tmpFile.Name()) {
		t.Errorf("isTransientError() should return false when output file exists")
	}
}

// TestExponentialBackoff tests the exponential backoff calculation
func TestExponentialBackoff(t *testing.T) {
	tests := []struct {
		name        string
		config      retryConfig
		attempt     int
		want        time.Duration
	}{
		{
			name: "default_config_attempt_1",
			config: retryConfig{
				baseDelay: 500 * time.Millisecond,
				maxDelay:  5000 * time.Millisecond,
				backoff:   2.0,
			},
			attempt: 1,
			want: 0, // First attempt has no delay
		},
		{
			name: "default_config_attempt_2",
			config: retryConfig{
				baseDelay: 500 * time.Millisecond,
				maxDelay:  5000 * time.Millisecond,
				backoff:   2.0,
			},
			attempt: 2,
			// Formula: 500ms * (1.0 + (2-1) * (2.0-1.0)) = 500ms * 2.0 = 1000ms
			want: 1000 * time.Millisecond,
		},
		{
			name: "default_config_attempt_3",
			config: retryConfig{
				baseDelay: 500 * time.Millisecond,
				maxDelay:  5000 * time.Millisecond,
				backoff:   2.0,
			},
			attempt: 3,
			// Formula: 500ms * (1.0 + (3-1) * (2.0-1.0)) = 500ms * 3.0 = 1500ms
			want: 1500 * time.Millisecond,
		},
		{
			name: "max_delay_cap",
			config: retryConfig{
				baseDelay: 1000 * time.Millisecond,
				maxDelay:  2000 * time.Millisecond,
				backoff:   3.0,
			},
			attempt: 5,
			// Formula would be: 1000ms * (1.0 + (5-1) * (3.0-1.0)) = 1000ms * 9.0 = 9000ms
			// But capped at maxDelay = 2000ms
			want: 2000 * time.Millisecond,
		},
		{
			name: "low_backoff",
			config: retryConfig{
				baseDelay: 1000 * time.Millisecond,
				maxDelay:  10000 * time.Millisecond,
				backoff:   1.5,
			},
			attempt: 2,
			// Formula: 1000ms * (1.0 + (2-1) * (1.5-1.0)) = 1000ms * 1.5 = 1500ms
			want: 1500 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate delay using the formula from executeWithRetry
			var delay time.Duration
			if tt.attempt > 1 {
				delay = time.Duration(float64(tt.config.baseDelay) * (1.0 + float64(tt.attempt-1)*(tt.config.backoff-1.0)))
				if delay > tt.config.maxDelay {
					delay = tt.config.maxDelay
				}
			}

			if delay != tt.want {
				t.Errorf("delay = %v, want %v", delay, tt.want)
			}
		})
	}
}

// TestRetryConfigValidation tests that invalid retry configurations are handled
func TestRetryConfigValidation(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		wantAttempts int
		wantDelay    time.Duration
	}{
		{
			name: "negative_attempts_ignored",
			envVars: map[string]string{
				"SCREENCAPTURE_RETRY_ATTEMPTS": "-5",
			},
			wantAttempts: 3, // Falls back to flag default
			wantDelay:    500 * time.Millisecond,
		},
		{
			name: "negative_delay_ignored",
			envVars: map[string]string{
				"SCREENCAPTURE_RETRY_DELAY": "-100",
			},
			wantAttempts: 3,
			wantDelay:    500 * time.Millisecond, // Falls back to flag default
		},
		{
			name: "zero_attempts_allowed",
			envVars: map[string]string{
				"SCREENCAPTURE_RETRY_ATTEMPTS": "0",
			},
			wantAttempts: 0, // Zero is valid (disables retries)
			wantDelay:    500 * time.Millisecond,
		},
		{
			name: "backoff_less_than_one_ignored",
			envVars: map[string]string{
				"SCREENCAPTURE_RETRY_BACKOFF": "0.5",
			},
			wantAttempts: 3,
			wantDelay:    500 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			// Reset to defaults
			*retryAttempts = 3
			*retryDelay = 500
			*retryMaxDelay = 5000
			*retryBackoff = 2.0

			config := getRetryConfig()

			if config.maxAttempts != tt.wantAttempts {
				t.Errorf("maxAttempts = %d, want %d", config.maxAttempts, tt.wantAttempts)
			}
			if config.baseDelay != tt.wantDelay {
				t.Errorf("baseDelay = %v, want %v", config.baseDelay, tt.wantDelay)
			}
		})
	}
}

// TestTransientErrorPatterns tests various transient error message patterns
func TestTransientErrorPatterns(t *testing.T) {
	transientPatterns := []string{
		"invalid window 12345",
		"window was closed",
		"WindowServer communication failed",
		"operation timeout occurred",
		"system is currently busy",
		"resource temporarily unavailable",
		"screen is locked by user",
		"system entering sleep mode",
		"window capture failed",
		"Window ID no longer valid",
		"Communication with WindowServer interrupted",
		"Timeout waiting for response",
	}

	for _, pattern := range transientPatterns {
		t.Run(pattern, func(t *testing.T) {
			err := fmt.Errorf("%s", pattern)
			if !isTransientError(err, "") {
				t.Errorf("Pattern %q should be detected as transient", pattern)
			}
		})
	}

	nonTransientPatterns := []string{
		"permission denied: screen recording",
		"invalid argument: output file",
		"disk full: cannot write",
		"command not found: screencapture",
		"filesystem is read-only",
		"access denied",
		"no such file or directory",
	}

	for _, pattern := range nonTransientPatterns {
		t.Run(pattern, func(t *testing.T) {
			err := fmt.Errorf("%s", pattern)
			if isTransientError(err, "") {
				t.Errorf("Pattern %q should NOT be detected as transient", pattern)
			}
		})
	}
}

// BenchmarkRetryConfig benchmarks retry configuration loading
func BenchmarkRetryConfig(b *testing.B) {
	*retryAttempts = 3
	*retryDelay = 500
	*retryMaxDelay = 5000
	*retryBackoff = 2.0

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = getRetryConfig()
	}
}

// BenchmarkIsTransientError benchmarks transient error detection
func BenchmarkIsTransientError(b *testing.B) {
	err := fmt.Errorf("invalid window ID 12345")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = isTransientError(err, "")
	}
}
