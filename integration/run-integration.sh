#!/bin/sh

if [ $# -lt 1 ]; then
    echo "Usage: $0 <test config name>"
    exit 1
fi

# Run the Go program with the provided arguments
go run . "$1"

# Capture the exit code of the Go program
exit_code=$?

# Check if the exit code is not 0
if [ $exit_code -ne 0 ]; then
    echo "Test failed, see stdout for more info"
    exit $exit_code
fi

printf "\n"

# Find the last occurrence of "Test Started" and print everything after that
last_occurrence=$(grep -n "Test Started" ./timing.log | tail -n 1 | cut -d: -f1)

# Check if the last occurrence is found
if [ -z "$last_occurrence" ]; then
    echo "No occurrences of 'Test Started' found in timing.log"
    exit 1
fi

# Print everything after the last occurrence
sed -n "${last_occurrence},\$p" ./timing.log