#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

import json
import uuid
from collections.abc import Iterator
from datetime import datetime
from typing import Any, TypeAlias, cast

import numpy as np
import numpy.typing as npt
import pandas as pd
from pydantic import BaseModel, GetCoreSchemaHandler
from pydantic_core import core_schema

from x.interop import ArrayOperators
from x.telem.telem import (
    Alignment,
    Bounds,
    CrudeAlignment,
    CrudeDataType,
    DataType,
    Size,
    TimeRange,
    TimeSpan,
    TimeStamp,
)

_VARIABLE_LENGTH_PREFIX_SIZE = 4


def _encode_samples(samples: list[Any], data_type: DataType) -> bytes:
    if data_type == DataType.JSON:
        return _encode_variable([json.dumps(d).encode("utf-8") for d in samples])
    if data_type == DataType.STRING:
        return _encode_variable([str(d).encode("utf-8") for d in samples])
    if data_type == DataType.UUID:
        uuids = [d for d in samples if isinstance(d, uuid.UUID)]
        if len(uuids) != len(samples):
            raise TypeError("[Series] - a UUID series accepts only UUIDs")
        return b"".join(d.bytes for d in uuids)
    return np.array(samples, dtype=data_type.np).tobytes()


def _encode_variable(parts: list[bytes]) -> bytes:
    return b"".join(
        len(p).to_bytes(_VARIABLE_LENGTH_PREFIX_SIZE, "little") + p for p in parts
    )


class _Payload(BaseModel):
    time_range: TimeRange | None = None
    data_type: DataType
    data: bytes
    alignment: Alignment = Alignment(0, 0)
    alignment_multiple: int = 1


def _from_payload(p: _Payload) -> Series:
    return Series(
        p.data,
        data_type=p.data_type,
        time_range=p.time_range,
        alignment=p.alignment,
        alignment_multiple=p.alignment_multiple,
    )


def _to_payload(s: Series) -> _Payload:
    return _Payload(
        time_range=s.time_range,
        data_type=s.data_type,
        data=s.data,
        alignment=s.alignment,
        alignment_multiple=s.alignment_multiple,
    )


