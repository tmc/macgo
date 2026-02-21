package macgo

import (
	"runtime"
	"testing"
)

func TestSetUIModeConstants(t *testing.T) {
	// Verify the UIMode constants have the expected string values.
	tests := []struct {
		mode UIMode
		want string
	}{
		{UIModeBackground, "background"},
		{UIModeAccessory, "accessory"},
		{UIModeRegular, "regular"},
	}
	for _, tt := range tests {
		if string(tt.mode) != tt.want {
			t.Errorf("UIMode %q != expected %q", tt.mode, tt.want)
		}
	}
}

func TestSetUIModeInvalidMode(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("skipping on non-darwin")
	}
	err := SetUIMode(UIMode("invalid"))
	if err == nil {
		t.Error("SetUIMode with invalid mode should return error")
	}
}
