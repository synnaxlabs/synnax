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
from typing import Any

from pydantic import GetCoreSchemaHandler
from pydantic_core import core_schema

_NON_FINITE = {"NaN": math.nan, "Infinity": math.inf, "-Infinity": -math.inf}


class Float(float):
    """Float is a 64-bit float. It is a JSON number when finite, and the text "NaN",
    "Infinity", or "-Infinity" when not.
    """

    def __new__(cls, value: object) -> Float:
        if isinstance(value, str):
            if value not in _NON_FINITE:
                raise ValueError(f"cannot parse a float from {value!r}")
            return super().__new__(cls, _NON_FINITE[value])
        if isinstance(value, bool) or not isinstance(value, (int, float)):
            raise ValueError(f"cannot parse a float from {value!r}")
        try:
            return super().__new__(cls, value)
        except OverflowError as e:
            raise ValueError(f"cannot parse a float from {value!r}") from e

    def encode(self) -> float | str:
        """:returns: the float as a JSON number, or its text when it is not finite."""
        if math.isnan(self):
            return "NaN"
        if math.isinf(self):
            return "Infinity" if self > 0 else "-Infinity"
        return float(self)

    @classmethod
    def __get_pydantic_core_schema__(
        cls, source_type: Any, handler: GetCoreSchemaHandler
    ) -> core_schema.CoreSchema:
        """Implemented for pydantic validation. Should not be used externally."""
        return core_schema.no_info_plain_validator_function(
            cls,
            serialization=core_schema.plain_serializer_function_ser_schema(
                lambda v: Float(v).encode(), when_used="json"
            ),
        )
