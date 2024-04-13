#  Copyright 2023 Synnax Labs, Inc.
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
from datetime import datetime

import numpy as np
import pandas as pd
from freighter import Payload
from pydantic import PrivateAttr

from synnax.telem.telem import (
    CrudeDataType,
    DataType,
    Size,
    TimeRange,
    TimeSpan,
    TimeStamp,
)
from synnax.util.interop import overload_comparison_operators


class Series(Payload):
    def __new__(cls, *args, **kwargs):
        return super().__new__(overload_comparison_operators(cls, "__array__"))

    """Series is a strongly typed array of telemetry samples backed by an underlying
    binary buffer. It is interoperable with np.ndarray, meaning that it can be safely
    passed as an argument to any function/method that accepts a numpy array.

    Series also have an optional 'time_range' property that can be used to define the
    time range occupied by the Series' data. This property is guaranteed to be defined
    when reading data from a Synnax cluster, and is particularly useful for
    understanding the alignment of samples in relation to another series. This is
    especially relevant in the context of a Frame (framer.Frame).
    """

    time_range: TimeRange | None = None
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
    alignment: int = 0
    __len_cache: int | None = PrivateAttr(None)

    def __len__(self) -> int:
        if self.data_type.has_fixed_density:
            return self.data_type.density.sample_count(len(self.data))
        if self.__len_cache is None:
            self.__len_cache = self.data.count(b"\n")
        return self.__len_cache

    def __init__(
        self,
        data: CrudeSeries,
        data_type: CrudeDataType | None = None,
        time_range: TimeRange | None = None,
        alignment: int = 0,
    ):
        if isinstance(data, (TimeStamp, int, float, np.number)):
            data_type = data_type or DataType(data)
            data_ = np.array([data], dtype=data_type.np).tobytes()
        elif isinstance(data, Series):
            data_type = data_type or data.data_type
            data_ = data.data
            time_range = data.time_range if time_range is None else time_range
        elif isinstance(data, pd.Series):
            data_type = data_type or DataType(data.dtype)
            data_ = data.to_numpy(dtype=data_type.np).tobytes()
        elif isinstance(data, np.ndarray):
            data_type = data_type or DataType(data.dtype)
            data_ = data.astype(data_type.np).tobytes()
        elif isinstance(data, list):
            data_type = data_type or DataType(data)
            if data_type == DataType.JSON:
                data_ = (
                    b"\n".join([json.dumps(d).encode("utf-8") for d in data]) + b"\n"
                )
            elif data_type == DataType.STRING:
                data_ = b"\n".join([d.encode("utf-8") for d in data]) + b"\n"
            elif data_type == DataType.UUID:
                data_ = b"".join(d.bytes for d in data)
            else:
                data_ = np.array(data, dtype=data_type.np).tobytes()
        else:
            if data_type is None:
                raise ValueError(
                    "[Series] - data_type must be specified if a buffer is given",
                )
            data_type = DataType(data_type)
            data_ = data
        super().__init__(
            data_type=data_type, data=data_, time_range=time_range, alignment=alignment
        )
        self.__len_cache = None

    class Config:
        arbitrary_types_allowed = True

    def __array__(self, *args, **kwargs) -> np.ndarray:
        """Implemented to that the Series can be passed around as a numpy array. See
        https://numpy.org/doc/stable/user/basics.interoperability.html#the-array-method.
        """
        if not self.data_type.has_np:
            raise ValueError(
                f"""
                [Series] - {self.data_type} does not have a numpy equivalent, so it can't
                be interpreted as a numpy array.
                """
            )

        if len(args) > 0:
            return np.array(self.__array__(), *args, **kwargs)
        return np.frombuffer(self.data, dtype=self.data_type.np, **kwargs)

    def to_numpy(self, *args, **kwargs) -> np.ndarray:
        """Converts the Series to a numpy array. This is necessary for matplotlib
        interop.
        """
        return self.__array__(*args, **kwargs)

    def __getitem__(self, index: int) -> SampleValue:
        if not self.data_type.has_np and index < 0:
            index = len(self) + index

        if self.data_type == DataType.UUID:
            start = self.data_type.density.size_span(index)
            end = start + self.data_type.density
            d = self.data[start:end]
            return uuid.UUID(bytes=d)

        if self.data_type == DataType.JSON:
            d = self.__newline_getitem__(index)
            return json.loads(d)

        if self.data_type == DataType.STRING:
            d = self.__newline_getitem__(index)
            return d.decode("utf-8")

        return self.__array__()[index]

    def __newline_getitem__(self, index: int) -> bytes:
        if index == 0:
            start = 0
        else:
            start = self.data.find(b"\n")
            while start >= 0 and index > 1:
                start = self.data.find(b"\n", start + 1)
                index -= 1
            start += 1

        if start < 0:
            raise IndexError(f"[Series] - Index {index} out of bounds")

        end = self.data.find(b"\n", start)
        if end < 0:
            end = len(self.data)
        return self.data[start:end]

    def __iter__(self):
        if self.data_type == DataType.UUID:
            yield from [self[i] for i in range(len(self))]
        elif self.data_type == DataType.JSON:
            for v in self.__iter__newline():
                yield json.loads(v)
        elif self.data_type == DataType.STRING:
            for v in self.__iter__newline():
                yield v.decode("utf-8")
        else:
            yield from self.__array__()

    def __iter__newline(self):
        curr = 0
        while curr < len(self.data):
            end = self.data.find(b"\n", curr)
            if end < 0:
                end = len(self.data)
            yield self.data[curr:end]
            curr = end + 1

    @property
    def size(self) -> Size:
        """:returns: A Size representing the number of bytes in the Series' data."""
        return Size(len(self.data))

    def astype(self, data_type: DataType) -> Series:
        return Series(
            data=self.__array__().astype(data_type.np),
            data_type=data_type,
            time_range=self.time_range,
        )

    def to_datetime(self) -> list[datetime]:
        return [pd.Timestamp(t).to_pydatetime() for t in self.__array__()]

    def __eq__(self, other):
        if isinstance(other, Series):
            return self.data == other.data
        elif isinstance(other, np.ndarray):
            return self.__array__() == other
        else:
            return False


