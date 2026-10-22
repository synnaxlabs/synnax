#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Writes <name>.vectors.json for each <name>.dbc: codec vectors from cantools.

Run from this directory with:

    uv run --no-project --with cantools python generate.py [directory]

The directory defaults to dbc/. Each vector holds a random payload, the physical values
cantools decodes from it, and the payload cantools encodes from those values onto a
zeroed frame. Messages that cantools cannot decode or encode are listed under "skipped"
with the reason, so that a corpus run reports them instead of stopping.
"""

import json
import math
import random
import sys
from pathlib import Path

import cantools

SEED = 4971
# Vectors per file, split across its messages, bounded per message.
VECTORS_PER_FILE = 1024
MIN_VECTORS = 4
MAX_VECTORS = 64


def bit_positions(signal) -> list[int]:
    """Returns the payload bit of each signal bit, least significant first."""
    if signal.byte_order == "little_endian":
        return [signal.start + i for i in range(signal.length)]
    msb_first = []
    pos = signal.start
    for _ in range(signal.length):
        msb_first.append(pos)
        pos = pos + 15 if pos % 8 == 0 else pos - 1
    return msb_first[::-1]


def set_bits(payload: bytearray, signal, value: int) -> None:
    for i, bit in enumerate(bit_positions(signal)):
        mask = 1 << (bit % 8)
        if (value >> i) & 1:
            payload[bit // 8] |= mask
        else:
            payload[bit // 8] &= ~mask


def multiplexor_values(message) -> dict:
    """Returns, for each multiplexor, the values that select at least one signal."""
    values: dict = {}
    for s in message.signals:
        if s.multiplexer_signal is not None:
            values.setdefault(s.multiplexer_signal, set()).update(s.multiplexer_ids)
    return {name: sorted(v) for name, v in values.items()}


def random_payload(rng: random.Random, message, choices: dict) -> bytes:
    payload = bytearray(rng.getrandbits(8) for _ in range(message.length))
    for signal in message.signals:
        if signal.name in choices:
            raw = rng.choice(choices[signal.name])
            set_bits(payload, signal, raw & ((1 << signal.length) - 1))
    return bytes(payload)


def finite(values: dict) -> bool:
    return all(not isinstance(v, float) or math.isfinite(v) for v in values.values())


def vectors(rng: random.Random, message, count: int) -> list:
    choices = multiplexor_values(message)
    out = []
    attempts = 0
    while len(out) < count:
        attempts += 1
        if attempts > count * 50:
            raise ValueError("no finite values after many random payloads")
        payload = random_payload(rng, message, choices)
        values = message.decode(payload, decode_choices=False, scaling=True)
        if not finite(values):
            continue
        encoded = message.encode(values, strict=False)
        out.append(
            {"payload": payload.hex(), "values": values, "encoded": encoded.hex()}
        )
    return out


def generate(path: Path) -> None:
    db = cantools.database.load_file(path, strict=False)
    rng = random.Random(SEED)
    messages = sorted(db.messages, key=lambda m: (m.frame_id, m.name))
    count = max(MIN_VECTORS, min(MAX_VECTORS, VECTORS_PER_FILE // len(messages)))
    out = {"messages": [], "skipped": {}}
    for m in messages:
        try:
            out["messages"].append({"name": m.name, "vectors": vectors(rng, m, count)})
        except Exception as e:  # noqa: BLE001 - every cantools failure is reported.
            out["skipped"][m.name] = f"{type(e).__name__}: {e}"
    text = json.dumps(out, indent=1)
    path.with_suffix(".vectors.json").write_text(text + "\n")


def main() -> None:
    here = Path(__file__).parent
    directory = Path(sys.argv[1]) if len(sys.argv) > 1 else here / "dbc"
    for path in sorted(directory.glob("*.dbc")):
        generate(path)


if __name__ == "__main__":
    main()
