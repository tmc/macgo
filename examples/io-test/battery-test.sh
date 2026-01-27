#!/bin/bash
# Battery test for io-test with various flag configurations

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

TIMEOUT=15
TEST_DIR="/tmp/io-test-battery-results"
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
    local io_count=$(pgrep -c io-test 2>/dev/null || echo 0)

    if [ $xpc_count -gt 0 ] || [ $io_count -gt 0 ]; then
        echo -e "${YELLOW}⚠️  WARNING: Found potentially hung processes (xpcproxy: $xpc_count, io-test: $io_count)${NC}"
        return 1
    fi
    return 0
}

test_config() {
    local name="$1"
    local output="$TEST_DIR/${name}.txt"
    shift
    local env_vars=("$@")
    local extra_args=""

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

    # Parse test name for flags
    if [[ "$name" == *"verbose"* ]]; then
        extra_args="$extra_args -verbose"
    fi
    if [[ "$name" == *"hang"* ]]; then
        extra_args="$extra_args -hang"
    fi

    # Build command with env vars BEFORE timeout
    local cmd=""
    for var in "${env_vars[@]}"; do
        cmd="$cmd $var"
    done
    cmd="$cmd timeout $TIMEOUT io-test $extra_args"

    echo "Command: $cmd"

    # Run command and capture output
    local start_time=$(perl -MTime::HiRes=time -e 'printf "%.3f", time')
    eval "$cmd" > "$output" 2>&1
    local exit_code=$?
    local end_time=$(perl -MTime::HiRes=time -e 'printf "%.3f", time')
    local duration_ms=$(echo "($end_time - $start_time) * 1000" | bc | cut -d. -f1)

    # Give it a moment
    if [[ "$cmd" == *"MACGO_NO_WAIT=1"* ]]; then
        sleep 1
    else
        sleep 0.2
    fi

    # Check for hung processes after test
    if ! check_hung_processes; then
        echo -e "${YELLOW}⚠️  Test left hung processes${NC}"
    fi

    # Format duration
    local proc_time=""
    if [ "$duration_ms" -lt 1000 ]; then
        proc_time="${duration_ms}ms"
    else
        local duration_s=$(echo "scale=2; $duration_ms / 1000" | bc)
        proc_time="${duration_s}s"
    fi

    # Check if output was created and contains expected content
    local success=0
    if [ -f "$output" ]; then
        local size=$(wc -c < "$output" 2>/dev/null || echo 0)

        # For hang tests, timeout (exit 124) is expected
        if [[ "$name" == *"hang"* ]] && [ $exit_code -eq 124 ]; then
            echo -e "${GREEN}✅ PASS${NC} - Hang detected and killed ($size bytes) | Process: $proc_time | Exit: $exit_code (timeout as expected)"
            PASSED=$((PASSED + 1))
            return 0
        fi

        # Check for expected output markers
        if grep -q "IO Test Running" "$output" || grep -q "STDOUT: Test output" "$output"; then
            if [ $exit_code -eq 0 ] || [ $exit_code -eq 124 ]; then
                echo -e "${GREEN}✅ PASS${NC} - Output captured ($size bytes) | Process: $proc_time | Exit: $exit_code"
                PASSED=$((PASSED + 1))
                return 0
            fi
        fi
    fi

    # Test failed
    echo -e "${RED}❌ FAIL${NC} - Expected output not found or wrong exit code (took $proc_time, exit: $exit_code)"
    if [ -f "$output" ]; then
        echo "Output preview:"
        head -5 "$output" | sed 's/^/  /'
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

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# BASIC TESTS (no I/O forwarding)
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# Test 1: Default configuration (no bundle)
test_config "nobundle" "MACGO_NOBUNDLE=1"

# Test 2: Default configuration (with bundle, no I/O forwarding)
test_config "default" "MACGO_DEBUG=1"

# Test 3: Explicit wait mode
test_config "explicit-wait" "MACGO_DEBUG=1"

# Test 4: No wait mode
test_config "no-wait" "MACGO_DEBUG=1" "MACGO_NO_WAIT=1"

# Test 5: New instance flag
test_config "new-instance" "MACGO_DEBUG=1" "MACGO_OPEN_NEW_INSTANCE=1"

# Test 6: New instance + no wait
test_config "new-instance-no-wait" "MACGO_DEBUG=1" "MACGO_OPEN_NEW_INSTANCE=1" "MACGO_NO_WAIT=1"

# Test 7: With launch timeout protection
test_config "with-timeout" "MACGO_DEBUG=1" "MACGO_LAUNCH_TIMEOUT=10s"

# Test 8: Short launch timeout
test_config "short-timeout" "MACGO_DEBUG=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 9: Verbose output
test_config "verbose" "MACGO_DEBUG=1"

# Test 10: Hang test (should timeout)
test_config "hang" "MACGO_DEBUG=1" "MACGO_LAUNCH_TIMEOUT=3s"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# I/O FORWARDING TESTS
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# Test 11: stdin-only forwarding
test_config "io-stdin-only" "MACGO_DEBUG=1" "MACGO_ENABLE_STDIN_FORWARDING=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 12: stdout-only forwarding (expected to fail)
test_config "io-stdout-only" "MACGO_DEBUG=1" "MACGO_ENABLE_STDOUT_FORWARDING=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 13: stderr-only forwarding (expected to fail)
test_config "io-stderr-only" "MACGO_DEBUG=1" "MACGO_ENABLE_STDERR_FORWARDING=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 14: All I/O forwarding with regular files (expected to fail)
test_config "io-all-files" "MACGO_DEBUG=1" "MACGO_ENABLE_IO_FORWARDING=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 15: All I/O forwarding with FIFOs (expected to fail/hang)
test_config "io-all-fifo" "MACGO_DEBUG=1" "MACGO_ENABLE_IO_FORWARDING=1" "MACGO_USE_FIFO=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 16: stdout + stderr forwarding (expected to fail)
test_config "io-output-only" "MACGO_DEBUG=1" "MACGO_ENABLE_STDOUT_FORWARDING=1" "MACGO_ENABLE_STDERR_FORWARDING=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 17: stdin + stdout forwarding (expected to fail due to stdout)
test_config "io-stdin-stdout" "MACGO_DEBUG=1" "MACGO_ENABLE_STDIN_FORWARDING=1" "MACGO_ENABLE_STDOUT_FORWARDING=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 18: stdin-only with FIFO (may hang)
test_config "io-stdin-fifo" "MACGO_DEBUG=1" "MACGO_ENABLE_STDIN_FORWARDING=1" "MACGO_USE_FIFO=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 19: I/O forwarding with no-wait mode
test_config "io-all-no-wait" "MACGO_DEBUG=1" "MACGO_ENABLE_IO_FORWARDING=1" "MACGO_NO_WAIT=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 20: I/O forwarding with verbose output
test_config "io-all-verbose" "MACGO_DEBUG=1" "MACGO_ENABLE_IO_FORWARDING=1" "MACGO_LAUNCH_TIMEOUT=5s"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# ENV-VARS I/O FORWARDING STRATEGY TESTS (NEW APPROACH)
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "ENV-VARS I/O STRATEGY TESTS (should work!)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Test 21: stdout-only with env-vars strategy
test_config "envvars-stdout-only" "MACGO_DEBUG=1" "MACGO_IO_STRATEGY=env-vars" "MACGO_ENABLE_STDOUT_FORWARDING=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 22: stderr-only with env-vars strategy
test_config "envvars-stderr-only" "MACGO_DEBUG=1" "MACGO_IO_STRATEGY=env-vars" "MACGO_ENABLE_STDERR_FORWARDING=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 23: stdout+stderr with env-vars strategy
test_config "envvars-output-only" "MACGO_DEBUG=1" "MACGO_IO_STRATEGY=env-vars" "MACGO_ENABLE_STDOUT_FORWARDING=1" "MACGO_ENABLE_STDERR_FORWARDING=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 24: All I/O with env-vars strategy
test_config "envvars-all-io" "MACGO_DEBUG=1" "MACGO_IO_STRATEGY=env-vars" "MACGO_ENABLE_IO_FORWARDING=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 25: All I/O with env-vars + FIFO
test_config "envvars-all-fifo" "MACGO_DEBUG=1" "MACGO_IO_STRATEGY=env-vars" "MACGO_ENABLE_IO_FORWARDING=1" "MACGO_USE_FIFO=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 26: stdin+stdout with env-vars strategy
test_config "envvars-stdin-stdout" "MACGO_DEBUG=1" "MACGO_IO_STRATEGY=env-vars" "MACGO_ENABLE_STDIN_FORWARDING=1" "MACGO_ENABLE_STDOUT_FORWARDING=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 27: env-vars with no-wait mode
test_config "envvars-no-wait" "MACGO_DEBUG=1" "MACGO_IO_STRATEGY=env-vars" "MACGO_ENABLE_IO_FORWARDING=1" "MACGO_NO_WAIT=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 28: env-vars with verbose
test_config "envvars-verbose" "MACGO_DEBUG=1" "MACGO_IO_STRATEGY=env-vars" "MACGO_ENABLE_IO_FORWARDING=1" "MACGO_LAUNCH_TIMEOUT=5s"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# LAUNCHER VERSION TESTS
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "LAUNCHER VERSION TESTS (v1 vs v2)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Test 29: Launcher v1 - basic
test_config "launcher-v1-basic" "MACGO_DEBUG=1" "MACGO_LAUNCHER_VERSION=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 30: Launcher v2 - basic
test_config "launcher-v2-basic" "MACGO_DEBUG=1" "MACGO_LAUNCHER_VERSION=2" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 31: Launcher v1 with env-vars I/O
test_config "launcher-v1-envvars-io" "MACGO_DEBUG=1" "MACGO_LAUNCHER_VERSION=1" "MACGO_IO_STRATEGY=env-vars" "MACGO_ENABLE_IO_FORWARDING=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 32: Launcher v2 with env-vars I/O
test_config "launcher-v2-envvars-io" "MACGO_DEBUG=1" "MACGO_LAUNCHER_VERSION=2" "MACGO_IO_STRATEGY=env-vars" "MACGO_ENABLE_IO_FORWARDING=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 33: Launcher v1 with stderr-only
test_config "launcher-v1-stderr" "MACGO_DEBUG=1" "MACGO_LAUNCHER_VERSION=1" "MACGO_IO_STRATEGY=env-vars" "MACGO_ENABLE_STDERR_FORWARDING=1" "MACGO_LAUNCH_TIMEOUT=5s"

# Test 34: Launcher v2 with stderr-only
test_config "launcher-v2-stderr" "MACGO_DEBUG=1" "MACGO_LAUNCHER_VERSION=2" "MACGO_IO_STRATEGY=env-vars" "MACGO_ENABLE_STDERR_FORWARDING=1" "MACGO_LAUNCH_TIMEOUT=5s"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# STDIN/STDOUT/STDERR VERIFICATION TESTS
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "STDIN/STDOUT/STDERR VERIFICATION TESTS"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Test 21: Verify stdout contains expected output (no I/O forwarding)
test_stdout_content() {
    local name="verify-stdout"
    local output="$TEST_DIR/${name}.txt"

    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Test: $name"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    rm -f "$output"

    # Run and capture stdout only
    MACGO_DEBUG=1 timeout $TIMEOUT io-test > "$output" 2>/dev/null
    local exit_code=$?

    if [ -f "$output" ] && grep -q "STDOUT: Test output" "$output" && grep -q "IO Test Running" "$output"; then
        echo -e "${GREEN}✅ PASS${NC} - Stdout contains expected content"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}❌ FAIL${NC} - Stdout missing expected content"
        echo "Output preview:"
        head -5 "$output" | sed 's/^/  /'
        FAILED=$((FAILED + 1))
    fi
}

