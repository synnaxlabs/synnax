#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
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

# Detect __batch mode early; in that mode the script processes a chunk of files
# handed to it by xargs and emits TAB-separated "STATUS<TAB>path" lines.
BATCH_MODE=0
if [ "${1:-}" = "__batch" ]; then
    BATCH_MODE=1
    shift
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

# Read .copyrightignore patterns
declare -a IGNORE_PATTERNS
if [ -f "$GIT_ROOT/.copyrightignore" ]; then
    while IFS= read -r pattern; do
        [[ -z "$pattern" || "$pattern" =~ ^[[:space:]]*# ]] && continue
        pattern="${pattern#"${pattern%%[![:space:]]*}"}"
        pattern="${pattern%"${pattern##*[![:space:]]}"}"
        [[ -n "$pattern" ]] && IGNORE_PATTERNS+=("$pattern")
    done < "$GIT_ROOT/.copyrightignore"
fi

should_ignore_file() {
    local file="$1"
    local relative_path="${file#$GIT_ROOT/}"
    local pattern
    for pattern in ${IGNORE_PATTERNS[@]+"${IGNORE_PATTERNS[@]}"}; do
        case "$relative_path" in
            $pattern) return 0 ;;
        esac
    done
    return 1
}

has_supported_extension() {
    local file="$1"
    local ext="${file##*.}"
    case "$ext" in
        go | py | ts | tsx | js | jsx | cpp | hpp | h | cc | cxx | css | oracle | rs | sh | zsh | html | svg) return 0 ;;
        *) return 1 ;;
    esac
}

# Resolve per-extension header properties into globals: HEADER, HEADER_LINES,
# SUPPORTS_SHEBANG (1 if the extension preserves a leading #! line), and
# TRAILING_BLANK (1 if the canonical layout puts a blank line between the
# header and the file body, 0 otherwise). HEADER_LINES describes the canonical
# new header — the size of an *existing* header is detected dynamically.
resolve_header_for_ext() {
    local ext="$1"
    case "$ext" in
        py)
            HEADER="$HEADER_HASH_TWO"
            HEADER_LINES=8
            SUPPORTS_SHEBANG=0
            TRAILING_BLANK=1
            ;;
        sh | zsh)
            HEADER="$HEADER_HASH_ONE"
            HEADER_LINES=8
            SUPPORTS_SHEBANG=1
            TRAILING_BLANK=1
            ;;
        css)
            HEADER="$HEADER_C_STYLE"
            HEADER_LINES=10
            SUPPORTS_SHEBANG=0
            TRAILING_BLANK=1
            ;;
        html)
            HEADER="$HEADER_HTML"
            HEADER_LINES=10
            SUPPORTS_SHEBANG=0
            TRAILING_BLANK=1
            ;;
        svg)
            HEADER="$HEADER_HTML"
            HEADER_LINES=10
            SUPPORTS_SHEBANG=0
            TRAILING_BLANK=0
            ;;
        *)
            HEADER="$HEADER_SLASHES"
            HEADER_LINES=8
            SUPPORTS_SHEBANG=0
            TRAILING_BLANK=1
            ;;
    esac
}

# Read the entire file into the LINES global. Uses only bash builtins so no
# subprocesses are spawned per file.
read_whole_file() {
    local file="$1"
    LINES=()
    local line
    local n=0
    while IFS= read -r line || [ -n "$line" ]; do
        LINES[n]="$line"
        n=$((n + 1))
    done < "$file"
}

