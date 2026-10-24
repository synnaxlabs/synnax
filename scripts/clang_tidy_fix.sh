#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Applies the fixes clang-tidy suggests for the C++ files in the given Bazel targets,
# then formats the changed code. Findings without a fix still need a hand edit, so run
# bazel build --config=clang-tidy <targets> afterwards.

set -euo pipefail

if [ "$#" -eq 0 ]; then
    echo "Usage: $0 <bazel targets...>"
    exit 1
fi

root=$(git rev-parse --show-toplevel)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

status=0
bazel build --config=clang-tidy-fix --keep_going \
    --build_event_json_file="$tmp/events.json" //:clang_apply_replacements "$@" \
    || status=$?
replacements=$(bazel info output_base)/$(bazel cquery --config=clang-tidy-fix \
    --output_groups=default --output=files //:clang_apply_replacements 2> /dev/null)

# Each action exports its fixes against the sandbox it ran in. Collect the files this
# build produced and point their paths back at the checkout. A template reports one
# fix per instantiation, each with its own message, and clang-apply-replacements
# applies every copy within one file, so keep one of each.
mkdir "$tmp/fixes"
python3 - "$tmp/events.json" "$tmp/fixes" "$root" << 'PY'
import json
import pathlib
import re
import sys
from urllib.parse import unquote, urlparse

events, out, root = sys.argv[1], pathlib.Path(sys.argv[2]), sys.argv[3]
sandbox = re.compile(r"[^'\"\s]*/execroot/_main\b")
relative = re.compile(r"((?:FilePath|MainSourceFile):\s+')(?:\./)?(?=[^/'])")
diagnostic = re.compile(r"(?m)^(?=  - DiagnosticName:)")
message = re.compile(r"(?m)^ +Message: .*\n")
count = 0
for line in open(events):
    files = json.loads(line).get("namedSetOfFiles", {}).get("files", [])
    for file in files:
        path = pathlib.Path(unquote(urlparse(file["uri"]).path))
        if not path.name.endswith(".clang-tidy.yaml") or path.stat().st_size == 0:
            continue
        text = relative.sub(rf"\g<1>{root}/", sandbox.sub(root, path.read_text()))
        head, _, body = text.partition("Diagnostics:\n")
        body = body.removesuffix("...\n")
        blocks = {}
        for block in diagnostic.split(body):
            blocks.setdefault(message.sub("", block), block)
        unique = "".join(blocks.values())
        (out / f"{count}.yaml").write_text(f"{head}Diagnostics:\n{unique}...\n")
        count += 1
print(f"Applying fixes from {count} files.")
PY

# Two checks can insert at one spot, such as [[nodiscard]] and static. Apply both, and
# let the next lint run catch any result that does not compile.
"$replacements" --ignore-insert-conflict --format --style=file "$tmp/fixes"
exit "$status"
