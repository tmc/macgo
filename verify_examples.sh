#!/bin/bash
set -u

echo "=== Verifying Examples ==="
FAILURES=0
PASSED=0
SKIPPED=0

for d in examples/*; do
    if [ -d "$d" ]; then
        if [ ! -f "$d/main.go" ]; then
             # Check if it has any go files
             if ls "$d"/*.go 1> /dev/null 2>&1; then
                 : # has go files
             else
                 echo "⚠️  Skipping $d (no go files)"
                 SKIPPED=$((SKIPPED+1))
                 continue
             fi
        fi

        if [[ "$d" == *"app-to-web"* ]]; then
            echo "⚠️  Skipping $d (requested)"
            SKIPPED=$((SKIPPED+1))
            continue
        fi

        echo -n "Building $d... "
        if (cd "$d" && go mod tidy >/dev/null 2>&1 && go build -v ./... >/dev/null 2>&1); then
            echo "✅"
            PASSED=$((PASSED+1))
        else
            echo "❌ FAILED"
            # Compile again to show error
            (cd "$d" && go build -v ./...)
            FAILURES=$((FAILURES+1))
        fi
    fi
done

echo "=========================="
echo "Summary: $PASSED passed, $FAILURES failed, $SKIPPED skipped"

if [ $FAILURES -eq 0 ]; then
    exit 0
else
    exit 1
fi
