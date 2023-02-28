#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from datetime import datetime, timedelta, timezone, tzinfo
from typing import TypeAlias, Union, get_args, no_type_check
from numpy.typing import DTypeLike

import numpy as np
import pandas as pd
from freighter import Payload
from pydantic import BaseModel, ValidationError

from .exceptions import ContiguityError


class TimeStamp(int):
    """TimeStamp represents a 64 bit nanosecond-precision UTC timestamp. The TimeStamp
    constructor accepts a variety of types and will attempt to convert them to a
    TimeStamp. The following types are supported:

    * TimeStamp - returns the TimeStamp.
    * TimeSpan - treats the TimeSpan as a duration since the Unix epoch in UTC.
    * pd.TimeStamp - converts the pandas TimeStamp to a TimeStamp. If the timestamp is
    not timezone aware, it is assumed to be in the local timezone.
    * datetime - converts the datetime to a TimeStamp.  If the datetime is not timezone
    aware, it is assumed to be in the local timezone.
    * timedelta - treats the timedelta as a duration since the Unix epoch in UTC.
    * np.datetime64 - treats the numpy datetime64 as a duration since the Unix epoch in
    UTC.
    * int - treats the int as a nanosecond duration since the Unix epoch and in UTC.
    TimeStamp.

    :param value: An unparsed timestamp value that can be converted to a TimeStamp.
    """

    def __new__(cls, value: UnparsedTimeStamp, *args, **kwargs):
        if isinstance(value, TimeStamp):
            return value
        if isinstance(value, TimeSpan):
            value = int(value)
        elif isinstance(value, pd.Timestamp):
            # Convert the timestamp to a timezone aware datetime
            value = int(
                float(TimeSpan.SECOND) * value.to_pydatetime().astimezone().timestamp()
            )
        elif isinstance(value, datetime):
            value = int(float(TimeSpan.SECOND) * value.timestamp())
        elif isinstance(value, timedelta):
            value = int(float(TimeSpan.SECOND) * value.total_seconds())
        elif isinstance(value, np.datetime64):
            # Assume the datetime64 is in UTC
            value = int(pd.Timestamp(value).asm8.view(np.int64))
        elif isinstance(value, np.int64) or isinstance(value, np.float64):
            value = int(value)
        elif isinstance(value, int):
            return super().__new__(cls, int(value))
        else:
            raise TypeError(f"Cannot convert {type(value)} to TimeStamp")

        return super().__new__(cls, value, *args, **kwargs)

    def __init__(self, value: UnparsedTimeStamp, *args, **kwargs):
        pass

    @classmethod
    def __get_validators__(cls):
        yield cls.validate

    @classmethod
    def validate(cls, v):
        return cls(v)

    @classmethod
    def now(cls) -> TimeStamp:
        """:returns : the current time as a TimeStamp."""
        return TimeStamp(datetime.now())

    @classmethod
    def since(cls, ts: UnparsedTimeStamp) -> TimeSpan:
        """: returns the amount of time elapsed since the given TimeStamp."""
        return TimeStamp.now().span(ts)

    def span(self, other: UnparsedTimeStamp) -> TimeSpan:
        """Returns the TimeSpan between two timestamps. This span is guaranteed to be
        positive.
        """
        return TimeRange(self, other).make_valid().span()

    def datetime(self, tzinfo: tzinfo | None = None) -> datetime:
        """Returns the TimeStamp represented as a timezone aware datetime object.

        :param tzinfo: the timezone to use for the datetime. If None, the local timezone
        is used.
        :return: a datetime object
        """
        return (
            datetime.utcfromtimestamp(self / TimeSpan.SECOND)
            .replace(tzinfo=timezone.utc)
            .astimezone(tzinfo)
        )

    def is_zero(self) -> bool:
        """Checks if the timestamp is the Unix epoch.
        :return: True if the TimeStamp is zero, False otherwise
        """
        return self == 0

    def after(self, ts: UnparsedTimeStamp) -> bool:
        """Returns true if the TimeStamp is after the given TimeStamp.
        :param ts: the TimeStamp to compare to
        :return: True if the TimeStamp is after the given TimeStamp, False otherwise
        """
        return super().__gt__(TimeStamp(ts))

    def after_eq(self, ts: UnparsedTimeStamp) -> bool:
        """Returns true if the TimeStamp is after or equal to the given TimeStamp.
        :param ts: the TimeStamp to compare to
        :return: True if the TimeStamp is after or equal to the given TimeStamp an d False
        otherwise.
        """
        return super().__ge__(TimeStamp(ts))

    def before(self, ts: UnparsedTimeStamp) -> bool:
        """Returns true if the TimeStamp is before the given TimeStamp.
        :param ts: the TimeStamp to compare to
        :return: True if the TimeStamp is before the given TimeStamp, False otherwise
        """
        return super().__lt__(TimeStamp(ts))

    def before_eq(self, ts: UnparsedTimeStamp) -> bool:
        """Returns true if the TimeStamp is before or equal to the given TimeStamp.
        :param ts: the TimeStamp to compare to
        :return: True if the TimeStamp is before or equal to the given TimeStamp, and False
        otherwise.
        """
        return super().__le__(TimeStamp(ts))

    def span_range(self, span: TimeSpan) -> TimeRange:
        """Returns a TimeRange that spans the given TimeSpan.
        :param span: the TimeSpan to span
        :return: a TimeRange that spans the given TimeSpan
        """
        return TimeRange(self, self + span).make_valid()

    def range(self, ts: UnparsedTimeStamp) -> TimeRange:
        """Returns a new TimeRange spanning the provided time stamps
        :param ts: the second time stamp
        :return: a new TimeRange spanning the provided time stamps
        """
        return TimeRange(self, TimeStamp(ts)).make_valid()

    def add(self, ts: UnparsedTimeStamp) -> TimeStamp:
        """Returns a new TimeStamp that is the sum of the two TimeStamps.
        :param ts: the second TimeStamp
        :return: a new TimeStamp that is the sum of the two TimeStamps
        """
        return TimeStamp(super().__add__(TimeStamp(ts)))

    def sub(self, ts: UnparsedTimeStamp) -> TimeStamp:
        """Returns a new TimeStamp that is the difference of the two TimeStamps.
        :param ts: the second TimeStamp
        :return: a new TimeStamp that is the difference of the two TimeStamps
        """
        return TimeStamp(super().__sub__(TimeStamp(ts)))

    def __add__(self, rhs: UnparsedTimeStamp) -> TimeStamp:
        return self.add(rhs)

    def __sub__(self, rhs: UnparsedTimeStamp) -> TimeStamp:
        return self.sub(rhs)

    def __lt__(self, rhs: UnparsedTimeStamp) -> bool:
        return self.before(rhs)

    def __le__(self, rhs: UnparsedTimeStamp) -> bool:
        return self.before_eq(rhs)

    def __ge__(self, rhs: UnparsedTimeStamp) -> bool:
        return self.after_eq(rhs)

    def __gt__(self, rhs: UnparsedTimeStamp) -> bool:
        return self.after(rhs)

    def __eq__(self, rhs: object) -> bool:
        if isinstance(rhs, get_args(UnparsedTimeStamp)):
            return super().__eq__(TimeStamp(rhs))
        return NotImplemented

    def __str__(self) -> str:
        return self.datetime().isoformat()

    MIN: TimeStamp
    MAX: TimeStamp


