#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from pydantic import BaseModel, ConfigDict, Field, NonNegativeInt


class Envelope(BaseModel):
    """The portable format for a single importable/exportable resource.

    On the wire, an envelope is a flat JSON object with three promoted fields — version,
    type, and name — and any number of additional schema-specific fields at the same top
    level::

        {"version": 1, "type": "log", "name": "...", "channels": [...]}

    Pydantic's extra="allow" carries arbitrary additional fields through; they are
    accessible via Envelope.model_extra and round-trip cleanly through
    Envelope.model_dump_json().
    """

    model_config = ConfigDict(extra="allow")

    version: NonNegativeInt
    """Per-schema integer version stamped on every envelope."""
    type: str = Field(min_length=1)
    """Resource type identifier (e.g., 'log', 'schematic')."""
    name: str = Field(min_length=1)
    """Human-readable name of the resource."""
