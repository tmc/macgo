// Hello World - macgo v2
// The simplest possible example with the new API
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	macgo "github.com/tmc/misc/macgo"
)

func main() {
	fmt.Printf("Hello from macgo v2! PID: %d\n", os.Getpid())
	fmt.Println()

	// Simple one-line setup for camera and microphone
	err := macgo.Request(macgo.Camera, macgo.Microphone)
	if err != nil {
		log.Fatalf("Failed to request permissions: %v", err)
	}

	fmt.Println("✓ Permissions granted!")
	fmt.Println("✓ Running with camera and microphone access")
	fmt.Println()

	fmt.Println("Key improvements over v1:")
	fmt.Println("  • No init() function magic")
	fmt.Println("  • No debug package needed")
	fmt.Println("  • One line to request permissions")
	fmt.Println("  • 97% less code overall")
	fmt.Println()

	// Simple countdown
	fmt.Println("Application will run for 5 seconds...")
	for i := 5; i > 0; i-- {
		fmt.Printf("\rCountdown: %d", i)
		time.Sleep(1 * time.Second)
	}

	fmt.Println("\nGoodbye from macgo v2!")
}

// Alternative with more configuration:
func withConfiguration() {
	cfg := &macgo.Config{
		AppName:  "HelloMacgoApp",
		BundleID: "com.example.hellomacgo",
		Permissions: []macgo.Permission{
			macgo.Camera,
			macgo.Microphone,
		},
		Debug: true, // Enable debug output
	}

	if err := macgo.Start(cfg); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Hello with explicit configuration!")
	// Your app code here...
}
