#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from typing import Any

from pydantic import BaseModel, ConfigDict, NonNegativeInt, field_validator


class Envelope(BaseModel):
    """The portable format for a single importable/exportable resource.

    On the wire, an envelope is a flat JSON object with three promoted fields —
    version, type, and name — and any number of additional schema-specific
    fields at the same top level::

        {"version": 1, "type": "log", "name": "...", "channels": [...]}

    Pydantic's extra="allow" carries arbitrary additional fields through; they
    are accessible via Envelope.model_extra and round-trip cleanly through
    Envelope.model_dump_json().
    """

    model_config = ConfigDict(extra="allow")

    version: NonNegativeInt
    """Per-schema integer version stamped on every envelope."""
    type: str
    """Resource type identifier (e.g., 'log', 'schematic')."""
    name: str
    """Human-readable name of the resource."""

    @field_validator("version", mode="before")
    @classmethod
    def _coerce_version(cls, v: Any) -> Any:
        """Accept the legacy ``N.0.0`` semver string in addition to integers.

        Mirrors ``legacyToNumeric`` in ``core/pkg/service/imex/imex.go`` —
        older Console exports stamped the major component as a semver string.
        """
        if isinstance(v, str):
            parts = v.split(".")
            if len(parts) != 3:
                raise ValueError(f"invalid version {v!r}: expected N.0.0")
            major, minor, patch = parts
            if minor != "0" or patch != "0":
                raise ValueError(f"invalid version {v!r}: only N.0.0 is supported")
            return int(major)
        return v
