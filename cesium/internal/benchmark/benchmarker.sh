#!/usr/bin/env bash

# Copyright 2025 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

OUTPUT=$(go test -bench=. -run=^# -benchtime=20s -d="$1" -s="$2" -index="$3" -data="$4" -rate="$5" -mem="$6" -w="$7" -g="$8" -only_stream="$9" -commit="${10}")

WRITE_TIME=$(echo "$OUTPUT" | sed -nE 's#BenchmarkCesium/write-[0-9]+[[:space:]]*[0-9]+[[:space:]]*([0-9]+)[[:space:]]*ns/op#\1#p')
READ_TIME=$(echo "$OUTPUT" | sed -nE 's#BenchmarkCesium\/read-[0-9]+[[:space:]]*[0-9]+[[:space:]]*([0-9]+)[[:space:]]*ns\/op#\1#p')
STREAM_TIME=$(echo "$OUTPUT" | sed -nE 's#BenchmarkCesium\/stream-[0-9]+[[:space:]]*[0-9]+[[:space:]]*([0-9]+)[[:space:]]*ns\/op#\1#p')

if [ -z "$WRITE_TIME" ] || [ -z "$READ_TIME" ] || [ -z "$STREAM_TIME" ]; then
    echo "$OUTPUT"
else
    echo -en "${WRITE_TIME}"'\n'"${READ_TIME}"'\n'"${STREAM_TIME}"
fi
