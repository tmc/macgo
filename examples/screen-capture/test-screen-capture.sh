#!/bin/bash
# Test script for screen-capture with various configurations

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="/tmp/screen-capture-tests"
MARKER_FILE="/tmp/screen-capture-test-marker"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Setup
echo "Setting up test environment..."
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR"
cd "$SCRIPT_DIR"
go install

# Test counter
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

run_test() {
    local test_name="$1"
    local output_file="$2"
    shift 2
    local cmd=("$@")

    TESTS_RUN=$((TESTS_RUN + 1))
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Test $TESTS_RUN: $test_name"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    # Clean up from previous test
    rm -f "$output_file" "$MARKER_FILE"

    # Run the command
    echo "Running: ${cmd[*]}"
    if "${cmd[@]}" 2>&1; then
        cmd_exit=$?
    else
        cmd_exit=$?
    fi

    # Check results
    local success=false
    local reason=""

    if [ -f "$MARKER_FILE" ]; then
        echo "Marker file content:"
        cat "$MARKER_FILE"
        if grep -q "SUCCESS:" "$MARKER_FILE"; then
            success=true
        else
            reason="Marker file indicates failure"
        fi
    elif [ -f "$output_file" ]; then
        local size=$(stat -f%z "$output_file" 2>/dev/null || echo 0)
        if [ "$size" -gt 0 ]; then
            success=true
            echo "File created: $output_file ($size bytes)"
        else
            reason="File created but empty"
        fi
    else
        reason="No output file created"
    fi

    if [ "$success" = true ]; then
        echo -e "${GREEN}✓ PASS${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}✗ FAIL${NC}: $reason"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# Test 1: Default mode (should work)
run_test "Default mode with -W flag" \
    "$TEST_DIR/test1.png" \
    env MACGO_DEBUG=1 MACGO_TEST_MARKER="$MARKER_FILE" \
    screen-capture -app "System Settings" "$TEST_DIR/test1.png"

# Test 2: NO_WAIT mode
run_test "NO_WAIT mode (no -W flag)" \
    "$TEST_DIR/test2.png" \
    env MACGO_DEBUG=1 MACGO_NO_WAIT=1 MACGO_TEST_MARKER="$MARKER_FILE" \
    screen-capture -app "System Settings" "$TEST_DIR/test2.png"

# Test 3: NEW_INSTANCE mode
run_test "NEW_INSTANCE mode (-n flag)" \
    "$TEST_DIR/test3.png" \
    env MACGO_DEBUG=1 MACGO_OPEN_NEW_INSTANCE=1 MACGO_TEST_MARKER="$MARKER_FILE" \
    screen-capture -app "System Settings" "$TEST_DIR/test3.png"

# Test 4: With I/O forwarding enabled (should fail or work depending on fix)
run_test "I/O forwarding enabled (experimental)" \
    "$TEST_DIR/test4.png" \
    env MACGO_DEBUG=1 MACGO_ENABLE_IO_FORWARDING=1 MACGO_LAUNCH_TIMEOUT=5s MACGO_TEST_MARKER="$MARKER_FILE" \
    timeout 10 screen-capture -app "System Settings" "$TEST_DIR/test4.png"

# Test 5: Capture with timeout protection
run_test "With launch timeout protection" \
    "$TEST_DIR/test5.png" \
    env MACGO_DEBUG=1 MACGO_LAUNCH_TIMEOUT=10s MACGO_TEST_MARKER="$MARKER_FILE" \
    screen-capture -app "System Settings" "$TEST_DIR/test5.png"

# Summary
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST SUMMARY"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Total:  $TESTS_RUN"
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Failed: $TESTS_FAILED${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi
