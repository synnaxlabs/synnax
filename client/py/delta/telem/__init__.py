from __future__ import annotations

import datetime
from dataclasses import dataclass


@dataclass
class TimeStamp:
    """TimeStamp represents a 64 bit nanosecond-precision UTC timestamp.

    :param value: The nanosecond value of the timestamp
    """
    value: int

    def __init__(self, value: UnparsedTimeStamp):
        t = type(value)
        if t is int:
            self.value = value
            return
        if t is float:
            self.value = int(value)
            return
        if t is datetime.datetime:
            self.value = int((SECOND * (value - epoch).total_seconds()).value)
            return
        if t is datetime.timedelta:
            self.value = int((SECOND * value.total_seconds()).value)
            return
        if t is TimeSpan:
            self.value = int(value.value)
            return
        if t is TimeStamp:
            self.value = value.value
            return

        raise TypeError(f"Unable to convert {value} to a TimeStamp")

    def time(self) -> datetime.datetime:
        """Returns the TimeStamp represented as a datetime.datetime object.
        :return: a datetime.datetime object
        """
        return datetime.datetime.utcfromtimestamp(self.value / SECOND.value)

    def is_zero(self) -> bool:
        """Returns true if the TimeStamp is zero.
        :return: True if the TimeStamp is zero, False otherwise
        """
        return self.value == 0

    def after(self, ts: UnparsedTimeStamp) -> bool:
        """Returns true if the TimeStamp is after the given TimeStamp.
        :param ts: the TimeStamp to compare to
        :return: True if the TimeStamp is after the given TimeStamp, False otherwise
        """
        return self.value > TimeStamp(ts).value

    def after_eq(self, ts: UnparsedTimeStamp) -> bool:
        """Returns true if the TimeStamp is after or equal to the given TimeStamp.
        :param ts: the TimeStamp to compare to
        :return: True if the TimeStamp is after or equal to the given TimeStamp, False otherwise
        """
        return self.value >= TimeStamp(ts).value

    def before(self, ts: UnparsedTimeStamp) -> bool:
        """Returns true if the TimeStamp is before the given TimeStamp.
        :param ts: the TimeStamp to compare to
        :return: True if the TimeStamp is before the given TimeStamp, False otherwise
        """
        return self.value < TimeStamp(ts).value

    def before_eq(self, ts: UnparsedTimeStamp) -> bool:
        """Returns true if the TimeStamp is before or equal to the given TimeStamp.
        :param ts: the TimeStamp to compare to
        :return: True if the TimeStamp is before or equal to the given TimeStamp, False otherwise
        """
        return self.value <= TimeStamp(ts).value

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
        return TimeStamp(self.value + TimeStamp(ts).value)

    def sub(self, ts: UnparsedTimeStamp) -> TimeStamp:
        """Returns a new TimeStamp that is the difference of the two TimeStamps.
        :param ts: the second TimeStamp
        :return: a new TimeStamp that is the difference of the two TimeStamps
        """
        return TimeStamp(self.value - TimeStamp(ts).value)

    def __add__(self, other: UnparsedTimeStamp) -> TimeStamp:
        return self.add(other)

    def __sub__(self, other: UnparsedTimeStamp) -> TimeStamp:
        return self.sub(other)


epoch = datetime.datetime.utcfromtimestamp(0)


def now() -> TimeStamp:
    return TimeStamp(datetime.datetime.now())


TIME_STAMP_MIN = TimeStamp(0)
TIME_STAMP_MAX = TimeStamp(0xFFFFFFFFFFFFFFFF)


