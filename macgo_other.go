//go:build !darwin

package macgo

import "context"

func startDarwin(_ context.Context, _ *Config) error {
	return nil
}

func writeDoneFile() {}
