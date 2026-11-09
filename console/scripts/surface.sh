#!/usr/bin/env bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Prints the shape of the state the Console saves, one sorted line per field. Compare
# the output of two revisions to see if a pull request changes it. The report runs under
# Vitest, which resolves the Console's imports. Build the Console's dependencies first
# (turbo build --filter=@synnaxlabs/console^...).

set -euo pipefail

console=$(cd "$(dirname "$0")/.." && pwd)
out=$(mktemp)
trap 'rm -f "$out"' EXIT
(cd "$console" && SURFACE_OUT=$out pnpm exec vitest run --project console \
    scripts/surface.spec.ts) >&2
cat "$out"
