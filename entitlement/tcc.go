package entitlement

import "github.com/tmc/misc/macgo"

// TCC (Transparency, Consent, and Control) permission helper functions.
// These functions provide convenient ways to register TCC-related entitlements.

// SetCamera enables camera access entitlement.
// This allows the app to access the camera after user consent.
func (r *Registry) SetCamera() {
	r.Register(Camera, true)
}

// SetMicrophone enables microphone access entitlement.
// This allows the app to access the microphone after user consent.
func (r *Registry) SetMicrophone() {
	r.Register(Microphone, true)
}

// SetLocation enables location services entitlement.
// This allows the app to access location services after user consent.
func (r *Registry) SetLocation() {
	r.Register(Location, true)
}

// SetContacts enables contacts access entitlement.
// This allows the app to access the user's contacts after user consent.
func (r *Registry) SetContacts() {
	r.Register(AddressBook, true)
}

// SetPhotos enables photos library access entitlement.
// This allows the app to access the Photos library after user consent.
func (r *Registry) SetPhotos() {
	r.Register(Photos, true)
}

// SetCalendars enables calendar access entitlement.
// This allows the app to access calendar data after user consent.
func (r *Registry) SetCalendars() {
	r.Register(Calendars, true)
}

// SetReminders enables reminders access entitlement.
// This allows the app to access reminders after user consent.
func (r *Registry) SetReminders() {
	r.Register(Reminders, true)
}

// SetAllTCCPermissions enables all common TCC permissions.
// This is a convenience function that enables camera, microphone, location,
// contacts, photos, calendars, and reminders access.
func (r *Registry) SetAllTCCPermissions() {
	r.SetCamera()
	r.SetMicrophone()
	r.SetLocation()
	r.SetContacts()
	r.SetPhotos()
	r.SetCalendars()
	r.SetReminders()
}

// Standalone convenience functions that work with a global registry
var globalRegistry = NewRegistry()

// Camera enables camera access entitlement in the global registry.
func EnableCamera() {
	globalRegistry.SetCamera()
}

// EnableMicrophone enables microphone access entitlement in the global registry.
func EnableMicrophone() {
	globalRegistry.SetMicrophone()
}

// EnableLocation enables location services entitlement in the global registry.
func EnableLocation() {
	globalRegistry.SetLocation()
}

// EnableContacts enables contacts access entitlement in the global registry.
func EnableContacts() {
	globalRegistry.SetContacts()
}

// EnablePhotos enables photos library access entitlement in the global registry.
func EnablePhotos() {
	globalRegistry.SetPhotos()
}

// EnableCalendars enables calendar access entitlement in the global registry.
func EnableCalendars() {
	globalRegistry.SetCalendars()
}

// EnableReminders enables reminders access entitlement in the global registry.
func EnableReminders() {
	globalRegistry.SetReminders()
}

// EnableAllTCCPermissions enables all common TCC permissions in the global registry.
func EnableAllTCCPermissions() {
	globalRegistry.SetAllTCCPermissions()
}

// RequestEntitlements requests multiple entitlements in the global registry.
func RequestEntitlements(entitlements ...macgo.Entitlement) {
	for _, ent := range entitlements {
		globalRegistry.Register(ent, true)
	}
}

// GetGlobalRegistry returns the global entitlement registry.
// This can be used to integrate with the main macgo configuration.
func GetGlobalRegistry() *Registry {
	return globalRegistry
}