# Locate an existing copyright header inside LINES, starting the search at
# zero-based index $1. Sets:
#   OLD_HEADER_FOUND   1 if a header was found, 0 otherwise
#   OLD_HEADER_START   first index (0-based) of the existing header
#   OLD_HEADER_END     last index (0-based) of the existing header
#   OLD_COPYRIGHT_YEAR year string from the existing header (e.g. "2025")
#
# The end is detected by walking forward from OLD_HEADER_START: for block
# styles (`/*` / `<!--`), to the closing marker; for line styles (`//` / `#`),
# until the first blank line or first line that doesn't start with the same
# comment prefix. This handles non-canonical old headers that are shorter or
# longer than the new canonical header.
locate_existing_header() {
    local scan_start_idx="$1"
    OLD_HEADER_FOUND=0
    OLD_HEADER_START=0
    OLD_HEADER_END=0
    OLD_COPYRIGHT_YEAR=""

    local scan_end_idx=$((scan_start_idx + 14))
    if [ $scan_end_idx -ge ${#LINES[@]} ]; then
        scan_end_idx=$((${#LINES[@]} - 1))
    fi

    # Find the first "Copyright ... Synnax Labs" line in the scan window.
    local cr_idx=-1
    local i
    for ((i = scan_start_idx; i <= scan_end_idx; i++)); do
        if [[ "${LINES[i]}" == *"Copyright"*"Synnax Labs"* ]]; then
            cr_idx=$i
            break
        fi
    done
    if [ "$cr_idx" -lt 0 ]; then
        return
    fi

    OLD_HEADER_FOUND=1

    # Extract the year from the matched line (e.g. "Copyright 2025 Synnax Labs").
    local cr_line="${LINES[cr_idx]}"
    local year_re='Copyright[[:space:]]+([0-9]+)[[:space:]]+Synnax Labs'
    if [[ "$cr_line" =~ $year_re ]]; then
        OLD_COPYRIGHT_YEAR="${BASH_REMATCH[1]}"
    fi

    # Determine block style by looking at the line preceding the match.
    local style="line"
    local prev_line=""
    if [ "$cr_idx" -gt 0 ]; then
        prev_line="${LINES[cr_idx - 1]}"
    fi
    case "$prev_line" in
        "/*")
            style="c_block"
            OLD_HEADER_START=$((cr_idx - 1))
            ;;
        "<!--")
            style="html_block"
            OLD_HEADER_START=$((cr_idx - 1))
            ;;
        *)
            OLD_HEADER_START=$cr_idx
            ;;
    esac

    local max_scan_idx=$((OLD_HEADER_START + 30))
    if [ $max_scan_idx -ge ${#LINES[@]} ]; then
        max_scan_idx=$((${#LINES[@]} - 1))
    fi

    local end_idx=$OLD_HEADER_START
    case "$style" in
        c_block)
            for ((i = OLD_HEADER_START + 1; i <= max_scan_idx; i++)); do
                end_idx=$i
                if [ "${LINES[i]}" = " */" ]; then
                    break
                fi
            done
            ;;
        html_block)
            local html_close_re='-->[[:space:]]*$'
            for ((i = OLD_HEADER_START + 1; i <= max_scan_idx; i++)); do
                end_idx=$i
                if [[ "${LINES[i]}" =~ $html_close_re ]]; then
                    break
                fi
            done
            ;;
        *)
            local first_line="${LINES[OLD_HEADER_START]}"
            local prefix
            case "$first_line" in
                "//"*) prefix="//" ;;
                "#"*) prefix="#" ;;
                *) prefix="" ;;
            esac
            if [ -n "$prefix" ]; then
                end_idx=$OLD_HEADER_START
                for ((i = OLD_HEADER_START + 1; i <= max_scan_idx; i++)); do
                    local line="${LINES[i]}"
                    if [ -z "$line" ]; then
                        break
                    fi
                    case "$line" in
                        "$prefix"*) end_idx=$i ;;
                        *) break ;;
                    esac
                done
            fi
            ;;
    esac

    OLD_HEADER_END=$end_idx
}

