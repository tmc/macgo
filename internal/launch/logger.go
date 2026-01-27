package launch

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Logger wraps slog.Logger with macgo-specific configuration
type Logger struct {
	*slog.Logger
}

// NewLogger creates a configured logger based on environment variables
func NewLogger() *Logger {
	var handler slog.Handler
	var writers []io.Writer

	// Determine log level
	level := slog.LevelInfo
	if os.Getenv("MACGO_DEBUG") == "1" {
		level = slog.LevelDebug
	}

	// Configure output destinations
	logDest := os.Getenv("MACGO_LOG_DEST") // Can be "stderr", "file:<path>", or "both:<path>"

	switch {
	case strings.HasPrefix(logDest, "file:"):
		// Log only to file
		logPath := strings.TrimPrefix(logDest, "file:")
		if f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
			writers = append(writers, f)
		} else {
			fmt.Fprintf(os.Stderr, "macgo: failed to open log file %s: %v\n", logPath, err)
			writers = append(writers, os.Stderr)
		}
	case strings.HasPrefix(logDest, "both:"):
		// Log to both stderr and file
		logPath := strings.TrimPrefix(logDest, "both:")
		writers = append(writers, os.Stderr)
		if f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
			writers = append(writers, f)
		} else {
			fmt.Fprintf(os.Stderr, "macgo: failed to open log file %s: %v\n", logPath, err)
		}
	default:
		// Default to stderr
		writers = append(writers, os.Stderr)
	}

	// Create multi-writer if needed
	var output io.Writer
	if len(writers) == 1 {
		output = writers[0]
	} else {
		output = io.MultiWriter(writers...)
	}

	// Configure handler based on format preference
	if os.Getenv("MACGO_LOG_JSON") == "1" {
		handler = slog.NewJSONHandler(output, &slog.HandlerOptions{
			Level: level,
		})
	} else {
		// Use text handler with custom format
		handler = slog.NewTextHandler(output, &slog.HandlerOptions{
			Level: level,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				// Customize the output format
				if a.Key == slog.TimeKey {
					// Omit time for cleaner output (can be enabled via env var)
					if os.Getenv("MACGO_LOG_TIME") != "1" {
						return slog.Attr{}
					}
				}
				if a.Key == slog.LevelKey {
					// Simplify level output
					if os.Getenv("MACGO_LOG_LEVEL") != "1" {
						return slog.Attr{}
					}
				}
				return a
			},
		})
	}

	return &Logger{
		Logger: slog.New(handler).With("component", "macgo"),
	}
}

// Debug logs at debug level with "macgo:" prefix for compatibility
func (l *Logger) Debug(msg string, args ...any) {
	l.Logger.Debug("macgo: "+msg, args...)
}

// Info logs at info level with "macgo:" prefix for compatibility
func (l *Logger) Info(msg string, args ...any) {
	l.Logger.Info("macgo: "+msg, args...)
}

// Error logs at error level with "macgo:" prefix for compatibility
func (l *Logger) Error(msg string, args ...any) {
	l.Logger.Error("macgo: "+msg, args...)
}

// Warn logs at warn level with "macgo:" prefix for compatibility
func (l *Logger) Warn(msg string, args ...any) {
	l.Logger.Warn("macgo: "+msg, args...)
}