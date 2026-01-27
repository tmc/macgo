package launch

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strings"
)

// IOWrapper wraps IO streams to add prefixes, indentation, or styling
type IOWrapper struct {
	dest     io.Writer
	prefix   string
	indent   string
	colorize bool
	stream   string // "stdout" or "stderr"
}

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorGray   = "\033[90m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorDim    = "\033[2m"
)

// NewIOWrapper creates a new IO wrapper based on environment configuration
func NewIOWrapper(dest io.Writer, stream string) io.Writer {
	// Check if wrapping is disabled
	if os.Getenv("MACGO_IO_WRAP") != "1" {
		return dest
	}

	wrapper := &IOWrapper{
		dest:     dest,
		stream:   stream,
		colorize: os.Getenv("MACGO_IO_COLOR") == "1" || os.Getenv("TERM") != "dumb",
	}

	// Configure prefix and indentation
	prefix := os.Getenv("MACGO_IO_PREFIX")
	if prefix == "" {
		// Default prefix shows the stream type
		if stream == "stdout" {
			prefix = "[out] "
		} else if stream == "stderr" {
			prefix = "[err] "
		}
	}
	wrapper.prefix = prefix

	// Configure indentation
	indentStr := os.Getenv("MACGO_IO_INDENT")
	if indentStr == "" {
		indentStr = "  " // Default 2 spaces
	}
	wrapper.indent = indentStr

	return wrapper
}

// Write implements io.Writer, adding prefix and styling to each line
func (w *IOWrapper) Write(p []byte) (int, error) {
	// Split input into lines
	scanner := bufio.NewScanner(bytes.NewReader(p))
	var output bytes.Buffer

	for scanner.Scan() {
		line := scanner.Text()

		// Apply styling if colorization is enabled
		if w.colorize {
			// Choose color based on stream type
			var color string
			switch w.stream {
			case "stdout":
				color = colorDim + colorGreen // Dim green for stdout
			case "stderr":
				color = colorDim + colorYellow // Dim yellow for stderr
			default:
				color = colorGray
			}

			// Write styled line
			output.WriteString(color)
		}

		// Add indentation and prefix
		output.WriteString(w.indent)
		output.WriteString(w.prefix)
		output.WriteString(line)

		// Reset color if applied
		if w.colorize {
			output.WriteString(colorReset)
		}

		output.WriteString("\n")
	}

	// Handle any remaining bytes without newline
	if remainder := p[len(p)-1]; len(p) > 0 && remainder != '\n' {
		// Preserve the lack of newline for partial writes
		outputBytes := output.Bytes()
		if len(outputBytes) > 0 {
			outputBytes = outputBytes[:len(outputBytes)-1]
		}
		_, err := w.dest.Write(outputBytes)
		return len(p), err
	}

	// Write the processed output
	_, err := w.dest.Write(output.Bytes())
	return len(p), err
}

// LineWriter wraps a writer to buffer and process complete lines
type LineWriter struct {
	w      io.Writer
	buffer bytes.Buffer
}

// NewLineWriter creates a writer that processes complete lines
func NewLineWriter(w io.Writer) *LineWriter {
	return &LineWriter{w: w}
}

// Write buffers data and writes complete lines
func (lw *LineWriter) Write(p []byte) (int, error) {
	n := len(p)
	lw.buffer.Write(p)

	// Process complete lines
	for {
		line, err := lw.buffer.ReadString('\n')
		if err != nil {
			// If we have a partial line, put it back
			if len(line) > 0 {
				lw.buffer = bytes.Buffer{}
				lw.buffer.WriteString(line)
			}
			break
		}

		// Write the complete line
		if _, err := lw.w.Write([]byte(line)); err != nil {
			return n, err
		}
	}

	return n, nil
}

// Flush writes any remaining buffered data
func (lw *LineWriter) Flush() error {
	if lw.buffer.Len() > 0 {
		remaining := lw.buffer.String()
		if !strings.HasSuffix(remaining, "\n") {
			remaining += "\n"
		}
		_, err := lw.w.Write([]byte(remaining))
		lw.buffer.Reset()
		return err
	}
	return nil
}

// Close flushes and closes the writer if it implements io.Closer
func (lw *LineWriter) Close() error {
	if err := lw.Flush(); err != nil {
		return err
	}
	if closer, ok := lw.w.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}