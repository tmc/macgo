// sigquit-test is a simple program to test SIGQUIT stack dumps with macgo.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/tmc/macgo"
)

func main() {
	cfg := &macgo.Config{Debug: true}
	if err := macgo.Start(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "macgo.Start: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("PID: %d\n", os.Getpid())
	fmt.Println("Running... send SIGQUIT (Ctrl+\\) to dump stacks")

	// Start some goroutines to make the stack dump interesting
	go worker("worker-1")
	go worker("worker-2")

	// Main loop
	for {
		time.Sleep(time.Second)
		fmt.Print(".")
	}
}

func worker(name string) {
	for {
		doWork(name)
		time.Sleep(500 * time.Millisecond)
	}
}

func doWork(name string) {
	// Nested function to show in stack trace
	innerWork(name)
}

func innerWork(name string) {
	time.Sleep(100 * time.Millisecond)
}
