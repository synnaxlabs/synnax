#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Writes cobs.json: COBS vectors encoded by the Python cobs package.

Run from this directory with:

    uv run --no-project --with cobs python generate.py

Each vector holds a frame and its wire bytes: the cobs encoding followed by the zero
byte that ends the frame.
"""

import json
import random
from pathlib import Path

from cobs import cobs

SEED = 4971


def runs() -> list[bytes]:
    out = []
    for n in [1, 2, 253, 254, 255, 256, 507, 508, 509, 510, 1000]:
        run = bytes((i % 255) + 1 for i in range(n))
        out += [run, run + b"\x00", b"\x00" + run, run + b"\x00" + run]
    return out


def edges() -> list[bytes]:
    return [
        b"\x00",
        b"\x00\x00",
        b"\x00" * 300,
        b"\x11\x22\x00\x33",
        b"\x11\x22\x33\x44",
        b"\x11\x00\x00\x00",
        b"\x01" * 254 + b"\x00\x00",
        b"\xff" * 254,
    ]


def randoms(rng: random.Random) -> list[bytes]:
    out = []
    for i in range(48):
        size = rng.randint(1, 600 if i % 8 == 0 else 80)
        zeros = rng.choice([0.0, 0.01, 0.25, 0.75])
        out.append(
            bytes(
                0 if rng.random() < zeros else rng.randint(1, 255) for _ in range(size)
            )
        )
    return out


def main() -> None:
    frames = runs() + edges() + randoms(random.Random(SEED))
    vectors = [
        {"frame": f.hex(), "wire": (cobs.encode(f) + b"\x00").hex()} for f in frames
    ]
    out = Path(__file__).parent / "cobs.json"
    out.write_text(json.dumps({"vectors": vectors}, indent=1) + "\n")


if __name__ == "__main__":
    main()
