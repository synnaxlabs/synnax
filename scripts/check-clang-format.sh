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

# Check if any files need formatting
needs_formatting=false
for file in $files; do
  # Format the file and capture the output
  formatted_content=$(clang-format "$file")
  original_content=$(cat "$file")
  
  # Compare the original with the formatted content
  if [ "$formatted_content" != "$original_content" ]; then
    if [ "$needs_formatting" = false ]; then
      echo "The following files need to be formatted:"
      needs_formatting=true
    fi
    echo "$file"
  fi
done

if [ "$needs_formatting" = true ]; then
  echo "Run 'clang-format -i <file>' to format the files."
  exit 1
else
  echo "All files are properly formatted."
  exit 0
fi