@dataclass
class TimeSpan:
    """TimeSpan represents a 64 bit nanosecond-precision duration.
    """
    value: int

    def __init__(self, value: UnparsedTimeSpan):
        t = type(value)
        if t is int:
            self.value = value
            return
        if t is TimeStamp:
            self.value = value.value
            return
        if t is datetime.timedelta:
            self.value = int(SECOND.value * value.total_seconds())
            return
        if t is float:
            self.value = int(value)
            return
        if t is TimeSpan:
            self.value = value.value
            return
        raise TypeError(f"Unable to convert {value} to a TimeSpan")

    def delta(self) -> datetime.timedelta:
        """Returns the TimeSpan represented as a datetime.timedelta object.
        :return: a datetime.timedelta object
        """
        return datetime.timedelta(seconds=self.seconds())

    def seconds(self) -> float:
        """Returns the TimeSpan represented as a number of seconds.
        :return: a number of seconds
        """
        return self.value / SECOND.value

    def is_zero(self) -> bool:
        """Returns true if the TimeSpan is zero.
        :return: True if the TimeSpan is zero, False otherwise
        """
        return self.value == 0

    def add(self, ts: UnparsedTimeSpan) -> TimeSpan:
        """Returns a new TimeSpan that is the sum of the two TimeSpans.
        :param ts: the second TimeSpan
        :return: a new TimeSpan that is the sum of the two TimeSpans
        """
        return TimeSpan(self.value + TimeSpan(ts).value)

    def sub(self, ts: UnparsedTimeSpan) -> TimeSpan:
        """Returns a new TimeSpan that is the difference of the two TimeSpans.
        :param ts: the second TimeSpan
        :return: a new TimeSpan that is the difference of the two TimeSpans
        """
        return TimeSpan(self.value - TimeSpan(ts).value)

    def byte_size(self, data_rate: Rate, data_type: Density) -> int:
        return (self / data_rate.period() * data_type).value

    def __add__(self, other: UnparsedTimeSpan) -> TimeSpan:
        return self.add(other)

    def __sub__(self, other: UnparsedTimeSpan) -> TimeSpan:
        return self.sub(other)

    def __mul__(self, other: UnparsedTimeSpan) -> TimeSpan:
        return TimeSpan(TimeSpan(other).value * self.value)

    def __rmul__(self, other: UnparsedTimeSpan) -> TimeSpan:
        return self.__mul__(other)

    def __gt__(self, other: UnparsedTimeSpan) -> bool:
        return self.value > TimeSpan(other).value

    def __ge__(self, other) -> bool:
        return self.value >= TimeSpan(other).value

    def __lt__(self, other) -> bool:
        return self.value < TimeSpan(other).value

    def __le__(self, other: UnparsedTimeSpan) -> bool:
        return self.value <= TimeSpan(other).value

    def __eq__(self, other: UnparsedTimeSpan) -> bool:
        return self.value == TimeSpan(other).value

    def __truediv__(self, other: UnparsedTimeSpan) -> TimeSpan:
        return TimeSpan(self.value / TimeSpan(other).value)


NANOSECOND = TimeSpan(1)
MICROSECOND = TimeSpan(1000) * NANOSECOND
MILLISECOND = TimeSpan(1000) * MICROSECOND
SECOND = TimeSpan(1000) * MILLISECOND
MINUTE = TimeSpan(60) * SECOND
HOUR = TimeSpan(60) * MINUTE


@dataclass
class Rate:
    value: float

    def period(self) -> TimeSpan:
        return TimeSpan(1) / self * SECOND

    def sample_count(self, time_span: TimeSpan) -> int:
        return int(time_span.seconds() * self.value)

    def span(self, sample_count: int) -> TimeSpan:
        return self.period() * TimeSpan(sample_count)

    def size_span(self, size: Size, density: Density) -> TimeSpan:
        return self.span(size.value * density.value)

    def __mul__(self, other):
        return Rate(self.value * other.value)

    def __truediv__(self, other):
        return Rate(self.value / other.value)


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
        return TimeSpan((self.end - self.start).value)

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


@dataclass
class Density:
    value: int

    def __mul__(self, other):
        return Density(self.value * other.value)


@dataclass
class Size:
    value: int

    def __str__(self):
        return str(self.value) + "B"

    def __mul__(self, other):
        return Size(self.value * other.value)


UNKNOWN = Density(0)
FLOAT_64 = Density(8)
UINT_64 = Density(8)
INT_64 = Density(8)
FLOAT_32 = Density(4)
INT_32 = Density(4)
UINT_32 = Density(4)
INT_16 = Density(2)
UINT_16 = Density(2)
INT_8 = Density(1)
UINT_8 = Density(1)

UnparsedTimeStamp = TimeSpan | TimeStamp | int | float | datetime.datetime | datetime.timedelta
UnparsedTimeSpan = TimeSpan | TimeStamp | int | float | datetime.timedelta
