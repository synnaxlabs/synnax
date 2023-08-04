#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from math import trunc

from datetime import datetime, timedelta, timezone, tzinfo
from typing import TypeAlias, Union, get_args, TypeVar, overload
from numpy.typing import DTypeLike

import numpy as np
import pandas as pd
from freighter import Payload
from pydantic import BaseModel

from synnax.exceptions import ContiguityError

T = TypeVar("T")


def _validate_options_contains(value: T, options: list[T]) -> None:
    if not any(value == o for o in options):
        raise ValueError(f"""
            Invalid divisor argument for remainder. Divisor must be one of the following
            options:
            {[o.__str__() for o in options]}
            """)


def _semantic_mod(value: T, divisor: T, options: list[T]) -> T:
    _validate_options_contains(divisor, options)
    return value % divisor


def _semantic_trunc(value: T, span: T, options: list[T]) -> T:
    _validate_options_contains(span, options)
    return trunc(value / span) * span


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

    def __new__(cls, value: CrudeTimeStamp, *args, **kwargs):
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
            value = int(pd.Timestamp(value).value)
        elif isinstance(value, np.int64) or isinstance(value, np.float64):
            value = int(value)
        elif isinstance(value, int):
            return super().__new__(cls, int(value))
        else:
            raise TypeError(f"Cannot convert {type(value)} to TimeStamp")

        return super().__new__(cls, value, *args, **kwargs)

    def __init__(self, value: CrudeTimeStamp, *args, **kwargs):
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
    def since(cls, ts: CrudeTimeStamp) -> TimeSpan:
        """:returns the amount of time elapsed since the given TimeStamp."""
        return TimeStamp.now().span(ts)

    def span(self, other: CrudeTimeStamp) -> TimeSpan:
        """:returns: the TimeSpan between two timestamps. This span is guaranteed to be
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

    def after(self, ts: CrudeTimeStamp) -> bool:
        """Returns true if the TimeStamp is after the given TimeStamp.
        :param ts: the TimeStamp to compare to
        :return: True if the TimeStamp is after the given TimeStamp, False otherwise
        """
        return super().__gt__(TimeStamp(ts))

    def after_eq(self, ts: CrudeTimeStamp) -> bool:
        """Returns true if the TimeStamp is after or equal to the given TimeStamp.
        :param ts: the TimeStamp to compare to
        :return: True if the TimeStamp is after or equal to the given TimeStamp an d False
        otherwise.
        """
        return super().__ge__(TimeStamp(ts))

    def before(self, ts: CrudeTimeStamp) -> bool:
        """Returns true if the TimeStamp is before the given TimeStamp.
        :param ts: the TimeStamp to compare to
        :return: True if the TimeStamp is before the given TimeStamp, False otherwise
        """
        return super().__lt__(TimeStamp(ts))

    def before_eq(self, ts: CrudeTimeStamp) -> bool:
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

    def range(self, ts: CrudeTimeStamp) -> TimeRange:
        """Returns a new TimeRange spanning the provided time stamps
        :param ts: the second time stamp
        :return: a new TimeRange spanning the provided time stamps
        """
        return TimeRange(self, TimeStamp(ts)).make_valid()

    def add(self, ts: CrudeTimeStamp) -> TimeStamp:
        """Returns a new TimeStamp that is the sum of the two TimeStamps.
        :param ts: the second TimeStamp
        :return: a new TimeStamp that is the sum of the two TimeStamps
        """
        return TimeStamp(super().__add__(TimeStamp(ts)))

    def sub(self, ts: CrudeTimeStamp) -> TimeStamp:
        """Returns a new TimeStamp that is the difference of the two TimeStamps.
        :param ts: the second TimeStamp
        :return: a new TimeStamp that is the difference of the two TimeStamps
        """
        return TimeStamp(super().__sub__(TimeStamp(ts)))

    def __add__(self, rhs: CrudeTimeStamp) -> TimeStamp:
        return self.add(rhs)

    def __sub__(self, rhs: CrudeTimeStamp) -> TimeStamp:
        return self.sub(rhs)

    def __lt__(self, rhs: CrudeTimeStamp) -> bool:
        return self.before(rhs)

    def __le__(self, rhs: CrudeTimeStamp) -> bool:
        return self.before_eq(rhs)

    def __ge__(self, rhs: CrudeTimeStamp) -> bool:
        return self.after_eq(rhs)

    def __gt__(self, rhs: CrudeTimeStamp) -> bool:
        return self.after(rhs)

    def __eq__(self, rhs: object) -> bool:
        if isinstance(rhs, get_args(CrudeTimeStamp)):
            return super().__eq__(TimeStamp(rhs))
        return NotImplemented

    def __str__(self) -> str:
        return self.datetime().isoformat()

    MIN: TimeStamp
    MAX: TimeStamp
    ZERO: TimeStamp


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

    def __new__(cls, value: CrudeTimeSpan, *args, **kwargs):
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

    def __init__(self, value: CrudeTimeSpan, *args, **kwargs):
        pass

    @classmethod
    def __get_validators__(cls):
        yield cls.validate

    @classmethod
    def validate(cls, value):
        return cls(value)

    def __repr__(self) -> str:
        return f"TimeSpan({super().__repr__()})"

    def __str__(self) -> str:
        tot_days = self.trunc(TimeSpan.DAY)
        tot_hours = self.trunc(TimeSpan.HOUR)
        tot_minutes = self.trunc(TimeSpan.MINUTE)
        tot_seconds = self.trunc(TimeSpan.SECOND)
        tot_milliseconds = self.trunc(TimeSpan.MILLISECOND)
        tot_micros = self.trunc(TimeSpan.MICROSECOND)
        tot_nanos = self.trunc(TimeSpan.NANOSECOND)
        days = tot_days
        hours = tot_hours.sub(tot_days)
        minutes = tot_minutes.sub(tot_hours)
        seconds = tot_seconds.sub(tot_minutes)
        milliseconds = tot_milliseconds.sub(tot_seconds)
        microseconds = tot_micros.sub(tot_milliseconds)
        nanoseconds = tot_nanos.sub(tot_micros)

        v = ""
        if not days == 0:
            v += f"{days.days}d "
        if not hours == 0:
            v += f"{hours.hours}h "
        if not minutes == 0:
            v += f"{minutes.minutes}m "
        if not seconds == 0:
            v += f"{seconds.seconds}s "
        if not milliseconds == 0:
            v += f"{milliseconds.milliseconds}ms "
        if not microseconds == 0:
            v += f"${microseconds.microseconds}µs "
        if not nanoseconds == 0:
            v += f"{nanoseconds.nanoseconds}ns"
        return v.strip()

    @property
    def days(self) -> float:
        """:returns: The decimal number of days in the TimeSpan, including fractional
        days.
        """
        return float(self / TimeSpan.DAY)

    @property
    def days_int(self) -> int:
        """:returns:  The integer number of days in the TimeSpan, NOT including
        fractional days.
        """
        return int(self / TimeSpan.DAY)

    @property
    def hours(self) -> float:
        """:returns: The decimal number of hours in the TimeSpan, including fractional
        hours.
        """
        return float(self / TimeSpan.HOUR)

    @property
    def hours_int(self) -> int:
        """:returns:  The integer number of hours in the TimeSpan, NOT including
        fractional hours.
        """
        return int(self / TimeSpan.DAY)

    @property
    def minutes(self) -> float:
        """:returns: The decimal number of minutes in the TimeSpan, including fractional
        minutes.
        """
        return float(self / TimeSpan.MINUTE)

    @property
    def minutes_int(self) -> int:
        """:returns:  THe integer number of minutes in the TimeSpan, NOT including
        fractional minutes.
        """
        return int(self / TimeSpan.MINUTE)

    @property
    def seconds(self) -> float:
        """:returns: The decimal number of seconds in the TimeSpan, including fractional
        seconds.
        """
        return float(self / TimeSpan.SECOND)

    @property
    def seconds_int(self) -> int:
        """:returns: The integer number of seconds in the TimeSpan, NOT including
        fractional seconds.
        """
        return int(self / TimeSpan.SECOND)

    @property
    def milliseconds(self) -> float:
        """:returns: The decimal number of milliseconds in the TimeSpan, including the
        fractional milliseconds.
        """
        return float(self / TimeSpan.MILLISECOND)

    @property
    def milliseconds_int(self) -> int:
        """:returns: The integer number of milliseconds in the TimeSpan, NOT including
        fractional milliseconds.
        """
        return int(self / TimeSpan.MILLISECOND)

    @property
    def microseconds(self) -> float:
        """:returns:  The decimal number of microseconds in the TimeSpan, including
        the fractional microseconds.
        """
        return float(self / TimeSpan.MICROSECOND)

    @property
    def microseconds_int(self) -> int:
        """:returns:  The integer number of microseconds in the TimeSpan, NOT including
        fractional microseconds.
        """
        return int(self / TimeSpan.MICROSECOND)

    @property
    def nanoseconds(self) -> int:
        """:returns: The integer number of nanoseconds in the TimeSpan.
        """
        return int(self / TimeSpan.NANOSECOND)

    def trunc(self, span: CrudeTimeSpan) -> TimeSpan:
        """Truncates the TimeSpan to the nearest integer multiple of the given span.

        For example,

        (TimeSpan.DAY + TimeSpan.HOUR + TimeSpan.SECOND).trunc(TimeSpan.DAY)

        would return

        TimeSpan.DAY

        :param span: The TimeSpan to truncate by. The span MUST be one of TimeSpan.DAY,
        TimeSpan.HOUR, TimeSpan.SECOND, TimeSpan.MILLISECOND, TimeSpan.MICROSECOND,
        TimeSpan.NANOSECOND
        """
        return _semantic_trunc(self, TimeSpan(span), list(self.UNITS.values()))

    def mod(self, span: CrudeTimeSpan) -> TimeSpan:
        """Calculates the remainder of the TimeSpan using the given span.

        For example,

        (TimeSpan.DAY + TimeSpan.HOUR).trunc(TimeSpan.DAY)

        would return

        TimeSpan.HOUR

        :param span: The TimeSpan to divide by for the remainder. The span MUST be one
        of TimeSpan.DAY, TimeSpan.HOUR, TimeSpan.SECOND, TimeSpan.MILLISECOND,
        TimeSpan.MICROSECOND, TimeSpan.NANOSECOND
        """
        return _semantic_mod(self, TimeSpan(span), list(self.UNITS.values()))

    def delta(self) -> timedelta:
        """:return: The TimeSpan represented as a datetime.timedelta.
        """
        return timedelta(seconds=self.seconds)

    def is_zero(self) -> bool:
        """:return: True if the TimeSpan is zero, False otherwise
        """
        return self == 0

    def add(self, ts: CrudeTimeSpan) -> TimeSpan:
        """Adds the TimesSpan and given TimeSpan.
        :param ts: The second TimeSpan
        :return: A new TimeSpan that is the sum of the two TimeSpans
        """
        return TimeSpan(super().__add__(TimeSpan(ts)))

    def sub(self, ts: CrudeTimeSpan) -> TimeSpan:
        """Returns a new TimeSpan that is the difference of the two TimeSpans.
        :param ts: the second TimeSpan
        :return: a new TimeSpan that is the difference of the two TimeSpans
        """
        return TimeSpan(super().__sub__(TimeSpan(ts)))

    def __add__(self, other: CrudeTimeSpan) -> TimeSpan:
        return self.add(other)

    def __sub__(self, other: CrudeTimeSpan) -> TimeSpan:
        return self.sub(other)

    def __mul__(self, other: CrudeTimeSpan) -> TimeSpan:
        return TimeSpan(super().__mul__(TimeSpan(other)))

    def __rmul__(self, other: CrudeTimeSpan) -> TimeSpan:
        return self.__mul__(other)

    def __gt__(self, other: CrudeTimeSpan) -> bool:
        return super().__gt__(TimeSpan(other))

    def __ge__(self, other: CrudeTimeSpan) -> bool:
        return super().__ge__(TimeSpan(other))

    def __lt__(self, other: CrudeTimeSpan) -> bool:
        return super().__lt__(TimeSpan(other))

    def __le__(self, other: CrudeTimeSpan) -> bool:
        return super().__le__(TimeSpan(other))

    def __eq__(self, other: object) -> bool:
        if isinstance(other, get_args(CrudeTimeSpan)):
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
    MIN: TimeSpan
    ZERO: TimeSpan
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
TimeSpan.MIN = TimeSpan(0)
TimeSpan.ZERO = TimeSpan(0)
TimeSpan.UNITS = {
    TimeSpan.NANOSECOND_UNITS: TimeSpan.NANOSECOND,
    TimeSpan.MICROSECOND_UNITS: TimeSpan.MICROSECOND,
    TimeSpan.MILLISECOND_UNITS: TimeSpan.MILLISECOND,
    TimeSpan.SECOND_UNITS: TimeSpan.SECOND,
    TimeSpan.MINUTE_UNITS: TimeSpan.MINUTE,
    TimeSpan.HOUR_UNITS: TimeSpan.HOUR,
}
TimeStamp.MIN = TimeStamp(0)
TimeStamp.ZERO = TimeStamp.MIN
TimeStamp.MAX = TimeStamp(2 ** 63 - 1)


def convert_time_units(data: np.ndarray, _from: str, to: str):
    """Converts the data from one time unit to another.
    :param data: the data to convert
    :param _from: the units of the data
    :param to: the units to convert to
    :return: the data in the new units
    """
    if _from == "iso":
        data = datetime.fromisoformat(data).timestamp()
        f = TimeSpan.SECOND
    else:
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

    def __new__(cls, value: CrudeRate):
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

    def __init__(self, value: CrudeRate):
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

    def sample_count(self, time_span: CrudeTimeSpan) -> int:
        """Returns the number of samples in the given TimeSpan at this rate"""
        return int(TimeSpan(time_span).seconds() * self)

    def byte_size(self, time_span: CrudeTimeSpan, density: Density) -> Size:
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

    def __mul__(self, other: CrudeRate) -> Rate:
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

    def __init__(
        self,
        start: CrudeTimeStamp | TimeRange,
        end: CrudeTimeStamp | None = None,
        **kwargs,
    ):
        if isinstance(start, TimeRange):
            start, end = start.start, start.end
        end = start if end is None else end
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

    @property
    def span(self) -> TimeSpan:
        return TimeSpan(self.end - self.start)

    def make_valid(self) -> TimeRange:
        if not self.is_valid():
            return self.swap()
        return self

    @property
    def is_zero(self) -> bool:
        return self.span.is_zero()

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

    def contains(self, value: TimeRange | CrudeTimeStamp) -> bool:
        """Checks if the TimeRange contains the given TimeRange or TimeStamp. If
        value is a TimeRange, it is considered contained if value's start is after or
        equal to the TimeRange start and value's end is before or equal to the TimeRange
        end. This means that two equal time ranges contain each other.

        If the value is a TimeStamp, it is considered contained if value is after
        or equal to the TimeRange start and value is STRICTLY before the TimeRange end.

        :param value: The TimeRange or TimeStamp to check.
        """
        if isinstance(value, TimeRange):
            return value.start.after_eq(self.start) and value.end.before_eq(self.end)
        pv = TimeStamp(value)
        return pv.after_eq(self.start) and pv.before(self.end)

    def overlaps_with(self, tr: TimeRange) -> bool:
        """:returns: True if the time ranges overlap.
        :param tr: The time range to compare against.
        """
        return self == tr or self.contains(tr.end) or tr.contains(self)

    def swap(self) -> TimeRange:
        """:returns: A new TimeRange with the start and end values swapped. Note that
        this may make the TimeRange invalid.
        """
        self.start, self.end = self.end, self.start
        return self

    def is_valid(self) -> bool:
        """:returns: True if the TimeRange start is before or equal to the TimeRange end.
        """
        return self.span >= TimeSpan.ZERO

    def __str__(self) -> str:
        return str(self.start) + " - " + str(self.end)

    def __eq__(self, other: TimeRange) -> bool:
        return self.start == other.start and self.end == other.end

    # Assigning these values to None prevents pydantic
    # from throwing a nasty missing attribute error.
    MAX: TimeRange = None  # type: ignore
    MIN: TimeRange = None  # type: ignore
    ZERO: TimeRange = None  # type: ignore


TimeRange.MAX = TimeRange(TimeStamp.MIN, TimeStamp.MAX)
TimeRange.MIN = TimeRange(TimeStamp.MAX, TimeStamp.MIN)
TimeRange.ZERO = TimeRange(0, 0)


class Density(int):
    def __new__(cls, value: CrudeDensity):
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

    def sample_count(self, byte_length: int) -> int:
        return int(byte_length / self)

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

    def __new__(cls, value: CrudeDataType):
        if isinstance(value, DataType):
            return value
        if isinstance(value, str):
            return super().__new__(cls, value)
        if np.issctype(value):
            value = DataType._FROM_NUMPY.get(np.dtype(value), None)
            if value is not None:
                return value
        raise TypeError(f"Cannot convert {type(value)} to DataType")

    def __init__(self, value: CrudeDataType):
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

CrudeTimeStamp: TypeAlias = Union[
    int,
    TimeStamp,
    TimeSpan,
    datetime,
    timedelta,
    np.datetime64,
    np.int64,
]
CrudeTimeSpan: TypeAlias = Union[
    int | TimeSpan | TimeStamp,
    timedelta,
    np.timedelta64,
]
CrudeRate: TypeAlias = int | float | TimeSpan | Rate
CrudeDensity: TypeAlias = Density | int
CrudeDataType: TypeAlias = DTypeLike | DataType | str
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


class Series(Payload):
    time_range: TimeRange | None = None
    data_type: DataType
    data: bytes

    def __len__(self) -> int:
        return self.data_type.density.sample_count(len(self.data))

    def __init__(
        self,
        data: bytes | pd.Series | np.ndarray | Series,
        data_type: CrudeDataType | None = None,
        time_range: TimeRange | None = None,
    ):
        if isinstance(data, Series):
            data_type = data.data_type if data_type is None else data_type
            data_ = data.data
            time_range = data.time_range if time_range is None else time_range
        elif isinstance(data, pd.Series):
            data_type = DataType(data.dtype if data_type is None else data_type)
            data_ = data.to_numpy(dtype=data_type.np).tobytes()
        elif isinstance(data, np.ndarray):
            data_type = DataType(data.dtype if data_type is None else data_type)
            data_ = data.tobytes()
        else:
            if data_type is None:
                raise ValueError("data_type must be specified if a buffer is given")
            data_type = DataType(data_type)
            data_ = data
        super().__init__(data_type=data_type, data=data_, time_range=time_range)

    class Config:
        arbitrary_types_allowed = True

    def __array__(self) -> np.ndarray:
        return np.frombuffer(self.data, dtype=self.data_type.np)

    def __getitem__(self, index: int) -> float:
        return self.__array__()[index]

    def astype(self, data_type: DataType) -> Series:
        return Series(
            data=self.__array__().astype(data_type.np),
            data_type=data_type,
            time_range=self.time_range,
        )

    def to_numpy(self) -> np.ndarray:
        return self.__array__()

    def to_list(self) -> list:
        return self.__array__().tolist()

    def to_datetime(self) -> list[datetime]:
        return [pd.Timestamp(t).to_pydatetime() for t in self.__array__()]
