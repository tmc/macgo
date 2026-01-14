package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tmc/macgo"
)

// stdioConn wraps stdin/stdout to ensure NDJSON framing
type stdioConn struct {
	io.ReadCloser
	io.Writer
}

func (c *stdioConn) Write(p []byte) (n int, err error) {
	n, err = c.Writer.Write(p)
	if err != nil {
		return n, err
	}
	// Append newline to ensure NDJSON framing
	_, err = c.Writer.Write([]byte{'\n'})
	return n, err
}

func main() {
	log.Printf("Starting server PID=%d", os.Getpid())

	// 1. Initialize macgo for permissions and bundle management
	cfg := macgo.NewConfig().
		WithAppName("ScreenCaptureMCP").
		WithPermissions(macgo.ScreenRecording, macgo.Accessibility).
		WithDebug()

	if err := macgo.Start(cfg); err != nil {
		log.Fatalf("macgo start failed: %v", err)
	}
	defer macgo.Cleanup()

	// 2. Initialize MCP Server
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "screen-capture-mcp",
		Version: "1.0.0",
	}, nil)

	// 3. Register Tools

	// Tool: list_screens
	type ListScreensArgs struct{}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_screens",
		Description: "List connected displays via system_profiler",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args ListScreensArgs) (*mcp.CallToolResult, any, error) {
		cmd := exec.Command("system_profiler", "SPDisplaysDataType")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to list screens: %w", err)
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(out)},
			},
		}, nil, nil
	})

	// Tool: list_windows
	type ListWindowsArgs struct {
		IncludeOffscreen bool `json:"include_offscreen,omitempty" jsonschema:"Include minimized/hidden windows"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_windows",
		Description: "List application windows",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args ListWindowsArgs) (*mcp.CallToolResult, any, error) {
		windows, err := getWindowList(args.IncludeOffscreen)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get window list: %w", err)
		}

		// Format output as text table for readability
		output := "WINDOW_ID\tDISP\tPID\tOWNER\tWxH\n"
		for _, w := range windows {
			output += fmt.Sprintf("%d\t%d\t%d\t%s\t%.0fx%.0f\n",
				w.WindowID, w.DisplayID, w.OwnerPID, w.OwnerName, w.Width, w.Height)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: output},
			},
		}, nil, nil
	})

	// Tool: capture_screen
	type CaptureScreenArgs struct {
		DisplayID int `json:"display_id,omitempty" jsonschema:"Optional display ID to capture"`
		WindowID  int `json:"window_id,omitempty" jsonschema:"Optional window ID to capture"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "capture_screen",
		Description: "Capture a screenshot and return as base64 png",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args CaptureScreenArgs) (*mcp.CallToolResult, any, error) {
		tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("capture_%d.png", time.Now().UnixNano()))
		defer os.Remove(tmpFile)

		// Build screencapture command
		// -x: muted (no sound)
		// -r: do not add shadow
		// -t png: image format
		cmdArgs := []string{"-x", "-r", "-t", "png"}

		if args.WindowID != 0 {
			cmdArgs = append(cmdArgs, "-l", fmt.Sprintf("%d", args.WindowID))
		}

		cmdArgs = append(cmdArgs, tmpFile)

		cmd := exec.Command("screencapture", cmdArgs...)
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, nil, fmt.Errorf("screencapture failed: %v\nOutput: %s", err, out)
		}

		// Read image data
		data, err := os.ReadFile(tmpFile)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read capture: %w", err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.ImageContent{
					Data:     data,
					MIMEType: "image/png",
				},
			},
		}, nil, nil
	})

	// Tool: type_text
	type TypeTextArgs struct {
		Text string `json:"text" jsonschema:"Text to type"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "type_text",
		Description: "Type text using keyboard simulation",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args TypeTextArgs) (*mcp.CallToolResult, any, error) {
		if err := typeString(args.Text); err != nil {
			return nil, nil, fmt.Errorf("failed to type text: %w", err)
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Typed: %s", args.Text)},
			},
		}, nil, nil
	})

	// Tool: press_key
	type PressKeyArgs struct {
		Key   string `json:"key" jsonschema:"Key to press (e.g. 'a', 'enter', 'space')"`
		Cmd   bool   `json:"cmd,omitempty" jsonschema:"Command key modifier"`
		Shift bool   `json:"shift,omitempty" jsonschema:"Shift key modifier"`
		Opt   bool   `json:"opt,omitempty" jsonschema:"Option/Alt key modifier"`
		Ctrl  bool   `json:"ctrl,omitempty" jsonschema:"Control key modifier"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "press_key",
		Description: "Press a key with optional modifiers",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args PressKeyArgs) (*mcp.CallToolResult, any, error) {
		if err := pressKey(args.Key, args.Cmd, args.Shift, args.Opt, args.Ctrl); err != nil {
			return nil, nil, fmt.Errorf("failed to press key: %w", err)
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Pressed: %s", args.Key)},
			},
		}, nil, nil
	})

	// 4. Run Server
	// Wrap os.Stdout to ensure NDJSON framing (append \n to each JSON message)
	realStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		log.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	// Goroutine to frame output
	go func() {
		defer realStdout.Close()
		dec := json.NewDecoder(r)
		for {
			var v json.RawMessage
			if err := dec.Decode(&v); err != nil {
				if err != io.EOF {
					log.Printf("frame-enforcer: decode error: %v", err)
				}
				return
			}
			// Write JSON + Newline atomically-ish
			realStdout.Write(v)
			realStdout.Write([]byte{'\n'})
		}
	}()

	// Use standard StdioTransport. It writes to os.Stdout (now our pipe)
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Printf("Server failed: %v", err)
	}
}
