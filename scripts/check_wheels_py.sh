#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Usage: check_wheels_py.sh <directory> <python>...
#
# Builds the Python packages into <directory>/dist as the release does. For each
# <python>, it installs the wheels in a new virtual environment in <directory>, with no
# workspace on the path. There it imports every module, runs `sy --help`, and runs mypy
# in strict mode on a script that uses the public API of the installed synnax.

set -euo pipefail

work=$(mkdir -p "$1" && cd "$1" && pwd)
shift
repo=$(cd "$(dirname "$0")/.." && pwd)
mypy=$(sed -n '/^name = "mypy"$/{n;s/^version = "\(.*\)"$/\1/p;}' "$repo/uv.lock")

rm -rf "$work/dist"
"$repo/.github/scripts/build_py.sh" "$work/dist" > /dev/null

cat > "$work/check.py" << 'EOF'
import numpy as np

import synnax as sy


def read_write(client: sy.Synnax) -> float:
    index = client.channels.create(
        name="time", is_index=True, data_type=sy.DataType.TIMESTAMP
    )
    data = client.channels.create(
        name="data", index=index.key, data_type=sy.DataType.FLOAT32
    )
    start = sy.TimeStamp.now()
    with client.open_writer(start, [index.key, data.key]) as writer:
        writer.write({index.key: [start], data.key: np.array([1.0], np.float32)})
    with client.open_streamer(data.key) as streamer:
        frame = streamer.read(timeout=sy.TimeSpan.SECOND)
        if frame is None:
            return 0.0
    series = data.read(sy.TimeRange(start, sy.TimeStamp.now()))
    return float(series.to_numpy().mean())
EOF

# uv reads its settings from the working directory, so it runs away from the workspace.
cd "$work"
for python in "$@"; do
    echo "Checking the wheels on Python $python"
    venv=$work/venv-$python
    rm -rf "$venv"
    uv venv --quiet --python "$python" "$venv"
    uv pip install --quiet --python "$venv" "$work"/dist/*.whl "alamos[dev]" \
        "mypy==$mypy"
    # The filter makes the lazy imports of Python 3.15 eager, so that a missing module
    # fails at its import.
    "$venv/bin/python" -I -c '
import importlib, pkgutil, sys
if hasattr(sys, "set_lazy_imports_filter"):
    sys.set_lazy_imports_filter(lambda *_: False)
for name in ("alamos", "freighter", "synnax", "x"):
    package = importlib.import_module(name)
    if not package.__file__.startswith(sys.prefix):
        sys.exit(f"{name} loads from {package.__file__}, not the wheel")
    for module in pkgutil.walk_packages(package.__path__, name + "."):
        importlib.import_module(module.name)
'
    "$venv/bin/sy" --help > /dev/null
    "$venv/bin/python" -I -m mypy --strict --no-incremental check.py
done
