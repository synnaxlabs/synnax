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

# In main mode the first positional is an optional subdirectory; in batch mode
# the positionals are file paths handed in by xargs.
SUBDIR=""
if [ "$BATCH_MODE" = "0" ]; then
    SUBDIR="${1:-}"
    SUBDIR="${SUBDIR#./}"
fi

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

# Generate expected headers for different comment styles
EXPECTED_HEADER_SLASHES=$(generate_line_header "//" 1)
EXPECTED_HEADER_HASH_TWO=$(generate_line_header "#" 2)
EXPECTED_HEADER_HASH_ONE=$(generate_line_header "#" 1)
EXPECTED_HEADER_C_STYLE=$(generate_block_header "/*" " *" " * " " */")
EXPECTED_HEADER_HTML=$(generate_block_header "<!--" "" "  " "  -->")
EXPECTED_HEADER_REM=$(generate_line_header "rem" 1)
EXPECTED_HEADER_SEMI=$(generate_line_header ";" 1)

# Read .copyrightignore patterns
declare -a IGNORE_PATTERNS
if [ -f "$GIT_ROOT/.copyrightignore" ]; then
    while IFS= read -r pattern; do
        # Skip empty lines and comments
        [[ -z "$pattern" || "$pattern" =~ ^[[:space:]]*# ]] && continue
        # Remove leading/trailing whitespace
        pattern="${pattern#"${pattern%%[![:space:]]*}"}"
        pattern="${pattern%"${pattern##*[![:space:]]}"}"
        [[ -n "$pattern" ]] && IGNORE_PATTERNS+=("$pattern")
    done < "$GIT_ROOT/.copyrightignore"
fi

# Check if a file matches any ignore pattern. Patterns are evaluated as bash
# glob patterns against the path relative to GIT_ROOT.
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

# Resolve per-extension header properties into globals. TRAILING_BLANK is 1
# when the canonical layout requires a blank line between the header and the
# file body. LEADING_LINE_RE is a regex; when non-empty and the file's first
# line matches it, that line is preserved above the header (shebangs, the cmd
# `@echo off` directive that must stay first to suppress command echo, and the
# astro `---` frontmatter fence that must open the file — the header then lives
# inside the frontmatter as a // comment). LEADING_BLANK is 1 when a blank line
# separates the leading line from the header; astro sets it to 0 because
# prettier strips blank lines adjacent to the frontmatter fences.
resolve_header_for_ext() {
    local ext="$1"
    LEADING_LINE_RE=""
    LEADING_BLANK=1
    case "$ext" in
        py | pyi)
            EXPECTED_HEADER="$EXPECTED_HEADER_HASH_TWO"
            HEADER_LINES=8
            TRAILING_BLANK=1
            ;;
        sh | zsh)
            EXPECTED_HEADER="$EXPECTED_HEADER_HASH_ONE"
            HEADER_LINES=8
            LEADING_LINE_RE='^#!'
            TRAILING_BLANK=1
            ;;
        ps1 | bazel | bzl | yaml | yml | toml)
            EXPECTED_HEADER="$EXPECTED_HEADER_HASH_ONE"
            HEADER_LINES=8
            TRAILING_BLANK=1
            ;;
        nsi | def)
            EXPECTED_HEADER="$EXPECTED_HEADER_SEMI"
            HEADER_LINES=8
            TRAILING_BLANK=1
            ;;
        astro)
            EXPECTED_HEADER="$EXPECTED_HEADER_SLASHES"
            HEADER_LINES=8
            LEADING_LINE_RE='^---$'
            LEADING_BLANK=0
            TRAILING_BLANK=0
            ;;
        cmd)
            EXPECTED_HEADER="$EXPECTED_HEADER_REM"
            HEADER_LINES=8
            LEADING_LINE_RE='^@[Ee][Cc][Hh][Oo]'
            TRAILING_BLANK=1
            ;;
        css)
            EXPECTED_HEADER="$EXPECTED_HEADER_C_STYLE"
            HEADER_LINES=10
            TRAILING_BLANK=1
            ;;
        html)
            EXPECTED_HEADER="$EXPECTED_HEADER_HTML"
            HEADER_LINES=10
            TRAILING_BLANK=1
            ;;
        xml)
            EXPECTED_HEADER="$EXPECTED_HEADER_HTML"
            HEADER_LINES=10
            TRAILING_BLANK=0
            ;;
        svg)
            EXPECTED_HEADER="$EXPECTED_HEADER_HTML"
            HEADER_LINES=10
            TRAILING_BLANK=0
            ;;
        *)
            EXPECTED_HEADER="$EXPECTED_HEADER_SLASHES"
            HEADER_LINES=8
            TRAILING_BLANK=1
            ;;
    esac
}