class TimeSpan(int):
    """TimeSpan represents a 64 bit nanosecond-precision duration. The TimeSpan constructor
    accepts a variety of different types and will attempt to convert them to a TimeSpan.
    The following types are supported:

    * int: the number of nanoseconds.
    * np.int64: the number of nanoseconds.
    * TimeSpan: returns a copy of the TimeSpan.
    * TimeStamp: the difference between the TimeStamp and the Unix epoch
    * timedelta: the duration of the timedelta.
    * pands.Timedelta: the duration of the Timedelta.
    * np.timedelta64: the duration of the timedelta.
    """

    def __new__(cls, value: UnparsedTimeSpan, *args, **kwargs):
        if isinstance(value, int):
            return super().__new__(cls, value)
        elif isinstance(value, TimeSpan):
            return value

        if isinstance(value, timedelta):
            value = int(float(TimeSpan.SECOND) * value.total_seconds())
        elif isinstance(value, pd.Timedelta):
            value = int(float(TimeSpan.SECOND) * value.total_seconds())
        elif isinstance(value, np.timedelta64):
            value = int(float(TimeSpan.SECOND) * pd.Timedelta(value).total_seconds())
        elif isinstance(value, np.int64):
            value = int(value)
        else:
            raise TypeError(f"Cannot convert {type(value)} to TimeSpan")

        return super().__new__(cls, value)

    def __init__(self, value: UnparsedTimeSpan, *args, **kwargs):
        pass

    @classmethod
    def __get_validators__(cls):
        yield cls.validate

    @classmethod
    def validate(cls, value):
        return cls(value)

    def __repr__(self) -> str:
        return f"TimeSpan({super().__repr__()})"

    def delta(self) -> timedelta:
        """Returns the TimeSpan represented as a timedelta object.
        :return: a timedelta object
        """
        return timedelta(seconds=self.seconds())

    def seconds(self) -> float:
        """Returns the TimeSpan represented as a number of seconds.
        :return: a number of seconds
        """
        return float(self / TimeSpan.SECOND)

    def is_zero(self) -> bool:
        """Returns true if the TimeSpan is zero.
        :return: True if the TimeSpan is zero, False otherwise
        """
        return self == 0

    def add(self, ts: UnparsedTimeSpan) -> TimeSpan:
        """Returns a new TimeSpan that is the sum of the two TimeSpans.
        :param ts: the second TimeSpan
        :return: a new TimeSpan that is the sum of the two TimeSpans
        """
        return TimeSpan(super().__add__(TimeSpan(ts)))

    def sub(self, ts: UnparsedTimeSpan) -> TimeSpan:
        """Returns a new TimeSpan that is the difference of the two TimeSpans.
        :param ts: the second TimeSpan
        :return: a new TimeSpan that is the difference of the two TimeSpans
        """
        return TimeSpan(super().__sub__(TimeSpan(ts)))

    def __add__(self, other: UnparsedTimeSpan) -> TimeSpan:
        return self.add(other)

    def __sub__(self, other: UnparsedTimeSpan) -> TimeSpan:
        return self.sub(other)

    def __mul__(self, other: UnparsedTimeSpan) -> TimeSpan:
        return TimeSpan(super().__mul__(TimeSpan(other)))

    def __rmul__(self, other: UnparsedTimeSpan) -> TimeSpan:
        return self.__mul__(other)

    def __gt__(self, other: UnparsedTimeSpan) -> bool:
        return super().__gt__(TimeSpan(other))

    def __ge__(self, other: UnparsedTimeSpan) -> bool:
        return super().__ge__(TimeSpan(other))

    def __lt__(self, other: UnparsedTimeSpan) -> bool:
        return super().__lt__(TimeSpan(other))

    def __le__(self, other: UnparsedTimeSpan) -> bool:
        return super().__le__(TimeSpan(other))

    def __eq__(self, other: object) -> bool:
        if not isinstance(other, get_args(UnparsedTimeSpan)):
            return NotImplemented
        return super().__eq__(TimeSpan(other))

    NANOSECOND: TimeSpan
    NANOSECOND_UNITS: str
    MICROSECOND: TimeSpan
    MICROSECOND_UNITS: str
    MILLISECOND: TimeSpan
    MILLISECOND_UNITS: str
    SECOND: TimeSpan
    SECOND_UNITS: str
    MINUTE: TimeSpan
    MINUTE_UNITS: str
    HOUR: TimeSpan
    HOUR_UNITS: str
    DAY: TimeSpan
    DAY_UNITS: str
    MAX: TimeSpan
    UNITS: dict[str, TimeSpan]


