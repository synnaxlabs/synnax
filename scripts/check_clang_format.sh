#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
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
    exit 1
fi

path="$1"

# Check if the provided path exists and is a directory
if [ ! -d "$path" ]; then
    echo "Error: Path '$path' does not exist or is not a directory."
    exit 1
fi

# Check if clang-format is installed
if ! command -v clang-format &> /dev/null; then
    echo "Error: clang-format is not installed."
    exit 1
fi

# Find all .cpp, .hpp, .h, and .cc files in the directory. Excluded files are handled
# by clang-format itself via the root .clang-format-ignore file, so we don't filter
# them here.
files=$(git -C "$path" ls-files -- "*.cpp" "*.hpp" "*.h" "*.cc" | grep -v "vendor/")

# Exit successfully if no files were found
if [ -z "$files" ]; then
    echo "No C++ files found in $path."
    exit 0
fi

jobs=$(getconf _NPROCESSORS_ONLN 2> /dev/null || echo 4)

# Run clang-format in parallel batches. --dry-run --Werror checks formatting in a single
# pass (no separate diff) and exits non-zero on any violation, printing the offending
# locations. xargs propagates that non-zero status.
if echo "$files" \
    | sed "s|^|$path/|" \
    | xargs -P "$jobs" -n 32 clang-format --dry-run --Werror; then
    echo "All files are properly formatted."
    exit 0
fi

echo "Run 'scripts/clang_format.sh $path' to format the files."
exit 1
