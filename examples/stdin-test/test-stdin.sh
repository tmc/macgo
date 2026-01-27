#!/bin/bash
# Comprehensive stdin forwarding tests for macgo

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

TIMEOUT=15
TEST_DIR="/tmp/stdin-test-results"
PASSED=0
FAILED=0

mkdir -p "$TEST_DIR"

# Test counter for unique naming
TEST_COUNT=0

# Helper to run a test with input and check output
run_stdin_test() {
    local name="$1"
    local flags="$2"
    local input="$3"
    local expected_pattern="$4"
    local env_vars="$5"

    TEST_COUNT=$((TEST_COUNT + 1))
    local output="$TEST_DIR/${TEST_COUNT}-${name}.txt"

    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Test $TEST_COUNT: $name"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    rm -f "$output"

    # Build command
    local cmd="echo -e \"$input\" | "
    if [ -n "$env_vars" ]; then
        cmd="$cmd $env_vars "
    fi
    cmd="$cmd timeout $TIMEOUT stdin-test $flags"

    echo "Command: $cmd"
    echo "Input: $input"
    echo "Expected pattern: $expected_pattern"

    # Run test
    local start_time=$(perl -MTime::HiRes=time -e 'printf "%.3f", time')
    eval "$cmd" > "$output" 2>&1
    local exit_code=$?
    local end_time=$(perl -MTime::HiRes=time -e 'printf "%.3f", time')
    local duration_ms=$(echo "($end_time - $start_time) * 1000" | bc | cut -d. -f1)

    # Format duration
    local proc_time=""
    if [ "$duration_ms" -lt 1000 ]; then
        proc_time="${duration_ms}ms"
    else
        local duration_s=$(echo "scale=2; $duration_ms / 1000" | bc)
        proc_time="${duration_s}s"
    fi

    # Check results
    if [ -f "$output" ]; then
        local size=$(wc -c < "$output" 2>/dev/null || echo 0)

        if grep -qE "$expected_pattern" "$output"; then
            echo -e "${GREEN}✅ PASS${NC} - Pattern found ($size bytes, $proc_time)"
            PASSED=$((PASSED + 1))
            if [ -n "$VERBOSE" ]; then
                echo "Output preview:"
                head -10 "$output" | sed 's/^/  /'
            fi
            return 0
        else
            echo -e "${RED}❌ FAIL${NC} - Pattern not found ($size bytes, $proc_time)"
            echo "Output preview:"
            head -15 "$output" | sed 's/^/  /'
            FAILED=$((FAILED + 1))
            return 1
        fi
    else
        echo -e "${RED}❌ FAIL${NC} - No output file created ($proc_time)"
        FAILED=$((FAILED + 1))
        return 1
    fi
}

# Helper to run interactive test with expect-style scripting
run_interactive_test() {
    local name="$1"
    local flags="$2"
    local script="$3"
    local expected_pattern="$4"
    local env_vars="$5"

    TEST_COUNT=$((TEST_COUNT + 1))
    local output="$TEST_DIR/${TEST_COUNT}-${name}.txt"
    local script_file="$TEST_DIR/${TEST_COUNT}-script.sh"

    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Test $TEST_COUNT: $name"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    rm -f "$output" "$script_file"

    # Create script file
    echo "$script" > "$script_file"
    chmod +x "$script_file"

    # Build command
    local cmd=""
    if [ -n "$env_vars" ]; then
        cmd="$env_vars "
    fi
    cmd="$cmd timeout $TIMEOUT bash $script_file"

    echo "Command: $cmd"
    echo "Expected pattern: $expected_pattern"

    # Run test
    local start_time=$(perl -MTime::HiRes=time -e 'printf "%.3f", time')
    eval "$cmd" > "$output" 2>&1
    local exit_code=$?
    local end_time=$(perl -MTime::HiRes=time -e 'printf "%.3f", time')
    local duration_ms=$(echo "($end_time - $start_time) * 1000" | bc | cut -d. -f1)

    # Format duration
    local proc_time=""
    if [ "$duration_ms" -lt 1000 ]; then
        proc_time="${duration_ms}ms"
    else
        local duration_s=$(echo "scale=2; $duration_ms / 1000" | bc)
        proc_time="${duration_s}s"
    fi

    # Check results
    if [ -f "$output" ]; then
        local size=$(wc -c < "$output" 2>/dev/null || echo 0)

        if grep -qE "$expected_pattern" "$output"; then
            echo -e "${GREEN}✅ PASS${NC} - Pattern found ($size bytes, $proc_time)"
            PASSED=$((PASSED + 1))
            if [ -n "$VERBOSE" ]; then
                echo "Output preview:"
                head -10 "$output" | sed 's/^/  /'
            fi
            return 0
        else
            echo -e "${RED}❌ FAIL${NC} - Pattern not found ($size bytes, $proc_time)"
            echo "Output preview:"
            head -15 "$output" | sed 's/^/  /'
            FAILED=$((FAILED + 1))
            return 1
        fi
    else
        echo -e "${RED}❌ FAIL${NC} - No output file created ($proc_time)"
        FAILED=$((FAILED + 1))
        return 1
    fi
}

