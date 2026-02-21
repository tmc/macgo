//go:build !darwin

package macgo

import "fmt"

// SetUIMode is only supported on macOS.
func SetUIMode(mode UIMode) error {
	return fmt.Errorf("SetUIMode is only supported on macOS")
}