TimeSpan.NANOSECOND = TimeSpan(1)
TimeSpan.NANOSECOND_UNITS = "ns"
TimeSpan.MICROSECOND = TimeSpan(1000) * TimeSpan.NANOSECOND
TimeSpan.MICROSECOND_UNITS = "us"
TimeSpan.MILLISECOND = TimeSpan(1000) * TimeSpan.MICROSECOND
TimeSpan.MILLISECOND_UNITS = "ms"
TimeSpan.SECOND = TimeSpan(1000) * TimeSpan.MILLISECOND
TimeSpan.SECOND_UNITS = "s"
TimeSpan.MINUTE = TimeSpan(60) * TimeSpan.SECOND
TimeSpan.MINUTE_UNITS = "m"
TimeSpan.HOUR = TimeSpan(60) * TimeSpan.MINUTE
TimeSpan.HOUR_UNITS = "h"
TimeSpan.MAX = TimeSpan(0xFFFFFFFFFFFFFFFF)
TimeSpan.UNITS = {
    TimeSpan.NANOSECOND_UNITS: TimeSpan.NANOSECOND,
    TimeSpan.MICROSECOND_UNITS: TimeSpan.MICROSECOND,
    TimeSpan.MILLISECOND_UNITS: TimeSpan.MILLISECOND,
    TimeSpan.SECOND_UNITS: TimeSpan.SECOND,
    TimeSpan.MINUTE_UNITS: TimeSpan.MINUTE,
    TimeSpan.HOUR_UNITS: TimeSpan.HOUR,
}
TimeStamp.MIN = TimeStamp(0)
TimeStamp.MAX = TimeStamp(2**63 - 1)


