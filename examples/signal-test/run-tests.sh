#!/bin/bash
# Signal and I/O test runner for macgo
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
MACGO_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
TEST_BIN="/tmp/signal-test"
BUNDLE_PATH="$HOME/go/bin/signal-test.app"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log() { echo -e "${CYAN}==>${NC} $*"; }
pass() { echo -e "${GREEN}PASS${NC}: $*"; }
fail() { echo -e "${RED}FAIL${NC}: $*"; }
warn() { echo -e "${YELLOW}WARN${NC}: $*"; }

# Build test binary
build() {
    log "Building signal-test..."
    cd "$MACGO_DIR"
    go build -o "$TEST_BIN" ./examples/signal-test/
    rm -rf "$BUNDLE_PATH"
    log "Built: $TEST_BIN"
}

# Run a signal test and capture output
# Args: signal_name timeout
test_signal() {
    local sig="$1"
    local timeout="${2:-5}"
    local output_file="/tmp/signal-test-$sig.out"

    log "Testing $sig..."

    # Start the test program
    MACGO_DEBUG=1 "$TEST_BIN" > "$output_file" 2>&1 &
    local parent_pid=$!

    # Wait for it to start and get child PID
    sleep 2

    # Find the actual app PID (child of open command, or in the bundle)
    local child_pid=$(pgrep -P "$parent_pid" 2>/dev/null | head -1)
    if [ -z "$child_pid" ]; then
        # Try finding by bundle name
        child_pid=$(pgrep -f "signal-test.app" 2>/dev/null | grep -v "$parent_pid" | head -1)
    fi

    if [ -z "$child_pid" ]; then
        warn "Could not find child PID, using parent PID"
        child_pid=$parent_pid
    fi

    log "  Parent PID: $parent_pid, Child PID: $child_pid"

    # Send the signal to the child
    kill -"$sig" "$child_pid" 2>/dev/null || true

    # Wait for process to exit
    local waited=0
    while kill -0 "$parent_pid" 2>/dev/null && [ $waited -lt $timeout ]; do
        sleep 0.5
        waited=$((waited + 1))
    done

    # Force kill if still running
    if kill -0 "$parent_pid" 2>/dev/null; then
        warn "Process still running after ${timeout}s, killing..."
        kill -9 "$parent_pid" 2>/dev/null || true
    fi

    # Check exit code
    wait "$parent_pid" 2>/dev/null
    local exit_code=$?

    # Display results
    echo "  Exit code: $exit_code"
    echo "  Output (last 20 lines):"
    tail -20 "$output_file" | sed 's/^/    /'

    # Return the output file for further analysis
    echo "$output_file"
}

# Test: SIGINT should exit cleanly
test_sigint() {
    local out=$(test_signal INT)
    if grep -q "Received signal: interrupt" "$out"; then
        pass "SIGINT received by app"
    else
        warn "SIGINT may not have been logged"
    fi
}

# Test: SIGQUIT should dump stacks
test_sigquit() {
    local out=$(test_signal QUIT)
    if grep -q "goroutine dump" "$out"; then
        pass "SIGQUIT dumped goroutine stacks"
    else
        fail "SIGQUIT did not dump stacks"
    fi
}

# Test: SIGHUP
test_sighup() {
    local out=$(test_signal HUP)
    if grep -q "Received signal: hangup" "$out"; then
        pass "SIGHUP received by app"
    else
        warn "SIGHUP may not have been received"
    fi
}

# Test: TTY info
test_tty() {
    log "Testing TTY passthrough..."
    echo "=== Without MACGO_TTY_PASSTHROUGH ==="
    MACGO_DEBUG=1 go run "$SCRIPT_DIR/tty-info.go" 2>&1 | head -30

    echo ""
    echo "=== With MACGO_TTY_PASSTHROUGH=1 ==="
    MACGO_TTY_PASSTHROUGH=1 MACGO_DEBUG=1 go run "$SCRIPT_DIR/tty-info.go" 2>&1 | head -30
}

# Interactive test mode
interactive() {
    log "Starting interactive signal test..."
    log "Use Ctrl+C, Ctrl+\\, etc. to test signals"
    log "Press Ctrl+D or type 'exit' to quit"
    MACGO_DEBUG=1 "$TEST_BIN"
}

# Main
case "${1:-all}" in
    build)
        build
        ;;
    sigint)
        build
        test_sigint
        ;;
    sigquit)
        build
        test_sigquit
        ;;
    sighup)
        build
        test_sighup
        ;;
    tty)
        test_tty
        ;;
    interactive|i)
        build
        interactive
        ;;
    all)
        build
        echo ""
        test_sigint
        echo ""
        test_sigquit
        echo ""
        test_sighup
        ;;
    *)
        echo "Usage: $0 {build|sigint|sigquit|sighup|tty|interactive|all}"
        exit 1
        ;;
esac