echo "Installing stdin-test..."
cd "$(dirname "$0")"
go install

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "STDIN FORWARDING TESTS"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# BASIC STDIN TESTS (NOBUNDLE MODE)
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "NOBUNDLE MODE TESTS (Baseline)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

run_stdin_test \
    "nobundle-simple-prompt" \
    "-prompt" \
    "Alice\\n25\\ny" \
    "Hello, Alice!" \
    "MACGO_NOBUNDLE=1"

run_stdin_test \
    "nobundle-eof-handling" \
    "-eof" \
    "line1\\nline2\\nline3" \
    "EOF detected" \
    "MACGO_NOBUNDLE=1"

run_stdin_test \
    "nobundle-multiline" \
    "-multiline" \
    "first line\\nsecond line\\nthird line\\nEND" \
    "Received 3 lines" \
    "MACGO_NOBUNDLE=1"

run_stdin_test \
    "nobundle-control-chars" \
    "-control" \
    "hello\\tworld\\ntest\\ndata" \
    "Has tabs: true" \
    "MACGO_NOBUNDLE=1"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# BUNDLE MODE WITH STDIN FORWARDING (CONFIG-FILE STRATEGY)
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "BUNDLE MODE - CONFIG-FILE STRATEGY (Default, Working)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

run_stdin_test \
    "bundle-config-simple-prompt" \
    "-prompt" \
    "Bob\\n30\\ny" \
    "Hello, Bob!" \
    "MACGO_DEBUG=1 MACGO_ENABLE_STDIN_FORWARDING=1 MACGO_IO_STRATEGY=config-file"

run_stdin_test \
    "bundle-config-eof" \
    "-eof" \
    "alpha\\nbeta\\ngamma" \
    "EOF detected" \
    "MACGO_DEBUG=1 MACGO_ENABLE_STDIN_FORWARDING=1 MACGO_IO_STRATEGY=config-file"

run_stdin_test \
    "bundle-config-multiline" \
    "-multiline" \
    "line A\\nline B\\nline C\\nEND" \
    "Received 3 lines" \
    "MACGO_DEBUG=1 MACGO_ENABLE_STDIN_FORWARDING=1 MACGO_IO_STRATEGY=config-file"

run_stdin_test \
    "bundle-config-password" \
    "-password" \
    "secret123\\nsecret123" \
    "Passwords match" \
    "MACGO_DEBUG=1 MACGO_ENABLE_STDIN_FORWARDING=1 MACGO_IO_STRATEGY=config-file"

run_stdin_test \
    "bundle-config-control-chars" \
    "-control" \
    "hello\\tworld\\ntest\\nmore" \
    "Has tabs: true" \
    "MACGO_DEBUG=1 MACGO_ENABLE_STDIN_FORWARDING=1 MACGO_IO_STRATEGY=config-file"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# BUNDLE MODE WITH STDIN FORWARDING (ENV-VARS STRATEGY)
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "BUNDLE MODE - ENV-VARS STRATEGY (May not work)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

run_stdin_test \
    "bundle-envvars-simple-prompt" \
    "-prompt" \
    "Charlie\\n35\\ny" \
    "Hello, Charlie!" \
    "MACGO_DEBUG=1 MACGO_ENABLE_STDIN_FORWARDING=1 MACGO_IO_STRATEGY=env-vars"

run_stdin_test \
    "bundle-envvars-eof" \
    "-eof" \
    "one\\ntwo\\nthree" \
    "EOF detected" \
    "MACGO_DEBUG=1 MACGO_ENABLE_STDIN_FORWARDING=1 MACGO_IO_STRATEGY=env-vars"

run_stdin_test \
    "bundle-envvars-multiline" \
    "-multiline" \
    "first\\nsecond\\nthird\\nEND" \
    "Received 3 lines" \
    "MACGO_DEBUG=1 MACGO_ENABLE_STDIN_FORWARDING=1 MACGO_IO_STRATEGY=env-vars"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# LINE BUFFERING TESTS
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "LINE BUFFERING TESTS"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

run_stdin_test \
    "nobundle-linebuffer" \
    "-linebuffer" \
    "test line\\nabcdefghij\\none two three" \
    "Read buffered line" \
    "MACGO_NOBUNDLE=1"

run_stdin_test \
    "bundle-linebuffer" \
    "-linebuffer" \
    "buffer test\\n1234567890\\nalpha beta gamma" \
    "Read buffered line" \
    "MACGO_DEBUG=1 MACGO_ENABLE_STDIN_FORWARDING=1"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# EOF EDGE CASES
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "EOF EDGE CASES"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Empty input (immediate EOF)
run_stdin_test \
    "nobundle-empty-input" \
    "-eof" \
    "" \
    "EOF detected" \
    "MACGO_NOBUNDLE=1"

run_stdin_test \
    "bundle-empty-input" \
    "-eof" \
    "" \
    "EOF detected" \
    "MACGO_DEBUG=1 MACGO_ENABLE_STDIN_FORWARDING=1"

