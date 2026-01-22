package applicationservices

import (
	"runtime"
	"testing"
)

func TestIsProcessTrusted(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("only runs on darwin")
	}

	// Just verify it doesn't panic - actual result depends on TCC state
	result := IsProcessTrusted()
	t.Logf("IsProcessTrusted() = %v", result)
}
