#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from pydantic import BaseModel, Field, model_validator

from x import spatial

_CORNERS = ("top_left", "top_right", "bottom_left", "bottom_right")


def _xy(v: object) -> object:
    if isinstance(v, int | float) and not isinstance(v, bool):
        return spatial.XY(x=v, y=v)
    return v


class Radius(BaseModel):
    """A per-corner border radius with independent x and y components, expressed as
    percentages of the element's dimensions. Validation also accepts shorthand forms:
    a bare number, a single { x, y } pair, or per-corner numbers. Each expands to the
    canonical shape, and a missing corner is zero.
    """

    top_left: spatial.XY = Field(default_factory=lambda: spatial.XY(x=0, y=0))
    top_right: spatial.XY = Field(default_factory=lambda: spatial.XY(x=0, y=0))
    bottom_left: spatial.XY = Field(default_factory=lambda: spatial.XY(x=0, y=0))
    bottom_right: spatial.XY = Field(default_factory=lambda: spatial.XY(x=0, y=0))

    @model_validator(mode="before")
    @classmethod
    def _expand(cls, v: object) -> object:
        if isinstance(v, dict) and "x" in v:
            return dict.fromkeys(_CORNERS, v)
        if isinstance(v, dict):
            return {k: _xy(c) for k, c in v.items()}
        xy = _xy(v)
        if xy is v:
            return v
        return dict.fromkeys(_CORNERS, xy)
