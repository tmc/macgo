package macgo

import (
	"runtime"
	"testing"
)

// TestCrossPlatformCompatibility verifies that macgo functions work gracefully on all platforms
func TestCrossPlatformCompatibility(t *testing.T) {
	// These functions should not panic on any platform
	t.Run("Basic API functions", func(t *testing.T) {
		// Configuration functions should work without panicking
		RequestEntitlements("com.apple.security.device.camera")
		RequestEntitlement("com.apple.security.device.microphone")
		EnableSigning("")
		SetAppName("TestApp")
		SetBundleID("com.example.test")
		EnableDockIcon()
		DisableRelaunch()
		EnableDebug()

		// These should return sensible values on all platforms
		inBundle := IsInAppBundle()
		if runtime.GOOS != "darwin" {
			// On non-Darwin platforms, should always return false
			if inBundle {
				t.Error("IsInAppBundle() should return false on non-Darwin platforms")
			}
		}

		// Start should not panic on any platform
		Start()

		// Stop should not panic
		Stop()
	})

	t.Run("Configuration functions", func(t *testing.T) {
		cfg := NewConfig()
		if cfg == nil {
			t.Error("NewConfig() should never return nil")
		}

		// Configuration methods should not panic
		cfg.RequestEntitlements("com.apple.security.app-sandbox")
		cfg.AddEntitlement("com.apple.security.device.camera")
		cfg.AddPlistEntry("TestKey", "TestValue")
	})
}

// TestPlatformSpecificBehavior tests platform-specific behavior
func TestPlatformSpecificBehavior(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Log("Running on Darwin - macgo should be fully functional")
		// On Darwin, functions should work normally
		// (actual app bundle creation would require integration tests)
	} else {
		t.Logf("Running on %s - macgo should be no-op", runtime.GOOS)
		// On non-Darwin, functions should be no-ops but not panic

		// Test that Start() is a no-op
		Start()

		// Test that IsInAppBundle() returns false
		if IsInAppBundle() {
			t.Error("IsInAppBundle() should return false on non-Darwin platforms")
		}
	}
}

// TestConcurrentPlatformAccess tests that concurrent access to macgo functions is safe
func TestConcurrentPlatformAccess(t *testing.T) {
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	// Test concurrent access to configuration functions
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Goroutine %d panicked: %v", id, r)
				}
				done <- true
			}()

			// These should be safe to call concurrently
			RequestEntitlements("com.apple.security.device.camera")
			SetAppName("ConcurrentTest")
			IsInAppBundle()
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}
