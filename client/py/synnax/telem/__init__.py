from __future__ import annotations

import datetime
from dataclasses import dataclass
import numpy as np
import pandas as pd
from typing import get_args
from typing import Union

import synnax.errors

_EPOCH = datetime.datetime.utcfromtimestamp(0)


class TimeStamp(int):
    """TimeStamp represents a 64 bit nanosecond-precision UTC timestamp.

    :param value: The nanosecond value of the timestamp
    """

    def __new__(cls, value: UnparsedTimeStamp, *args, **kwargs):
        if isinstance(value, TimeStamp):
            return value

        if isinstance(value, TimeSpan):
            value = int(value)
        elif isinstance(value, pd.Timestamp):
            value = int(value.asm8.view(np.int64))
        elif isinstance(value, datetime.datetime):
            value = SECOND * int((value - _EPOCH).total_seconds())
        elif isinstance(value, datetime.timedelta):
            value = SECOND * int(value.total_seconds())
        elif isinstance(value, np.datetime64):
            value = int(pd.Timestamp(value).asm8.view(np.int64))
        elif isinstance(value, int):
            return super().__new__(cls, int(value))
        else:
            raise TypeError(f"Cannot convert {type(value)} to TimeStamp")

        return super().__new__(cls, value)

    def __init__(self, value: UnparsedTimeStamp, *args, **kwargs):
        pass

    def time(self) -> datetime.datetime:
        """Returns the TimeStamp represented as a datetime.datetime object.
        :return: a datetime.datetime object
        """
        return datetime.datetime.utcfromtimestamp(self / SECOND)

    def is_zero(self) -> bool:
        """Returns true if the TimeStamp is zero.
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
        rng = TimeRange(self, self + span)
        if not rng.is_valid():
            rng = rng.swap()
        return rng

    def range(self, ts: UnparsedTimeStamp) -> TimeRange:
        """Returns a new TimeRange spanning the provided time stamps
        :param ts: the second time stamp
        :return: a new TimeRange spanning the provided time stamps
        """
        return TimeRange(self, TimeStamp(ts))

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


def now() -> TimeStamp:
    return TimeStamp(datetime.datetime.now())


class TimeSpan(int):
    """TimeSpan represents a 64 bit nanosecond-precision duration."""

    def __new__(cls, value: UnparsedTimeSpan, *args, **kwargs):
        if isinstance(value, int):
            return super().__new__(cls, value)
        elif isinstance(value, TimeSpan):
            return value

        if isinstance(value, datetime.timedelta):
            value = int(float(SECOND) * value.total_seconds())
        return super().__new__(cls, value)

    def __init__(self, value: UnparsedTimeSpan, *args, **kwargs):
        pass

    def delta(self) -> datetime.timedelta:
        """Returns the TimeSpan represented as a datetime.timedelta object.
        :return: a datetime.timedelta object
        """
        return datetime.timedelta(seconds=self.seconds())

    def seconds(self) -> float:
        """Returns the TimeSpan represented as a number of seconds.
        :return: a number of seconds
        """
        return float(self / SECOND)

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


TIME_STAMP_MIN = TimeStamp(0)
TIME_STAMP_MAX = TimeStamp(0xFFFFFFFFFFFFFFFF)
NANOSECOND = TimeSpan(1)
MICROSECOND = TimeSpan(1000) * NANOSECOND
MILLISECOND = TimeSpan(1000) * MICROSECOND
SECOND = TimeSpan(1000) * MILLISECOND
MINUTE = TimeSpan(60) * SECOND
HOUR = TimeSpan(60) * MINUTE
TIME_SPAN_MAX = TimeSpan(0xFFFFFFFFFFFFFFFF)


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

    def period(self) -> TimeSpan:
        """Returns the period of the rate as a TimeSpan"""
        return TimeSpan(int(1 / self * float(SECOND)))

    def sample_count(self, time_span: UnparsedTimeSpan) -> int:
        """Returns the number of samples in the given TimeSpan at this rate"""
        return int(TimeSpan(time_span) / self.period())

    def byte_size(self, time_span: UnparsedTimeSpan, density: Density) -> Size:
        """Calculates the amount of bytes occupied by the given TimeSpan at the given rate and sample density."""
        return Size(self.sample_count(time_span) * int(density))

    def span(self, sample_count: int) -> TimeSpan:
        """Returns the TimeSpan that corresponds to the given number of samples at this rate."""
        return self.period() * sample_count

    def size_span(self, size: Size, density: Density) -> TimeSpan:
        """Returns the TimeSpan that corresponds to the given number of bytes at this rate and sample density."""
        if size % density != 0:
            raise synnax.errors.ContiguityError(
                f"Size {size} is not a multiple of density {density}"
            )
        return self.span(int(size / density))

    def __str__(self):
        return str(int(self)) + "Hz"

    def __repr__(self):
        return str(int(self)) + "Hz"


HZ = Rate(1)
KHZ = Rate(1000) * HZ
MHZ = Rate(1000) * KHZ


@dataclass
class TimeRange:
    start: TimeStamp
    end: TimeStamp

    def __init__(self, start: UnparsedTimeStamp, end: UnparsedTimeStamp):
        self.start = TimeStamp(start)
        self.end = TimeStamp(end)

    def span(self) -> TimeSpan:
        return TimeSpan(self.end - self.start)

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


class Density(int):
    ...

    def __new__(cls, value: UnparsedDensity):
        if isinstance(value, Density):
            return value
        if isinstance(value, int):
            return super().__new__(cls, value)
        raise TypeError(f"Cannot convert {type(value)} to Density")

    def __init__(self, value):
        pass


class Size(int):
    def __str__(self):
        return super(Size, self).__str__() + "B"


TIME_RANGE_MAX = TimeRange(TIME_STAMP_MIN, TIME_STAMP_MAX)
DENSITY_UNKNOWN = Density(0)
BIT64 = Density(8)
BIT32 = Density(4)
BIT16 = Density(2)
BIT8 = Density(1)


class DataType(str):

    def __new__(cls, value: UnparsedDataType):
        if isinstance(value, DataType):
            return value
        if isinstance(value, str):
            return super().__new__(cls, value)
        try:
            if issubclass(value, np.ScalarType):
                return from_numpy_type(value)
        except TypeError:
            pass
        raise TypeError(f"Cannot convert {type(value)} to DataType")

    def __init__(self, value: UnparsedDataType):
        pass


def to_numpy_type(data_type: DataType) -> np.ScalarType:
    return NUMPY_TYPES.get(data_type, None)


def from_numpy_type(np_type: np.ScalarType) -> DataType:
    return DATA_TYPES.get(np_type, None)


DATA_TYPE_UNKNOWN = DataType("")
FLOAT64 = DataType("float64")
FLOAT32 = DataType("float32")
INT64 = DataType("int64")
INT32 = DataType("int32")
INT16 = DataType("int16")
INT8 = DataType("int8")
UINT64 = DataType("uint64")
UINT32 = DataType("uint32")
UINT16 = DataType("uint16")
UINT8 = DataType("uint8")

UnparsedTimeStamp = Union[
    TimeStamp,
    TimeSpan,
    int,
    datetime.datetime,
    datetime.timedelta,
]
UnparsedTimeSpan = Union[TimeSpan | TimeStamp | int | datetime.timedelta]
UnparsedRate = TimeSpan | Rate | float
UnparsedDensity = Density | int
UnparsedDataType = (*np.ScalarType, DataType, str)

NUMPY_TYPES: dict[str, np.ScalarType] = {
    FLOAT64: np.float64,
    FLOAT32: np.float32,
    INT64: np.int64,
    INT32: np.int32,
    INT16: np.int16,
    INT8: np.int8,
    UINT64: np.uint64,
    UINT32: np.uint32,
    UINT16: np.uint16,
    UINT8: np.uint8,
}

DATA_TYPES: dict[np.ScalarType, DataType] = {
    np.float64: FLOAT64,
    np.float32: FLOAT32,
    np.int64: INT64,
    np.int32: INT32,
    np.int16: INT16,
    np.int8: INT8,
    np.uint64: UINT64,
    np.uint32: UINT32,
    np.uint16: UINT16,
    np.uint8: UINT8,
}