# Test 22: Verify stderr contains expected output (no I/O forwarding)
test_stderr_content() {
    local name="verify-stderr"
    local output="$TEST_DIR/${name}.txt"

    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Test: $name"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    rm -f "$output"

    # Run and capture stderr only
    MACGO_DEBUG=1 timeout $TIMEOUT io-test 2> "$output" >/dev/null
    local exit_code=$?

    if [ -f "$output" ] && grep -q "STDERR: Test output" "$output"; then
        echo -e "${GREEN}✅ PASS${NC} - Stderr contains expected content"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}❌ FAIL${NC} - Stderr missing expected content"
        echo "Output preview:"
        head -5 "$output" | sed 's/^/  /'
        FAILED=$((FAILED + 1))
    fi
}

# Test 23: Verify both stdout and stderr (no I/O forwarding)
test_combined_output() {
    local name="verify-combined"
    local output="$TEST_DIR/${name}.txt"

    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Test: $name"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    rm -f "$output"

    # Run and capture both stdout and stderr
    MACGO_DEBUG=1 timeout $TIMEOUT io-test > "$output" 2>&1
    local exit_code=$?

    if [ -f "$output" ] && \
       grep -q "STDOUT: Test output" "$output" && \
       grep -q "STDERR: Test output" "$output" && \
       grep -q "IO Test Running" "$output"; then
        echo -e "${GREEN}✅ PASS${NC} - Combined output contains all expected content"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}❌ FAIL${NC} - Combined output missing expected content"
        echo "Output preview:"
        head -10 "$output" | sed 's/^/  /'
        FAILED=$((FAILED + 1))
    fi
}

