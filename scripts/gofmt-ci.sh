#!/bin/bash

files=$(gofmt -l -s -e .)
if [[ -n "$files" ]]; then
    echo "The following files need to be formatted:"
    echo "$files"
    exit 1
else
    echo "All files are properly formatted."
fi
