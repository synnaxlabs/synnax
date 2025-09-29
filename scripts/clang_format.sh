#!/bin/bash

# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

echo "Clang format version"
clang-format --version

# Check for correct usage
if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <path>"
    exit 1
fi

path="$1"

# Check if the provided path exists and is a directory
if [ ! -d "$path" ]; then
    echo "Error: Path '$path' does not exist or is not a directory."
    exit 1
fi

# Find all .cpp, .hpp, .h, and .cc files in the directory
files=$(git -C "$path" ls-files -- "*.cpp" "*.hpp" "*.h" "*.cc" | grep -v "vendor/")

# Exit successfully if no files were found
if [ -z "$files" ]; then
    echo "No C++ files found in $path."
    exit 0
fi

# Check if clang-format is installed
if ! command -v clang-format &> /dev/null; then
    echo "Error: clang-format is not installed."
    exit 1
fi

# Use the root .clang-format-ignore file
ignore_file="$(git -C "$path" rev-parse --show-toplevel)/.clang-format-ignore"

# Create a temporary file to store filtered files
declare -a files_to_format=()

while IFS= read -r file; do
    should_format=true

    while IFS= read -r pattern || [ -n "$pattern" ]; do
        # Skip empty lines and comments
        [[ -z "$pattern" || "$pattern" =~ ^# ]] && continue

        # Clean up pattern
        pattern=$(echo "$pattern" | tr -d '[:space:]')
        filename=$(basename "$file")

        if [[ "$filename" == "$pattern" ]]; then
            echo "Skipping $file (ignored by pattern $pattern)..."
            should_format=false
            break
        fi
    done < "$ignore_file"

    if [ "$should_format" = true ]; then
        files_to_format+=("$file")
    fi
done <<< "$files"

# Format all files and report
formatted_count=0
for file in "${files_to_format[@]}"; do
    fullpath="$path/$file"
    echo "Formatting $file..."
    clang-format -i "$fullpath"
    if [ $? -eq 0 ]; then
        formatted_count=$((formatted_count + 1))
    else
        echo "Warning: Failed to format $file"
    fi
done

echo "Completed! Formatted $formatted_count files."
exit 0
