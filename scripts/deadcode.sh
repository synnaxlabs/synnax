#!/usr/bin/env bash
set -euo pipefail

# Parse go.work to get list of packages
packages=$(grep -E '^\s+\.' go.work | sed 's/^[[:space:]]*//' | sed 's/^\.\/*//')

# Build the deadcode command with all package paths
package_paths=""
for package in $packages; do
    package_paths="$package_paths ./$package/..."
done

echo "deadcode -test -tags=driver $package_paths"

# Run deadcode once for all packages
output="$(deadcode -test -tags=driver $package_paths 2>&1 || true)"
printf '%s\n' "$output"

if [[ -n "$output" ]]; then
  exit 1
fi
