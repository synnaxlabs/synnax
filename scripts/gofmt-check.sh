#!/bin/bash

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

# Check formatting of all Go files in the provided path
files=$(gofmt -l -s -e "$path")
if [[ -n "$files" ]]; then
    echo "The following files need to be formatted:"
    echo "$files"
    exit 1
else
    echo "All files are properly formatted."
fi
