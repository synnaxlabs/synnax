#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from pydantic import BaseModel, model_validator


class Color(BaseModel):
    """An RGBA color with 8-bit RGB channels and a float alpha."""

    r: int = 0
    g: int = 0
    b: int = 0
    a: float = 0

    def __init__(self, __value: object = None, /, **kwargs: object):
        if __value is not None:
            validated = type(self).model_validate(__value)
            super().__init__(r=validated.r, g=validated.g, b=validated.b, a=validated.a)
        else:
            super().__init__(**kwargs)

    @model_validator(mode="before")
    @classmethod
    def _parse(cls, v: object) -> object:
        if isinstance(v, Color):
            return v
        if isinstance(v, str):
            return _from_hex(v)
        if isinstance(v, (list, tuple)):
            if len(v) == 3:
                return {"r": int(v[0]), "g": int(v[1]), "b": int(v[2]), "a": 1.0}
            if len(v) == 4:
                return {
                    "r": int(v[0]),
                    "g": int(v[1]),
                    "b": int(v[2]),
                    "a": float(v[3]),
                }
            raise ValueError(f"Invalid color array length: {len(v)}")
        if isinstance(v, dict):
            return v
        raise ValueError(f"Cannot parse color from: {v!r}")

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


def _from_hex(s: str) -> dict:
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


Crude = Color | str | list[int] | tuple[int, ...]
"""Types that can be coerced into a Color."""
