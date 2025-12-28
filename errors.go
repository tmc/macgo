package macgo

import "fmt"

// Error represents a macgo error with additional context and actionable guidance.
type Error struct {
	Op   string // Operation that failed (e.g., "create bundle", "code sign")
	Err  error  // Underlying error
	Help string // Actionable guidance for the user
}

func (e *Error) Error() string {
	if e.Help != "" {
		return fmt.Sprintf("macgo: %s: %v\n  hint: %s", e.Op, e.Err, e.Help)
	}
	return fmt.Sprintf("macgo: %s: %v", e.Op, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}

// errBundleCreate returns an error with guidance for bundle creation failures.
func errBundleCreate(err error) error {
	return &Error{
		Op:   "create bundle",
		Err:  err,
		Help: "ensure write permissions to ~/go/bin or set MACGO_KEEP_BUNDLE=0 to use temp directory",
	}
}

// errCodeSign returns an error with guidance for code signing failures.
func errCodeSign(err error) error {
	return &Error{
		Op:   "code sign",
		Err:  err,
		Help: "for development, use WithAdHocSign(); for distribution, install Xcode and run 'security find-identity -v -p codesigning'",
	}
}

// errLaunch returns an error with guidance for launch failures.
func errLaunch(err error) error {
	return &Error{
		Op:   "launch bundle",
		Err:  err,
		Help: "check that the bundle was created correctly and has valid code signing",
	}
}

// errPermission returns an error with guidance for permission-related failures.
func errPermission(err error) error {
	return &Error{
		Op:   "request permission",
		Err:  err,
		Help: "ensure the app has been granted the required permissions in System Settings > Privacy & Security",
	}
}

// errConfig returns an error with guidance for configuration issues.
func errConfig(err error) error {
	return &Error{
		Op:   "validate config",
		Err:  err,
		Help: "check that all required fields are set and permissions are valid",
	}
}
