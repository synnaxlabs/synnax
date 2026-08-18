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

from pydantic import BaseModel, Field, model_validator


class Color(BaseModel):
    """An RGBA color with 8-bit RGB channels and a float alpha."""

    r: int = Field(default=0, ge=0, le=255)
    g: int = Field(default=0, ge=0, le=255)
    b: int = Field(default=0, ge=0, le=255)
    a: float = 1

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

    def hex(self) -> str:
        """Return the hex string representation of the color."""
        alpha_byte = round(self.a * 255)
        if alpha_byte == 255:
            return f"#{self.r:02x}{self.g:02x}{self.b:02x}"
        return f"#{self.r:02x}{self.g:02x}{self.b:02x}{alpha_byte:02x}"

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
        if len(v) == 3:
            return {
                "r": _parse_channel(v[0]),
                "g": _parse_channel(v[1]),
                "b": _parse_channel(v[2]),
                "a": 1.0,
            }
        if len(v) == 4:
            return {
                "r": _parse_channel(v[0]),
                "g": _parse_channel(v[1]),
                "b": _parse_channel(v[2]),
                "a": _normalize_alpha(float(v[3])),
            }
        raise ValueError(f"Invalid color array length: {len(v)}")
    if isinstance(v, dict):
        rgba255 = v.get("rgba255")
        if isinstance(rgba255, (list, tuple)):
            return _coerce(list(rgba255))
        a = v.get("a")
        if isinstance(a, (int, float)) and a > 1:
            return {**v, "a": _normalize_alpha(float(a))}
        return v
    raise ValueError(f"Cannot parse color from: {v!r}")


def _parse_channel(v: object) -> int:
    """Parse a single RGB channel. A fractional value is rejected rather than
    truncated. The Color field bounds enforce the 0-255 range."""
    if isinstance(v, bool) or not isinstance(v, (int, float)):
        raise ValueError(f"Invalid color channel: {v!r}")
    if isinstance(v, float) and not v.is_integer():
        raise ValueError(f"Color channel {v} is not a whole number")
    return int(v)


def _normalize_alpha(a: float) -> float:
    """Lift a legacy 0-255 alpha onto the current 0-1 scale. Old Consoles persisted
    alpha as a fourth 0-255 channel; the current format caps alpha at 1, so any larger
    value is legacy. Values in (1, 2] clamp to 1 instead of dividing: a legacy alpha
    that small means a sub-1% opacity no user sets, while a current-scale value nudged
    past 1 by float error means opaque. An alpha above 255 fits neither scale and
    fails validation. Mirrors the rule in the Go color decoders."""
    if a <= 1:
        return a
    if a <= 2:
        return 1.0
    if a > 255:
        raise ValueError(f"alpha {a} is above the 0-255 scale")
    return a / 255


def _from_hex(s: str) -> dict[str, int | float]:
    s = s.lstrip("#")
    if len(s) == 0:
        return {"r": 0, "g": 0, "b": 0, "a": 0}
    if len(s) == 6:
        return {
            "r": int(s[0:2], 16),
            "g": int(s[2:4], 16),
            "b": int(s[4:6], 16),
            "a": 1.0,
        }
    if len(s) == 8:
        return {
            "r": int(s[0:2], 16),
            "g": int(s[2:4], 16),
            "b": int(s[4:6], 16),
            "a": int(s[6:8], 16) / 255.0,
        }
    raise ValueError(f"Invalid hex color: #{s}")


def _from_rgb(s: str) -> dict[str, int | float]:
    vals = re.findall(r"[\d.]+", s)
    if len(vals) < 3:
        raise ValueError(f"Invalid rgb color: {s}")
    r, g, b = int(float(vals[0])), int(float(vals[1])), int(float(vals[2]))
    a = float(vals[3]) if len(vals) >= 4 else 1.0
    return {"r": r, "g": g, "b": b, "a": a}


Crude = Color | str | list[int] | tuple[int, ...]
"""Types that can be coerced into a Color."""
