#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Writes vectors.json: golden codec vectors produced by cantools from vectors.dbc.

Run from this directory with:

    uv run --no-project --with cantools python generate.py

Each vector holds a random payload, the physical values cantools decodes from it, and
the payload cantools encodes from those values onto a zeroed frame.
"""

import json
import math
import random
from pathlib import Path

import cantools

VECTORS_PER_MESSAGE = 64
SEED = 4971

# Multiplexor values to force so that every multiplexed branch is exercised. cantools
# rejects values that select no branch, so the choices hold only known values.
MULTIPLEXOR_CHOICES = {
    "Mux": [0, 1, 2],
    "Mux1": [0, 1, 2],
    "Mux2": [0, 1, 3, 4, 5],
}


def set_bits(payload: bytearray, start: int, length: int, value: int) -> None:
    for i in range(length):
        bit = start + i
        mask = 1 << (bit % 8)
        if (value >> i) & 1:
            payload[bit // 8] |= mask
        else:
            payload[bit // 8] &= ~mask


def random_payload(rng: random.Random, message) -> bytes:
    payload = bytearray(rng.getrandbits(8) for _ in range(message.length))
    for signal in message.signals:
        choices = MULTIPLEXOR_CHOICES.get(signal.name)
        if choices is None:
            continue
        if signal.byte_order != "little_endian":
            raise ValueError("multiplexors in vectors.dbc must be little endian")
        set_bits(payload, signal.start, signal.length, rng.choice(choices))
    return bytes(payload)


def layout(message) -> dict:
    return {
        "name": message.name,
        "id": message.frame_id,
        "extended": message.is_extended_frame,
        "length": message.length,
        "signals": [
            {
                "name": s.name,
                "start_bit": s.start,
                "bit_length": s.length,
                "byte_order": s.byte_order,
                "signed": s.is_signed,
                "float": s.is_float,
                "scale": s.scale,
                "offset": s.offset,
                "multiplexor": s.multiplexer_signal,
                "multiplex_values": s.multiplexer_ids or [],
            }
            for s in message.signals
        ],
    }


def finite(values: dict) -> bool:
    return all(not isinstance(v, float) or math.isfinite(v) for v in values.values())


def vectors(rng: random.Random, message) -> list:
    out = []
    while len(out) < VECTORS_PER_MESSAGE:
        payload = random_payload(rng, message)
        values = message.decode(payload, decode_choices=False)
        if not finite(values):
            continue
        encoded = message.encode(values, strict=False)
        out.append(
            {"payload": payload.hex(), "values": values, "encoded": encoded.hex()}
        )
    return out


def main() -> None:
    here = Path(__file__).parent
    db = cantools.database.load_file(here / "vectors.dbc")
    rng = random.Random(SEED)
    messages = [
        {**layout(m), "vectors": vectors(rng, m)}
        for m in sorted(db.messages, key=lambda m: m.frame_id)
    ]
    text = json.dumps({"messages": messages}, indent=1)
    (here / "vectors.json").write_text(text + "\n")


if __name__ == "__main__":
    main()
