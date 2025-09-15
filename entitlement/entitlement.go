// Package entitlement provides consolidated entitlement management for macOS apps.
// This package replaces the many tiny entitlement packages with a single,
// focused package that handles all entitlement types.
package entitlement

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/tmc/misc/macgo"
)

// Registry implements the EntitlementRegistry interface.
// It provides thread-safe entitlement management.
type Registry struct {
	mu           sync.RWMutex
	entitlements macgo.Entitlements
}

// NewRegistry creates a new entitlement registry.
func NewRegistry() *Registry {
	return &Registry{
		entitlements: make(macgo.Entitlements),
	}
}

// Register registers an entitlement with the given value.
func (r *Registry) Register(entitlement macgo.Entitlement, value bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entitlements[entitlement] = value
}

// Get returns the value of an entitlement, or false if not set.
func (r *Registry) Get(entitlement macgo.Entitlement) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.entitlements[entitlement]
}

// GetAll returns a copy of all registered entitlements.
func (r *Registry) GetAll() macgo.Entitlements {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(macgo.Entitlements, len(r.entitlements))
	for k, v := range r.entitlements {
		result[k] = v
	}
	return result
}

// Clear removes all registered entitlements.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entitlements = make(macgo.Entitlements)
}

// LoadFromJSON loads entitlements from JSON data.
func (r *Registry) LoadFromJSON(data []byte) error {
	var entitlements map[string]bool
	if err := json.Unmarshal(data, &entitlements); err != nil {
		return fmt.Errorf("entitlement: parse JSON: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for key, value := range entitlements {
		r.entitlements[macgo.Entitlement(key)] = value
	}

	return nil
}

// LoadFromReader loads entitlements from a JSON reader.
func (r *Registry) LoadFromReader(reader io.Reader) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("entitlement: read JSON: %w", err)
	}
	return r.LoadFromJSON(data)
}

// LoadFromFile loads entitlements from a JSON file.
func (r *Registry) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("entitlement: read file: %w", err)
	}
	return r.LoadFromJSON(data)
}

// SaveToJSON saves entitlements to JSON format.
func (r *Registry) SaveToJSON() ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Convert to map[string]bool for JSON serialization
	data := make(map[string]bool, len(r.entitlements))
	for k, v := range r.entitlements {
		data[string(k)] = v
	}

	return json.MarshalIndent(data, "", "  ")
}

// Entitlement constants - re-exported from macgo for convenience
const (
	// App Sandbox entitlements
	AppSandbox    = macgo.EntAppSandbox
	NetworkClient = macgo.EntNetworkClient
	NetworkServer = macgo.EntNetworkServer

	// Device entitlements
	Camera     = macgo.EntCamera
	Microphone = macgo.EntMicrophone
	Bluetooth  = macgo.EntBluetooth
	USB        = macgo.EntUSB
	AudioInput = macgo.EntAudioInput
	Print      = macgo.EntPrint

	// Personal information entitlements
	AddressBook = macgo.EntAddressBook
	Location    = macgo.EntLocation
	Calendars   = macgo.EntCalendars
	Photos      = macgo.EntPhotos
	Reminders   = macgo.EntReminders

	// File entitlements
	UserSelectedReadOnly  = macgo.EntUserSelectedReadOnly
	UserSelectedReadWrite = macgo.EntUserSelectedReadWrite
	DownloadsReadOnly     = macgo.EntDownloadsReadOnly
	DownloadsReadWrite    = macgo.EntDownloadsReadWrite
	PicturesReadOnly      = macgo.EntPicturesReadOnly
	PicturesReadWrite     = macgo.EntPicturesReadWrite
	MusicReadOnly         = macgo.EntMusicReadOnly
	MusicReadWrite        = macgo.EntMusicReadWrite
	MoviesReadOnly        = macgo.EntMoviesReadOnly
	MoviesReadWrite       = macgo.EntMoviesReadWrite

	// Hardened Runtime entitlements
	AllowJIT                        = macgo.EntAllowJIT
	AllowUnsignedExecutableMemory   = macgo.EntAllowUnsignedExecutableMemory
	AllowDyldEnvVars                = macgo.EntAllowDyldEnvVars
	DisableLibraryValidation        = macgo.EntDisableLibraryValidation
	DisableExecutablePageProtection = macgo.EntDisableExecutablePageProtection
	Debugger                        = macgo.EntDebugger

	// Virtualization entitlements
	Virtualization = macgo.EntVirtualization
)

// Compile-time check that Registry implements EntitlementRegistry
var _ macgo.EntitlementRegistry = (*Registry)(nil)
