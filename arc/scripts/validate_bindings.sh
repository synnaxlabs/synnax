#!/bin/bash
# Validates that Go and C++ runtime bindings match Go compiler imports.
# Run from repo root: ./arc/scripts/validate_bindings.sh
#
# This script uses imports.go (via dump command) as the source of truth and verifies:
# 1. Go static_bindings_generated.go exports match imports.go order
# 2. C++ bindings.cpp imports match imports.go order
#
# WASM imports are index-based, so order is critical for correctness.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
GO_BINDINGS_FILE="$REPO_ROOT/arc/go/compiler/bindings/static_bindings_generated.go"
CPP_FILE="$REPO_ROOT/arc/cpp/runtime/wasm/bindings.cpp"

cd "$REPO_ROOT"

# Get expected imports from Go imports.go (the source of truth)
echo "Extracting Go compiler imports (source of truth)..."
GO_IMPORTS=$(go run ./arc/go/compiler/bindings/dump)

# Convert to array
IFS=$'\n' read -r -d '' -a GO_ARRAY <<< "$GO_IMPORTS" || true
GO_COUNT=${#GO_ARRAY[@]}
echo "  Found $GO_COUNT imports in imports.go"
echo ""

# ============================================================================
# PART 1: Validate Go static_bindings_generated.go against imports.go
# ============================================================================
echo "Checking Go static_bindings_generated.go..."

# Extract export names from Bind() method in order
# Pattern: .Export("function_name")
GO_STATIC_EXPORTS=$(grep -oE '\.Export\("[^"]+"\)' "$GO_BINDINGS_FILE" | sed 's/\.Export("//;s/")//')

# Convert to array
IFS=$'\n' read -r -d '' -a GO_STATIC_ARRAY <<< "$GO_STATIC_EXPORTS" || true

# Compare order
GO_ERRORS=0
for i in "${!GO_ARRAY[@]}"; do
    if [ "${GO_ARRAY[$i]}" != "${GO_STATIC_ARRAY[$i]:-}" ]; then
        if [ $GO_ERRORS -eq 0 ]; then
            echo "❌ GO BINDINGS ORDER MISMATCH:"
            echo "   (static_bindings_generated.go doesn't match imports.go)"
            echo ""
        fi
        echo "   Index $i: imports.go='${GO_ARRAY[$i]}', static_bindings='${GO_STATIC_ARRAY[$i]:-MISSING}'"
        GO_ERRORS=$((GO_ERRORS + 1))
        if [ $GO_ERRORS -ge 10 ]; then
            echo "   ... (showing first 10 mismatches)"
            break
        fi
    fi
done

if [ $GO_ERRORS -gt 0 ]; then
    echo ""
    echo "Go Summary:"
    echo "  imports.go expects: $GO_COUNT imports"
    echo "  static_bindings_generated.go provides: ${#GO_STATIC_ARRAY[@]} exports"
    echo "  Order mismatches: $GO_ERRORS"
    echo ""
    echo "❌ Go bindings are OUT OF SYNC!"
    echo "   Run 'go generate ./arc/go/compiler/bindings' after fixing gen/main.go"
    exit 1
fi

# Verify counts match
if [ "$GO_COUNT" != "${#GO_STATIC_ARRAY[@]}" ]; then
    echo "❌ GO COUNT MISMATCH:"
    echo "  imports.go expects: $GO_COUNT imports"
    echo "  static_bindings_generated.go provides: ${#GO_STATIC_ARRAY[@]} exports"
    exit 1
fi

echo "  ✓ Go bindings OK ($GO_COUNT exports, correct order)"
echo ""

# ============================================================================
# PART 2: Validate C++ bindings.cpp against imports.go
# ============================================================================
echo "Checking C++ bindings.cpp..."
echo "  Preprocessing to expand macros..."

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
    echo "  ❌ Failed to preprocess C++ file (empty output)"
    exit 1
fi

# Check for missing functions
MISSING=()
for import in "${GO_ARRAY[@]}"; do
    if ! grep -qF "$import" "$CPP_PREPROCESSED_FILE"; then
        MISSING+=("$import")
    fi
done

if [ ${#MISSING[@]} -gt 0 ]; then
    echo "  ❌ MISSING in C++ (required by imports.go):"
    for m in "${MISSING[@]}"; do
        echo "     - $m"
    done
    echo ""
    exit 1
fi

# Extract function registrations from create_imports() in order
# Each registration uses: wrap(runtime, &Bindings::FUNCTION_NAME)
CREATE_IMPORTS=$(sed -n '/create_imports.*{/,/return imports/p' "$CPP_PREPROCESSED_FILE")

if [ -z "$CREATE_IMPORTS" ]; then
    echo "  ❌ Could not find create_imports() function in preprocessed C++ output"
    exit 1
fi

# Extract all Bindings::FUNCTION calls in order
CPP_IMPORT_ORDER=$(echo "$CREATE_IMPORTS" \
    | grep -oE 'Bindings::[a-z_0-9]+' \
    | sed 's/Bindings:://')

# Convert to array
IFS=$'\n' read -r -d '' -a CPP_ARRAY <<< "$CPP_IMPORT_ORDER" || true

# Compare order
CPP_ERRORS=0
for i in "${!GO_ARRAY[@]}"; do
    if [ "${GO_ARRAY[$i]}" != "${CPP_ARRAY[$i]:-}" ]; then
        if [ $CPP_ERRORS -eq 0 ]; then
            echo "  ❌ C++ BINDINGS ORDER MISMATCH:"
            echo "     (bindings.cpp doesn't match imports.go)"
            echo ""
        fi
        echo "     Index $i: imports.go='${GO_ARRAY[$i]}', C++='${CPP_ARRAY[$i]:-MISSING}'"
        CPP_ERRORS=$((CPP_ERRORS + 1))
        if [ $CPP_ERRORS -ge 10 ]; then
            echo "     ... (showing first 10 mismatches)"
            break
        fi
    fi
done

if [ $CPP_ERRORS -gt 0 ]; then
    echo ""
    echo "C++ Summary:"
    echo "  imports.go expects: $GO_COUNT imports"
    echo "  bindings.cpp provides: ${#CPP_ARRAY[@]} imports"
    echo "  Order mismatches: $CPP_ERRORS"
    echo ""
    echo "❌ C++ bindings are OUT OF SYNC!"
    exit 1
fi

# Verify counts match
if [ "$GO_COUNT" != "${#CPP_ARRAY[@]}" ]; then
    echo "  ❌ C++ COUNT MISMATCH:"
    echo "     imports.go expects: $GO_COUNT imports"
    echo "     bindings.cpp provides: ${#CPP_ARRAY[@]} imports"
    exit 1
fi

echo "  ✓ C++ bindings OK (${#CPP_ARRAY[@]} imports, correct order)"
echo ""

# ============================================================================
# SUMMARY
# ============================================================================
echo "============================================"
echo "✅ All bindings are in sync!"
echo ""
echo "  Source of truth: imports.go ($GO_COUNT imports)"
echo "  Go runtime:      static_bindings_generated.go ✓"
echo "  C++ runtime:     bindings.cpp ✓"
echo "============================================"
