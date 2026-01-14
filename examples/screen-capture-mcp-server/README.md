# Screen Capture MCP Server

A Model Context Protocol (MCP) server that provides screen capture capabilities for macOS.

This example demonstrates:
1.  **Official Go SDK Integration**: Uses `github.com/modelcontextprotocol/go-sdk`.
2.  **Macgo Integration**: Uses `macgo` to bundle the app and manage permissions (Screen Recording).
3.  **Stdio Transport**: Enables usage with any MCP client (Claude Desktop, etc.) via helper scripts.

## Tools

*   `list_screens`: Lists connected displays using `system_profiler`.
*   `capture_screen`: Captures a screenshot and returns it as a base64-encoded PNG image.

## Usage

### 1. Build

```bash
go build -o screen-capture-mcp-server
```

### 2. Run / Permission Check

Run locally first to trigger the macOS Screen Recording permission prompt (TCC).

```bash
./screen-capture-mcp-server
```
*Note: The first run might look like it hangs while waiting for you to grant permission in System Settings. You may need to restart the terminal or app after granting permission.*

### 3. Use with MCP Client

Configure your MCP client (e.g., `claude_desktop_config.json`) to run the binary:

```json
{
  "mcpServers": {
    "screen-capture": {
      "command": "/absolute/path/to/screen-capture-mcp-server",
      "args": []
    }
  }
}
```

## How It Works

This server runs over `stdio`. It uses `macgo` with `WithForceDirectExecution(true)` to ensure that:
1.  Stdin/Stdout are preserved (not redirected to logs), allowing MCP JSON-RPC communication.
2.  The application still runs as a bundled macOS app (created on the fly) to satisfy TCC requirements for Screen Recording.
