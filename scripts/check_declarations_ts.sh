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
#        check_declarations_ts.sh --window
#
# Type-checks the declaration files of each entry point of @synnaxlabs/<package>,
# installed in <directory> by pack_ts.sh, as a strict consumer with skipLibCheck off
# sees them. It checks with the bundler and nodenext module resolutions, each with the
# oldest supported TypeScript and with the repo's TypeScript. With --window, it only
# fails if that oldest TypeScript is more than two years old.

set -euo pipefail

# The oldest TypeScript less than two years old. Move up a minor when it turns two.
oldest=5.8.3

if [ "${1:-}" = --window ]; then
    # The window starts at the first stable release of the minor.
    stale=$(curl -fsS https://registry.npmjs.org/typescript | node -p '
const minor = process.argv[1].replace(/\.\d+$/, ".");
const time = JSON.parse(require("fs").readFileSync(0)).time;
const stable = Object.keys(time).filter((v) => /^\d+\.\d+\.\d+$/.test(v));
const end = new Date(time[stable.find((v) => v.startsWith(minor))]);
end.setUTCFullYear(end.getUTCFullYear() + 2);
end < new Date();
' "$oldest")
    if [ "$stale" = true ]; then
        echo "TypeScript $oldest is past the two-year support window:" \
            'move `oldest` to the next minor' >&2
        exit 1
    fi
    exit
fi

package=$1
work=$(cd "$2" && pwd)
repo=$(cd "$(dirname "$0")/.." && pwd)

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
