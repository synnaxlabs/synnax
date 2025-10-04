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

# Define copyright headers for different comment styles
read -r -d '' HEADER_SLASHES << 'EOF' || true
// Copyright YEAR Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

EOF

read -r -d '' HEADER_HASH << 'EOF' || true
#  Copyright YEAR Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

EOF

read -r -d '' HEADER_C_STYLE << 'EOF' || true
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
HEADER_SLASHES="${HEADER_SLASHES//YEAR/$CURRENT_YEAR}"
HEADER_HASH="${HEADER_HASH//YEAR/$CURRENT_YEAR}"
HEADER_C_STYLE="${HEADER_C_STYLE//YEAR/$CURRENT_YEAR}"

# Counters
TOTAL_FILES=0
UPDATED_FILES=0
SKIPPED_FILES=0

# Arrays to store updated files
declare -a FILES_UPDATED

# Function to update copyright in a file
update_file() {
    local file="$1"
    local ext="${file##*.}"

    TOTAL_FILES=$((TOTAL_FILES + 1))

    # Determine comment style and header based on extension
    local new_header
    local header_lines_without_blank
    local header_lines_with_blank
    if [ "$ext" = "py" ]; then
        new_header="$HEADER_HASH"
        header_lines_without_blank=8
        header_lines_with_blank=9
    elif [ "$ext" = "css" ]; then
        new_header="$HEADER_C_STYLE"
        header_lines_without_blank=10
        header_lines_with_blank=11
    else
        new_header="$HEADER_SLASHES"
        header_lines_without_blank=8
        header_lines_with_blank=9
    fi

    # Read the first lines to check for existing copyright
    local first_line
    first_line=$(head -n 1 "$file" 2> /dev/null || true)

    # Check if file already has correct copyright
    if [ "$ext" = "css" ]; then
        local second_line
        second_line=$(head -n 2 "$file" 2> /dev/null | tail -n 1 || true)

        if [[ "$second_line" =~ Copyright.*$CURRENT_YEAR.*Synnax\ Labs ]]; then
            # Check if full header is correct
            local current_header
            current_header=$(head -n "$header_lines_with_blank" "$file")
            if [ "$current_header" = "$new_header" ]; then
                SKIPPED_FILES=$((SKIPPED_FILES + 1))
                return
            fi
        fi
    else
        if [[ "$first_line" =~ Copyright.*$CURRENT_YEAR.*Synnax\ Labs ]]; then
            # Check if full header is correct
            local current_header
            current_header=$(head -n "$header_lines_with_blank" "$file")
            if [ "$current_header" = "$new_header" ]; then
                SKIPPED_FILES=$((SKIPPED_FILES + 1))
                return
            fi
        fi
    fi

    # Create a temporary file
    local temp_file
    temp_file=$(mktemp)

    # Check if file has any copyright header (even old year)
    local has_copyright=0
    if [ "$ext" = "css" ]; then
        if [[ "$second_line" =~ Copyright.*Synnax\ Labs ]]; then
            has_copyright=1
        fi
    else
        if [[ "$first_line" =~ Copyright.*Synnax\ Labs ]]; then
            has_copyright=1
        fi
    fi

    if [ $has_copyright -eq 1 ]; then
        # Replace existing header
        # Check if the old header has a blank line after it (line 9 for most, line 11 for CSS)
        local line_after_header_no_blank
        line_after_header_no_blank=$(sed -n "$((header_lines_without_blank + 1))p" "$file" 2> /dev/null || true)

        # Write new header to temp file
        printf "%s\n\n" "$new_header" > "$temp_file"

        # Append rest of file after old header
        # If line 9 is blank, old header had blank line (skip 9 lines)
        # If line 9 is not blank, old header didn't have blank line (skip 8 lines)
        if [ -z "$line_after_header_no_blank" ]; then
            # Line 9 is blank, so old header had the blank line - skip it
            tail -n +$((header_lines_with_blank + 1)) "$file" >> "$temp_file"
        else
            # Line 9 is not blank, old header didn't have blank line - skip 8 lines
            tail -n +$((header_lines_without_blank + 1)) "$file" >> "$temp_file"
        fi
    else
        # Prepend new header
        printf "%s\n\n" "$new_header" > "$temp_file"
        cat "$file" >> "$temp_file"
    fi

    # Replace original file with updated content
    mv "$temp_file" "$file"

    FILES_UPDATED+=("$file")
    UPDATED_FILES=$((UPDATED_FILES + 1))
}

# Find and update all files
echo "Updating copyright headers in source files..."
echo "Git root: $GIT_ROOT"
echo "Search path: $SEARCH_PATH"
echo "Current year: $CURRENT_YEAR"
echo ""

# Build find command with excludes
while IFS= read -r -d '' file; do
    update_file "$file"
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
echo "Processed $TOTAL_FILES files"
echo ""

if [ $UPDATED_FILES -gt 0 ]; then
    echo "✅ Updated $UPDATED_FILES file(s):"
    for file in "${FILES_UPDATED[@]}"; do
        echo "  - $file"
    done
    echo ""
fi

if [ $SKIPPED_FILES -gt 0 ]; then
    echo "⏭️  Skipped $SKIPPED_FILES file(s) (already up to date)"
fi

echo ""
echo "✅ Copyright update complete"
