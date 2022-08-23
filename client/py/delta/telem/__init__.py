from __future__ import annotations

import datetime
from dataclasses import dataclass
from typing import Any


class TimeStamp(int):
    """TimeStamp represents a 64 bit nanosecond-precision UTC timestamp.

    :param value: The nanosecond value of the timestamp
    """

    def __new__(cls, value: UnparsedTimeStamp, *args, **kwargs):
        t = type(value)
        if t is float:
            value = int(value)
        elif t is datetime.datetime:
            value = int((SECOND * (value - epoch).total_seconds()))
        elif t is datetime.timedelta:
            value = int((SECOND * value.total_seconds()))
        elif t is TimeSpan:
            value = int(value)
        elif t is TimeStamp:
            value = int(value)
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
        return self > TimeStamp(ts)

    def after_eq(self, ts: UnparsedTimeStamp) -> bool:
        """Returns true if the TimeStamp is after or equal to the given TimeStamp.
        :param ts: the TimeStamp to compare to
        :return: True if the TimeStamp is after or equal to the given TimeStamp, False otherwise
        """
        return self >= TimeStamp(ts)

    def before(self, ts: UnparsedTimeStamp) -> bool:
        """Returns true if the TimeStamp is before the given TimeStamp.
        :param ts: the TimeStamp to compare to
        :return: True if the TimeStamp is before the given TimeStamp, False otherwise
        """
        return self < TimeStamp(ts)

    def before_eq(self, ts: UnparsedTimeStamp) -> bool:
        """Returns true if the TimeStamp is before or equal to the given TimeStamp.
        :param ts: the TimeStamp to compare to
        :return: True if the TimeStamp is before or equal to the given TimeStamp, False otherwise
        """
        return self <= TimeStamp(ts)

    def span_range(self, span: TimeSpan) -> TimeRange:
        """Returns a TimeRange that spans the given TimeSpan.
        :param span: the TimeSpan to span
        :return: a TimeRange that spans the given TimeSpan
        """
        rng = TimeRange(self, self + span)
        if not rng.is_valid():
            rng = rng.swap()
        return rng

    def range(self, ts: TimeStamp) -> TimeRange:
        """Returns a new TimeRange spanning the provided time stamps
        :param ts: the second time stamp
        :return: a new TimeRange spanning the provided time stamps
        """
        return TimeRange(self, ts)

    def add(self, ts: UnparsedTimeStamp) -> TimeStamp:
        """Returns a new TimeStamp that is the sum of the two TimeStamps.
        :param ts: the second TimeStamp
        :return: a new TimeStamp that is the sum of the two TimeStamps
        """
        return TimeStamp(int(self) + int(TimeStamp(ts)))

    def sub(self, ts: UnparsedTimeStamp) -> TimeStamp:
        """Returns a new TimeStamp that is the difference of the two TimeStamps.
        :param ts: the second TimeStamp
        :return: a new TimeStamp that is the difference of the two TimeStamps
        """
        return TimeStamp(int(self) - TimeStamp(ts))

    def __add__(self, other: UnparsedTimeStamp) -> TimeStamp:
        return self.add(other)

    def __sub__(self, other: UnparsedTimeStamp) -> TimeStamp:
        return self.sub(other)


epoch = datetime.datetime.utcfromtimestamp(0)


def now() -> TimeStamp:
    return TimeStamp(datetime.datetime.now())


class TimeSpan(int):
    """TimeSpan represents a 64 bit nanosecond-precision duration.
    """

    def __new__(cls, value: UnparsedTimeSpan, *args, **kwargs):
        t = type(value)
        if t is TimeStamp:
            value = int(value)
        elif t is datetime.timedelta:
            value = int(SECOND) * value.total_seconds()
        elif t is float:
            value = int(value)
        elif t is TimeSpan:
            value = value
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
        return TimeSpan(int(self) + int(TimeSpan(ts)))

    def sub(self, ts: UnparsedTimeSpan) -> TimeSpan:
        """Returns a new TimeSpan that is the difference of the two TimeSpans.
        :param ts: the second TimeSpan
        :return: a new TimeSpan that is the difference of the two TimeSpans
        """
        return TimeSpan(int(self) - int(TimeSpan(ts)))

    def byte_size(self, data_rate: Rate, density: Density) -> int:
        return self / data_rate.period() * int(density)

    def __add__(self, other: UnparsedTimeSpan) -> TimeSpan:
        return self.add(other)

    def __sub__(self, other: UnparsedTimeSpan) -> TimeSpan:
        return self.sub(other)

    def __mul__(self, other: UnparsedTimeSpan) -> TimeSpan:
        return TimeSpan(int(TimeSpan(other)) * int(self))

    def __rmul__(self, other: UnparsedTimeSpan) -> TimeSpan:
        return self.__mul__(other)

    def __gt__(self, other: UnparsedTimeSpan) -> bool:
        return int(self) > int(TimeSpan(other))

    def __ge__(self, other) -> bool:
        return int(self) >= int(TimeSpan(other))

    def __lt__(self, other) -> bool:
        return int(self) < int(TimeSpan(other))

    def __le__(self, other: UnparsedTimeSpan) -> bool:
        return int(self) <= int(TimeSpan(other))

    def __eq__(self, other: UnparsedTimeSpan) -> bool:
        return int(self) == int(TimeSpan(other))

    def __truediv__(self, other: UnparsedTimeSpan) -> TimeSpan:
        return TimeSpan(int(self) / int(TimeSpan(other)))