# Process a single file. Prints "UPDATED\t$file" or "SKIPPED\t$file".
process_file() {
    local file="$1"
    local ext="${file##*.}"
    resolve_header_for_ext "$ext"
    local new_header="$HEADER"
    local trailing_blank="$TRAILING_BLANK"

    read_whole_file "$file"

    # Detect a leading shebang on shell scripts. The canonical layout always
    # restores: shebang / blank / header / [blank] / body.
    local has_shebang=0
    if [ "$SUPPORTS_SHEBANG" = "1" ] && [ ${#LINES[@]} -gt 0 ]; then
        if [[ "${LINES[0]}" =~ ^#! ]]; then
            has_shebang=1
        fi
    fi
    local scan_start_idx=$has_shebang

    locate_existing_header "$scan_start_idx"
    local has_copyright=$OLD_HEADER_FOUND
    local old_header_start=$OLD_HEADER_START
    local old_header_end=$OLD_HEADER_END
    local old_copyright_year=$OLD_COPYRIGHT_YEAR

    # Detect "between" content (between scan_start_idx and old_header_start).
    # Preserves e.g. a `set -euo pipefail` line that lived between the shebang
    # and the old header. A canonical shebang+blank prefix yields a blank-only
    # window, which we ignore so the layout is recognized as already canonical.
    local between_start=-1
    local between_end=-1
    if [ "$has_copyright" = "1" ] && [ "$old_header_start" -gt "$scan_start_idx" ]; then
        local i
        local has_nonblank=0
        for ((i = scan_start_idx; i < old_header_start; i++)); do
            if [[ "${LINES[i]}" =~ [^[:space:]] ]]; then
                has_nonblank=1
                break
            fi
        done
        if [ $has_nonblank = 1 ]; then
            between_start=$scan_start_idx
            between_end=$((old_header_start - 1))
        fi
    fi

    # Determine where the body starts. Skip the detected old header and an
    # optional trailing blank.
    local body_start_idx
    if [ "$has_copyright" = "1" ]; then
        local line_after_old=""
        if [ $((old_header_end + 1)) -lt ${#LINES[@]} ]; then
            line_after_old="${LINES[old_header_end + 1]}"
        fi
        if [ -z "$line_after_old" ]; then
            body_start_idx=$((old_header_end + 2))
        else
            body_start_idx=$((old_header_end + 1))
        fi
    else
        body_start_idx=$scan_start_idx
    fi

    # Skip if the file already conforms to the canonical layout.
    if [ "$has_copyright" = "1" ] \
        && [ "$old_copyright_year" = "$CURRENT_YEAR" ] \
        && [ "$between_start" = "-1" ]; then
        local canonical_header_start_idx=$scan_start_idx
        if [ "$has_shebang" = "1" ]; then
            local line2=""
            if [ ${#LINES[@]} -ge 2 ]; then
                line2="${LINES[1]}"
            fi
            if [ -z "$line2" ]; then
                canonical_header_start_idx=2
            else
                canonical_header_start_idx=-1
            fi
        fi
        if [ "$canonical_header_start_idx" != "-1" ] \
            && [ "$old_header_start" = "$canonical_header_start_idx" ]; then
            local current_header=""
            local i
            for ((i = old_header_start; i <= old_header_end; i++)); do
                if [ $i -gt $old_header_start ]; then
                    current_header+=$'\n'
                fi
                current_header+="${LINES[i]}"
            done
            local line_after_canonical=""
            if [ $((old_header_end + 1)) -lt ${#LINES[@]} ]; then
                line_after_canonical="${LINES[old_header_end + 1]}"
            fi
            local format_ok=1
            [ "$current_header" != "$new_header" ] && format_ok=0
            if [ "$trailing_blank" = "1" ] && [ -n "$line_after_canonical" ]; then format_ok=0; fi
            if [ "$trailing_blank" = "0" ] && [ -z "$line_after_canonical" ]; then format_ok=0; fi
            if [ "$format_ok" = "1" ]; then
                printf 'SKIPPED\t%s\n' "$file"
                return
            fi
        fi
    fi

    # Build the new file content. We assemble: shebang / blank / new header /
    # [blank] / preserved-between / body-with-leading-blanks-stripped. Then we
    # write atomically via a temp file, preserving the original file mode.
    # Assemble the body in-memory, then write the full new file content in a
    # single redirection.
    local body=""
    local first_nonblank_seen=0
    local i line
    if [ "$between_start" -ge 0 ]; then
        for ((i = between_start; i <= between_end; i++)); do
            line="${LINES[i]}"
            if [ "$first_nonblank_seen" = "0" ] && [[ ! "$line" =~ [^[:space:]] ]]; then
                continue
            fi
            first_nonblank_seen=1
            body+="$line"$'\n'
        done
    fi
    for ((i = body_start_idx; i < ${#LINES[@]}; i++)); do
        line="${LINES[i]}"
        if [ "$first_nonblank_seen" = "0" ] && [[ ! "$line" =~ [^[:space:]] ]]; then
            continue
        fi
        first_nonblank_seen=1
        body+="$line"$'\n'
    done

    # Write atomically via a temp file, then move it back to preserve the
    # destination's file mode (e.g. executable bit on shell scripts).
    local temp_file
    temp_file=$(mktemp)
    {
        if [ "$has_shebang" = "1" ]; then
            printf '%s\n\n' "${LINES[0]}"
        fi
        printf '%s\n' "$new_header"
        if [ "$trailing_blank" = "1" ]; then
            printf '\n'
        fi
        printf '%s' "$body"
    } > "$temp_file"
    cat "$temp_file" > "$file"
    rm -f "$temp_file"

    printf 'UPDATED\t%s\n' "$file"
}

if [ "$BATCH_MODE" = "1" ]; then
    for file in "$@"; do
        process_file "$file"
    done
    exit 0
fi

# ---------------------------------------------------------------------------
# Main mode: parse args, enumerate files, dispatch, aggregate.
# ---------------------------------------------------------------------------

EXPLICIT_FILES=()
SEARCH_DIRS=()
PATTERNS=()

if [ $# -gt 0 ]; then
    for arg in "$@"; do
        if [ -f "$arg" ]; then
            if [[ "$arg" = /* ]]; then
                EXPLICIT_FILES+=("$arg")
            else
                EXPLICIT_FILES+=("$GIT_ROOT/$arg")
            fi
        elif [ -d "$arg" ]; then
            if [[ "$arg" = /* ]]; then
                SEARCH_DIRS+=("$arg")
            else
                SEARCH_DIRS+=("$GIT_ROOT/$arg")
            fi
        elif [[ "$arg" == *"*"* ]] || [[ "$arg" == *"?"* ]] || [[ "$arg" == *"["* ]]; then
            PATTERNS+=("$arg")
        else
            echo "Warning: '$arg' is not a file, directory, or pattern, skipping"
        fi
    done
else
    SEARCH_DIRS+=("$GIT_ROOT")
fi

echo "Updating copyright headers in source files..."
echo "Git root: $GIT_ROOT"
[ ${#EXPLICIT_FILES[@]} -gt 0 ] && echo "Files: ${#EXPLICIT_FILES[@]}"
[ ${#SEARCH_DIRS[@]} -gt 0 ] && echo "Directories: ${SEARCH_DIRS[*]}"
[ ${#PATTERNS[@]} -gt 0 ] && echo "Patterns: ${PATTERNS[*]}"
echo "Current year: $CURRENT_YEAR"
echo ""

echo -n "Counting files..."
cd "$GIT_ROOT" || exit 1
declare -a FILES_TO_UPDATE

try_add_file() {
    local abs_file="$1"
    if [ -f "$abs_file" ] && has_supported_extension "$abs_file" && ! should_ignore_file "$abs_file"; then
        FILES_TO_UPDATE+=("$abs_file")
    fi
}

for abs_file in ${EXPLICIT_FILES[@]+"${EXPLICIT_FILES[@]}"}; do
    try_add_file "$abs_file"
done

for search_dir in ${SEARCH_DIRS[@]+"${SEARCH_DIRS[@]}"}; do
    while IFS= read -r file; do
        abs_file="$GIT_ROOT/$file"
        if [[ "$abs_file" == "$search_dir"* ]]; then
            try_add_file "$abs_file"
        fi
    done < <(git ls-files)
done

for pattern in ${PATTERNS[@]+"${PATTERNS[@]}"}; do
    while IFS= read -r abs_file; do
        try_add_file "$abs_file"
    done < <(find "$GIT_ROOT" -type f -name "$pattern" 2> /dev/null)
done

TOTAL_TO_UPDATE=${#FILES_TO_UPDATE[@]}
printf '\r\033[KFound %d files to process\n\n' "$TOTAL_TO_UPDATE"

if [ "$TOTAL_TO_UPDATE" -eq 0 ]; then
    echo "Nothing to process."
    exit 0
fi

TOTAL_FILES=0
UPDATED_FILES=0
SKIPPED_FILES=0
declare -a FILES_UPDATED

NUM_WORKERS=$(getconf _NPROCESSORS_ONLN 2> /dev/null \
    || sysctl -n hw.ncpu 2> /dev/null \
    || nproc 2> /dev/null \
    || echo 4)

while IFS=$'\t' read -r status path; do
    TOTAL_FILES=$((TOTAL_FILES + 1))
    case "$status" in
        UPDATED)
            FILES_UPDATED+=("$path")
            UPDATED_FILES=$((UPDATED_FILES + 1))
            ;;
        SKIPPED)
            SKIPPED_FILES=$((SKIPPED_FILES + 1))
            ;;
        *)
            echo "Warning: unexpected worker output: $status $path" >&2
            ;;
    esac
    if ((TOTAL_FILES % 100 == 0)) || [ "$TOTAL_FILES" -eq "$TOTAL_TO_UPDATE" ]; then
        printf '\r\033[KProcessed %d/%d files' "$TOTAL_FILES" "$TOTAL_TO_UPDATE"
    fi
done < <(
    printf '%s\0' "${FILES_TO_UPDATE[@]}" \
        | xargs -0 -P "$NUM_WORKERS" -n 100 bash "$0" __batch
)

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
