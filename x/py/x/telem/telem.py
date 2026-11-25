#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

# On Python 3.15 and later, pandas loads on first use. Older versions ignore this.
__lazy_modules__ = ["pandas"]

import uuid
from typing import Any, ClassVar, TypeAlias, override

import numpy as np
from numpy.typing import DTypeLike
from pydantic import GetCoreSchemaHandler
from pydantic_core import core_schema

from x.telem.quantity import Density, TimeSpan, TimeStamp
from x.telem.types_gen import (
    DATA_TYPE_NAME_BOOLEAN,
    DATA_TYPE_NAME_BYTES,
    DATA_TYPE_NAME_FLOAT32,
    DATA_TYPE_NAME_FLOAT64,
    DATA_TYPE_NAME_INT8,
    DATA_TYPE_NAME_INT16,
    DATA_TYPE_NAME_INT32,
    DATA_TYPE_NAME_INT64,
    DATA_TYPE_NAME_JSON,
    DATA_TYPE_NAME_STRING,
    DATA_TYPE_NAME_TIMESTAMP,
    DATA_TYPE_NAME_UINT8,
    DATA_TYPE_NAME_UINT16,
    DATA_TYPE_NAME_UINT32,
    DATA_TYPE_NAME_UINT64,
    DATA_TYPE_NAME_UUID,
)


class DataType(str):
    """DataType represents a data type as a string."""

    def __new__(cls, value: CrudeDataType) -> DataType:
        if isinstance(value, DataType):
            return value

        if isinstance(value, str):
            if value not in DataType.ALL:
                raise ValueError(f"{value!r} is not a valid data type")
            return super().__new__(cls, value)

        if isinstance(value, (int, float, np.number, np.bool_, dict, uuid.UUID)):
            return DataType._from_sample(value)

        if isinstance(value, list):
            if len(value) == 0:
                raise ValueError("Cannot extract a data type from an empty list")

            types = {DataType._from_sample(v) for v in value}
            if len(types) == 1:
                return types.pop()
            return DataType._promote(types, value)

        return DataType._from_dtype(np.dtype(value))

    @staticmethod
    def _from_dtype(dtype: np.dtype) -> DataType:
        if dtype.kind == "M":
            return DataType.TIMESTAMP
        if dtype.kind == "U":
            return DataType.STRING
        result = DataType._FROM_NUMPY.get(dtype, None)
        if result is None:
            raise TypeError(f"Synnax has no data type for the NumPy type {dtype}")
        return result

    @staticmethod
    def _from_sample(value: Any) -> DataType:
        if isinstance(value, str):
            return DataType.STRING
        if isinstance(value, bytes):
            return DataType.BYTES
        if isinstance(value, dict):
            return DataType.JSON
        if isinstance(value, TimeStamp):
            return DataType.TIMESTAMP
        if isinstance(value, (bool, np.bool_)):
            return DataType.BOOLEAN
        if isinstance(value, float):
            return DataType.FLOAT64
        if isinstance(value, int):
            return DataType.INT64
        if isinstance(value, uuid.UUID):
            return DataType.UUID
        if isinstance(value, (np.number, np.datetime64)):
            return DataType._from_dtype(value.dtype)
        raise TypeError(f"Cannot convert {type(value)} to DataType")

    @staticmethod
    def _promote(types: set[DataType], samples: list[Any]) -> DataType:
        """Returns one numeric type that holds every sample exactly, or raises."""
        if not all(
            t != DataType.TIMESTAMP and t.has_np and t.np.kind in "iuf" for t in types
        ):
            raise TypeError(f"Cannot infer one data type from {sorted(types)}")
        promoted = DataType(np.result_type(*(t.np for t in types)))
        if promoted.np.kind != "f":
            return promoted
        ints = [int(v) for v in samples if isinstance(v, (int, np.integer))]
        if len(ints) == len(samples):
            if all(-(2**63) <= i < 2**63 for i in ints):
                return DataType.INT64
            if all(0 <= i < 2**64 for i in ints):
                return DataType.UINT64
        elif all(int(promoted.np.type(i)) == i for i in ints):
            return promoted
        raise TypeError(f"No data type in {sorted(types)} holds every value exactly")

    @classmethod
    def __get_pydantic_core_schema__(
        cls, source_type: Any, handler: GetCoreSchemaHandler
    ) -> core_schema.CoreSchema:
        """Implemented for pydantic validation. Should not be used externally."""
        return core_schema.no_info_after_validator_function(cls, handler(str))

    @classmethod
    def validate(cls, v: Any, *args: Any, **kwargs: Any) -> DataType:
        """Implemented for pydantic validation. Should not be used externally."""
        return cls(v)

    @classmethod
    def __modify_schema__(cls, field_schema: dict[str, Any]) -> None:
        """Implemented for pydantic validation. Should not be used externally."""
        field_schema.update(type="string")

    @classmethod
    def __get_pydantic_json_schema__(
        cls, _schema_generator: Any, _field_schema: Any
    ) -> dict[str, str]:
        """Implemented for pydantic validation. Should not be used externally."""
        return {"type": "string"}

    @property
    def np(self) -> np.dtype:
        """Converts a built-in DataType to a numpy type Scalar Type
        :returns: The numpy type
        """
        npt = DataType._TO_NUMPY.get(self, None)
        if not isinstance(npt, np.dtype):
            raise TypeError(f"Cannot convert {self} to numpy type")
        return npt

    @property
    def is_variable(self) -> bool:
        return (
            self == DataType.STRING or self == DataType.JSON or self == DataType.BYTES
        )

    @property
    def has_np(self) -> bool:
        """:returns: True if the DataType has a corresponding numpy type"""
        return self in DataType._TO_NUMPY

    @property
    def density(self) -> Density:
        """:returns: The density of the DataType. If the density can't be determined,
        returns Density.UNKNOWN.
        """
        return DataType._DENSITIES.get(self, Density.UNKNOWN)

    @override
    def __repr__(self) -> str:
        return f"DataType({super().__repr__()})"

    UNKNOWN: DataType
    UUID: DataType
    TIMESTAMP: DataType
    FLOAT64: DataType
    FLOAT32: DataType
    INT64: DataType
    INT32: DataType
    INT16: DataType
    INT8: DataType
    UINT64: DataType
    UINT32: DataType
    UINT16: DataType
    UINT8: DataType
    STRING: DataType
    JSON: DataType
    BYTES: DataType
    BOOLEAN: DataType
    ALL: tuple[DataType, ...]
    _TO_NUMPY: dict[DataType, DTypeLike]
    _FROM_NUMPY: dict[DTypeLike, DataType]
    _DENSITIES: dict[DataType, Density]


