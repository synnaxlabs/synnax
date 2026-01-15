#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from typing import Any, List, Union

from pydantic_core import core_schema

# Type alias for the 4-element RGBA tuple
RGBA = tuple[int, int, int, int]


class Color(tuple[int, int, int, int]):
    """RGBA color as 4 bytes (0-255 each).

    Can be constructed from:
    - Hex string: "#RRGGBB" or "#RRGGBBAA"
    - RGBA tuple/list: [R, G, B, A] or (R, G, B, A)
    - 4 bytes: bytes([R, G, B, A])
    - Another Color instance
    """

    def __new__(
        cls, value: Union[bytes, list[int], tuple[int, ...], str, "Color"]
    ) -> "Color":
        if isinstance(value, cls):
            return value
        if isinstance(value, str):
            rgba = cls._from_hex(value)
        elif isinstance(value, bytes) and len(value) == 4:
            rgba = (value[0], value[1], value[2], value[3])
        elif isinstance(value, (list, tuple)) and len(value) == 4:
            rgba = (int(value[0]), int(value[1]), int(value[2]), int(value[3]))
        else:
            raise ValueError("Color must be hex string, [R,G,B,A], or 4 bytes")
        return super().__new__(cls, rgba)

    @property
    def r(self) -> int:
        """Red component (0-255)."""
        return self[0]

    @property
    def g(self) -> int:
        """Green component (0-255)."""
        return self[1]

    @property
    def b(self) -> int:
        """Blue component (0-255)."""
        return self[2]

    @property
    def a(self) -> int:
        """Alpha component (0-255)."""
        return self[3]

    def hex(self, include_alpha: bool = True) -> str:
        """Convert to hex string.

        Args:
            include_alpha: If True, always includes alpha. If False, omits alpha
                when it's 255 (fully opaque).

        Returns:
            Hex string in format "#RRGGBB" or "#RRGGBBAA".
        """
        if include_alpha or self[3] != 255:
            return f"#{self[0]:02x}{self[1]:02x}{self[2]:02x}{self[3]:02x}"
        return f"#{self[0]:02x}{self[1]:02x}{self[2]:02x}"

    def is_zero(self) -> bool:
        """Return True if all components are 0."""
        return all(c == 0 for c in self)

    @staticmethod
    def _from_hex(hex_str: str) -> tuple[int, int, int, int]:
        """Parse hex string to RGBA tuple."""
        hex_str = hex_str.lstrip("#")
        if len(hex_str) == 6:
            hex_str += "ff"
        if len(hex_str) != 8:
            raise ValueError("Hex color must be 6 or 8 hex digits")
        return (
            int(hex_str[0:2], 16),
            int(hex_str[2:4], 16),
            int(hex_str[4:6], 16),
            int(hex_str[6:8], 16),
        )

    @classmethod
    def __get_pydantic_core_schema__(
        cls, source_type: Any, handler: Any
    ) -> core_schema.CoreSchema:
        """Pydantic v2 schema for validation and serialization."""
        return core_schema.no_info_plain_validator_function(
            cls._validate,
            serialization=core_schema.plain_serializer_function_ser_schema(
                lambda c: list(c)  # Serialize as [R, G, B, A]
            ),
        )

    @classmethod
    def _validate(cls, v: Any) -> "Color":
        """Validate and convert input to Color."""
        if isinstance(v, cls):
            return v
        return cls(v)

    def __repr__(self) -> str:
        return f"Color({self.hex()})"
