#!/bin/bash
# Battery test for screen-capture with various flag configurations

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

TIMEOUT=15
TEST_DIR="/tmp/screen-capture-battery-tests"
PASSED=0
FAILED=0

mkdir -p "$TEST_DIR"

# Helper function to kill hung xpcproxy processes
kill_xpcproxy() {
    local count=0
    echo -e "${BLUE}🔍 Checking for hung xpcproxy processes...${NC}"
    while killall -9 xpcproxy 2>/dev/null; do
        count=$((count + 1))
        echo -e "${YELLOW}⚠️  Killed xpcproxy instance #$count${NC}"
        sleep 0.5
    done
    if [ $count -gt 0 ]; then
        echo -e "${GREEN}✓ Cleaned up $count hung xpcproxy process(es)${NC}"
    else
        echo -e "${GREEN}✓ No hung xpcproxy processes found${NC}"
    fi
}

# Helper function to check for and report hung processes
check_hung_processes() {
    local xpc_count=$(pgrep -c xpcproxy 2>/dev/null || echo 0)
    local sc_count=$(pgrep -c screen-capture 2>/dev/null || echo 0)

    if [ $xpc_count -gt 0 ] || [ $sc_count -gt 0 ]; then
        echo -e "${YELLOW}⚠️  WARNING: Found potentially hung processes (xpcproxy: $xpc_count, screen-capture: $sc_count)${NC}"
        return 1
    fi
    return 0
}

test_config() {
    local name="$1"
    local output="$TEST_DIR/${name}.png"
    shift
    local env_vars=("$@")

    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Test: $name (timeout: ${TIMEOUT}s)"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    # Check for hung processes before starting test
    if ! check_hung_processes; then
        echo -e "${BLUE}🧹 Cleaning up before test...${NC}"
        kill_xpcproxy
    fi

    rm -f "$output"

    # Build command with env vars BEFORE timeout
    local cmd=""
    for var in "${env_vars[@]}"; do
        cmd="$cmd $var"
    done
    cmd="$cmd timeout $TIMEOUT screen-capture -app 'System Settings' '$output'"

    echo "Command: $cmd"

    # Run command with NO redirection - isPipeOutput() detects any output redirection
    # Just run it and capture the exit code
    # Use nanoseconds for accurate timing (macOS date doesn't support %N, use gdate if available)
    if command -v gdate >/dev/null 2>&1; then
        local start_ms=$(gdate +%s%3N)
        eval "$cmd" >/dev/null 2>&1
        local exit_code=$?
        local end_ms=$(gdate +%s%3N)
        local duration_ms=$((end_ms - start_ms))
    else
        # Fallback to seconds with millisecond estimation
        local start_time=$(perl -MTime::HiRes=time -e 'printf "%.3f", time')
        eval "$cmd" >/dev/null 2>&1
        local exit_code=$?
        local end_time=$(perl -MTime::HiRes=time -e 'printf "%.3f", time')
        local duration_ms=$(echo "($end_time - $start_time) * 1000" | bc | cut -d. -f1)
    fi

    # Give it a moment for file to be written
    # For no-wait mode, need extra time since app runs in background
    if [[ "$cmd" == *"MACGO_NO_WAIT=1"* ]]; then
        sleep 3
    else
        sleep 0.5
    fi

    # Check for hung processes after test
    if ! check_hung_processes; then
        echo -e "${YELLOW}⚠️  Test left hung processes${NC}"
    fi

    # Check if file was created
    if [ -f "$output" ]; then
        local size=$(stat -f%z "$output" 2>/dev/null || echo 0)
        if [ "$size" -gt 0 ]; then
            # Get file creation time and calculate age
            local file_mtime=$(stat -f%m "$output" 2>/dev/null || echo 0)
            local current_time=$(date +%s)
            local file_age=$((current_time - file_mtime))

            # Format process duration
            local proc_time=""
            if [ "$duration_ms" -lt 1000 ]; then
                proc_time="${duration_ms}ms"
            else
                local duration_s=$(echo "scale=2; $duration_ms / 1000" | bc)
                proc_time="${duration_s}s"
            fi

            echo -e "${GREEN}✅ PASS${NC} - File created ($size bytes) | Process: $proc_time | File age: ${file_age}s"
            PASSED=$((PASSED + 1))
            return 0
        fi
    fi

    # Format duration for failure too
    if [ "$duration_ms" -lt 1000 ]; then
        echo -e "${RED}❌ FAIL${NC} - File not created or empty (took ${duration_ms}ms)"
    else
        local duration_s=$(echo "scale=2; $duration_ms / 1000" | bc)
        echo -e "${RED}❌ FAIL${NC} - File not created or empty (took ${duration_s}s)"
    fi
    FAILED=$((FAILED + 1))
    return 1
}

