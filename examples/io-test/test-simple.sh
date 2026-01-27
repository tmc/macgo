#!/bin/bash
# Simple test script to verify io-test behavior

echo "=== Testing io-test without macgo (should work) ==="
MACGO_NOBUNDLE=1 ./io-test 2>&1 | head -5
echo "Exit code: $?"
echo

echo "=== Testing io-test with macgo (should work now) ==="
./io-test 2>&1 | head -5
echo "Exit code: $?"
echo

echo "=== Checking for hung xpcproxy processes ==="
ps aux | grep xpcproxy | grep io-test | grep -v grep || echo "No hung processes found"