SampleValue = np.number | uuid.UUID | dict | str
TypedCrudeSeries = Series | pd.Series | np.ndarray
CrudeSeries = (
    Series
    | bytes
    | pd.Series
    | np.ndarray
    | list[float]
    | list[str]
    | list[dict]
    | float
    | int
    | TimeStamp
)


def elapsed_seconds(d: np.ndarray) -> np.ndarray:
    """Converts a Series of timestamps to elapsed seconds since the first timestamp.

    :param d: A Series of timestamps.
    :returns: A Series of elapsed seconds.
    """
    return (d - d[0]) / TimeSpan.SECOND


class MultiSeries:
    series: list[Series]

    def __new__(cls, *args, **kwargs):
        return super().__new__(overload_comparison_operators(cls, "__array__"))

    def __init__(self, series: list[Series]):
        self.series = series
        if len(series) > 0:
            first_dt = series[0].data_type
            same_dt = all(s.data_type == first_dt for s in series)
            if not same_dt:
                raise ValueError(
                    f"""
                    [MultiSeries] - All series must have the same data type. Received
                    {first_dt} and {set(s.data_type for s in series)}.
                    """
                )

    def __array__(self) -> np.ndarray:
        pre_alloc = np.empty((len(self),), dtype=self.series[0].data_type.np)
        start = 0
        for s in self.series:
            end = start + len(s)
            pre_alloc[start:end] = s.__array__()
            start = end
        return pre_alloc

    def to_numpy(self) -> np.ndarray:
        return self.__array__()

    @property
    def time_range(self) -> TimeRange | None:
        if len(self.series) == 0:
            return TimeRange.ZERO
        first = self.series[0].time_range
        last = self.series[-1].time_range
        if first is None or last is None:
            return None
        return TimeRange(start=first.start, end=last.end)

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

    def __iter__(self):
        for s in self.series:
            yield from s

    @property
    def size(self) -> Size:
        return Size(sum(s.size for s in self.series))