def _const(name: str) -> DataType:
    return str.__new__(DataType, name)


DataType.UUID = _const(DATA_TYPE_NAME_UUID)
DataType.UNKNOWN = _const("")
DataType.TIMESTAMP = _const(DATA_TYPE_NAME_TIMESTAMP)
DataType.FLOAT64 = _const(DATA_TYPE_NAME_FLOAT64)
DataType.FLOAT32 = _const(DATA_TYPE_NAME_FLOAT32)
DataType.INT64 = _const(DATA_TYPE_NAME_INT64)
DataType.INT32 = _const(DATA_TYPE_NAME_INT32)
DataType.INT16 = _const(DATA_TYPE_NAME_INT16)
DataType.INT8 = _const(DATA_TYPE_NAME_INT8)
DataType.UINT64 = _const(DATA_TYPE_NAME_UINT64)
DataType.UINT32 = _const(DATA_TYPE_NAME_UINT32)
DataType.UINT16 = _const(DATA_TYPE_NAME_UINT16)
DataType.UINT8 = _const(DATA_TYPE_NAME_UINT8)
DataType.JSON = _const(DATA_TYPE_NAME_JSON)
DataType.STRING = _const(DATA_TYPE_NAME_STRING)
DataType.BYTES = _const(DATA_TYPE_NAME_BYTES)
DataType.BOOLEAN = _const(DATA_TYPE_NAME_BOOLEAN)
DataType.ALL = (
    DataType.UNKNOWN,
    DataType.UINT8,
    DataType.UINT16,
    DataType.UINT32,
    DataType.UINT64,
    DataType.INT8,
    DataType.INT16,
    DataType.INT32,
    DataType.INT64,
    DataType.FLOAT32,
    DataType.FLOAT64,
    DataType.BOOLEAN,
    DataType.TIMESTAMP,
    DataType.UUID,
    DataType.STRING,
    DataType.JSON,
    DataType.BYTES,
)

