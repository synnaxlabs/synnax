#!/bin/bash

# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Check for correct usage
if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <path>"
    echo "Example: $0 driver"
    echo "         $0 x/cpp"
    exit 1
fi

path="$1"

# Check if the provided path exists and is a directory
if [ ! -d "$path" ]; then
    echo "Error: Path '$path' does not exist or is not a directory."
    exit 1
fi

# Check if clang-tidy is installed
if ! command -v clang-tidy &> /dev/null; then
    echo "Error: clang-tidy is not installed."
    exit 1
fi

# Get the repository root
repo_root=$(git rev-parse --show-toplevel)

echo "Generating compilation database with Bazel..."
cd "$repo_root" || exit 1

# Generate compile_commands.json using hedron_compile_commands
# This creates a symlink at the workspace root
bazel run @hedron_compile_commands//:refresh_all

if [ ! -f "compile_commands.json" ]; then
    echo "Error: Failed to generate compile_commands.json"
    exit 1
fi

echo "Running clang-tidy on $path..."

# Find all .cpp, .hpp, .h, and .cc files in the directory
# Exclude vendor directory and generated protobuf files
files=$(git -C "$path" ls-files -- "*.cpp" "*.hpp" "*.h" "*.cc" \
    | grep -v "vendor/" \
    | grep -v "\.pb\.h$" \
    | grep -v "\.pb\.cc$" \
    | grep -v "\.grpc\.pb\.h$" \
    | grep -v "\.grpc\.pb\.cc$")

# Exit successfully if no files were found
if [ -z "$files" ]; then
    echo "No C++ files found in $path."
    exit 0
fi

# Count total files
total_files=$(echo "$files" | wc -l | tr -d ' ')
echo "Found $total_files files to check"

# Run clang-tidy on each file
exit_code=0
current=0

while IFS= read -r file; do
    current=$((current + 1))
    full_path="$path/$file"

    echo "[$current/$total_files] Checking $file..."

    # Run clang-tidy with the compilation database
    # -p points to the directory containing compile_commands.json
    if ! clang-tidy -p "$repo_root" "$full_path"; then
        exit_code=1
    fi
done <<< "$files"

if [ $exit_code -eq 0 ]; then
    echo "✓ All files passed clang-tidy checks"
else
    echo "✗ Some files have clang-tidy violations"
    echo ""
    echo "To automatically fix some issues, run:"
    echo "  clang-tidy -p . --fix-errors <file>"
fi

exit $exit_code
