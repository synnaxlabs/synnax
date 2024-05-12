#!/usr/bin/env bash

OUTPUT=$(go test -bench=. -benchtime=20x -d="$1" -s="$2" -index="$3" -data="$4" -rate="$5" -mem="$6" -w="$7" -g="$8" -only_stream="$9" -commit="${10}")

WRITE_TIME=$(echo "$OUTPUT" | sed -nE 's#BenchmarkCesium/write-[0-9]+[[:space:]]*[0-9]+[[:space:]]*([0-9]+)[[:space:]]*ns/op#\1#p')
READ_TIME=$(echo "$OUTPUT" | sed -nE 's#BenchmarkCesium\/read-[0-9]+[[:space:]]*[0-9]+[[:space:]]*([0-9]+)[[:space:]]*ns\/op#\1#p')
STREAM_TIME=$(echo "$OUTPUT" | sed -nE 's#BenchmarkCesium\/stream-[0-9]+[[:space:]]*[0-9]+[[:space:]]*([0-9]+)[[:space:]]*ns\/op#\1#p')

echo -en "${WRITE_TIME}"'\n'"${READ_TIME}"'\n'"${STREAM_TIME}"