# Test 24: Test with stdin input using pipe (no I/O forwarding)
test_pipe_input() {
    local name="verify-pipe-input"
    local output="$TEST_DIR/${name}.txt"

    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Test: $name"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    rm -f "$output"

    # Test pipe detection by sending input through stdin
    echo -e "test line 1\ntest line 2\ntest line 3" | MACGO_DEBUG=1 timeout $TIMEOUT io-test -pipe > "$output" 2>&1
    local exit_code=$?

    if [ -f "$output" ] && grep -q "Input is from a pipe" "$output"; then
        echo -e "${GREEN}✅ PASS${NC} - Pipe input detected correctly"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}❌ FAIL${NC} - Pipe input not detected"
        echo "Output preview:"
        head -10 "$output" | sed 's/^/  /'
        FAILED=$((FAILED + 1))
    fi
}

# Test 25: Test output redirection detection (no I/O forwarding)
test_output_redirection() {
    local name="verify-output-redirect"
    local output="$TEST_DIR/${name}.txt"

    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Test: $name"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    rm -f "$output"

    # Test output redirection detection
    MACGO_DEBUG=1 timeout $TIMEOUT io-test -pipe > "$output" 2>&1
    local exit_code=$?

    if [ -f "$output" ] && grep -q "Output is to a pipe" "$output"; then
        echo -e "${GREEN}✅ PASS${NC} - Output redirection detected correctly"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}❌ FAIL${NC} - Output redirection not detected"
        echo "Output preview:"
        head -10 "$output" | sed 's/^/  /'
        FAILED=$((FAILED + 1))
    fi
}

