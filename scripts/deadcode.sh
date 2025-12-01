#!/usr/bin/env bash
set -euo pipefail

# Parse go.work to get list of packages
packages=$(grep -E '^\s+\.' go.work | sed 's/^[[:space:]]*//' | sed 's/^\.\/*//')

echo "Found packages:"
echo "$packages"
echo ""

# Build the deadcode command with all package paths
package_paths=""
for package in $packages; do
  package_paths="$package_paths ./$package/..."
done

echo "Running deadcode with: deadcode -test -tags=driver $package_paths"
echo "=================================="
echo ""

# Run deadcode once for all packages
output=$(deadcode -test -tags=driver $package_paths 2>&1 || true)

if [ -n "$output" ]; then
  echo "❌ Dead code detected!"
  echo ""
  echo "$output"
  exit 1
else
  echo "✅ No dead code found in any package"
  exit 0
fi