echo "Installing latest version..."
cd "$(dirname "$0")"
go install

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "PRE-TEST CLEANUP"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
kill_xpcproxy

echo ""
echo "Running battery tests with ${TIMEOUT}s timeout per test..."

# Test 1: Default configuration
test_config "default" "MACGO_DEBUG=1"

# Test 2: Explicit wait mode (should be same as default)
test_config "explicit-wait" "MACGO_DEBUG=1"

# Test 3: No wait mode
test_config "no-wait" "MACGO_DEBUG=1" "MACGO_NO_WAIT=1"

# Test 4: New instance flag
test_config "new-instance" "MACGO_DEBUG=1" "MACGO_OPEN_NEW_INSTANCE=1"

# Test 5: New instance + no wait
test_config "new-instance-no-wait" "MACGO_DEBUG=1" "MACGO_OPEN_NEW_INSTANCE=1" "MACGO_NO_WAIT=1"

# Test 6: With launch timeout protection
test_config "with-timeout" "MACGO_DEBUG=1" "MACGO_LAUNCH_TIMEOUT=10s"

# Test 7: Short launch timeout (should still work)
test_config "short-timeout" "MACGO_DEBUG=1" "MACGO_LAUNCH_TIMEOUT=5s"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# I/O FORWARDING TESTS
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# NOTE: These are known to NOT work with .app bundles due to macOS limitations
# Testing with short timeout (5s) to prevent long hangs

# Test 8: All I/O forwarding (stdin+stdout+stderr) with regular files
test_config "io-all-files" "MACGO_DEBUG=1" "MACGO_ENABLE_IO_FORWARDING=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 9: All I/O forwarding with FIFOs (may hang more)
test_config "io-all-fifo" "MACGO_DEBUG=1" "MACGO_ENABLE_IO_FORWARDING=1" "MACGO_USE_FIFO=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 10: Only stdin forwarding
test_config "io-stdin-only" "MACGO_DEBUG=1" "MACGO_ENABLE_STDIN_FORWARDING=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 11: Only stdout forwarding
test_config "io-stdout-only" "MACGO_DEBUG=1" "MACGO_ENABLE_STDOUT_FORWARDING=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 12: Only stderr forwarding
test_config "io-stderr-only" "MACGO_DEBUG=1" "MACGO_ENABLE_STDERR_FORWARDING=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 13: stdout + stderr (no stdin)
test_config "io-output-only" "MACGO_DEBUG=1" "MACGO_ENABLE_STDOUT_FORWARDING=1" "MACGO_ENABLE_STDERR_FORWARDING=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 14: stdin + stdout (no stderr)
test_config "io-stdin-stdout" "MACGO_DEBUG=1" "MACGO_ENABLE_STDIN_FORWARDING=1" "MACGO_ENABLE_STDOUT_FORWARDING=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 15: stdin only with FIFO
test_config "io-stdin-fifo" "MACGO_DEBUG=1" "MACGO_ENABLE_STDIN_FORWARDING=1" "MACGO_USE_FIFO=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 16: stdout only with FIFO
test_config "io-stdout-fifo" "MACGO_DEBUG=1" "MACGO_ENABLE_STDOUT_FORWARDING=1" "MACGO_USE_FIFO=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 17: I/O forwarding with no-wait mode
test_config "io-all-no-wait" "MACGO_DEBUG=1" "MACGO_ENABLE_IO_FORWARDING=1" "MACGO_NO_WAIT=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 18: I/O forwarding with new instance
test_config "io-all-new-instance" "MACGO_DEBUG=1" "MACGO_ENABLE_IO_FORWARDING=1" "MACGO_OPEN_NEW_INSTANCE=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Final cleanup check
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "POST-TEST CLEANUP"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
kill_xpcproxy

# Summary
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "SUMMARY"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
TOTAL=$((PASSED + FAILED))
echo "Total:  $TOTAL"
echo -e "${GREEN}Passed: $PASSED${NC}"
echo -e "${RED}Failed: $FAILED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All tests passed!${NC}"
    exit 0
else
    echo -e "${YELLOW}⚠️  Some tests failed${NC}"
    echo "Created files in: $TEST_DIR"
    ls -lh "$TEST_DIR"/*.png 2>/dev/null || echo "No PNG files created"
    exit 1
fi
