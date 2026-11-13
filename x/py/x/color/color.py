#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

import math
import re
from typing import override

from pydantic import (
    BaseModel,
    ConfigDict,
    Field,
    StrictInt,
    model_serializer,
    model_validator,
)


class Color(BaseModel):
    """An RGBA color with every channel an integer from 0 to 255. A color serializes
    to its hex string."""

    model_config = ConfigDict(extra="forbid")

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
        if isinstance(v, str):
            return _from_string(v)
        if isinstance(v, (list, tuple)):
            return _from_tuple(v)
        return v

    @model_serializer
    def _serialize(self) -> str:
        return self.hex()

    def hex(self) -> str:
        """Return "#rrggbb" for an opaque color and "#rrggbbaa" otherwise."""
        rgb = f"#{self.r:02x}{self.g:02x}{self.b:02x}"
        return rgb if self.a == 255 else f"{rgb}{self.a:02x}"

    @override
    def __eq__(self, other: object) -> bool:
        if isinstance(other, (str, list, tuple)):
            try:
                other = Color(other)
            except (ValueError, TypeError):
                return NotImplemented
        return super().__eq__(other)


Crude = Color | str | list[int] | tuple[int, ...]
"""Types that can be coerced into a Color."""


def _from_tuple(v: list[object] | tuple[object, ...]) -> dict[str, int]:
    """Parse an [r, g, b] or [r, g, b, a] tuple of bytes. The alpha of a 3-tuple is
    255."""
    if len(v) not in (3, 4):
        raise ValueError(f"Invalid color array length: {len(v)}")
    channels = [_parse_channel(c) for c in v]
    return {
        "r": channels[0],
        "g": channels[1],
        "b": channels[2],
        "a": channels[3] if len(channels) == 4 else 255,
    }


def _parse_channel(v: object) -> int:
    if isinstance(v, bool) or not isinstance(v, (int, float)):
        raise ValueError(f"Invalid color channel: {v!r}")
    if isinstance(v, float) and not v.is_integer():
        raise ValueError(f"Color channel {v} is not a whole number")
    return int(v)


def _round(v: float) -> int:
    """Round a non-negative value half away from zero, as Go, TypeScript and C++ do."""
    return math.floor(v + 0.5)


_HEX = re.compile(r"#?([0-9a-f]{3,8})")
_FUNCTION = re.compile(r"(rgba?|hsla?)\((.*)\)")
_NUMBER = re.compile(r"[+-]?(\d+\.?\d*|\.\d+)")


def _from_string(s: str) -> dict[str, int]:
    t = s.strip().lower()
    if m := _HEX.fullmatch(t):
        return _from_hex(m.group(1), s)
    if m := _FUNCTION.fullmatch(t):
        return _from_function(m.group(1), m.group(2), s)
    raise ValueError(f"Invalid color: {s!r}")


def _from_hex(digits: str, s: str) -> dict[str, int]:
    if len(digits) in (3, 4):
        digits = "".join(d + d for d in digits)
    if len(digits) not in (6, 8):
        raise ValueError(f"Invalid hex color: {s!r}")
    b = bytes.fromhex(digits)
    return {"r": b[0], "g": b[1], "b": b[2], "a": b[3] if len(b) == 4 else 255}


def _split_args(body: str, s: str) -> tuple[list[str], str | None]:
    """Split the arguments of a color function into its three channels and an
    optional alpha, in comma syntax or space syntax with a slash."""
    if "," in body:
        args = [a.strip() for a in body.split(",")]
        if len(args) not in (3, 4) or "/" in body:
            raise ValueError(f"Invalid color: {s!r}")
        return args[:3], args[3] if len(args) == 4 else None
    main, slash, alpha = body.partition("/")
    args = main.split()
    if len(args) != 3 or (slash and not alpha.strip()):
        raise ValueError(f"Invalid color: {s!r}")
    return args, alpha.strip() if slash else None


def _number(arg: str, s: str, unit: str = "") -> float:
    if unit and arg.endswith(unit):
        arg = arg[: -len(unit)]
    if not _NUMBER.fullmatch(arg):
        raise ValueError(f"Invalid color: {s!r}")
    return float(arg)


def _bounded(arg: str, scale: float, s: str) -> float:
    """Parse a number in [0, scale], or a percentage of scale."""
    n = _number(arg, s, "%")
    v = n * scale / 100 if arg.endswith("%") else n
    if not 0 <= v <= scale:
        raise ValueError(f"Color value {arg} is out of range: {s!r}")
    return v


def _from_function(name: str, body: str, s: str) -> dict[str, int]:
    channels, alpha = _split_args(body, s)
    a = 255 if alpha is None else _round(_bounded(alpha, 1, s) * 255)
    if name.startswith("rgb"):
        r, g, b = (_round(_bounded(c, 255, s)) for c in channels)
        return {"r": r, "g": g, "b": b, "a": a}
    h = _number(channels[0], s, "deg") % 360
    sat = _number(channels[1], s, "%")
    light = _number(channels[2], s, "%")
    if not (0 <= sat <= 100 and 0 <= light <= 100):
        raise ValueError(f"Color value is out of range: {s!r}")
    r, g, b = _hsl_to_rgb(h, sat / 100, light / 100)
    return {"r": r, "g": g, "b": b, "a": a}


def _hsl_to_rgb(h: float, sat: float, light: float) -> tuple[int, int, int]:
    c = (1 - abs(2 * light - 1)) * sat
    x = c * (1 - abs((h / 60) % 2 - 1))
    m = light - c / 2
    sector = int(h // 60) % 6
    r, g, b = [(c, x, 0), (x, c, 0), (0, c, x), (0, x, c), (x, 0, c), (c, 0, x)][sector]
    return _round((r + m) * 255), _round((g + m) * 255), _round((b + m) * 255)
