#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Runs `gh api` with the given arguments. It retries a transient failure: an empty, cut,
# or plain text body, an HTTP 5xx, or a network error. Every other failure, such as a
# 404 or 422, exits at once with gh's output and exit code. A failed write can still
# land, so only send requests that are safe to repeat.
# GH_API_RETRY_DELAY sets the seconds before the first retry; it doubles after that. The
# default waits 30 seconds in all, which is longer than a short GitHub fault.

set -uo pipefail

ATTEMPTS=5
TRANSIENT='unexpected end of JSON input|invalid character|HTTP 5[0-9]{2}([^0-9]|$)'
TRANSIENT+='|dial tcp|connection reset|i/o timeout|TLS handshake timeout|unexpected EOF'

delay=${GH_API_RETRY_DELAY:-2}
out=$(mktemp)
err=$(mktemp)
trap 'rm -f "$out" "$err"' EXIT

for ((attempt = 1; ; attempt++)); do
    code=0
    gh api "$@" > "$out" 2> "$err" || code=$?
    if ((code == 0 || attempt == ATTEMPTS)) \
        || ! grep -qE "$TRANSIENT" "$err"; then
        break
    fi
    echo "gh api: transient failure, retry $attempt in ${delay}s" >&2
    sleep "$delay"
    delay=$((delay * 2))
done

cat "$out"
cat "$err" >&2
exit "$code"
