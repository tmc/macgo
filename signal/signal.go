// Package signal provides consolidated signal handling functionality for macgo.
// This package consolidates all signal-related functionality that was previously
// scattered across improvedsignals.go, signalforwarder.go, and signalhandler/ package.
package signal

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// Handler manages signal forwarding and handling for macgo processes.
type Handler struct {
	disabled bool
}

// NewHandler creates a new signal handler.
func NewHandler() *Handler {
	return &Handler{}
}

// DisableSignalHandling allows opting out of the default signal handling
// when necessary for compatibility with specific environments or requirements.
// By default, signal handling is enabled for better Ctrl+C and signal behavior.
var DisableSignalHandling = false

// Disable disables signal handling for this handler.
func (h *Handler) Disable() {
	h.disabled = true
}

// Enable enables signal handling for this handler.
func (h *Handler) Enable() {
	h.disabled = false
}

// IsDisabled returns true if signal handling is disabled.
func (h *Handler) IsDisabled() bool {
	return h.disabled || DisableSignalHandling
}

// Forward sets up signal forwarding from the current process to the target process.
// This implements the SignalForwarder interface.
func (h *Handler) Forward(ctx context.Context, target *os.Process) error {
	if h.IsDisabled() {
		return nil
	}

	return h.forwardSignalsToProcess(ctx, target)
}

// Stop stops signal forwarding (placeholder for interface compliance).
func (h *Handler) Stop() error {
	// Signal forwarding goroutines will stop when their context is cancelled
	return nil
}

// forwardSignalsToProcess forwards signals to a specific process.
func (h *Handler) forwardSignalsToProcess(ctx context.Context, target *os.Process) error {
	sigCh := make(chan os.Signal, 16)

	// Handle comprehensive set of signals
	signal.Notify(sigCh,
		syscall.SIGABRT,
		syscall.SIGALRM,
		syscall.SIGBUS,
		syscall.SIGCHLD,
		syscall.SIGCONT,
		syscall.SIGFPE,
		syscall.SIGHUP,
		syscall.SIGILL,
		syscall.SIGINT,
		syscall.SIGIO,
		syscall.SIGPIPE,
		syscall.SIGPROF,
		syscall.SIGQUIT,
		syscall.SIGSEGV,
		syscall.SIGSYS,
		syscall.SIGTERM,
		syscall.SIGTRAP,
		syscall.SIGTSTP,
		syscall.SIGTTIN,
		syscall.SIGTTOU,
		syscall.SIGUSR1,
		syscall.SIGUSR2,
		syscall.SIGVTALRM,
		syscall.SIGWINCH,
		syscall.SIGXCPU,
		syscall.SIGXFSZ,
		// explicitly not catching SIGKILL and SIGSTOP (cannot be caught)
	)

	go func() {
		defer signal.Stop(sigCh)
		defer close(sigCh)

		for {
			select {
			case <-ctx.Done():
				return
			case sig := <-sigCh:
				h.handleSignal(sig, target)
			}
		}
	}()

	return nil
}

// handleSignal processes a single signal and forwards it appropriately.
func (h *Handler) handleSignal(sig os.Signal, target *os.Process) {
	sigNum, ok := sig.(syscall.Signal)
	if !ok {
		return
	}

	// Skip SIGCHLD as we don't need to forward it
	if sigNum == syscall.SIGCHLD {
		return
	}

	debugf("Forwarding signal %v to process group", sigNum)

	// Forward the signal to the process group of the child
	// Using negative PID sends to the entire process group
	if err := syscall.Kill(-target.Pid, sigNum); err != nil {
		debugf("Error forwarding signal %v: %v", sigNum, err)
	}

	// Handle terminal stop signals
	if sigNum == syscall.SIGTSTP || sigNum == syscall.SIGTTIN || sigNum == syscall.SIGTTOU {
		// Use SIGSTOP for these terminal signals
		syscall.Kill(syscall.Getpid(), syscall.SIGSTOP)
	}

	// Special handling for SIGINT and SIGTERM
	if sigNum == syscall.SIGINT || sigNum == syscall.SIGTERM {
		debugf("Received termination signal %v", sigNum)
		// Let the main process handle cleanup and exit
	}
}

// SetupSignalHandling sets up signal handling for a command and returns a channel.
// This is a utility function for backward compatibility.
func (h *Handler) SetupSignalHandling(cmd *os.Process) chan os.Signal {
	c := make(chan os.Signal, 100)
	signal.Notify(c)
	go func() {
		for sig := range c {
			debugf("Forwarding signal %v to process", sig)
			cmd.Signal(sig)
		}
	}()
	return c
}

// debugf prints debug messages if MACGO_DEBUG=1 is set
func debugf(format string, args ...any) {
	if os.Getenv("MACGO_DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "[macgo/signal] "+format+"\n", args...)
	}
}
