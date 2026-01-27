// launcher_test_helper is a test helper program for integration testing ServicesLauncher V1 and V2.
// This program is built and executed by the integration tests to verify launcher behavior.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	var (
		stdout     = flag.String("stdout", "", "Message to write to stdout")
		stderr     = flag.String("stderr", "", "Message to write to stderr")
		exitCode   = flag.Int("exit", 0, "Exit code to return")
		sleepMs    = flag.Int("sleep", 0, "Milliseconds to sleep before exiting")
		multiline  = flag.Bool("multiline", false, "Output multiple lines")
	)
	flag.Parse()

	// Sleep if requested (to test timeout behavior)
	if *sleepMs > 0 {
		time.Sleep(time.Duration(*sleepMs) * time.Millisecond)
	}

	// Output to stdout
	if *stdout != "" {
		if *multiline {
			fmt.Println(*stdout)
			fmt.Println("Line 2 of stdout")
			fmt.Println("Line 3 of stdout")
		} else {
			fmt.Println(*stdout)
		}
	}

	// Output to stderr
	if *stderr != "" {
		if *multiline {
			fmt.Fprintln(os.Stderr, *stderr)
			fmt.Fprintln(os.Stderr, "Line 2 of stderr")
			fmt.Fprintln(os.Stderr, "Line 3 of stderr")
		} else {
			fmt.Fprintln(os.Stderr, *stderr)
		}
	}

	// Exit with specified code
	os.Exit(*exitCode)
}