class Series(ArrayOperators):
    """Series is a strongly typed array of telemetry samples backed by an underlying
    binary buffer. It is interoperable with np.ndarray, meaning that it can be safely
    passed as an argument to any function/method that accepts a numpy array.

    Series also have an optional 'time_range' property that can be used to define the
    time range occupied by the Series' data. This property is guaranteed to be defined
    when reading data from a Synnax cluster, and is particularly useful for
    understanding the alignment of samples in relation to another series. This is
    especially relevant in the context of a Frame (framer.Frame).
    """

    time_range: TimeRange | None
    """An optional property defines the time range occupied by the
    Series' data. This property is guaranteed to be defined when reading data from
    a Synnax cluster, and is particularly useful for understanding the alignment of
    samples in relation to another series. This is especially relevant in the context
    of a Frame (framer.Frame). When set by the Cluster, the start of the time range
    represents the timestamp of the first sample in the array (inclusive), while the
    end of the time range is set to the nanosecond AFTER the last sample in the array
    (exclusive).
    """
    data_type: DataType
    """The data type of the Series"""
    data: bytes
    """The underlying buffer"""
    alignment: Alignment
    """The alignment of the Series, representing the position within an array of arrays"""
    alignment_multiple: int
    """The number of alignment steps between consecutive samples. Above one when the
    Core reduced the Series from a longer one."""
    _len_cache: int | None

    def __len__(self) -> int:
        if not self.data_type.is_variable:
            return self.data_type.density.sample_count(len(self.data))
        if self._len_cache is None:
            count = 0
            offset = 0
            d = self.data
            while offset + _VARIABLE_LENGTH_PREFIX_SIZE <= len(d):
                length = int.from_bytes(
                    d[offset : offset + _VARIABLE_LENGTH_PREFIX_SIZE], "little"
                )
                offset += _VARIABLE_LENGTH_PREFIX_SIZE + length
                count += 1
            self._len_cache = count
        return self._len_cache

    def __init__(
        self,
        data: CrudeSeries | MultiSeries,
        *,
        data_type: CrudeDataType | None = None,
        time_range: TimeRange | None = None,
        alignment: CrudeAlignment | None = None,
        alignment_multiple: int | None = None,
    ):
        if data_type is not None:
            data_type = DataType(data_type)
        align = None if alignment is None else Alignment(alignment)
        if isinstance(data, (TimeStamp, int, float, np.number, np.bool_)):
            data_type = data_type or DataType(data)
            data_ = np.array([data], dtype=data_type.np).tobytes()
        elif isinstance(data, Series):
            data_type = data_type or data.data_type
            data_ = data.data
            time_range = data.time_range if time_range is None else time_range
            align = data.alignment if align is None else align
            if alignment_multiple is None:
                alignment_multiple = data.alignment_multiple
        elif isinstance(data, MultiSeries):
            if len(data.series) == 1:
                data_type = data_type or data.series[0].data_type
                data_ = data.series[0].data
                time_range = (
                    data.series[0].time_range if time_range is None else time_range
                )
                align = data.series[0].alignment if align is None else align
                if alignment_multiple is None:
                    alignment_multiple = data.series[0].alignment_multiple
            else:
                raise ValueError(
                    "[Series] - MultiSeries with more than one series cannot be converted to a Series"
                )
            data_type = data_type or data.data_type
        elif isinstance(data, pd.Series):
            if data_type is None:
                if not isinstance(data.dtype, np.dtype):
                    raise TypeError(f"Cannot convert {data.dtype} to DataType")
                data_type = DataType(data.dtype)
            data_ = data.to_numpy(dtype=data_type.np).tobytes()
        elif isinstance(data, np.ndarray):
            data_type = data_type or DataType(data.dtype)
            data_ = data.astype(data_type.np).tobytes()
        elif isinstance(data, list):
            data_type = data_type or DataType(data)
            data_ = _encode_samples(data, data_type)
        elif isinstance(data, (str, dict, uuid.UUID)):
            data_type = data_type or DataType([data])
            data_ = _encode_samples([data], data_type)
        else:
            if data_type is None:
                raise ValueError(
                    "[Series] - data_type must be specified if a buffer is given",
                )
            data_type = DataType(data_type)
            data_ = data
        self.data_type = data_type
        self.data = data_
        self.time_range = time_range
        self.alignment = Alignment(0) if align is None else align
        self.alignment_multiple = (
            1 if alignment_multiple is None else alignment_multiple
        )
        self._len_cache = None

    @classmethod
    def __get_pydantic_core_schema__(
        cls, source_type: object, handler: GetCoreSchemaHandler
    ) -> core_schema.CoreSchema:
        """Implemented for pydantic validation. Should not be used externally."""
        payload = handler.generate_schema(_Payload)
        return core_schema.union_schema(
            [
                core_schema.is_instance_schema(cls),
                core_schema.no_info_after_validator_function(_from_payload, payload),
            ],
            serialization=core_schema.plain_serializer_function_ser_schema(
                _to_payload, return_schema=payload
            ),
        )

    def __array__(
        self, dtype: np.dtype | None = None, copy: bool | None = None
    ) -> np.ndarray:
        """Implemented so that the Series can be passed around as a numpy array. See
        https://numpy.org/doc/stable/user/basics.interoperability.html#the-array-method.
        """
        if not self.data_type.has_np:
            raise ValueError(f"""
                [Series] - {self.data_type} does not have a numpy equivalent, so it can't
                be interpreted as a numpy array.
                """)

        arr = np.frombuffer(self.data, dtype=self.data_type.np)
        if dtype is not None and dtype != arr.dtype:
            arr = arr.astype(dtype, copy=True)
        elif copy:
            arr = arr.copy()
        return arr

    def to_numpy(
        self, dtype: np.dtype | None = None, *, copy: bool | None = None
    ) -> np.ndarray:
        """Converts the Series to a numpy array. This is necessary for matplotlib
        interop.
        """
        return self.__array__(dtype=dtype, copy=copy)

    def __getitem__(self, index: int) -> SampleValue:
        if not self.data_type.has_np and index < 0:
            index = len(self) + index

        if self.data_type == DataType.UUID:
            start = self.data_type.density.size_span(index)
            end = start + self.data_type.density
            d = self.data[start:end]
            return uuid.UUID(bytes=d)

        if self.data_type == DataType.JSON:
            d = self.__variable_getitem__(index)
            return json.loads(d)  # type: ignore[no-any-return]

        if self.data_type == DataType.STRING:
            d = self.__variable_getitem__(index)
            return d.decode("utf-8")

        # numpy stubs type scalar indexing as Any.
        return cast(np.number | np.bool_, self.__array__()[index])

    def __variable_getitem__(self, index: int) -> bytes:
        offset = 0
        d = self.data
        current = 0
        while offset + _VARIABLE_LENGTH_PREFIX_SIZE <= len(d):
            length = int.from_bytes(
                d[offset : offset + _VARIABLE_LENGTH_PREFIX_SIZE], "little"
            )
            offset += _VARIABLE_LENGTH_PREFIX_SIZE
            if current == index:
                return d[offset : offset + length]
            offset += length
            current += 1
        raise IndexError(f"[Series] - Index {index} out of bounds")

    def __iter__(self) -> Iterator[SampleValue]:
        if self.data_type == DataType.UUID:
            yield from [self[i] for i in range(len(self))]
        elif self.data_type == DataType.JSON:
            for v in self._iter_variable():
                yield json.loads(v)
        elif self.data_type == DataType.STRING:
            for v in self._iter_variable():
                yield v.decode("utf-8")
        else:
            yield from self.__array__()

    def _iter_variable(self) -> Iterator[bytes]:
        offset = 0
        d = self.data
        while offset + _VARIABLE_LENGTH_PREFIX_SIZE <= len(d):
            length = int.from_bytes(
                d[offset : offset + _VARIABLE_LENGTH_PREFIX_SIZE], "little"
            )
            offset += _VARIABLE_LENGTH_PREFIX_SIZE
            yield d[offset : offset + length]
            offset += length

    def __str__(self) -> str:
        return str(list(self))

    def __repr__(self) -> str:
        return (
            f"Series(data_type={self.data_type!r}, len={len(self)}, "
            f"time_range={self.time_range!r}, alignment={self.alignment!r})"
        )

    @property
    def size(self) -> Size:
        """:returns: A Size representing the number of bytes in the Series' data."""
        return Size(len(self.data))

    @property
    def alignment_bounds(self) -> Bounds:
        """:returns: The bounds of alignments covered by this Series.

        The lower bound is the alignment of the first sample (inclusive), and the
        upper bound is the alignment just after the last sample (exclusive).
        This represents the range [alignment, alignment + length * alignment_multiple).
        """
        return Bounds(
            lower=int(self.alignment),
            upper=int(self.alignment) + len(self) * self.alignment_multiple,
        )

    def astype(self, data_type: DataType) -> Series:
        return Series(
            data=self.__array__().astype(data_type.np),
            data_type=data_type,
            time_range=self.time_range,
            alignment=self.alignment,
            alignment_multiple=self.alignment_multiple,
        )

    def to_datetime(self) -> list[datetime]:
        return [pd.Timestamp(t).to_pydatetime() for t in self.__array__()]