def convert_time_units(data: np.ndarray, _from: str, to: str):
    """Converts the data from one time unit to another.
    :param data: the data to convert
    :param _from: the units of the data
    :param to: the units to convert to
    :return: the data in the new units
    """
    if _from == to:
        return data
    f = TimeSpan.UNITS.get(_from, None)
    if f is None:
        raise ValueError(f"Invalid input time unit {_from}")
    t = TimeSpan.UNITS.get(to, None)
    if t is None:
        raise ValueError(f"Invalid output time unit {to}")
    converted = data * f / t
    if to == TimeSpan.SECOND_UNITS:
        return converted.astype(np.float64)
    if to == TimeSpan.NANOSECOND_UNITS:
        return converted.astype(np.int64)
    return converted


class Rate(float):
    """Rate represents a data rate in Hz"""

    def __new__(cls, value: UnparsedRate):
        if isinstance(value, float):
            return super().__new__(cls, value)
        if isinstance(value, Rate):
            return value
        if isinstance(value, TimeSpan):
            value = 1 / value.seconds()
        elif isinstance(value, int):
            value = float(value)
        else:
            raise TypeError(f"Cannot convert {type(value)} to Rate")
        return super().__new__(cls, value)

    def __init__(self, value: UnparsedRate):
        pass

    @classmethod
    def __get_validators__(cls):
        yield cls.validate

    @classmethod
    def validate(cls, v):
        return cls(v)

    def period(self) -> TimeSpan:
        """Returns the period of the rate as a TimeSpan"""
        return TimeSpan(int(1 / self * float(TimeSpan.SECOND)))

    def sample_count(self, time_span: UnparsedTimeSpan) -> int:
        """Returns the number of samples in the given TimeSpan at this rate"""
        return int(TimeSpan(time_span).seconds() * self)

    def byte_size(self, time_span: UnparsedTimeSpan, density: Density) -> Size:
        """Calculates the amount of bytes occupied by the given TimeSpan at the given
        rate and sample density."""
        return Size(self.sample_count(time_span) * int(density))

    def span(self, sample_count: int) -> TimeSpan:
        """Returns the TimeSpan that corresponds to the given number of samples at this
        rate."""
        return self.period() * sample_count

    def size_span(self, size: Size, density: Density) -> TimeSpan:
        """Returns the TimeSpan that corresponds to the given number of bytes at this
        rate and sample density."""
        if size % density != 0:
            raise ContiguityError(f"Size {size} is not a multiple of density {density}")
        return self.span(int(size / density))

    def __str__(self):
        return str(int(self)) + "Hz"

    def __repr__(self):
        return f"Rate({super().__repr__()} Hz)"

    def __mul__(self, other: UnparsedRate) -> Rate:
        return Rate(super().__mul__(Rate(other)))

    HZ: Rate
    KHZ: Rate
    MHZ: Rate


