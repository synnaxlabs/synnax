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

# Collect search inputs: files, directories, and patterns
declare -a EXPLICIT_FILES
declare -a SEARCH_DIRS
declare -a PATTERNS

if [ $# -gt 0 ]; then
    for arg in "$@"; do
        if [ -f "$arg" ]; then
            # Explicit file path
            if [[ "$arg" = /* ]]; then
                EXPLICIT_FILES+=("$arg")
            else
                EXPLICIT_FILES+=("$GIT_ROOT/$arg")
            fi
        elif [ -d "$arg" ]; then
            # Directory to search
            if [[ "$arg" = /* ]]; then
                SEARCH_DIRS+=("$arg")
            else
                SEARCH_DIRS+=("$GIT_ROOT/$arg")
            fi
        elif [[ "$arg" == *"*"* ]] || [[ "$arg" == *"?"* ]] || [[ "$arg" == *"["* ]]; then
            # Glob pattern
            PATTERNS+=("$arg")
        else
            echo "Warning: '$arg' is not a file, directory, or pattern, skipping"
        fi
    done
else
    # Default: search entire repo
    SEARCH_DIRS+=("$GIT_ROOT")
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

# Generate headers for different comment styles
HEADER_SLASHES=$(generate_header_with_comment_style "//" "false")
HEADER_HASH=$(generate_header_with_comment_style "#" "false")
HEADER_C_STYLE=$(generate_header_with_comment_style "*" "true")

# Counters
TOTAL_FILES=0
UPDATED_FILES=0
SKIPPED_FILES=0

# Arrays to store updated files
declare -a FILES_UPDATED

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
[ ${#EXPLICIT_FILES[@]} -gt 0 ] && echo "Files: ${#EXPLICIT_FILES[@]}"
[ ${#SEARCH_DIRS[@]} -gt 0 ] && echo "Directories: ${SEARCH_DIRS[*]}"
[ ${#PATTERNS[@]} -gt 0 ] && echo "Patterns: ${PATTERNS[*]}"
echo "Current year: $CURRENT_YEAR"
echo ""

# Helper to check if file has a supported extension
has_supported_extension() {
    local file="$1"
    local ext="${file##*.}"
    case "$ext" in
        go | py | ts | tsx | js | jsx | cpp | hpp | h | cc | cxx | css | oracle) return 0 ;;
        *) return 1 ;;
    esac
}

# Helper to add file if it passes all checks
try_add_file() {
    local abs_file="$1"
    if [ -f "$abs_file" ] && has_supported_extension "$abs_file" && ! should_ignore_file "$abs_file"; then
        FILES_TO_UPDATE+=("$abs_file")
    fi
}

# First pass: count total files to process
echo -n "Counting files..."
cd "$GIT_ROOT" || exit 1
declare -a FILES_TO_UPDATE

# 1. Add explicit files
for abs_file in ${EXPLICIT_FILES[@]+"${EXPLICIT_FILES[@]}"}; do
    try_add_file "$abs_file"
done

# 2. Search directories using git ls-files
for search_dir in ${SEARCH_DIRS[@]+"${SEARCH_DIRS[@]}"}; do
    while IFS= read -r file; do
        abs_file="$GIT_ROOT/$file"
        # Check if file is within the search directory
        if [[ "$abs_file" == "$search_dir"* ]]; then
            try_add_file "$abs_file"
        fi
    done < <(git ls-files)
done

# 3. Find files matching patterns
for pattern in ${PATTERNS[@]+"${PATTERNS[@]}"}; do
    while IFS= read -r abs_file; do
        try_add_file "$abs_file"
    done < <(find "$GIT_ROOT" -type f -name "$pattern" 2> /dev/null)
done

TOTAL_TO_UPDATE=${#FILES_TO_UPDATE[@]}
echo -e "\r\033[KFound $TOTAL_TO_UPDATE files to process"
echo ""

# Second pass: update each file with progress
CURRENT_FILE_NUM=0
for file in "${FILES_TO_UPDATE[@]}"; do
    CURRENT_FILE_NUM=$((CURRENT_FILE_NUM + 1))
    relative_file="${file#$GIT_ROOT/}"

    # Show progress (overwrite same line)
    printf "\r\033[KProcessing file %d/%d: %s" "$CURRENT_FILE_NUM" "$TOTAL_TO_UPDATE" "$relative_file"

    update_file "$file"
done

# Clear progress line and print results
echo -e "\r\033[K"
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