SampleValue: TypeAlias = (
    np.number | np.bool_ | uuid.UUID | dict[str, Any] | str | int | float | TimeStamp
)
TypedCrudeSeries: TypeAlias = Series | pd.Series | np.ndarray


def to_float(value: SampleValue) -> float:
    """Converts a sample from a numeric channel to a float.

    :raises TypeError: If the sample is not numeric.
    """
    if isinstance(value, (int, float, np.number, np.bool_)):
        return float(value)
    raise TypeError(f"Expected a numeric sample, got {value!r}")


def to_int(value: SampleValue) -> int:
    """Converts a sample from an integer or timestamp channel to an int.

    :raises TypeError: If the sample is not an integer.
    """
    if isinstance(value, (int, np.integer, np.bool_)):
        return int(value)
    raise TypeError(f"Expected an integer sample, got {value!r}")


CrudeSeries: TypeAlias = (
    Series
    | bytes
    | pd.Series
    | np.ndarray
    | list[float]
    | list[int]
    | list[bool]
    | list[str]
    | list[dict[str, Any]]
    | list[uuid.UUID]
    | list[TimeStamp]
    | np.number
    | np.bool_
    | str
    | uuid.UUID
    | dict[str, Any]
    | float
    | int
    | bool
    | TimeStamp
)


