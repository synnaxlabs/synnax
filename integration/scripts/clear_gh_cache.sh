#!/bin/bash

# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Requires: jq, gh CLI

# Get current time and calculate cutoff (4 hours ago) - macOS compatible
cutoff=$(date -u -v-12H +%s)

# List all caches with creation time
gh cache list --json id,createdAt | jq -c '.[]' | while read cache; do
  id=$(echo "$cache" | jq -r '.id')
  createdAt=$(echo "$cache" | jq -r '.createdAt')
  # Convert ISO date to epoch - macOS compatible
  createdAtEpoch=$(date -u -j -f "%Y-%m-%dT%H:%M:%SZ" "$createdAt" +%s 2>/dev/null || date -u -j -f "%Y-%m-%dT%H:%M:%S" "${createdAt%Z}" +%s)

  if [ "$createdAtEpoch" -lt "$cutoff" ]; then
    echo "Deleting cache $id (created at $createdAt)..."
    gh cache delete "$id"
  fi
done