CrudeDataType: TypeAlias = (
    DTypeLike | DataType | str | list[str] | np.number | int | float
)

DataType._TO_NUMPY = {
    DataType.FLOAT64: np.dtype(np.float64),
    DataType.FLOAT32: np.dtype(np.float32),
    DataType.TIMESTAMP: np.dtype(np.int64),
    DataType.INT64: np.dtype(np.int64),
    DataType.INT32: np.dtype(np.int32),
    DataType.INT16: np.dtype(np.int16),
    DataType.INT8: np.dtype(np.int8),
    DataType.UINT64: np.dtype(np.uint64),
    DataType.UINT32: np.dtype(np.uint32),
    DataType.UINT16: np.dtype(np.uint16),
    DataType.UINT8: np.dtype(np.uint8),
    DataType.BOOLEAN: np.dtype(np.bool_),
}
DataType._FROM_NUMPY = {
    np.dtype(np.float64): DataType.FLOAT64,
    np.dtype(np.float32): DataType.FLOAT32,
    np.dtype(np.int64): DataType.INT64,
    np.dtype(np.int32): DataType.INT32,
    np.dtype(np.int16): DataType.INT16,
    np.dtype(np.int8): DataType.INT8,
    np.dtype(np.uint64): DataType.UINT64,
    np.dtype(np.uint32): DataType.UINT32,
    np.dtype(np.uint16): DataType.UINT16,
    np.dtype(np.uint8): DataType.UINT8,
    np.dtype(np.bool_): DataType.BOOLEAN,
}

DataType._DENSITIES = {
    DataType.UUID: Density.BIT128,
    DataType.FLOAT64: Density.BIT64,
    DataType.FLOAT32: Density.BIT32,
    DataType.TIMESTAMP: Density.BIT64,
    DataType.INT64: Density.BIT64,
    DataType.INT32: Density.BIT32,
    DataType.INT16: Density.BIT16,
    DataType.INT8: Density.BIT8,
    DataType.UINT64: Density.BIT64,
    DataType.UINT32: Density.BIT32,
    DataType.UINT16: Density.BIT16,
    DataType.UINT8: Density.BIT8,
    DataType.STRING: Density.UNKNOWN,
    DataType.JSON: Density.UNKNOWN,
    DataType.BYTES: Density.UNKNOWN,
    DataType.BOOLEAN: Density.BIT8,
}


class Bounds:
    lower: float
    upper: float

    def __init__(self, lower: float, upper: float):
        self.lower = lower
        self.upper = upper

    def contains(self, value: float) -> bool:
        """:returns: True if lower <= value < upper."""
        return self.lower <= value < self.upper


def _pack_alignment(domain_idx: int, sample_idx: int) -> int:
    """Packs uint32 indices, wrapping each like the Go Alignment's uint32 fields."""
    return ((domain_idx & 0xFFFFFFFF) << 32) | (sample_idx & 0xFFFFFFFF)