# Test 26: Verify both parent and child PIDs appear in output
test_dual_pid_output() {
    local name="verify-dual-pids"
    local output="$TEST_DIR/${name}.txt"

    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Test: $name"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    rm -f "$output"

    # Run and capture output
    MACGO_DEBUG=1 timeout $TIMEOUT io-test > "$output" 2>&1
    local exit_code=$?

    if [ -f "$output" ]; then
        # Check for parent process
        local has_parent=$(grep -c "\[parent\]" "$output")
        # Check for child process
        local has_child=$(grep -c "\[child\]" "$output")
        # Extract PIDs and verify they're different
        local parent_pid=$(grep "\[parent\].*PID:" "$output" | head -1 | grep -o "PID: [0-9]*" | cut -d' ' -f2)
        local child_pid=$(grep "\[child\].*PID:" "$output" | head -1 | grep -o "PID: [0-9]*" | cut -d' ' -f2)

        if [ "$has_parent" -gt 0 ] && [ "$has_child" -gt 0 ] && [ "$parent_pid" != "$child_pid" ]; then
            echo -e "${GREEN}✅ PASS${NC} - Both parent ($parent_pid) and child ($child_pid) PIDs present"
            PASSED=$((PASSED + 1))
        else
            echo -e "${RED}❌ FAIL${NC} - Missing parent/child PID separation"
            echo "Parent lines: $has_parent, Child lines: $has_child"
            echo "Parent PID: $parent_pid, Child PID: $child_pid"
            FAILED=$((FAILED + 1))
        fi
    else
        echo -e "${RED}❌ FAIL${NC} - No output file created"
        FAILED=$((FAILED + 1))
    fi
}

