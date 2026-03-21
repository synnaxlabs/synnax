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

# Read the base header template from licenses/headers/template.txt
HEADER_TEMPLATE_FILE="$GIT_ROOT/licenses/headers/template.txt"

if [ ! -f "$HEADER_TEMPLATE_FILE" ]; then
    echo "Error: Header template file not found at $HEADER_TEMPLATE_FILE"
    exit 1
fi

BASE_HEADER=$(cat "$HEADER_TEMPLATE_FILE")

# Replace {{YEAR}} with current year
BASE_HEADER="${BASE_HEADER//\{\{YEAR\}\}/$CURRENT_YEAR}"

# Function to generate header with specific comment style
generate_header_with_comment_style() {
    local comment_prefix="$1"
    local use_block_comment="$2"

    if [ "$use_block_comment" = "true" ]; then
        # C-style block comment
        echo "/*"
        while IFS= read -r line; do
            if [ -z "$line" ]; then
                echo " *"
            else
                echo " * $line"
            fi
        done <<< "$BASE_HEADER"
        echo " */"
        echo ""
    else
        # Line-based comments (// or #)
        # Note: # uses 2 spaces, // uses 1 space
        local space_count=1
        [[ "$comment_prefix" == "#" ]] && space_count=2

        while IFS= read -r line; do
            if [ -z "$line" ]; then
                echo "$comment_prefix"
            else
                printf "%s%*s%s\n" "$comment_prefix" "$space_count" "" "$line"
            fi
        done <<< "$BASE_HEADER"
        echo ""
    fi
}

# Generate expected headers for different comment styles
EXPECTED_HEADER_SLASHES=$(generate_header_with_comment_style "//" "false")
EXPECTED_HEADER_HASH=$(generate_header_with_comment_style "#" "false")
EXPECTED_HEADER_C_STYLE=$(generate_header_with_comment_style "*" "true")

# Counters
TOTAL_FILES=0
MISSING_HEADER=0
WRONG_YEAR=0
MALFORMED_HEADER=0
DUPLICATE_HEADER=0

# Arrays to store problematic files
declare -a FILES_MISSING_HEADER
declare -a FILES_WRONG_YEAR
declare -a FILES_MALFORMED_HEADER
declare -a FILES_DUPLICATE_HEADER

# Read .copyrightignore patterns
declare -a IGNORE_PATTERNS
if [ -f "$GIT_ROOT/.copyrightignore" ]; then
    while IFS= read -r pattern; do
        # Skip empty lines and comments
        [[ -z "$pattern" || "$pattern" =~ ^[[:space:]]*# ]] && continue
        # Remove leading/trailing whitespace
        pattern=$(echo "$pattern" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        [[ -n "$pattern" ]] && IGNORE_PATTERNS+=("$pattern")
    done < "$GIT_ROOT/.copyrightignore"
fi

# Function to check if a file matches any ignore pattern
should_ignore_file() {
    local file="$1"
    local relative_path="${file#$GIT_ROOT/}"

    for pattern in "${IGNORE_PATTERNS[@]}"; do
        # Convert glob pattern to regex-like matching
        case "$relative_path" in
            $pattern) return 0 ;;
        esac
    done
    return 1
}

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

    # Check for duplicate copyright headers anywhere in the file
    local copyright_count
    copyright_count=$(grep -c "Copyright.*Synnax Labs" "$file" 2> /dev/null || true)
    if [ "$copyright_count" -gt 1 ]; then
        FILES_DUPLICATE_HEADER+=("$file")
        DUPLICATE_HEADER=$((DUPLICATE_HEADER + 1))
        return
    fi
}

# Find and check all files
echo "Checking copyright headers in source files..."
echo "Git root: $GIT_ROOT"
echo "Search path: $SEARCH_PATH"
echo "Current year: $CURRENT_YEAR"
echo ""

# First pass: count total files to process
echo -n "Counting files..."
cd "$GIT_ROOT" || exit 1
declare -a FILES_TO_CHECK
while IFS= read -r file; do
    # Convert relative path to absolute
    abs_file="$GIT_ROOT/$file"

    # Check if file is in the search path
    if [[ "$abs_file" != "$SEARCH_PATH"* ]]; then
        continue
    fi

    # Check file extension
    ext="${file##*.}"
    case "$ext" in
        go | py | ts | tsx | js | jsx | cpp | hpp | h | cc | cxx | css | oracle)
            # Check if file should be ignored per .copyrightignore
            if ! should_ignore_file "$abs_file"; then
                [ -f "$abs_file" ] && FILES_TO_CHECK+=("$abs_file")
            fi
            ;;
    esac
done < <(git ls-files)

TOTAL_TO_CHECK=${#FILES_TO_CHECK[@]}
echo -e "\r\033[KFound $TOTAL_TO_CHECK files to check"
echo ""

# Second pass: check each file with progress
CURRENT_FILE_NUM=0
for file in "${FILES_TO_CHECK[@]}"; do
    CURRENT_FILE_NUM=$((CURRENT_FILE_NUM + 1))
    relative_file="${file#$GIT_ROOT/}"

    # Show progress (overwrite same line)
    printf "\r\033[KChecking file %d/%d: %s" "$CURRENT_FILE_NUM" "$TOTAL_TO_CHECK" "$relative_file"

    check_file "$file"
done

# Clear progress line and print results
echo -e "\r\033[K"
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

if [ $DUPLICATE_HEADER -gt 0 ]; then
    echo "❌ $DUPLICATE_HEADER file(s) with duplicate copyright headers:"
    for file in "${FILES_DUPLICATE_HEADER[@]}"; do
        echo "  - $file"
    done
    echo ""
    EXIT_CODE=1
fi

if [ $EXIT_CODE -eq 0 ]; then
    echo "✅ All files have correct copyright headers with year $CURRENT_YEAR"
fi

exit $EXIT_CODE