TIME_STAMP_MIN = TimeStamp(0)
TIME_STAMP_MAX = TimeStamp(0xFFFFFFFFFFFFFFFF)
NANOSECOND = TimeSpan(1)
MICROSECOND = TimeSpan(1000) * NANOSECOND
MILLISECOND = TimeSpan(1000) * MICROSECOND
SECOND = TimeSpan(1000) * MILLISECOND
MINUTE = TimeSpan(60) * SECOND
HOUR = TimeSpan(60) * MINUTE


class Rate(float):

    def __new__(cls, value: UnparsedRate):
        t = type(value)
        if t is int:
            value = float(value)
        elif t is Rate:
            value = float(value)
        elif t is TimeSpan:
            value = float(1 / value.seconds())
        return super().__new__(cls, value)

    def __init__(self, value: UnparsedRate):
        pass

    def period(self) -> TimeSpan:
        return TimeSpan(1) / self * SECOND

    def sample_count(self, time_span: TimeSpan) -> int:
        return int(time_span.seconds() * self)

    def span(self, sample_count: int) -> TimeSpan:
        return self.period() * TimeSpan(sample_count)

    def size_span(self, size: Size, density: Density) -> TimeSpan:
        return self.span(size * density)

    def __mul__(self, other):
        return Rate(float(self) * float(Rate(other)))

    def __truediv__(self, other):
        return Rate(float(self) / float(other))

    def __rmul__(self, other):
        return self.__mul__(other)

    def __str__(self):
        return str(int(self)) + "Hz"

    def __repr__(self):
        return str(int(self)) + "Hz"


HZ = Rate(1)
KHZ = Rate(1000) * HZ
MHZ = Rate(1000) * KHZ


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
        return self.contains_stamp(tr.start) or self.contains_stamp(tr.end) or tr.contains_range(self)

    def swap(self) -> TimeRange:
        self.start, self.end = self.end, self.start
        return self

    def is_valid(self) -> bool:
        return self.span() >= TimeSpan(0)


class Density(int):
    ...

    def __new__(cls, value):
        return super().__new__(cls, value)

    def __init__(self, value):
        pass


class Size(int):

    def __str__(self):
        return str(self) + "B"

    def __mul__(self, other):
        return self * other


DENSITY_UNKNOWN = Density(0)
BIT_64 = Density(8)
BIT_32 = Density(4)
BIT_16 = Density(2)
BIT_8 = Density(1)


@dataclass
class DataType:
    key: str
    density: Density

    def __str__(self):
        return self.key

    def __eq__(self, other):
        return self.key == other.key and self.density == other.density


DATA_TYPE_UNKNOWN = DataType("", DENSITY_UNKNOWN)
FLOAT_64 = DataType("float64", BIT_64)
FLOAT_32 = DataType("float32", BIT_32)
INT_64 = DataType("int64", BIT_64)
INT_32 = DataType("int32", BIT_32)
INT_16 = DataType("int16", BIT_16)
INT_8 = DataType("int8", BIT_8)
UINT_64 = DataType("uint64", BIT_64)
UINT_32 = DataType("uint32", BIT_32)
UINT_16 = DataType("uint16", BIT_16)
UINT_8 = DataType("uint8", BIT_8)

UnparsedTimeStamp = TimeSpan | TimeStamp | int | float | datetime.datetime | datetime.timedelta
UnparsedTimeSpan = TimeSpan | TimeStamp | int | float | datetime.timedelta
UnparsedRate = TimeSpan | Rate | int | float
