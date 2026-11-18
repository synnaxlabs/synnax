#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Usage: pack_ts.sh <package> <directory>
#
# Builds and packs @synnaxlabs/<package> and its Synnax dependencies into <directory> as
# <name>.tgz, then installs the package there in a new ES module project outside the
# workspace, with the layout that npm makes.

set -euo pipefail

package=$1
work=$(mkdir -p "$2" && cd "$2" && pwd)
repo=$(cd "$(dirname "$0")/.." && pwd)
cd "$repo"

pnpm build:"$package"
deps=$(pnpm --filter "@synnaxlabs/$package" exec node -p \
    'Object.keys(require("./package.json").dependencies ?? {}).join("\n")' \
    | sed -n 's|^@synnaxlabs/||p')
for name in "$package" $deps; do
    pnpm --filter "@synnaxlabs/$name" pack --out "$work/$name.tgz" > /dev/null
done

# The overrides keep pnpm from fetching the Synnax dependencies from the registry.
cd "$work"
cat > package.json << EOF
{ "type": "module", "dependencies": { "@synnaxlabs/$package": "file:$package.tgz" } }
EOF
{
    echo "nodeLinker: hoisted"
    echo "trustPolicy: no-downgrade"
    echo "fetchRetries: 5"
    if [ -n "$deps" ]; then
        echo "overrides:"
        for name in $deps; do echo "  \"@synnaxlabs/$name\": file:$work/$name.tgz"; done
    fi
} > pnpm-workspace.yaml
pnpm install --reporter=silent
