#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

import re
from typing import override

from pydantic import BaseModel, Field, StrictInt, model_serializer, model_validator


class Color(BaseModel):
    """An RGBA color with every channel an integer from 0 to 255. A color serializes
    to an 8 digit "#rrggbbaa" hex string."""

    r: int = Field(default=0, ge=0, le=255)
    g: int = Field(default=0, ge=0, le=255)
    b: int = Field(default=0, ge=0, le=255)
    a: StrictInt = Field(default=255, ge=0, le=255)

    def __init__(self, __value: object = None, /, **kwargs: object):
        if __value is not None:
            validated = type(self).model_validate(__value)
            super().__init__(r=validated.r, g=validated.g, b=validated.b, a=validated.a)
        else:
            super().__init__(**kwargs)

    @model_validator(mode="before")
    @classmethod
    def _parse(cls, v: object) -> object:
        return _coerce(v)

    @model_serializer
    def _serialize(self) -> str:
        return f"#{self.r:02x}{self.g:02x}{self.b:02x}{self.a:02x}"

    def hex(self) -> str:
        """Return the hex string of the color, without alpha digits when the color is
        opaque."""
        if self.a == 255:
            return f"#{self.r:02x}{self.g:02x}{self.b:02x}"
        return f"#{self.r:02x}{self.g:02x}{self.b:02x}{self.a:02x}"

    @override
    def __eq__(self, other: object) -> bool:
        if isinstance(other, (str, list, tuple)):
            try:
                other = Color(other)
            except (ValueError, TypeError):
                return NotImplemented
        return super().__eq__(other)

    @property
    def is_zero(self) -> bool:
        return self.r == 0 and self.g == 0 and self.b == 0 and self.a == 0


def _coerce(v: object) -> object:
    if isinstance(v, Color):
        return v
    if isinstance(v, str):
        if v.startswith("rgb"):
            return _from_rgb(v)
        return _from_hex(v)
    if isinstance(v, (list, tuple)):
        return _from_tuple(v)
    if isinstance(v, dict):
        rgba255 = v.get("rgba255")
        if isinstance(rgba255, (list, tuple)):
            return _from_tuple(rgba255)
        if len(v) == 0:
            return {"r": 0, "g": 0, "b": 0, "a": 0}
        return v
    raise ValueError(f"Cannot parse color from: {v!r}")


def _from_tuple(v: list[object] | tuple[object, ...]) -> dict[str, int]:
    """Parse a legacy [r, g, b] or [r, g, b, a] tuple, whose alpha is on the 0-1
    scale."""
    if len(v) not in (3, 4):
        raise ValueError(f"Invalid color array length: {len(v)}")
    c = {
        "r": _parse_channel(v[0]),
        "g": _parse_channel(v[1]),
        "b": _parse_channel(v[2]),
        "a": 255,
    }
    if len(v) == 4:
        a = v[3]
        if isinstance(a, bool) or not isinstance(a, (int, float)):
            raise ValueError(f"Invalid color alpha: {a!r}")
        c["a"] = _legacy_alpha(float(a))
    return c


def _parse_channel(v: object) -> int:
    """Parse a single RGB channel. A fractional value is rejected rather than
    truncated. The Color field bounds enforce the 0-255 range."""
    if isinstance(v, bool) or not isinstance(v, (int, float)):
        raise ValueError(f"Invalid color channel: {v!r}")
    if isinstance(v, float) and not v.is_integer():
        raise ValueError(f"Color channel {v} is not a whole number")
    return int(v)


def _legacy_alpha(a: float) -> int:
    """Convert a legacy alpha, written on the 0-1 scale, to a byte. Old Consoles wrote
    a 0-255 alpha into the same slot, so a value above 2 is taken as a byte. A value in
    (1, 2] is a 0-1 alpha nudged past 1 by float error and means opaque. Mirrors the
    rule in the Go color decoders."""
    if not 0 <= a <= 255:
        raise ValueError(f"alpha {a} is outside the 0-255 range")
    if a <= 1:
        return round(a * 255)
    if a <= 2:
        return 255
    return round(a)


def _from_hex(s: str) -> dict[str, int]:
    s = s.lstrip("#")
    if len(s) not in (6, 8):
        raise ValueError(f"Invalid hex color: #{s}")
    try:
        channels = [int(s[i : i + 2], 16) for i in range(0, len(s), 2)]
    except ValueError as e:
        raise ValueError(f"Invalid hex color: #{s}") from e
    a = channels[3] if len(channels) == 4 else 255
    return {"r": channels[0], "g": channels[1], "b": channels[2], "a": a}


def _from_rgb(s: str) -> dict[str, int]:
    vals = re.findall(r"[\d.]+", s)
    if len(vals) < 3:
        raise ValueError(f"Invalid rgb color: {s}")
    r, g, b = int(float(vals[0])), int(float(vals[1])), int(float(vals[2]))
    a = float(vals[3]) if len(vals) >= 4 else 1.0
    if not 0 <= a <= 1:
        raise ValueError(f"rgba alpha must be 0-1: {s}")
    return {"r": r, "g": g, "b": b, "a": round(a * 255)}


Crude = Color | str | list[int] | tuple[int, ...]
"""Types that can be coerced into a Color."""
