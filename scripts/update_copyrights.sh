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

# Generate a line-comment header. Empty body lines emit just the prefix; non-empty
# lines emit prefix + space_count spaces + line.
generate_line_header() {
    local comment_prefix="$1"
    local space_count="$2"
    while IFS= read -r line; do
        if [ -z "$line" ]; then
            echo "$comment_prefix"
        else
            printf "%s%*s%s\n" "$comment_prefix" "$space_count" "" "$line"
        fi
    done <<< "$BASE_HEADER"
}

# Generate a block-comment header bracketed by open/close markers. Empty body
# lines emit middle_empty; non-empty lines emit middle_full + line.
generate_block_header() {
    local open="$1"
    local middle_empty="$2"
    local middle_full="$3"
    local close="$4"
    echo "$open"
    while IFS= read -r line; do
        if [ -z "$line" ]; then
            printf "%s\n" "$middle_empty"
        else
            printf "%s%s\n" "$middle_full" "$line"
        fi
    done <<< "$BASE_HEADER"
    echo "$close"
}

# Generate headers for different comment styles
HEADER_SLASHES=$(generate_line_header "//" 1)
HEADER_HASH_TWO=$(generate_line_header "#" 2)
HEADER_HASH_ONE=$(generate_line_header "#" 1)
HEADER_C_STYLE=$(generate_block_header "/*" " *" " * " " */")
HEADER_HTML=$(generate_block_header "<!--" "" "  " "  -->")

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

# Resolve per-extension header properties into globals: HEADER, HEADER_LINES,
# COPYRIGHT_OFFSET (1-based line within the header containing "Copyright YYYY"),
# and SUPPORTS_SHEBANG (1 if the extension preserves a leading #! line).
resolve_header_for_ext() {
    local ext="$1"
    case "$ext" in
        py)
            HEADER="$HEADER_HASH_TWO"
            HEADER_LINES=8
            COPYRIGHT_OFFSET=1
            SUPPORTS_SHEBANG=0
            ;;
        sh | zsh)
            HEADER="$HEADER_HASH_ONE"
            HEADER_LINES=8
            COPYRIGHT_OFFSET=1
            SUPPORTS_SHEBANG=1
            ;;
        css)
            HEADER="$HEADER_C_STYLE"
            HEADER_LINES=10
            COPYRIGHT_OFFSET=2
            SUPPORTS_SHEBANG=0
            ;;
        html | svg)
            HEADER="$HEADER_HTML"
            HEADER_LINES=10
            COPYRIGHT_OFFSET=2
            SUPPORTS_SHEBANG=0
            ;;
        *)
            HEADER="$HEADER_SLASHES"
            HEADER_LINES=8
            COPYRIGHT_OFFSET=1
            SUPPORTS_SHEBANG=0
            ;;
    esac
}

# Function to update copyright in a file
update_file() {
    local file="$1"
    local ext="${file##*.}"

    TOTAL_FILES=$((TOTAL_FILES + 1))

    resolve_header_for_ext "$ext"
    local new_header="$HEADER"
    local header_lines_without_blank=$HEADER_LINES
    local header_lines_with_blank=$((HEADER_LINES + 1))
    local copyright_offset_in_header=$COPYRIGHT_OFFSET

    # Detect a leading shebang on shell scripts. The canonical layout is:
    # shebang, blank line, header, blank line, body. If the second line is not
    # blank we still preserve the shebang and rewrite the surrounding format.
    local has_shebang=0
    local prefix_lines=0
    if [ "$SUPPORTS_SHEBANG" = "1" ]; then
        local first_line
        first_line=$(head -n 1 "$file" 2> /dev/null || true)
        if [[ "$first_line" =~ ^#! ]]; then
            has_shebang=1
            local line2
            line2=$(sed -n '2p' "$file" 2> /dev/null || true)
            if [ -z "$line2" ]; then
                prefix_lines=2
            else
                prefix_lines=1
            fi
        fi
    fi

    # Detect any existing copyright header in the window after the prefix. The
    # old header may be in a different comment style than the new one (e.g. a
    # `.rs` file that was formatted as a `/* */` block), so we scan all the
    # lines a header could occupy rather than checking a single offset.
    local scan_window
    scan_window=$(sed -n "$((prefix_lines + 1)),$((prefix_lines + 12))p" "$file" 2> /dev/null || true)
    local has_copyright=0
    local old_header_lines=0
    local old_copyright_year=""
    if echo "$scan_window" | grep -q "Copyright.*Synnax Labs"; then
        has_copyright=1
        local first_after_prefix
        first_after_prefix=$(sed -n "$((prefix_lines + 1))p" "$file" 2> /dev/null || true)
        case "$first_after_prefix" in
            "/*" | "<!--")
                old_header_lines=10
                ;;
            *)
                old_header_lines=8
                ;;
        esac
        old_copyright_year=$(echo "$scan_window" | grep -oE "Copyright [0-9]+ Synnax Labs" | head -n 1 | grep -oE "[0-9]+" || true)
    fi

    # Skip if the file already conforms to the canonical layout.
    if [ "$has_copyright" = "1" ] && [ "$old_copyright_year" = "$CURRENT_YEAR" ]; then
        local current_header
        current_header=$(sed -n "$((prefix_lines + 1)),$((prefix_lines + header_lines_without_blank))p" "$file")
        local line_after_header
        line_after_header=$(sed -n "$((prefix_lines + header_lines_without_blank + 1))p" "$file" 2> /dev/null || true)
        local shebang_ok=1
        if [ $has_shebang -eq 1 ] && [ $prefix_lines -ne 2 ]; then
            shebang_ok=0
        fi
        if [ "$current_header" = "$new_header" ] && [ -z "$line_after_header" ] && [ "$shebang_ok" = "1" ]; then
            SKIPPED_FILES=$((SKIPPED_FILES + 1))
            return
        fi
    fi

    # Determine where the body starts in the original file.
    local body_start_line
    if [ $has_copyright -eq 1 ]; then
        local line_after_old_header
        line_after_old_header=$(sed -n "$((prefix_lines + old_header_lines + 1))p" "$file" 2> /dev/null || true)
        if [ -z "$line_after_old_header" ]; then
            body_start_line=$((prefix_lines + old_header_lines + 2))
        else
            body_start_line=$((prefix_lines + old_header_lines + 1))
        fi
    else
        body_start_line=$((prefix_lines + 1))
    fi

    local temp_file
    temp_file=$(mktemp)

    if [ $has_shebang -eq 1 ]; then
        head -n 1 "$file" >> "$temp_file"
        printf "\n" >> "$temp_file"
    fi
    printf "%s\n\n" "$new_header" >> "$temp_file"
    # Strip leading blank lines from the body so we don't accumulate blanks on
    # files where the existing header was followed by extra spacing.
    tail -n +"$body_start_line" "$file" | sed -e '/./,$!d' >> "$temp_file"

    # Overwrite via cat so the destination file's mode (e.g. executable bit on
    # shell scripts) is preserved.
    cat "$temp_file" > "$file"
    rm -f "$temp_file"

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
        go | py | ts | tsx | js | jsx | cpp | hpp | h | cc | cxx | css | oracle | rs | sh | zsh | html | svg) return 0 ;;
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