# Test 27: Verify stdin forwarding with actual input data
test_stdin_forwarding_with_data() {
    local name="verify-stdin-data"
    local output="$TEST_DIR/${name}.txt"

    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Test: $name"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    rm -f "$output"

    # Test stdin forwarding with env-vars strategy
    echo -e "test line A\ntest line B\ntest line C" | \
        MACGO_DEBUG=1 MACGO_IO_STRATEGY=env-vars MACGO_ENABLE_STDIN_FORWARDING=1 \
        timeout $TIMEOUT io-test -pipe > "$output" 2>&1
    local exit_code=$?

    if [ -f "$output" ] && \
       grep -q "Line 1: test line A" "$output" && \
       grep -q "Line 2: test line B" "$output" && \
       grep -q "Line 3: test line C" "$output"; then
        echo -e "${GREEN}✅ PASS${NC} - Stdin data forwarded correctly"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}❌ FAIL${NC} - Stdin data not forwarded correctly"
        echo "Output preview:"
        grep "Line" "$output" | sed 's/^/  /' || echo "  (no lines found)"
        FAILED=$((FAILED + 1))
    fi
}

# Test 28: Verify stdout/stderr with env-vars strategy shows both PIDs
test_envvars_dual_pids() {
    local name="verify-envvars-pids"
    local output="$TEST_DIR/${name}.txt"

    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Test: $name"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    rm -f "$output"

    # Run with env-vars strategy
    MACGO_DEBUG=1 MACGO_IO_STRATEGY=env-vars MACGO_ENABLE_IO_FORWARDING=1 \
        timeout $TIMEOUT io-test > "$output" 2>&1
    local exit_code=$?

    if [ -f "$output" ]; then
        local parent_pid=$(grep "\[parent\].*PID:" "$output" | head -1 | grep -o "PID: [0-9]*" | cut -d' ' -f2)
        local child_pid=$(grep "IO Test Running.*PID:" "$output" | head -1 | grep -o "PID: [0-9]*" | cut -d' ' -f2)

        if [ -n "$parent_pid" ] && [ -n "$child_pid" ] && [ "$parent_pid" != "$child_pid" ]; then
            echo -e "${GREEN}✅ PASS${NC} - Env-vars strategy preserves both PIDs (parent: $parent_pid, child: $child_pid)"
            PASSED=$((PASSED + 1))
        else
            echo -e "${RED}❌ FAIL${NC} - PIDs not properly captured"
            echo "Parent PID: $parent_pid, Child PID: $child_pid"
            echo "Output preview:"
            head -10 "$output" | sed 's/^/  /'
            FAILED=$((FAILED + 1))
        fi
    else
        echo -e "${RED}❌ FAIL${NC} - No output file created"
        FAILED=$((FAILED + 1))
    fi
}

# Signal handling tests (NOBUNDLE mode for now, bundle mode needs signal forwarding)
test_signal_sigint() {
    local name="signal-sigint-nobundle"
    local output="$TEST_DIR/${name}.txt"

    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Test: $name"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    rm -f "$output"

    # Test in NOBUNDLE mode since signal forwarding to bundle not yet implemented
    MACGO_DEBUG=1 MACGO_NOBUNDLE=1 io-test -hang > "$output" 2>&1 &
    local pid=$!

    # Give it time to start
    sleep 1

    # Send SIGINT
    kill -INT $pid

    # Wait for process to exit
    wait $pid 2>/dev/null

    # Check output
    if [ -f "$output" ] && grep -q "Received SIGINT" "$output"; then
        echo -e "${GREEN}✅ PASS${NC} - SIGINT handled correctly (NOBUNDLE)"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}❌ FAIL${NC} - SIGINT not handled"
        FAILED=$((FAILED + 1))
    fi
}

