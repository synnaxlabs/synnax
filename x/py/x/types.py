#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Sized integer types for Pydantic models.

Python integers are arbitrary precision, so these types enforce the value ranges of
fixed-width integer types. They work with isinstance checks and Pydantic model
validation.

At runtime, each type is an int subclass with a custom metaclass that makes isinstance
check value ranges. For static type checking, they are treated as plain int aliases to
avoid spurious type errors when assigning int literals.
"""

from __future__ import annotations

from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    from typing import TypeAlias

    Int8: TypeAlias = int
    Int16: TypeAlias = int
    Int32: TypeAlias = int
    Int64: TypeAlias = int
    Uint8: TypeAlias = int
    Uint12: TypeAlias = int
    Uint16: TypeAlias = int
    Uint20: TypeAlias = int
    Uint32: TypeAlias = int
    Uint64: TypeAlias = int
else:
    from pydantic import GetCoreSchemaHandler
    from pydantic_core import CoreSchema, core_schema

    class IntRangeMeta(type):
        """Metaclass that makes isinstance check whether an int is within range."""

        _min: int
        _max: int

        def __instancecheck__(cls, instance: object) -> bool:
            if not type.__instancecheck__(int, instance) or type.__instancecheck__(
                bool, instance
            ):
                return False
            return cls._min <= instance <= cls._max

    def _int_range_schema(cls: type, _: Any, __: GetCoreSchemaHandler) -> CoreSchema:
        return core_schema.int_schema(ge=cls._min, le=cls._max)  # type: ignore[attr-defined]

    class Int8(int, metaclass=IntRangeMeta):
        _min = -(2**7)
        _max = 2**7 - 1
        __get_pydantic_core_schema__ = classmethod(_int_range_schema)

    class Int16(int, metaclass=IntRangeMeta):
        _min = -(2**15)
        _max = 2**15 - 1
        __get_pydantic_core_schema__ = classmethod(_int_range_schema)

    class Int32(int, metaclass=IntRangeMeta):
        _min = -(2**31)
        _max = 2**31 - 1
        __get_pydantic_core_schema__ = classmethod(_int_range_schema)

    class Int64(int, metaclass=IntRangeMeta):
        _min = -(2**63)
        _max = 2**63 - 1
        __get_pydantic_core_schema__ = classmethod(_int_range_schema)

    class Uint8(int, metaclass=IntRangeMeta):
        _min = 0
        _max = 2**8 - 1
        __get_pydantic_core_schema__ = classmethod(_int_range_schema)

    class Uint12(int, metaclass=IntRangeMeta):
        _min = 0
        _max = 2**12 - 1
        __get_pydantic_core_schema__ = classmethod(_int_range_schema)

    class Uint16(int, metaclass=IntRangeMeta):
        _min = 0
        _max = 2**16 - 1
        __get_pydantic_core_schema__ = classmethod(_int_range_schema)

    class Uint20(int, metaclass=IntRangeMeta):
        _min = 0
        _max = 2**20 - 1
        __get_pydantic_core_schema__ = classmethod(_int_range_schema)

    class Uint32(int, metaclass=IntRangeMeta):
        _min = 0
        _max = 2**32 - 1
        __get_pydantic_core_schema__ = classmethod(_int_range_schema)

    class Uint64(int, metaclass=IntRangeMeta):
        _min = 0
        _max = 2**64 - 1
        __get_pydantic_core_schema__ = classmethod(_int_range_schema)
