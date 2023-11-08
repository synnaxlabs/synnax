#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

import uuid
from datetime import datetime

import numpy as np
import pandas as pd
from freighter import Payload

from synnax.telem.telem import (
    CrudeDataType,
    DataType,
    Size,
    TimeRange,
    TimeStamp,
    TimeSpan,
)
from synnax.util.interop import overload_comparison_operators


class Series(Payload):
    def __new__(cls, *args, **kwargs):
        return super().__new__(overload_comparison_operators(cls, "__array__"))

    """Series is a strongly typed array of telemetry samples backed by an underlying
    binary buffers. It is interoperable with np.ndarray, meaning that it can be safely
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
    of a Frame (framer.Frame). The start of the time range represents the timestamp of
    the first sample in the array (inclusive), while the end of the time range is set
    to the nanosecond AFTER the last sample in the array (exclusive).
    """
    data_type: DataType
    """The data type of the Series"""
    data: bytes
    """The underlying buffer"""
    alignment: int = 0

    def __len__(self) -> int:
        return self.data_type.density.sample_count(len(self.data))

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

    class Config:
        arbitrary_types_allowed = True

    def __array__(self, *args, **kwargs) -> np.ndarray:
        """Implemented to that the Series can be passed around as a numpy array. See
        https://numpy.org/doc/stable/user/basics.interoperability.html#the-array-method.
        """
        if len(args) > 0:
            return np.array(self.__array__(), *args, **kwargs)
        return np.frombuffer(self.data, dtype=self.data_type.np, **kwargs)

    def to_numpy(self, *args, **kwargs) -> np.ndarray:
        """Converts the Series to a numpy array. This is necessary for matplotlib
        interop.
        """
        return self.__array__(*args, **kwargs)

    def __getitem__(self, index: int) -> float:
        if self.data_type == DataType.UUID:
            start = self.data_type.density.sample_count(index)
            end = start + self.data_type.density + 1
            d = self.data[start:end]
            return uuid.UUID(bytes=d)
        return self.__array__()[index]

    def __iter__(self):
        return iter(self.__array__())

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


TypedCrudeSeries = Series | pd.Series | np.ndarray
CrudeSeries = Series | bytes | pd.Series | np.ndarray | list | float | int | TimeStamp


def elapsed_seconds(d: np.ndarray) -> np.ndarray:
    """Converts a Series of timestamps to elapsed seconds since the first timestamp.

    :param d: A Series of timestamps.
    :returns: A Series of elapsed seconds.
    """
    return (d - d[0]) / TimeSpan.SECOND
