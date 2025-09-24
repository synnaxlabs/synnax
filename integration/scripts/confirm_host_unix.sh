#!/bin/bash

# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

echo "Host confirmed"
echo "OS: $(uname -s)"
echo "Kernel: $(uname -r)"
echo "Architecture: $(uname -m)"
echo "Hostname: $(hostname)"

if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    echo "CPU cores: $(sysctl -n hw.ncpu)"
    echo "Physical memory: $(($(sysctl -n hw.memsize) / 1024 / 1024 / 1024))GB"
    echo "macOS version: $(sw_vers -productVersion)"
else
    # Linux
    echo "CPU cores: $(nproc)"
    echo "Memory: $(free -h | grep '^Mem:' | awk '{print $2}')"
fi

echo "Disk space: $(df -h / | tail -1 | awk '{print $4}') available"
echo "Date: $(date)"
