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

# Packages to exclude from deadcode reporting. The deadcode tool has no built-in
# exclude mechanism, so we filter its output. Add entries as grep -v patterns.
exclude_patterns=(
    "x/go/lsp/protocol/" # Vendored LSP protocol spec with many intentionally unused funcs
)

# Run deadcode once for all packages
output="$(deadcode -test -tags=driver $package_paths 2>&1 || true)"

# Apply exclusions
filtered="$output"
for pattern in "${exclude_patterns[@]}"; do
    filtered="$(printf '%s\n' "$filtered" | grep -v "$pattern" || true)"
done
printf '%s\n' "$filtered"

if [[ -n "$filtered" ]]; then
    exit 1
fi
