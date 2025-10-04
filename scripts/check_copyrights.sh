#!/bin/bash
# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

set -euo pipefail

# Find git repository root
GIT_ROOT=$(git rev-parse --show-toplevel 2> /dev/null || echo ".")

# Get search path: git_root + optional subdirectory argument
SUBDIR="${1:-}"
# Remove leading ./ if present
SUBDIR="${SUBDIR#./}"
if [ -z "$SUBDIR" ]; then
    SEARCH_PATH="$GIT_ROOT"
else
    SEARCH_PATH="$GIT_ROOT/$SUBDIR"
fi

# Get the current year
CURRENT_YEAR=$(date +%Y)

# Define expected copyright headers for different comment styles
read -r -d '' EXPECTED_HEADER_SLASHES << 'EOF' || true
// Copyright YEAR Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

EOF

read -r -d '' EXPECTED_HEADER_HASH << 'EOF' || true
#  Copyright YEAR Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

EOF

read -r -d '' EXPECTED_HEADER_C_STYLE << 'EOF' || true
/*
 * Copyright YEAR Synnax Labs, Inc.
 *
 * Use of this software is governed by the Business Source License included in the file
 * licenses/BSL.txt.
 *
 * As of the Change Date specified in that file, in accordance with the Business Source
 * License, use of this software will be governed by the Apache License, Version 2.0,
 * included in the file licenses/APL.txt.
 */

EOF

# Replace YEAR placeholder with current year
EXPECTED_HEADER_SLASHES="${EXPECTED_HEADER_SLASHES//YEAR/$CURRENT_YEAR}"
EXPECTED_HEADER_HASH="${EXPECTED_HEADER_HASH//YEAR/$CURRENT_YEAR}"
EXPECTED_HEADER_C_STYLE="${EXPECTED_HEADER_C_STYLE//YEAR/$CURRENT_YEAR}"

# Counters
TOTAL_FILES=0
MISSING_HEADER=0
WRONG_YEAR=0
MALFORMED_HEADER=0

# Arrays to store problematic files
declare -a FILES_MISSING_HEADER
declare -a FILES_WRONG_YEAR
declare -a FILES_MALFORMED_HEADER

# Function to check a single file
check_file() {
    local file="$1"
    local ext="${file##*.}"

    TOTAL_FILES=$((TOTAL_FILES + 1))

    # Determine comment style and header length based on extension
    local expected_header
    local header_lines
    if [ "$ext" = "py" ]; then
        expected_header="$EXPECTED_HEADER_HASH"
        header_lines=9
    elif [ "$ext" = "css" ]; then
        expected_header="$EXPECTED_HEADER_C_STYLE"
        header_lines=11
    else
        expected_header="$EXPECTED_HEADER_SLASHES"
        header_lines=9
    fi

    # Read the appropriate number of lines from the file
    local file_header
    file_header=$(head -n "$header_lines" "$file" 2> /dev/null || true)

    # Check if file is empty or too short
    if [ -z "$file_header" ]; then
        FILES_MISSING_HEADER+=("$file")
        MISSING_HEADER=$((MISSING_HEADER + 1))
        return
    fi

    # Get first line to check for copyright
    local first_line
    first_line=$(head -n 1 "$file")

    # Check if copyright line exists (different format for CSS)
    if [ "$ext" = "css" ]; then
        local second_line
        second_line=$(head -n 2 "$file" | tail -n 1)
        if [[ ! "$second_line" =~ Copyright.*Synnax\ Labs ]]; then
            FILES_MISSING_HEADER+=("$file")
            MISSING_HEADER=$((MISSING_HEADER + 1))
            return
        fi

        # Check if year is correct
        if [[ ! "$second_line" =~ Copyright\ $CURRENT_YEAR\ Synnax\ Labs ]]; then
            FILES_WRONG_YEAR+=("$file")
            WRONG_YEAR=$((WRONG_YEAR + 1))
            return
        fi
    else
        if [[ ! "$first_line" =~ Copyright.*Synnax\ Labs ]]; then
            FILES_MISSING_HEADER+=("$file")
            MISSING_HEADER=$((MISSING_HEADER + 1))
            return
        fi

        # Check if year is correct
        if [[ ! "$first_line" =~ Copyright\ $CURRENT_YEAR\ Synnax\ Labs ]]; then
            FILES_WRONG_YEAR+=("$file")
            WRONG_YEAR=$((WRONG_YEAR + 1))
            return
        fi
    fi

    # Check if the full header matches exactly
    if [ "$file_header" != "$expected_header" ]; then
        FILES_MALFORMED_HEADER+=("$file")
        MALFORMED_HEADER=$((MALFORMED_HEADER + 1))
        return
    fi
}

# Find and check all files
echo "Checking copyright headers in source files..."
echo "Git root: $GIT_ROOT"
echo "Search path: $SEARCH_PATH"
echo "Current year: $CURRENT_YEAR"
echo ""

# Build find command with excludes
while IFS= read -r -d '' file; do
    check_file "$file"
done < <(find "$SEARCH_PATH" \
    -path "*/.git" -prune -o \
    -path "*/node_modules" -prune -o \
    -path "*/vendor" -prune -o \
    -path "*/dist" -prune -o \
    -path "*/build" -prune -o \
    -path "*/out" -prune -o \
    -path "*/target" -prune -o \
    -path "*/.venv" -prune -o \
    -path "*/venv" -prune -o \
    -path "*/__pycache__" -prune -o \
    -path "*/.turbo" -prune -o \
    -path "*/gen" -prune -o \
    -path "*/generated" -prune -o \
    -path "*/.tauri" -prune -o \
    -path "*/binaries" -prune -o \
    -type f \( -name '*.go' -o -name '*.py' -o -name '*.ts' -o -name '*.tsx' -o -name '*.js' -o -name '*.jsx' -o -name '*.cpp' -o -name '*.hpp' -o -name '*.h' -o -name '*.cc' -o -name '*.cxx' -o -name '*.css' \) -print0 2> /dev/null)

# Print results
echo "Checked $TOTAL_FILES files"
echo ""

EXIT_CODE=0

if [ $MISSING_HEADER -gt 0 ]; then
    echo "❌ $MISSING_HEADER file(s) missing copyright header:"
    for file in "${FILES_MISSING_HEADER[@]}"; do
        echo "  - $file"
    done
    echo ""
    EXIT_CODE=1
fi

if [ $WRONG_YEAR -gt 0 ]; then
    echo "❌ $WRONG_YEAR file(s) with incorrect copyright year (expected $CURRENT_YEAR):"
    for file in "${FILES_WRONG_YEAR[@]}"; do
        echo "  - $file"
    done
    echo ""
    EXIT_CODE=1
fi

if [ $MALFORMED_HEADER -gt 0 ]; then
    echo "❌ $MALFORMED_HEADER file(s) with malformed copyright header:"
    for file in "${FILES_MALFORMED_HEADER[@]}"; do
        echo "  - $file"
    done
    echo ""
    EXIT_CODE=1
fi

if [ $EXIT_CODE -eq 0 ]; then
    echo "✅ All files have correct copyright headers with year $CURRENT_YEAR"
fi

exit $EXIT_CODE
