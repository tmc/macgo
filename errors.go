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
