// Permission Reset Test - macgo
// Tests the MACGO_RESET_PERMISSIONS functionality
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/tmc/misc/macgo"
)

func main() {
	fmt.Println("Permission Reset Test - macgo")
	fmt.Println("============================")
	fmt.Println()

	// Enable debug output to see what's happening
	cfg := &macgo.Config{
		AppName: "PermissionResetTest",
		Debug:   true,
		Permissions: []macgo.Permission{
			macgo.Camera,
			macgo.Microphone,
		},
	}

	fmt.Println("Testing MACGO_RESET_PERMISSIONS flag...")
	fmt.Println("This will reset TCC permissions for this app before requesting new ones.")
	fmt.Println()

	// Set the reset flag
	os.Setenv("MACGO_RESET_PERMISSIONS", "1")
	defer os.Unsetenv("MACGO_RESET_PERMISSIONS")

	err := macgo.Start(cfg)
	if err != nil {
		log.Fatalf("Failed to start macgo: %v", err)
	}

	fmt.Println()
	fmt.Println("✅ Permission reset test completed!")
	fmt.Println("Check System Settings > Privacy & Security to verify permissions were reset.")
}