def elapsed_seconds(d: npt.NDArray[np.int64]) -> npt.NDArray[np.float64]:
    """Converts a Series of timestamps to elapsed seconds since the first timestamp.

    :param d: A Series of timestamps.
    :returns: A Series of elapsed seconds.
    """
    return (d - d[:1]) / TimeSpan.SECOND


class MultiSeries(ArrayOperators):
    series: list[Series]

    def __init__(self, series: list[Series]):
        self.series = series
        if len(series) > 0:
            first_dt = series[0].data_type
            same_dt = all(s.data_type == first_dt for s in series)
            if not same_dt:
                data_types = {s.data_type for s in series}
                raise ValueError(f"""
                    [MultiSeries] - All series must have the same data type. Received
                    {first_dt} and {data_types}.
                    """)

    def __array__(
        self, dtype: np.dtype | None = None, copy: bool | None = None
    ) -> np.ndarray:
        if len(self.series) == 0:
            target_dtype = dtype if dtype is not None else np.float64
            return np.empty((0,), dtype=target_dtype)
        if len(self.series) == 1:
            return self.series[0].__array__(dtype=dtype, copy=copy)
        target_dtype = dtype if dtype is not None else self.series[0].data_type.np
        pre_alloc = np.empty((len(self),), dtype=target_dtype)
        start = 0
        for s in self.series:
            end = start + len(s)
            pre_alloc[start:end] = s.__array__(dtype=target_dtype)
            start = end
        # copy is implicitly handled since we're creating a new array
        return pre_alloc

    def to_numpy(
        self, dtype: np.dtype | None = None, *, copy: bool | None = None
    ) -> np.ndarray:
        return self.__array__(dtype=dtype, copy=copy)

    @property
    def time_range(self) -> TimeRange | None:
        if len(self.series) == 0:
            return TimeRange.ZERO
        first = self.series[0].time_range
        last = self.series[-1].time_range
        if first is None or last is None:
            return None
        return TimeRange(start=first.start, end=last.end)

    @property
    def alignment(self) -> Alignment:
        """:returns: The alignment of the first sample in the MultiSeries.

        If the MultiSeries is empty, returns Alignment(0, 0).
        """
        if len(self.series) == 0:
            return Alignment(0, 0)
        return self.series[0].alignment

    @property
    def alignment_bounds(self) -> Bounds:
        """:returns: The bounds of alignments covered by this MultiSeries.

        The lower bound is the alignment of the first sample in the first series,
        and the upper bound is the alignment just after the last sample in the last series.
        If the MultiSeries is empty, returns Bounds(0, 0).
        """
        if len(self.series) == 0:
            return Bounds(0, 0)
        return Bounds(
            lower=self.series[0].alignment_bounds.lower,
            upper=self.series[-1].alignment_bounds.upper,
        )

    @property
    def data_type(self) -> DataType:
        if len(self.series) == 0:
            return DataType.UNKNOWN
        return self.series[0].data_type

    def __len__(self) -> int:
        return sum(len(s) for s in self.series)

    def __getitem__(self, index: int) -> SampleValue:
        if index < 0:
            index = len(self) + index
        for s in self.series:
            if index < len(s):
                return s[index]
            index -= len(s)
        raise IndexError(f"[MultiSeries] - Index {index} out of bounds for {len(self)}")

    def __iter__(self) -> Iterator[SampleValue]:
        for s in self.series:
            yield from s

    def __str__(self) -> str:
        return str(list(self))

    @property
    def size(self) -> Size:
        return Size(sum(s.size for s in self.series))
