#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Writes <name>.expected.json for each DBC fixture, decoded by cantools.

Run from this directory with `uv run --with cantools python generate.py`.
"""

import json
import pathlib

import cantools


def signal_of(s: cantools.database.Signal) -> dict:
    return {
        "name": s.name,
        "start_bit": s.start,
        "bit_length": s.length,
        "byte_order": s.byte_order,
        # The library has no signed float: a float's sign lives in its IEEE bits.
        "signed": s.is_signed and not s.is_float,
        "float": s.is_float,
        "scale": float(s.scale),
        "offset": float(s.offset),
        "units": s.unit or "",
        "multiplexor": s.multiplexer_signal,
        "multiplex_values": sorted(s.multiplexer_ids or []),
        "min": s.minimum,
        "max": s.maximum,
    }


def message_of(m: cantools.database.Message) -> dict:
    return {
        "name": m.name,
        "id": m.frame_id,
        "extended": m.is_extended_frame,
        "fd": m.is_fd,
        "length": m.length,
        "fields": [signal_of(s) for s in m.signals],
    }


def main() -> None:
    here = pathlib.Path(__file__).parent
    for path in sorted(here.glob("*.dbc")):
        db = cantools.database.load_file(path)
        expected = {"messages": [message_of(m) for m in db.messages]}
        out = path.with_suffix(".expected.json")
        out.write_text(json.dumps(expected, indent=2) + "\n")


if __name__ == "__main__":
    main()
