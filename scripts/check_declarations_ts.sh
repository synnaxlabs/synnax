#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Usage: check_declarations_ts.sh <package> <directory>
#
# Type-checks the declaration files of each entry point of @synnaxlabs/<package>,
# installed in <directory> by pack_ts.sh, as a strict consumer with skipLibCheck off
# sees them. It checks with the bundler and nodenext module resolutions, each with the
# oldest supported TypeScript and with the repo's TypeScript.

set -euo pipefail

package=$1
work=$(cd "$2" && pwd)
repo=$(cd "$(dirname "$0")/.." && pwd)

# The oldest TypeScript less than two years old. Move up a minor when it turns two.
oldest=5.7.3

manifest=$work/node_modules/@synnaxlabs/$package/package.json

# The testutil entry points need the optional peers, such as vitest.
read -ra peers < <(node -e '
const peers = require(process.argv[1]).peerDependencies ?? {};
console.log(Object.entries(peers).map(([name, range]) => name + "@" + range).join(" "));
' "$manifest")
if [ ${#peers[@]} -gt 0 ]; then
    pnpm --dir "$work" add --reporter=silent "${peers[@]}"
fi
pnpm --dir "$work" add --reporter=silent "typescript@$oldest"

# Wildcard exports such as ./dist/* name files, not entry points.
node -e '
const [manifest, name] = process.argv.slice(1);
Object.keys(require(manifest).exports)
  .filter((key) => !key.includes("*"))
  .forEach((key, i) =>
    console.log("export * as e" + i + " from " + JSON.stringify(name + key.slice(1))),
  );
' "$manifest" "@synnaxlabs/$package" > "$work/index.ts"
for resolution in bundler nodenext; do
    module=$([ "$resolution" = bundler ] && echo esnext || echo nodenext)
    cat > "$work/tsconfig.json" << EOF
{
  "compilerOptions": {
    "strict": true,
    "skipLibCheck": false,
    "noEmit": true,
    "target": "ES2024",
    "lib": ["ES2024", "DOM"],
    "module": "$module",
    "moduleResolution": "$resolution"
  },
  "files": ["index.ts"]
}
EOF
    echo "Checking @synnaxlabs/$package, $resolution, TypeScript $oldest"
    "$work/node_modules/.bin/tsc" --project "$work/tsconfig.json"
    echo "Checking @synnaxlabs/$package, $resolution, the repo's TypeScript"
    pnpm --dir "$repo" --filter "@synnaxlabs/$package" exec tsc \
        --project "$work/tsconfig.json"
done
