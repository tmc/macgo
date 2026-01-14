#!/bin/bash

# Screen-capture battery test suite
# Tests all MACGO environment variable flags and I/O redirection

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
TIMEOUT=10
TEST_DIR="/tmp/screen-capture-battery-results"
PASSED=0
FAILED=0

# Create test directory
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "SCREEN-CAPTURE BATTERY TEST SUITE"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Cleanup function
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

# Pre-test cleanup
kill_xpcproxy

# Build screen-capture to /tmp to avoid binary disappearing issues
echo -e "${BLUE}Building screen-capture...${NC}"
go build -o /tmp/screen-capture . || { echo -e "${RED}Build failed${NC}"; exit 1; }
echo -e "${GREEN}✓ Build successful${NC}"
echo ""

# Test 1: Bundle mode with launcher v1
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test: bundle-launcher-v1"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
rm -f "$TEST_DIR/bundle-v1.png"
MACGO_DEBUG=1 MACGO_LAUNCHER_VERSION=1 timeout $TIMEOUT /tmp/screen-capture "$TEST_DIR/bundle-v1.png" 2>&1 | head -20
if [ -f "$TEST_DIR/bundle-v1.png" ]; then
    echo -e "${GREEN}✅ PASS${NC} - Screenshot created with launcher v1"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAIL${NC} - No screenshot created"
    FAILED=$((FAILED + 1))
fi

# Test 2: Bundle mode with launcher v2
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test: bundle-launcher-v2"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
rm -f "$TEST_DIR/bundle-v2.png"
MACGO_DEBUG=1 MACGO_LAUNCHER_VERSION=2 timeout $TIMEOUT /tmp/screen-capture "$TEST_DIR/bundle-v2.png" 2>&1 | head -20
if [ -f "$TEST_DIR/bundle-v2.png" ]; then
    echo -e "${GREEN}✅ PASS${NC} - Screenshot created with launcher v2"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAIL${NC} - No screenshot created"
    FAILED=$((FAILED + 1))
fi

# Test 3: I/O forwarding with env-vars strategy (stderr) - launcher v2
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test: io-stderr-forwarding-v2"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
rm -f "$TEST_DIR/io-stderr.png" "$TEST_DIR/io-stderr.txt"
MACGO_DEBUG=1 MACGO_LAUNCHER_VERSION=2 MACGO_IO_STRATEGY=env-vars MACGO_ENABLE_STDERR_FORWARDING=1 \
    timeout $TIMEOUT /tmp/screen-capture "$TEST_DIR/io-stderr.png" > "$TEST_DIR/io-stderr.txt" 2>&1
if [ -f "$TEST_DIR/io-stderr.png" ] && [ -f "$TEST_DIR/io-stderr.txt" ]; then
    echo -e "${GREEN}✅ PASS${NC} - I/O stderr forwarding works (v2)"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAIL${NC} - I/O stderr forwarding failed"
    FAILED=$((FAILED + 1))
fi

# Test 4: All I/O forwarding - launcher v2
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test: io-all-forwarding-v2"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
rm -f "$TEST_DIR/io-all.png" "$TEST_DIR/io-all.txt"
MACGO_DEBUG=1 MACGO_LAUNCHER_VERSION=2 MACGO_IO_STRATEGY=env-vars MACGO_ENABLE_IO_FORWARDING=1 \
    timeout $TIMEOUT /tmp/screen-capture "$TEST_DIR/io-all.png" > "$TEST_DIR/io-all.txt" 2>&1
if [ -f "$TEST_DIR/io-all.png" ]; then
    echo -e "${GREEN}✅ PASS${NC} - All I/O forwarding works (v2)"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}❌ FAIL${NC} - All I/O forwarding failed"
    FAILED=$((FAILED + 1))
fi

# Final cleanup
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
    exit 1
fi
