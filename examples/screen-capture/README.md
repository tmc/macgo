# Screen Capture Example

This example demonstrates screen capture using macgo with coordinate support, including integration with iTerm2 session panes.

## Features

- **Basic screen capture**: Full screen, window selection, interactive selection
- **Coordinate support**: Capture specific regions by coordinates
- **iTerm2 integration**: Capture specific iTerm2 session panes using `it2` tool
- **Proper permissions**: Handles macOS screen recording permissions via macgo
- **Flexible output**: Custom output paths with timestamp defaults
- **Automatic retry logic**: Configurable retry with exponential backoff for transient failures

## Retry Logic for Transient Failures

The screen-capture tool includes automatic retry logic to handle transient failures such as:
- Window state changes (window closed, minimized, or moved)
- Temporary WindowServer communication issues
- System resource contention
- Permission check delays

### Retry Configuration

Control retry behavior via command-line flags:

```bash
# Default: 3 attempts with 500ms initial delay
./screen-capture -app Safari

# Disable retries (single attempt)
./screen-capture -retry-attempts 1 -app Safari

# More aggressive retries
./screen-capture -retry-attempts 5 -retry-delay 100 -retry-backoff 1.5 -app Safari

# Conservative retries with longer delays
./screen-capture -retry-attempts 3 -retry-delay 1000 -retry-max-delay 10000 -app Safari
```

Retry flags:
- `-retry-attempts`: Number of retry attempts (default: 3, set to 1 to disable)
- `-retry-delay`: Initial retry delay in milliseconds (default: 500ms)
- `-retry-max-delay`: Maximum retry delay in milliseconds (default: 5000ms)
- `-retry-backoff`: Exponential backoff multiplier (default: 2.0)

### Environment Variable Configuration

Override retry settings globally via environment variables:

```bash
# Disable retries for all captures
export SCREENCAPTURE_RETRY_ATTEMPTS=1
./screen-capture -app Safari

# Custom retry configuration
export SCREENCAPTURE_RETRY_ATTEMPTS=5
export SCREENCAPTURE_RETRY_DELAY=200
export SCREENCAPTURE_RETRY_MAX_DELAY=3000
export SCREENCAPTURE_RETRY_BACKOFF=1.8
./screen-capture -app Safari
```

Environment variables:
- `SCREENCAPTURE_RETRY_ATTEMPTS`: Override -retry-attempts flag
- `SCREENCAPTURE_RETRY_DELAY`: Override -retry-delay flag (milliseconds)
- `SCREENCAPTURE_RETRY_MAX_DELAY`: Override -retry-max-delay flag (milliseconds)
- `SCREENCAPTURE_RETRY_BACKOFF`: Override -retry-backoff flag (float)

### Permission Timeout Configuration

Control the timeout for waiting for Screen Recording permission to be granted:

```bash
# Wait up to 120 seconds for permission (default: 60 seconds)
export SCREENCAPTURE_PERMISSION_TIMEOUT=120
./screen-capture -app Safari

# Customize all permission waiting parameters
export SCREENCAPTURE_PERMISSION_TIMEOUT=90       # Total timeout in seconds
export SCREENCAPTURE_PERMISSION_ATTEMPTS=15      # Max attempts (safety net)
export SCREENCAPTURE_PERMISSION_DELAY=400        # Initial delay in milliseconds
export SCREENCAPTURE_PERMISSION_MAX_DELAY=4000   # Max delay between attempts
./screen-capture -app Safari
```

Permission environment variables:
- `SCREENCAPTURE_PERMISSION_TIMEOUT`: Total timeout in seconds (default: 60)
- `SCREENCAPTURE_PERMISSION_ATTEMPTS`: Maximum attempts (default: 10)
- `SCREENCAPTURE_PERMISSION_DELAY`: Initial retry delay in milliseconds (default: 500)
- `SCREENCAPTURE_PERMISSION_MAX_DELAY`: Maximum retry delay in milliseconds (default: 5000)

The permission check will now provide better feedback including:
- Elapsed time and time remaining
- Clear timeout messages with configuration hints
- Detailed instructions for granting permission

### Retry Behavior

**Transient Errors (retried automatically):**
- Window-related errors (invalid window ID, window closed)
- WindowServer communication timeouts
- Core Graphics errors (CGError, kCGError)
- System busy / temporarily unavailable
- Screen locked during capture
- Exit codes 1-9 (typically transient)

**Non-Transient Errors (fail immediately):**
- Permission denied (missing Screen Recording permission)
- Invalid command arguments
- File system errors (disk full, read-only filesystem)
- Command not found errors

**Exponential Backoff:**
The retry delay increases with each attempt using the formula:
```
delay = base_delay * (1 + (attempt - 1) * (backoff - 1))
delay = min(delay, max_delay)
```

Example with defaults (base=500ms, backoff=2.0):
- Attempt 1: immediate
- Attempt 2: after 500ms delay
- Attempt 3: after 1500ms delay (if max attempts = 3)

### Debug Logging

Enable detailed retry logging with `MACGO_DEBUG=1`:

```bash
MACGO_DEBUG=1 ./screen-capture -app Safari -retry-attempts 3
```

Debug output shows:
- Each retry attempt number
- Transient error detection
- Calculated retry delays
- Success/failure after retries
- Total attempts made

Example debug output:
```
[screen-capture:12345] Executing: screencapture [-l 456 /tmp/screenshot.png]
[screen-capture:12345] ⚠ Transient error (attempt 1/3), retrying in 500ms: window error
[screen-capture:12345] Retry attempt 2/3
[screen-capture:12345] ✓ Succeeded on retry attempt 2
```

## Usage

```bash
# Build the example
go build -o screen-capture

# Basic captures
./screen-capture                                    # Full screen
./screen-capture -window                            # Interactive window selection
./screen-capture -selection                         # Interactive area selection

# Coordinate captures
./screen-capture -region 100,100,800,600            # Specific pixel region
./screen-capture -it2-session ABC123                # iTerm2 session pane
./screen-capture -it2-session $(it2 session current) # Current iTerm2 pane

# Advanced options
./screen-capture -delay 3 -output ~/Desktop/shot.png # Delay and custom output
./screen-capture -display 2                         # Specific display
```

## iTerm2 Integration

When used with the `it2` tool, this example can automatically capture precise iTerm2 session panes:

```bash
# Capture current iTerm2 pane
./screen-capture -it2-session $(it2 session current)

# Capture specific session (get ID from: it2 session list)
./screen-capture -it2-session "D52126DF-26D9-44C8-B304-D1FEEC3F3A3C"
```

The tool will:
1. Call `it2 session get-info <session-id> --json`
2. Extract frame coordinates from the JSON response
3. Use those coordinates with `screencapture -R`
4. Capture the exact pane area

## Permissions

This example requests the following macOS permissions through macgo:
- **Screen Recording**: For capturing screen content
- **Files**: For saving screenshot files to disk

You may be prompted to grant these permissions in System Settings on first run.

## Requirements

- macOS (uses system `screencapture` command)
- For iTerm2 integration: `it2` command in PATH
- Go 1.21+ for building

## Implementation Details

The tool combines:
- macgo for proper macOS permission handling
- System `screencapture` command for actual capture
- `it2` tool integration for iTerm2 coordinate extraction
- JSON parsing for coordinate data processing

This creates a seamless workflow for capturing terminal content with precise positioning.