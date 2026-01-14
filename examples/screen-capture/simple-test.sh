#!/bin/bash
# Simple test for screen-capture with timeout and file verification

set -e

OUTPUT_FILE="/tmp/screen-capture-simple-test.png"
TIMEOUT=15

echo "Cleaning up previous test..."
rm -f "$OUTPUT_FILE"

echo "Running screen-capture with ${TIMEOUT}s timeout..."
echo "Command: timeout $TIMEOUT screen-capture -app 'System Settings' $OUTPUT_FILE"

if timeout $TIMEOUT screen-capture -app "System Settings" "$OUTPUT_FILE" 2>&1; then
    echo "Command completed successfully"
else
    EXIT_CODE=$?
    if [ $EXIT_CODE -eq 124 ]; then
        echo "ERROR: Command timed out after ${TIMEOUT}s"
    else
        echo "ERROR: Command failed with exit code $EXIT_CODE"
    fi
fi

echo ""
echo "Checking if file was created..."
if [ -f "$OUTPUT_FILE" ]; then
    SIZE=$(stat -f%z "$OUTPUT_FILE" 2>/dev/null || echo 0)
    if [ "$SIZE" -gt 0 ]; then
        echo "✓ SUCCESS: File created at $OUTPUT_FILE ($SIZE bytes)"
        file "$OUTPUT_FILE"
        exit 0
    else
        echo "✗ FAIL: File exists but is empty"
        exit 1
    fi
else
    echo "✗ FAIL: File was not created"
    exit 1
fi
