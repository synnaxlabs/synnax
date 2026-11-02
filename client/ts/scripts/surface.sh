#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Prints the public type surface of the TypeScript client: the declarations tsc emits
# for every source file, sorted by path, without comments or private members. Compare
# the output of two revisions to see if a pull request changes the client API. Build
# the client's dependencies first (turbo build --filter=@synnaxlabs/client^...), or
# their types resolve to any.
#
#   surface.sh [project-dir]
#
# project-dir holds the tsconfig.build.json to emit and defaults to the client.

set -euo pipefail

client=$(cd "$(dirname "$0")/.." && pwd)
project=$(cd "${1:-$client}" && pwd)
out=$(mktemp -d)
trap 'rm -rf "$out"' EXIT
pnpm --dir "$client" exec tsc -p "$project/tsconfig.build.json" --outDir "$out" \
    --removeComments --declarationMap false >&2
cd "$out"
find . -name '*.d.ts' | LC_ALL=C sort | while read -r file; do
    echo "# ${file#./}"
    grep -Ev '^export \{\};$|^ +private ' "$file" || true
done
