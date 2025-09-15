// Package entitlements provides macOS entitlements for app sandbox and TCC permissions.
// This package centralizes all entitlement-related functionality for the macgo library.
package entitlements

// Entitlement is a type for macOS entitlement identifiers.
// Entitlements are special permissions that allow apps to access protected
// resources and perform privileged operations on macOS.
type Entitlement string

// Entitlements is a map of entitlement identifiers to boolean values.
// When true, the entitlement is granted; when false, it is explicitly denied.
type Entitlements map[Entitlement]bool

// Available app sandbox entitlements
const (
	// EntAppSandbox enables the macOS App Sandbox, which provides a secure environment
	// by restricting access to system resources. Required for many other entitlements.
	EntAppSandbox Entitlement = "com.apple.security.app-sandbox"

	// Network entitlements
	EntNetworkClient Entitlement = "com.apple.security.network.client"
	EntNetworkServer Entitlement = "com.apple.security.network.server"

	// Device access entitlements
	EntCamera     Entitlement = "com.apple.security.device.camera"
	EntMicrophone Entitlement = "com.apple.security.device.microphone"
	EntBluetooth  Entitlement = "com.apple.security.device.bluetooth"
	EntUSB        Entitlement = "com.apple.security.device.usb"
	EntAudioInput Entitlement = "com.apple.security.device.audio-input"
	EntPrint      Entitlement = "com.apple.security.print"

	// Personal information access entitlements
	EntAddressBook Entitlement = "com.apple.security.personal-information.addressbook"
	EntLocation    Entitlement = "com.apple.security.personal-information.location"
	EntCalendars   Entitlement = "com.apple.security.personal-information.calendars"
	EntPhotos      Entitlement = "com.apple.security.personal-information.photos-library"
	EntReminders   Entitlement = "com.apple.security.personal-information.reminders"

	// File system access entitlements
	EntUserSelectedReadOnly  Entitlement = "com.apple.security.files.user-selected.read-only"
	EntUserSelectedReadWrite Entitlement = "com.apple.security.files.user-selected.read-write"
	EntDownloadsReadOnly     Entitlement = "com.apple.security.files.downloads.read-only"
	EntDownloadsReadWrite    Entitlement = "com.apple.security.files.downloads.read-write"
	EntPicturesReadOnly      Entitlement = "com.apple.security.assets.pictures.read-only"
	EntPicturesReadWrite     Entitlement = "com.apple.security.assets.pictures.read-write"
	EntMusicReadOnly         Entitlement = "com.apple.security.assets.music.read-only"
	EntMusicReadWrite        Entitlement = "com.apple.security.assets.music.read-write"
	EntMoviesReadOnly        Entitlement = "com.apple.security.assets.movies.read-only"
	EntMoviesReadWrite       Entitlement = "com.apple.security.assets.movies.read-write"

	// Code signing and debugging entitlements
	EntHardenedRuntime                 Entitlement = "com.apple.security.cs.runtime"
	EntAllowJIT                        Entitlement = "com.apple.security.cs.allow-jit"
	EntAllowUnsignedExecutableMemory   Entitlement = "com.apple.security.cs.allow-unsigned-executable-memory"
	EntAllowDyldEnvVars                Entitlement = "com.apple.security.cs.allow-dyld-environment-variables"
	EntDisableLibraryValidation        Entitlement = "com.apple.security.cs.disable-library-validation"
	EntDisableExecutablePageProtection Entitlement = "com.apple.security.cs.disable-executable-page-protection"
	EntDebugger                        Entitlement = "com.apple.security.cs.debugger"

	// Virtualization entitlements
	EntVirtualization Entitlement = "com.apple.security.virtualization"
)

// These functions are needed for tests to work properly
// They're imported here to avoid import cycles

func SetAllTCCPermissions() {
	// This function is used by tests - it should call into macgo
	// But we can't import macgo due to import cycles
	// For now, this is a placeholder
}

func SetCamera() {
	// Placeholder for test compatibility
}

func SetMic() {
	// Placeholder for test compatibility
}

func SetLocation() {
	// Placeholder for test compatibility
}

func SetContacts() {
	// Placeholder for test compatibility
}

func SetPhotos() {
	// Placeholder for test compatibility
}

func SetCalendar() {
	// Placeholder for test compatibility
}

func SetReminders() {
	// Placeholder for test compatibility
}
