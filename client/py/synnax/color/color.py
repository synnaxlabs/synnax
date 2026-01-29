#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from typing import Any, TypeAlias

from pydantic import model_validator

from synnax.color.types_gen import Payload


class Color(Payload):
    @model_validator(mode="before")
    @classmethod
    def _parse_input(cls, data: Any) -> dict[str, Any]:
        if isinstance(data, dict) and "r" in data:
            return data
        if isinstance(data, str):
            r, g, b, a = cls._from_hex(data)
            return {"r": r, "g": g, "b": b, "a": a}
        if isinstance(data, (list, tuple)):
            if len(data) == 4:
                return {
                    "r": int(data[0]),
                    "g": int(data[1]),
                    "b": int(data[2]),
                    "a": float(data[3]),
                }
            if len(data) == 3:
                return {
                    "r": int(data[0]),
                    "g": int(data[1]),
                    "b": int(data[2]),
                    "a": 1.0,
                }
        raise ValueError(
            "Color must be hex string, [R,G,B,A] array, [R,G,B] array, or {r,g,b,a} dict"
        )

    def hex(self, include_alpha: bool = True) -> str:
        alpha_byte = int(self.a * 255)
        if include_alpha or alpha_byte != 255:
            return f"#{self.r:02x}{self.g:02x}{self.b:02x}{alpha_byte:02x}"
        return f"#{self.r:02x}{self.g:02x}{self.b:02x}"

    def is_zero(self) -> bool:
        return self.r == 0 and self.g == 0 and self.b == 0 and self.a == 0

    @staticmethod
    def _from_hex(hex_str: str) -> tuple[int, int, int, float]:
        hex_str = hex_str.lstrip("#")
        if len(hex_str) == 0:
            return (0, 0, 0, 0.0)
        if len(hex_str) not in (6, 8):
            raise ValueError("Hex color must be 6 or 8 hex digits")
        alpha = int(hex_str[6:8], 16) / 255.0 if len(hex_str) == 8 else 1.0
        return (
            int(hex_str[0:2], 16),
            int(hex_str[2:4], 16),
            int(hex_str[4:6], 16),
            alpha,
        )

    def __repr__(self) -> str:
        return f"Color({self.hex()})"

    def __eq__(self, other: object) -> bool:
        other_color = Color.model_validate(other)
        return (
            other_color.r == self.r
            and other_color.g == self.g
            and other_color.b == self.b
            and abs(self.a - other_color.a) < 0.004
        )

    def __hash__(self) -> int:
        return hash((self.r, self.g, self.b, round(self.a, 2)))


# Crude color representations - defined after Color class to avoid forward reference issues
Crude: TypeAlias = (
    str
    | tuple[int, int, int, float]
    | tuple[int, int, int]
    | dict[str, int | float]
    | Color
)
"""
An unparsed representation of a color that can be converted into a Color object.
Supports:
- Hex strings: '#ff0000', '#ff0000ff'
- RGBA tuples: (255, 0, 0, 1.0)
- RGB tuples: (255, 0, 0) - alpha defaults to 1.0
- Dicts: {'r': 255, 'g': 0, 'b': 0, 'a': 1.0}
- Color objects
"""


def is_crude(value: Any) -> bool:
    """Check if a value can be parsed into a valid Color."""
    try:
        Color.model_validate(value)
        return True
    except (ValueError, TypeError):
        return False


def construct(color: Crude) -> Color:
    """Construct a Color from a crude representation."""
    return Color.model_validate(color)
