#!/bin/bash
# Test script for io-tree-test
# Verifies that a tree of macgo programs correctly forwards I/O

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

echo "=== Building io-tree-test ==="
go build -o io-tree-test .

echo ""
echo "=== Test 1: Simple tree (depth 2, 1 child per level) ==="
echo "Expected: 3 processes (root + 2 levels of children)"
echo ""

# Run with MACGO_DEBUG to see what's happening
MACGO_DEBUG=1 ./io-tree-test -max-depth=2 -children=1 2>&1 | head -100

echo ""
echo "=== Test 2: Without macgo bundle (baseline) ==="
MACGO_NOBUNDLE=1 ./io-tree-test -max-depth=2 -children=1 2>&1 | head -50

echo ""
echo "=== Test 3: Wider tree (depth 2, 2 children per level) ==="
MACGO_DEBUG=1 ./io-tree-test -max-depth=2 -children=2 -timeout=60s 2>&1 | head -150

echo ""
echo "=== Tests Complete ==="