Rate.HZ = Rate(1)
Rate.KHZ = Rate(1000) * Rate.HZ
Rate.MHZ = Rate(1000) * Rate.KHZ


class TimeRange(BaseModel):
    start: TimeStamp
    end: TimeStamp

    def __init__(self, start: UnparsedTimeStamp, end: UnparsedTimeStamp):
        super().__init__(start=TimeStamp(start), end=TimeStamp(end))

    @classmethod
    def __get_validators__(cls):
        yield cls.validate

    @classmethod
    def validate(cls, v):
        if isinstance(v, TimeRange):
            return cls(v.start, v.end)
        elif isinstance(v, dict):
            return cls(**v)
        return cls(start=v[0], end=v[1])

    def span(self) -> TimeSpan:
        return TimeSpan(self.end - self.start)

    def make_valid(self) -> TimeRange:
        if not self.is_valid():
            return self.swap()
        return self

    def is_zero(self) -> bool:
        return self.span().is_zero()

    def bound_by(self, otr: TimeRange) -> TimeRange:
        if otr.start.after(self.start):
            self.start = otr.start
        if otr.start.after(self.end):
            self.end = otr.start
        if otr.end.before(self.end):
            self.end = otr.end
        if otr.end.before(self.start):
            self.start = otr.end
        return self

    def contains_stamp(self, ts: TimeStamp) -> bool:
        return ts.after_eq(self.start) and ts.before(self.end)

    def contains_range(self, tr: TimeRange) -> bool:
        return self.start.before_eq(tr.start) and self.end.after_eq(tr.end)

    def overlaps_with(self, tr: TimeRange) -> bool:
        return (
            self.contains_stamp(tr.start)
            or self.contains_stamp(tr.end)
            or tr.contains_range(self)
        )

    def swap(self) -> TimeRange:
        self.start, self.end = self.end, self.start
        return self

    def is_valid(self) -> bool:
        return self.span() >= TimeSpan(0)

    def __str__(self):
        return str(self.start) + " - " + str(self.end)

    # Assigning these values to None prevents pydantic
    # from throwing a nasty missing attribute error.
    MAX: TimeRange = None  # type: ignore
    MIN: TimeRange = None  # type: ignore
    ZERO: TimeRange = None  # type: ignore


TimeRange.MAX = TimeRange(TimeStamp.MIN, TimeStamp.MAX)
TimeRange.MIN = TimeRange(TimeStamp.MAX, TimeStamp.MIN)
TimeRange.ZERO = TimeRange(0, 0)


class Density(int):
    def __new__(cls, value: UnparsedDensity):
        if isinstance(value, Density):
            return value
        if isinstance(value, int):
            return super().__new__(cls, value)
        raise TypeError(f"Cannot convert {type(value)} to Density")

    def __init__(self, value):
        pass

    @classmethod
    def __get_validators__(cls):
        yield cls.validate

    @classmethod
    def validate(cls, v):
        return cls(v)

    def __repr__(self):
        return f"Density({super().__repr__()})"

    UNKNOWN: Density
    BIT64: Density
    BIT32: Density
    BIT16: Density
    BIT8: Density


Density.UNKNOWN = Density(0)
Density.BIT64 = Density(8)
Density.BIT32 = Density(4)
Density.BIT16 = Density(2)
Density.BIT8 = Density(1)


class Size(int):
    def __str__(self):
        return super(Size, self).__str__() + "B"

    def __mul__(self, other: UnparsedSize):
        return Size(super(Size, self).__mul__(Size(other)))

    BYTE: Size
    KILOBYTE: Size
    MEGABYTE: Size
    GIGABYTE: Size