# Read up to MAX_PREFIX_LINES lines of "$1" into the LINES global. Uses only
# bash builtins so no subprocesses are spawned per file. Trailing newline of
# the last line is normalized away (matches `sed -n Np` behavior).
MAX_PREFIX_LINES=30
read_file_prefix() {
    local file="$1"
    LINES=()
    local line
    local n=0
    while IFS= read -r line || [ -n "$line" ]; do
        LINES[n]="$line"
        n=$((n + 1))
        if [ "$n" -ge "$MAX_PREFIX_LINES" ]; then
            break
        fi
    done < "$file"
}

# Classify a single file and print one of:
#   OK<TAB>$file
#   MISSING<TAB>$file
#   WRONG_YEAR<TAB>$file
#   MALFORMED<TAB>$file
#   DUPLICATE<TAB>$file
classify_file() {
    local file="$1"
    local ext="${file##*.}"
    resolve_header_for_ext "$ext"
    local expected_header="$EXPECTED_HEADER"
    local header_lines=$HEADER_LINES
    local trailing_blank=$TRAILING_BLANK

    read_file_prefix "$file"

    if [ ${#LINES[@]} -eq 0 ]; then
        printf 'MISSING\t%s\n' "$file"
        return
    fi

    # Canonical layout: [leading line / [blank] /] header / [blank /] body. Header
    # starts at index 0 normally; when a leading line is preserved, at index 2
    # iff line 2 is blank (LEADING_BLANK=1), or at index 1 directly against the
    # leading line (LEADING_BLANK=0, e.g. astro frontmatter).
    local header_start_idx=0
    if [ -n "$LEADING_LINE_RE" ] && [[ "${LINES[0]}" =~ $LEADING_LINE_RE ]]; then
        if [ "$LEADING_BLANK" = "0" ]; then
            header_start_idx=1
        elif [ -z "${LINES[1]:-}" ]; then
            header_start_idx=2
        else
            printf 'MALFORMED\t%s\n' "$file"
            return
        fi
    fi

    local header_end_idx=$((header_start_idx + header_lines - 1))

    # If the file has fewer lines than the canonical header, the existing
    # script returned MISSING when sed produced empty output. Match that.
    if [ $header_start_idx -ge ${#LINES[@]} ]; then
        printf 'MISSING\t%s\n' "$file"
        return
    fi

    # Build the header slice we'll compare against the canonical text.
    local file_header=""
    local i
    local last_idx=$header_end_idx
    if [ $last_idx -ge ${#LINES[@]} ]; then
        last_idx=$((${#LINES[@]} - 1))
    fi
    for ((i = header_start_idx; i <= last_idx; i++)); do
        if [ $i -gt $header_start_idx ]; then
            file_header+=$'\n'
        fi
        file_header+="${LINES[i]}"
    done

    # Scan header slice line-by-line for the "Copyright ... Synnax Labs" marker
    # and the expected current year. Mirrors `grep -q` semantics (per-line).
    local has_copyright=0
    local has_current_year=0
    for ((i = header_start_idx; i <= last_idx; i++)); do
        local line="${LINES[i]}"
        if [[ "$line" == *"Copyright"*"Synnax Labs"* ]]; then
            has_copyright=1
            if [[ "$line" == *"Copyright $CURRENT_YEAR Synnax Labs"* ]]; then
                has_current_year=1
            fi
        fi
    done

    if [ "$has_copyright" = "0" ]; then
        printf 'MISSING\t%s\n' "$file"
        return
    fi

    if [ "$has_current_year" = "0" ]; then
        printf 'WRONG_YEAR\t%s\n' "$file"
        return
    fi

    if [ "$file_header" != "$expected_header" ]; then
        printf 'MALFORMED\t%s\n' "$file"
        return
    fi

    # Trailing-blank rule: the line immediately after the header must be blank
    # (or absent) when TRAILING_BLANK=1, and non-blank otherwise.
    local line_after_header=""
    if [ $((header_end_idx + 1)) -lt ${#LINES[@]} ]; then
        line_after_header="${LINES[$((header_end_idx + 1))]}"
    fi
    if [ "$trailing_blank" = "1" ] && [ -n "$line_after_header" ]; then
        printf 'MALFORMED\t%s\n' "$file"
        return
    fi
    if [ "$trailing_blank" = "0" ] && [ -z "$line_after_header" ]; then
        printf 'MALFORMED\t%s\n' "$file"
        return
    fi

    # Duplicate check: count header copyright lines in the first 20 lines to
    # guard against a header accidentally emitted twice. Only lines that *begin*
    # with the notice (after stripping a leading comment marker and whitespace)
    # count — a line that merely embeds the notice as data, such as the
    # JetBrains copyright-profile XML's value="Copyright ..." attribute, is not
    # a second header and must not trip this check.
    local dup_limit=20
    if [ $dup_limit -gt ${#LINES[@]} ]; then
        dup_limit=${#LINES[@]}
    fi
    local copyright_count=0
    for ((i = 0; i < dup_limit; i++)); do
        local dl="${LINES[i]}"
        dl="${dl#"${dl%%[![:space:]]*}"}"
        case "$dl" in
            "//"*) dl="${dl#//}" ;;
            "#"*) dl="${dl#\#}" ;;
            "*"*) dl="${dl#\*}" ;;
            "~"*) dl="${dl#\~}" ;;
            "rem "*) dl="${dl#rem}" ;;
        esac
        dl="${dl#"${dl%%[![:space:]]*}"}"
        if [[ "$dl" == "Copyright"*"Synnax Labs"* ]]; then
            copyright_count=$((copyright_count + 1))
        fi
    done
    if [ "$copyright_count" -gt 1 ]; then
        printf 'DUPLICATE\t%s\n' "$file"
        return
    fi

    printf 'OK\t%s\n' "$file"
}

if [ "$BATCH_MODE" = "1" ]; then
    for file in "$@"; do
        classify_file "$file"
    done
    exit 0
fi

# ---------------------------------------------------------------------------
# Main mode: enumerate files, dispatch to parallel workers, aggregate.
# ---------------------------------------------------------------------------

echo "Checking copyright headers in source files..."
echo "Git root: $GIT_ROOT"
echo "Search path: $SEARCH_PATH"
echo "Current year: $CURRENT_YEAR"
echo ""

echo -n "Counting files..."
cd "$GIT_ROOT" || exit 1
declare -a FILES_TO_CHECK
while IFS= read -r file; do
    abs_file="$GIT_ROOT/$file"
    if [[ "$abs_file" != "$SEARCH_PATH"* ]]; then
        continue
    fi
    ext="${file##*.}"
    case "$ext" in
        go | py | pyi | ts | tsx | js | jsx | c | cpp | hpp | h | cc | cxx | css | oracle | rs | sh | zsh | html | xml | svg | proto | g4 | glsl | bazel | bzl | ps1 | cmd | yaml | yml | arc | astro | toml | nsi | def)
            if ! should_ignore_file "$abs_file"; then
                [ -f "$abs_file" ] && FILES_TO_CHECK+=("$abs_file")
            fi
            ;;
    esac
done < <(git ls-files)

TOTAL_TO_CHECK=${#FILES_TO_CHECK[@]}
printf '\r\033[KFound %d files to check\n\n' "$TOTAL_TO_CHECK"

if [ "$TOTAL_TO_CHECK" -eq 0 ]; then
    echo "Nothing to check."
    exit 0
fi

# Counters and result arrays (main process only).
TOTAL_FILES=0
MISSING_HEADER=0
WRONG_YEAR=0
MALFORMED_HEADER=0
DUPLICATE_HEADER=0
declare -a FILES_MISSING_HEADER
declare -a FILES_WRONG_YEAR
declare -a FILES_MALFORMED_HEADER
declare -a FILES_DUPLICATE_HEADER

# Pick a worker count from the OS.
NUM_WORKERS=$(getconf _NPROCESSORS_ONLN 2> /dev/null \
    || sysctl -n hw.ncpu 2> /dev/null \
    || nproc 2> /dev/null \
    || echo 4)

# Stream file paths to xargs, which spawns up to NUM_WORKERS bash workers in
# parallel. Each worker handles ~100 files per invocation so the script's
# own one-time setup cost (template parse, header generation) is amortized.
while IFS=$'\t' read -r status path; do
    TOTAL_FILES=$((TOTAL_FILES + 1))
    case "$status" in
        OK) ;;
        MISSING)
            FILES_MISSING_HEADER+=("$path")
            MISSING_HEADER=$((MISSING_HEADER + 1))
            ;;
        WRONG_YEAR)
            FILES_WRONG_YEAR+=("$path")
            WRONG_YEAR=$((WRONG_YEAR + 1))
            ;;
        MALFORMED)
            FILES_MALFORMED_HEADER+=("$path")
            MALFORMED_HEADER=$((MALFORMED_HEADER + 1))
            ;;
        DUPLICATE)
            FILES_DUPLICATE_HEADER+=("$path")
            DUPLICATE_HEADER=$((DUPLICATE_HEADER + 1))
            ;;
        *)
            # Defensive: surface any unexpected line so problems are visible
            # rather than silently dropped.
            echo "Warning: unexpected worker output: $status $path" >&2
            ;;
    esac
    if ((TOTAL_FILES % 100 == 0)) || [ "$TOTAL_FILES" -eq "$TOTAL_TO_CHECK" ]; then
        printf '\r\033[KChecked %d/%d files' "$TOTAL_FILES" "$TOTAL_TO_CHECK"
    fi
done < <(
    printf '%s\n' "${FILES_TO_CHECK[@]}" \
        | xargs -P "$NUM_WORKERS" -n 100 bash "$0" __batch
)

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
