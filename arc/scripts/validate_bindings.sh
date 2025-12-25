#!/bin/bash
# Validates that Go compiler bindings and C++ runtime bindings are in sync.
# Run from repo root: ./arc/scripts/validate_bindings.sh
#
# This script uses Go as the source of truth and verifies C++ matches.
# It preprocesses the C++ file to expand macros, then searches for
# the actual function names - no need to understand macro expansions.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CPP_FILE="$REPO_ROOT/arc/cpp/runtime/wasm/bindings.cpp"

cd "$REPO_ROOT"

# Get expected imports from Go (the source of truth)
echo "Extracting Go compiler imports..."
GO_IMPORTS=$(go run ./arc/go/compiler/bindings/dump)

# Convert to array
IFS=$'\n' read -r -d '' -a GO_ARRAY <<< "$GO_IMPORTS" || true
GO_COUNT=${#GO_ARRAY[@]}

echo "Preprocessing C++ bindings..."

# Create temp dir with stub headers for preprocessing
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

# Create minimal stubs for external headers
touch "$TMP_DIR/wasmtime.hh"
mkdir -p "$TMP_DIR/google/protobuf"
touch "$TMP_DIR/google/protobuf/struct.pb.h"

# Preprocess the C++ file to expand all macros and save to temp file
# Note: clang may return non-zero even on success, so check output file size
CPP_PREPROCESSED_FILE="$TMP_DIR/preprocessed.cpp"
clang -E -P \
    -I"$REPO_ROOT" \
    -I"$TMP_DIR" \
    -Wno-everything \
    "$CPP_FILE" > "$CPP_PREPROCESSED_FILE" 2> /dev/null || true

# Check if preprocessing produced output
if [ ! -s "$CPP_PREPROCESSED_FILE" ]; then
    echo "❌ Failed to preprocess C++ file (empty output)"
    exit 1
fi

echo "Checking C++ bindings ($GO_COUNT imports)..."
echo ""

# ===== PRESENCE CHECK =====
# For each Go import, verify the function exists in preprocessed C++
MISSING=()
for import in "${GO_ARRAY[@]}"; do
    # Look for the function name in preprocessed output
    # Use grep -F for fixed string matching (faster, no regex issues)
    if ! grep -qF "$import" "$CPP_PREPROCESSED_FILE"; then
        MISSING+=("$import")
    fi
done

if [ ${#MISSING[@]} -gt 0 ]; then
    echo "❌ MISSING in C++ (required by Go compiler):"
    for m in "${MISSING[@]}"; do
        echo "   - $m"
    done
    echo ""
    exit 1
fi

# ===== ORDER CHECK =====
# Extract function registrations from create_imports() in order
# Each registration looks like: runtime->FUNCTION_NAME(...)
# We extract these in order to determine the C++ import order

# Find create_imports function - extract from its opening brace to the return statement
CREATE_IMPORTS=$(sed -n '/create_imports.*{/,/return imports/p' "$CPP_PREPROCESSED_FILE")

if [ -z "$CREATE_IMPORTS" ]; then
    echo "❌ Could not find create_imports() function in preprocessed C++ output"
    exit 1
fi

# Extract all runtime->FUNCTION( calls in order
# This gives us the order of import registrations
CPP_IMPORT_ORDER=$(echo "$CREATE_IMPORTS" \
    | grep -oE 'runtime->[a-z_0-9]+\(' \
    | sed 's/runtime->//; s/($//')

# Convert to array (note: not using uniq here, we want all occurrences)
IFS=$'\n' read -r -d '' -a CPP_ARRAY <<< "$CPP_IMPORT_ORDER" || true

# Compare order
ERRORS=0
for i in "${!GO_ARRAY[@]}"; do
    if [ "${GO_ARRAY[$i]}" != "${CPP_ARRAY[$i]:-}" ]; then
        if [ $ERRORS -eq 0 ]; then
            echo "❌ ORDER MISMATCH (WASM imports are index-based!):"
            echo ""
        fi
        echo "   Index $i: Go='${GO_ARRAY[$i]}', C++='${CPP_ARRAY[$i]:-MISSING}'"
        ERRORS=$((ERRORS + 1))
        if [ $ERRORS -ge 10 ]; then
            echo "   ... (showing first 10 mismatches)"
            break
        fi
    fi
done

if [ $ERRORS -gt 0 ]; then
    echo ""
    echo "Summary:"
    echo "  Go compiler expects: $GO_COUNT imports"
    echo "  C++ provides: ${#CPP_ARRAY[@]} imports"
    echo "  Order mismatches: $ERRORS"
    echo ""
    echo "❌ Bindings are OUT OF SYNC!"
    exit 1
fi

# Verify counts match
if [ "$GO_COUNT" != "${#CPP_ARRAY[@]}" ]; then
    echo "❌ COUNT MISMATCH:"
    echo "  Go compiler expects: $GO_COUNT imports"
    echo "  C++ provides: ${#CPP_ARRAY[@]} imports"
    exit 1
fi

echo "Summary:"
echo "  Go compiler expects: $GO_COUNT imports"
echo "  C++ runtime provides: ${#CPP_ARRAY[@]} imports (all present, correct order)"
echo ""
echo "✅ Bindings are in sync!"
