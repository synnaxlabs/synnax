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
from typing import Any, ClassVar

from pydantic import GetCoreSchemaHandler
from pydantic_core import core_schema


class _Decimal(int):
    """_Decimal is a bounded integer that is a JSON string of decimal digits."""

    min: ClassVar[int]
    max: ClassVar[int]
    digits: ClassVar[re.Pattern[str]]

    def __new__(cls, value: int | str) -> _Decimal:
        valid_str = isinstance(value, str) and cls.digits.fullmatch(value) is not None
        if not valid_str and (isinstance(value, bool) or not isinstance(value, int)):
            raise ValueError(f"cannot parse {cls.__name__} from {value!r}")
        n = int(value)
        if not cls.min <= n <= cls.max:
            raise ValueError(f"{n} is out of range for {cls.__name__}")
        return super().__new__(cls, n)

    @classmethod
    def __get_pydantic_core_schema__(
        cls, source_type: Any, handler: GetCoreSchemaHandler
    ) -> core_schema.CoreSchema:
        """Implemented for pydantic validation. Should not be used externally."""
        return core_schema.no_info_plain_validator_function(
            cls,
            serialization=core_schema.plain_serializer_function_ser_schema(
                lambda v: str(int(v)), when_used="json"
            ),
        )


class Integer(_Decimal):
    """Integer is a signed 64-bit integer. It is a JSON string of decimal digits, so it
    keeps every digit.
    """

    min = -(2**63)
    max = 2**63 - 1
    digits = re.compile(r"-?[0-9]+")


class Unsigned(_Decimal):
    """Unsigned is an unsigned 64-bit integer. It is a JSON string of decimal digits,
    so it keeps every digit.
    """

    min = 0
    max = 2**64 - 1
    digits = re.compile(r"[0-9]+")
