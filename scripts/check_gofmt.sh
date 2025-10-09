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

# Check formatting of all Go files in the provided path
diffs=$(gofmt -d -s -e "$path")
if [[ -n "$diffs" ]]; then
    echo "$diffs"
    exit 1
else
    echo "All files are properly formatted."
fi