Size.BYTE = Size(1)
Size.KILOBYTE = Size(1024) * Size.BYTE
Size.MEGABYTE = Size(1024) * Size.KILOBYTE
Size.GIGABYTE = Size(1024) * Size.MEGABYTE


class DataType(str):
    """DataType represents a data type as a string"""

    def __new__(cls, value: UnparsedDataType):
        if isinstance(value, DataType):
            return value
        if isinstance(value, str):
            return super().__new__(cls, value)
        if np.issctype(value):
            value = DataType._FROM_NUMPY.get(np.dtype(value), None)
            if value is not None:
                return value
        raise TypeError(f"Cannot convert {type(value)} to DataType")

    def __init__(self, value: UnparsedDataType):
        pass

    @classmethod
    def __get_validators__(cls):
        yield cls.validate

    @classmethod
    def validate(cls, v):
        return cls(v)

    @classmethod
    def __modify_schema__(cls, field_schema):
        field_schema.update(type="string")

    @property
    def np(self) -> np.dtype:
        """Converts a built-in DataType to a numpy type Scalar Type
        :param _raise: If True, raises a TypeError if the DataType is not a numpy type.
        :return: The numpy type
        """
        npt = DataType._TO_NUMPY.get(self, None)
        if npt is None:
            raise TypeError(f"Cannot convert {self} to numpy type")
        return npt

    @property
    def density(self) -> Density:
        return DataType._DENSITIES.get(self, Density.UNKNOWN)

    def __repr__(self):
        return f"DataType({super().__repr__()})"

    def string(self):
        return str(self)

    UNKNOWN: DataType
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
    ALL: tuple[DataType, ...]
    _TO_NUMPY: dict[DataType, np.dtype]
    _FROM_NUMPY: dict[np.dtype, DataType]
    _DENSITIES: dict[DataType, Density]


DataType.UNKNOWN = DataType("")
DataType.TIMESTAMP = DataType("timestamp")
DataType.FLOAT64 = DataType("float64")
DataType.FLOAT32 = DataType("float32")
DataType.INT64 = DataType("int64")
DataType.INT32 = DataType("int32")
DataType.INT16 = DataType("int16")
DataType.INT8 = DataType("int8")
DataType.UINT64 = DataType("uint64")
DataType.UINT32 = DataType("uint32")
DataType.UINT16 = DataType("uint16")
DataType.UINT8 = DataType("uint8")
DataType.ALL = (
    DataType.FLOAT64,
    DataType.FLOAT32,
    DataType.INT64,
    DataType.INT32,
    DataType.INT16,
    DataType.INT8,
    DataType.UINT64,
    DataType.UINT32,
    DataType.UINT16,
    DataType.UINT8,
)

UnparsedTimeStamp: TypeAlias = Union[
    int,
    TimeStamp,
    TimeSpan,
    datetime,
    timedelta,
    np.datetime64,
    np.int64,
]
UnparsedTimeSpan: TypeAlias = Union[
    int | TimeSpan | TimeStamp,
    timedelta,
    np.timedelta64,
]
UnparsedRate: TypeAlias = int | float | TimeSpan | Rate
UnparsedDensity: TypeAlias = Density | int
UnparsedDataType: TypeAlias = DTypeLike | DataType | str
UnparsedSize: TypeAlias = int | Size

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
}
DataType._FROM_NUMPY = {v: k for k, v in DataType._TO_NUMPY.items()}
DataType._DENSITIES = {
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
}


class ArrayHeader(Payload):
    time_range: TimeRange | None = None
    data_type: DataType


class BinaryArray(ArrayHeader):
    data: bytes = b""


class NumpyArray(ArrayHeader):
    data: np.ndarray

    class Config:
        arbitrary_types_allowed = True

    @classmethod
    def from_binary(cls, arr: BinaryArray) -> NumpyArray:
        return NumpyArray(
            data_type=arr.data_type,
            time_range=arr.time_range,
            data=np.frombuffer(arr.data, dtype=arr.data_type.np),
        )

    def to_binary(self) -> BinaryArray:
        return BinaryArray(
            data_type=self.data_type,
            time_range=self.time_range,
            data=self.data.tobytes(),
        )