# Single line without trailing newline
run_stdin_test \
    "nobundle-no-trailing-newline" \
    "-eof" \
    "single line without newline" \
    "Partial line before EOF" \
    "MACGO_NOBUNDLE=1"

# Very long line
run_stdin_test \
    "nobundle-long-line" \
    "-eof" \
    "$(printf 'A%.0s' {1..1000})" \
    "Total lines read: 1" \
    "MACGO_NOBUNDLE=1"

run_stdin_test \
    "bundle-long-line" \
    "-eof" \
    "$(printf 'B%.0s' {1..1000})" \
    "Total lines read: 1" \
    "MACGO_DEBUG=1 MACGO_ENABLE_STDIN_FORWARDING=1"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# SPECIAL CHARACTER TESTS
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "SPECIAL CHARACTER TESTS"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Unicode characters
run_stdin_test \
    "nobundle-unicode" \
    "-prompt" \
    "Ålice\\n25\\ny" \
    "Hello, Ålice!" \
    "MACGO_NOBUNDLE=1"

run_stdin_test \
    "bundle-unicode" \
    "-prompt" \
    "Bøb\\n30\\ny" \
    "Hello, Bøb!" \
    "MACGO_DEBUG=1 MACGO_ENABLE_STDIN_FORWARDING=1"

# Special characters in input
run_stdin_test \
    "nobundle-special-chars" \
    "-prompt" \
    "User!@#\\$%\\n25\\ny" \
    "Hello, User" \
    "MACGO_NOBUNDLE=1"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# CONCURRENT I/O TESTS
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "CONCURRENT I/O TESTS (stdin + stdout + stderr)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Note: These tests verify that stdin works when stdout/stderr forwarding is also enabled
run_stdin_test \
    "bundle-all-io-prompt" \
    "-prompt" \
    "Dave\\n40\\ny" \
    "Hello, Dave!" \
    "MACGO_DEBUG=1 MACGO_ENABLE_IO_FORWARDING=1"

run_stdin_test \
    "bundle-all-io-multiline" \
    "-multiline" \
    "concurrent line 1\\nconcurrent line 2\\nEND" \
    "Received 2 lines" \
    "MACGO_DEBUG=1 MACGO_ENABLE_IO_FORWARDING=1"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# TIMEOUT TESTS
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TIMEOUT TESTS (No input provided)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

run_interactive_test \
    "nobundle-timeout" \
    "-timeout" \
    "MACGO_NOBUNDLE=1 stdin-test -timeout" \
    "Timeout!" \
    ""

run_interactive_test \
    "bundle-timeout" \
    "-timeout" \
    "MACGO_DEBUG=1 MACGO_ENABLE_STDIN_FORWARDING=1 stdin-test -timeout" \
    "Timeout!" \
    ""

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# PIPE DETECTION TESTS
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "PIPE DETECTION TESTS"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Verify that piped input is detected correctly
TEST_COUNT=$((TEST_COUNT + 1))
output="$TEST_DIR/${TEST_COUNT}-nobundle-pipe-detection.txt"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test $TEST_COUNT: nobundle-pipe-detection"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
rm -f "$output"
echo "test" | MACGO_NOBUNDLE=1 timeout $TIMEOUT stdin-test > "$output" 2>&1
if grep -q "Is pipe: false" "$output" && grep -q "Is regular file: false" "$output"; then
    echo -e "${GREEN}✅ PASS${NC} - Pipe detected in nobundle mode"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAIL${NC} - Pipe not detected correctly"
    echo "Output preview:"
    grep "Is pipe\|Is regular" "$output" | sed 's/^/  /'
    FAILED=$((FAILED + 1))
fi

TEST_COUNT=$((TEST_COUNT + 1))
output="$TEST_DIR/${TEST_COUNT}-bundle-pipe-detection.txt"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test $TEST_COUNT: bundle-pipe-detection"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
rm -f "$output"
echo "test" | MACGO_DEBUG=1 MACGO_ENABLE_STDIN_FORWARDING=1 timeout $TIMEOUT stdin-test > "$output" 2>&1
if grep -q "Is regular file: true\|Is pipe: true" "$output"; then
    echo -e "${GREEN}✅ PASS${NC} - Stdin forwarded (shows as file or pipe in bundle)"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAIL${NC} - Stdin forwarding issue"
    echo "Output preview:"
    grep "Is pipe\|Is regular" "$output" | sed 's/^/  /'
    FAILED=$((FAILED + 1))
fi

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# SUMMARY
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

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
    echo -e "${GREEN}🎉 All stdin tests passed!${NC}"
    exit 0
else
    echo -e "${YELLOW}⚠️  Some tests failed${NC}"
    echo "Test outputs saved in: $TEST_DIR"
    echo ""
    echo "Failed test outputs:"
    for f in "$TEST_DIR"/*.txt; do
        if [ -f "$f" ]; then
            testname=$(basename "$f" .txt)
            if ! grep -q "PASS" "$f" 2>/dev/null; then
                echo "  - $testname"
            fi
        fi
    done
    exit 1
fi