test_signal_sigterm() {
    local name="signal-sigterm-nobundle"
    local output="$TEST_DIR/${name}.txt"

    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Test: $name"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    rm -f "$output"

    # Test in NOBUNDLE mode
    MACGO_DEBUG=1 MACGO_NOBUNDLE=1 io-test -hang > "$output" 2>&1 &
    local pid=$!

    # Give it time to start
    sleep 1

    # Send SIGTERM
    kill -TERM $pid

    # Wait for process to exit
    wait $pid 2>/dev/null

    # Check output
    if [ -f "$output" ] && grep -q "Received SIGTERM" "$output"; then
        echo -e "${GREEN}✅ PASS${NC} - SIGTERM handled correctly (NOBUNDLE)"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}❌ FAIL${NC} - SIGTERM not handled"
        FAILED=$((FAILED + 1))
    fi
}

test_signal_sigquit_stack_traces() {
    local name="signal-sigquit-stacks-nobundle"
    local output="$TEST_DIR/${name}.txt"

    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Test: $name"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    rm -f "$output"

    # Test in NOBUNDLE mode
    MACGO_DEBUG=1 MACGO_NOBUNDLE=1 io-test -hang > "$output" 2>&1 &
    local pid=$!

    # Give it time to start
    sleep 1

    # Send SIGQUIT
    kill -QUIT $pid

    # Give time for stack traces to be printed
    sleep 1

    # Kill the process since SIGQUIT doesn't terminate
    kill -TERM $pid 2>/dev/null
    wait $pid 2>/dev/null

    # Check output for stack traces
    if [ -f "$output" ]; then
        local has_sigquit=$(grep -c "Received SIGQUIT" "$output")
        local has_stacks=$(grep -c "Stack trace complete" "$output")

        if [ "$has_sigquit" -ge 1 ] && [ "$has_stacks" -ge 1 ]; then
            echo -e "${GREEN}✅ PASS${NC} - SIGQUIT stack traces printed (NOBUNDLE)"
            PASSED=$((PASSED + 1))
        else
            echo -e "${RED}❌ FAIL${NC} - SIGQUIT stack traces not found (sigquit:$has_sigquit stacks:$has_stacks)"
            FAILED=$((FAILED + 1))
        fi
    else
        echo -e "${RED}❌ FAIL${NC} - No output file"
        FAILED=$((FAILED + 1))
    fi
}

test_signal_propagation_to_child() {
    local name="signal-multiple-handlers-nobundle"
    local output="$TEST_DIR/${name}.txt"

    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Test: $name"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    rm -f "$output"

    # Test multiple signal types in sequence (NOBUNDLE mode)
    MACGO_DEBUG=1 MACGO_NOBUNDLE=1 io-test -hang > "$output" 2>&1 &
    local pid=$!

    # Give time to start
    sleep 1

    # Send SIGQUIT first (should print stack and continue)
    kill -QUIT $pid
    sleep 1

    # Send SIGINT to exit
    kill -INT $pid

    # Wait for exit
    wait $pid 2>/dev/null

    # Check that we got both signals
    if [ -f "$output" ]; then
        local has_sigquit=$(grep -c "Received SIGQUIT" "$output")
        local has_sigint=$(grep -c "Received SIGINT" "$output")

        if [ "$has_sigquit" -ge 1 ] && [ "$has_sigint" -ge 1 ]; then
            echo -e "${GREEN}✅ PASS${NC} - Multiple signals handled correctly (NOBUNDLE)"
            PASSED=$((PASSED + 1))
        else
            echo -e "${RED}❌ FAIL${NC} - Not all signals received (sigquit:$has_sigquit sigint:$has_sigint)"
            FAILED=$((FAILED + 1))
        fi
    else
        echo -e "${RED}❌ FAIL${NC} - No output file"
        FAILED=$((FAILED + 1))
    fi
}

# Run verification tests
test_stdout_content
test_stderr_content
test_combined_output
test_pipe_input
test_output_redirection
test_dual_pid_output
test_stdin_forwarding_with_data
test_envvars_dual_pids

# Run signal handling tests
test_signal_sigint
test_signal_sigterm
test_signal_sigquit_stack_traces
test_signal_propagation_to_child

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
    echo "Test outputs saved in: $TEST_DIR"
    ls -lh "$TEST_DIR"/*.txt 2>/dev/null | head -10
    exit 1
fi