class Alignment(int):
    """Alignment is two array index values that can be used to represent the location of
    a sample within an array of arrays. For example, if you have two arrays that have 50
    elements each, and you want the 15th element of the second array, you would use
    Alignment(1, 15). The first index is called the 'domain index' and the second index
    is called the 'sample index'. The domain index is the index of the array, and the
    sample index is the index of the sample within that array.

    You may think a better design is to just use a single number that overflows the
    arrays before it, i.e., the value of our previous example would be 50 + 14 = 64.
    However, this requires us to know the size of all arrays, which is not always
    possible.

    While not as meaningful as a single number, Alignment is a uint64 that guarantees
    that a larger value is, in fact, 'positionally' after a smaller value. This is
    useful for ordering samples correctly.

    The Alignment constructor accepts various types:

    * int - Treats the int as a packed alignment value.
    * tuple[int, int] - Treats as (domain_index, sample_index).
    * Alignment - Returns a copy of the Alignment.
    """

    MAX_ALIGNMENT: ClassVar[int] = (1 << 64) - 1

    def __new__(
        cls, value: CrudeAlignment = 0, sample_idx: int | None = None
    ) -> Alignment:
        """Creates a new Alignment from various input types.

        :param value: Either a packed int value, a tuple of (domain_idx, sample_idx), or
                      an existing Alignment. Can also be a domain_idx if sample_idx is
                      provided.
        :param sample_idx: Optional sample index if value is provided as domain index.
        :returns: A new Alignment.
        """
        if isinstance(value, str):
            # The Go server marshals the uint64 alignment as a JSON string to avoid
            # float64 precision loss, so a packed value can arrive as a string.
            value = int(value)
        if isinstance(value, Alignment):
            return value
        if isinstance(value, tuple):
            if len(value) != 2:
                raise ValueError(
                    "Alignment tuple must have exactly 2 elements (domain_idx, "
                    f"sample_idx), got {len(value)}"
                )
            domain_idx, sample_idx = value
            packed_value = _pack_alignment(int(domain_idx), int(sample_idx))
        elif sample_idx is not None:
            packed_value = _pack_alignment(int(value), sample_idx)
        elif isinstance(value, (int, np.integer)):
            # Packed value or zero
            packed_value = int(value)
        else:
            raise TypeError(f"Cannot convert {type(value)} to Alignment")

        return super().__new__(cls, packed_value)

    @classmethod
    def __get_pydantic_core_schema__(
        cls, source_type: Any, handler: GetCoreSchemaHandler
    ) -> core_schema.CoreSchema:
        """Implemented for pydantic validation. Should not be used externally."""
        return core_schema.no_info_after_validator_function(cls._validate, handler(int))

    @classmethod
    def _validate(cls, value: int) -> Alignment:
        """Implemented for pydantic validation. Should not be used externally."""
        # Use the constructor to handle all conversion logic
        return cls(value)

    @property
    def domain_index(self) -> int:
        """Returns the domain index of the Alignment. This is the index
        in the array of arrays.

        :returns: The domain index.
        """
        return int(self) >> 32

    @property
    def sample_index(self) -> int:
        """Returns the sample index of the Alignment. This is the index within
        a particular array.

        :returns: The sample index.
        """
        return int(self) & 0xFFFFFFFF

    def add_samples(self, samples: int) -> Alignment:
        """Increments the sample index of the alignment.

        :param samples: The number of samples to add.
        :returns: A new Alignment with the sample index incremented.
        """
        return Alignment(self.domain_index, self.sample_index + samples)

    def add(self, other: Alignment) -> Alignment:
        """Adds another Alignment to this one.

        :param other: The other Alignment to add.
        :returns: A new Alignment with both domain and sample indices added.
        """
        return Alignment(
            self.domain_index + other.domain_index,
            self.sample_index + other.sample_index,
        )

    @override
    def __str__(self) -> str:
        """Returns a string representation of the Alignment in the format
        'domain-sample'.

        :returns: A string representation of the Alignment.
        """
        return f"{self.domain_index}-{self.sample_index}"

    @override
    def __repr__(self) -> str:
        """Returns a string representation of the Alignment for debugging.

        :returns: A string representation of the Alignment.
        """
        return f"Alignment({self.domain_index}, {self.sample_index})"


CrudeAlignment: TypeAlias = int | tuple[int, int] | Alignment | str


def seconds_linspace(start: int, *, count: int) -> list[TimeStamp]:
    """Generates count TimeStamps one second apart, starting start seconds after the
    Unix epoch."""
    return [TimeStamp(TimeSpan.SECOND * (start + i)) for i in range(count)]
