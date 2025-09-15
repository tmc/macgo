package macgo

import "github.com/tmc/misc/macgo/signal"

// EnableAutoInit is deprecated and no longer has any effect.
// For new code, use macgo.Start() explicitly.
//
// Example with auto package:
//
//	import (
//	    _ "github.com/tmc/misc/macgo/auto"
//	)

// EnableImprovedSignalHandling sets up the improved signal handling
// for better process control, especially for Ctrl+C handling.
// This approach uses named pipes for IO redirection and proper signal forwarding.
//
// This is equivalent to importing the signalhandler package:
//
//	import _ "github.com/tmc/misc/macgo/auto/sandbox/signalhandler"
//
// Example usage:
//
//	func init() {
//	    // Enable improved signal handling
//	    macgo.EnableImprovedSignalHandling()
//
//	    // Configure macgo with your desired permissions
//	    macgo.RequestEntitlements(macgo.EntCamera, macgo.EntMicrophone)
//	}
//
//	func main() {
//	    // Start macgo with improved signal handling
//	    macgo.Start()
//	}
func EnableImprovedSignalHandling() {
	// Use IO redirection with robust signal handling
	debugf("macgo: enabling improved signal handling with IO redirection")
	SetReLaunchFunction(improvedSignalAdapter)
}

// improvedSignalAdapter adapts signal handling to the ReLaunchFunction signature
func improvedSignalAdapter(appPath, execPath string, args []string) {
	debugf("macgo: using improved signal handling (adapter)")
	// Use the consolidated signal package for robust signal handling
	signal.RelaunchWithRobustSignalHandling(appPath, execPath, args)
}
