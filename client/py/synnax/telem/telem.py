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
from datetime import datetime, timedelta, timezone, tzinfo
from math import trunc
from typing import ClassVar, Literal, TypeAlias, cast, get_args

import numpy as np
import pandas as pd
from numpy.typing import DTypeLike
from pydantic import BaseModel

from synnax.exceptions import ContiguityError


class TimeStamp(int):
    """TimeStamp represents a 64 bit nanosecond-precision UTC timestamp. The TimeStamp
    constructor accepts a variety of types and will attempt to convert them to a
    TimeStamp. The following types are supported:

    * TimeStamp - Returns a copy of the TimeStamp.
    * TimeSpan - Treats the TimeSpan as a duration since the Unix epoch in UTC.
    * pd.TimeStamp - Converts the pandas TimeStamp to a TimeStamp. If the timestamp is
        not timezone aware, it is assumed to be in the local timezone.
    * datetime - Converts the datetime to a TimeStamp.  If the datetime is not timezone
        aware, it is assumed to be in the local timezone.
    * timedelta - Treats the timedelta as a duration since the Unix epoch in UTC.
    * np.datetime64 - Treats the datetime64 as a duration since the Unix epoch in UTC.
    * int - Treats the int as a nanosecond duration since the Unix epoch in UTC.
    """

    def __new__(cls, value: CrudeTimeStamp):
        if isinstance(value, str):
            value = int(value)
        if isinstance(value, TimeStamp):
            return value
        if isinstance(value, TimeSpan):
            value = int(value)
        elif isinstance(value, pd.Timestamp):
            value = int(
                float(TimeSpan.SECOND) * value.to_pydatetime().astimezone().timestamp()
            )
        elif isinstance(value, datetime):
            value = int(float(TimeSpan.SECOND) * value.timestamp())
        elif isinstance(value, timedelta):
            value = int(float(TimeSpan.SECOND) * value.total_seconds())
        elif isinstance(value, np.datetime64):
            value = int(pd.Timestamp(value).value)
        elif isinstance(value, (int, float, np.integer, np.floating)):
            value = int(value)
        else:
            raise TypeError(f"Cannot convert {type(value)} to TimeStamp")

        return super().__new__(cls, value)

    @classmethod
    def __get_validators__(cls):
        """Implemented for pydantic validation"""
        yield cls.validate

    @classmethod
    def validate(cls, v):
        """Implemented for pydantic validation"""
        # if its a string, cast the string to a number
        return cls(v)

    @classmethod
    def now(cls) -> TimeStamp:
        """:returns: the current time as a TimeStamp."""
        return TimeStamp(datetime.now())

    @classmethod
    def since(cls, ts: CrudeTimeStamp) -> TimeSpan:
        """:returns: a TimeSpan representing the amount of time elapsed since the given
        TimeStamp."""
        return TimeStamp.now().span(ts)

    def trunc(self, span: CrudeTimeSpan) -> TimeStamp:
        """Truncates the TimeSpan to the nearst integer multiple of the given span.

        For example,

        (TimeSpan.DAY + TimeSpan.HOUR + TimeSpan.SECOND).trunc(TimeSpan.DAY)

        would return

        TimeSpan.DAY

        :param span: The TimeSpan to truncate by.
        """
        return TimeStamp(TimeSpan(self).trunc(span))

    def span(self, other: CrudeTimeStamp) -> TimeSpan:
        """:returns: The TimeSpan between two timestamps. This span is guaranteed to be
        positive.
        """
        return TimeRange(self, other).make_valid().span

    def datetime(self, tz: tzinfo | None = None) -> datetime:
        """:returns: The TimeStamp represented as a timezone aware datetime object.
        :param tz: The timezone to use for the datetime. If not provided, the local
        timezone is used.
        """
        return (
            datetime.utcfromtimestamp(self / TimeSpan.SECOND)
            .replace(tzinfo=timezone.utc)
            .astimezone(tz)
        )

    def after(self, ts: CrudeTimeStamp) -> bool:
        """:returns: True if the TimeStamp is strictly after the given TimeStamp.
        :param ts: The TimeStamp to compare to.

        """
        return super().__gt__(TimeStamp(ts))

    def after_eq(self, ts: CrudeTimeStamp) -> bool:
        """:return: True if the TimeStamp is after or equal to the given TimeStamp.
        :param ts: The TimeStamp to compare to.
        """
        return super().__ge__(TimeStamp(ts))

    def before(self, ts: CrudeTimeStamp) -> bool:
        """:returns: True if the TimeStamp is strictly before the given TimeStamp.
        :param ts: The TimeStamp to compare to.
        """
        return super().__lt__(TimeStamp(ts))

    def before_eq(self, ts: CrudeTimeStamp) -> bool:
        """:returns: True if the TimeStamp is before or equal to the given TimeStamp.
        :param ts: The TimeStamp to compare to.
        """
        return super().__le__(TimeStamp(ts))

    def span_range(self, span: TimeSpan) -> TimeRange:
        """:returns: A TimeRange that spans the given TimeSpan
        :param span: The TimeSpan.
        """
        return TimeRange(self, self + span).make_valid()

    def range(self, ts: CrudeTimeStamp) -> TimeRange:
        """:return: A new TimeRange spanning the TimeStamp and provided TimeStamp. This
        TimeRange is guaranteed to be valid i.e. start before or equal to end.
        :param ts: The second time stamp
        """
        return TimeRange(self, TimeStamp(ts)).make_valid()

    def __add__(self, rhs: CrudeTimeStamp) -> TimeStamp:
        return TimeStamp(super().__add__(TimeStamp(rhs)))

    def __sub__(self, rhs: CrudeTimeStamp) -> TimeStamp:
        return TimeStamp(super().__sub__(TimeStamp(rhs)))

    def __lt__(self, rhs: CrudeTimeStamp) -> bool:
        return self.before(rhs)

    def __le__(self, rhs: CrudeTimeStamp) -> bool:
        return self.before_eq(rhs)

    def __ge__(self, rhs: CrudeTimeStamp) -> bool:
        return self.after_eq(rhs)

    def __gt__(self, rhs: CrudeTimeStamp) -> bool:
        return self.after(rhs)

    def __eq__(self, rhs: object) -> bool:
        if not isinstance(rhs, get_args(CrudeTimeStamp)):
            return False
        return super().__eq__(TimeStamp(cast(CrudeTimeStamp, rhs)))

    def __str__(self) -> str:
        return self.datetime().isoformat()

    MIN: TimeStamp
    """The minimum possible value of a TimeStamp"""
    MAX: TimeStamp
    """The maximum possible value of a TimeStamp"""
    ZERO: TimeStamp
    """The zero value of a TimeStamp"""


class TimeSpan(int):
    """TimeSpan represents a 64 bit nanosecond-precision duration. The TimeSpan
    constructor accepts a variety of different types and will attempt to convert them
    to a TimeSpan. The supported types are parsed as follows:

    * int - The number of nanoseconds.
    * np.int64 - The number of nanoseconds.
    * TimeSpan - Creates a copy of the TimeSpan.
    * TimeStamp - The difference between the TimeStamp and the Unix epoch
    * timedelta - The duration of the timedelta.
    * pands.timedelta - The duration of the timedelta.
    * np.timedelta64 - The duration of the timedelta64.
    """

    def __new__(cls, value: CrudeTimeSpan):
        if isinstance(value, str):
            value = int(value)
        if isinstance(value, timedelta):
            value = int(float(TimeSpan.SECOND) * value.total_seconds())
        elif isinstance(value, np.timedelta64):
            value = int(float(TimeSpan.SECOND) * pd.Timedelta(value).total_seconds())
        elif not isinstance(value, (int, float, np.floating, np.integer)):
            raise TypeError(f"Cannot convert {type(value)} to TimeSpan")
        return super().__new__(cls, value)

    @classmethod
    def __get_validators__(cls):
        """Implemented for pydantic validation. Should not be used externally."""
        yield cls.validate

    @classmethod
    def validate(cls, value):
        """Implemented for pydantic validation. Should not be used externally."""
        return cls(value)

    @classmethod
    def since(cls, ts: CrudeTimeStamp) -> TimeSpan:
        """:returns: a TimeStamp representing the amount of time elapsed since the given
        TimeStamp."""
        return TimeStamp.now().span(ts)

    def __repr__(self) -> str:
        return f"TimeSpan({super().__repr__()})"

    def __str__(self) -> str:
        tot_days = self.trunc(TimeSpan.DAY)
        tot_hours = self.trunc(TimeSpan.HOUR)
        tot_minutes = self.trunc(TimeSpan.MINUTE)
        tot_seconds = self.trunc(TimeSpan.SECOND)
        tot_millis = self.trunc(TimeSpan.MILLISECOND)
        tot_micros = self.trunc(TimeSpan.MICROSECOND)
        tot_nanos = self.trunc(TimeSpan.NANOSECOND)
        days = tot_days
        hours = tot_hours - tot_days
        minutes = tot_minutes - tot_hours
        seconds = tot_seconds - tot_minutes
        millis = tot_millis - tot_seconds
        micros = tot_micros - tot_millis
        nanos = tot_nanos - tot_micros

        v = ""
        if days != 0:
            v += f"{days.days_int}{TimeSpan.DAY_UNITS} "
        if hours != 0:
            v += f"{hours.hours_int}{TimeSpan.HOUR_UNITS} "
        if minutes != 0:
            v += f"{minutes.minutes_int}{TimeSpan.MINUTE_UNITS} "
        if seconds != 0:
            v += f"{seconds.seconds_int}{TimeSpan.SECOND_UNITS} "
        if millis != 0:
            v += f"{millis.milliseconds_int}{TimeSpan.MILLISECOND_UNITS} "
        if micros != 0:
            v += f"{micros.microseconds_int}{TimeSpan.MICROSECOND_UNITS} "
        if nanos != 0 or len(v) == 0:
            v += f"{nanos.nanoseconds}{TimeSpan.NANOSECOND_UNITS}"
        return v.strip()

    @property
    def days(self) -> float:
        """:returns: The decimal number of days in the TimeSpan, including fractional
        days.
        """
        return float(self) / float(TimeSpan.DAY)

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
        return float(self) / float(TimeSpan.HOUR)

    @property
    def hours_int(self) -> int:
        """:returns: The integer number of hours in the TimeSpan, NOT including
        fractional hours.
        """
        return int(self / TimeSpan.HOUR)

    @property
    def minutes(self) -> float:
        """:returns: The decimal number of minutes in the TimeSpan, including fractional
        minutes.
        """
        return float(self) / float(TimeSpan.MINUTE)

    @property
    def minutes_int(self) -> int:
        """:returns: The integer number of minutes in the TimeSpan, NOT including
        fractional minutes.
        """
        return int(self / TimeSpan.MINUTE)

    @property
    def seconds(self) -> float:
        """:returns: The decimal number of seconds in the TimeSpan, including fractional
        seconds.
        """
        return float(self) / float(TimeSpan.SECOND)

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
        return float(self) / float(TimeSpan.MILLISECOND)

    @property
    def milliseconds_int(self) -> int:
        """:returns: The integer number of milliseconds in the TimeSpan, NOT including
        fractional milliseconds.
        """
        return int(self / TimeSpan.MILLISECOND)

    @property
    def microseconds(self) -> float:
        """:returns: The decimal number of microseconds in the TimeSpan, including
        the fractional microseconds.
        """
        return float(self) / float(TimeSpan.MICROSECOND)

    @property
    def microseconds_int(self) -> int:
        """:returns: The integer number of microseconds in the TimeSpan, NOT including
        fractional microseconds.
        """
        return int(self / TimeSpan.MICROSECOND)

    @property
    def nanoseconds(self) -> int:
        """:returns: The integer number of nanoseconds in the TimeSpan."""
        return int(self / TimeSpan.NANOSECOND)

    def trunc(self, span: CrudeTimeSpan) -> TimeSpan:
        """Truncates the TimeSpan to the nearest integer multiple of the given span.

        For example,

        (TimeSpan.DAY + TimeSpan.HOUR + TimeSpan.SECOND).trunc(TimeSpan.DAY)

        would return

        TimeSpan.DAY

        :param span: The TimeSpan to truncate by.
        """
        return TimeSpan(trunc(self / span) * span)

    @property
    def timedelta(self) -> timedelta:
        """:returns: The TimeSpan represented as a datetime.timedelta."""
        return timedelta(seconds=self.seconds)

    def __add__(self, rhs: CrudeTimeSpan) -> TimeSpan:
        return TimeSpan(super().__add__(TimeSpan(rhs)))

    def __sub__(self, rhs: CrudeTimeSpan) -> TimeSpan:
        return TimeSpan(super().__sub__(TimeSpan(rhs)))

    def __mul__(self, rhs: CrudeTimeSpan) -> TimeSpan:
        return TimeSpan(super().__mul__(TimeSpan(rhs)))

    def __truediv__(self, rhs: CrudeTimeSpan) -> TimeSpan:
        return TimeSpan(super().__truediv__(TimeSpan(rhs)))

    def __mod__(self, rhs: CrudeTimeSpan) -> TimeSpan:
        return TimeSpan(super().__mod__(TimeSpan(rhs)))

    def __rmul__(self, rhs: CrudeTimeSpan) -> TimeSpan:
        return self.__mul__(rhs)

    def __gt__(self, rhs: CrudeTimeSpan) -> bool:
        return super().__gt__(TimeSpan(rhs))

    def __ge__(self, rhs: CrudeTimeSpan) -> bool:
        return super().__ge__(TimeSpan(rhs))

    def __lt__(self, rhs: CrudeTimeSpan) -> bool:
        return super().__lt__(TimeSpan(rhs))

    def __le__(self, rhs: CrudeTimeSpan) -> bool:
        return super().__le__(TimeSpan(rhs))

    def __eq__(self, rhs: object) -> bool:
        if not isinstance(rhs, get_args(CrudeTimeSpan)):
            return NotImplemented
        return super().__eq__(int(TimeSpan(cast(CrudeTimeSpan, rhs))))

    NANOSECOND: TimeSpan
    """A nanosecond."""
    NANOSECOND_UNITS: TimeSpanUnits
    """The unit string for nanoseconds: 'ns'."""
    MICROSECOND: TimeSpan
    """A microsecond."""
    MICROSECOND_UNITS: TimeSpanUnits
    """The unit string for microseconds: 'us'."""
    MILLISECOND: TimeSpan
    """A millisecond."""
    MILLISECOND_UNITS: TimeSpanUnits
    """The unit string for milliseconds: 'ms'."""
    SECOND: TimeSpan
    """A second."""
    SECOND_UNITS: TimeSpanUnits
    """The unit string for seconds: 's'."""
    MINUTE: TimeSpan
    """A minute."""
    MINUTE_UNITS: TimeSpanUnits
    """The unit string for minutes: 'm'."""
    HOUR: TimeSpan
    """An hour."""
    HOUR_UNITS: TimeSpanUnits
    """The unit string for hours: 'h'."""
    DAY: TimeSpan
    """A day."""
    DAY_UNITS: TimeSpanUnits
    """The unit string for days: 'd'."""
    MAX: TimeSpan
    """The maximum possible value for a TimeSpan"""
    MIN: TimeSpan
    """The minimum possible value for a TimeSpan"""
    ZERO: TimeSpan
    """A dictionary mapping unit string to its corresponding TimeSan"""
    UNITS: dict[TimeSpanUnits, TimeSpan]


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
TimeSpan.DAY = TimeSpan.HOUR * 24
TimeSpan.DAY_UNITS = "d"
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
    TimeSpan.DAY_UNITS: TimeSpan.DAY,
}
TimeStamp.MIN = TimeStamp(0)
TimeStamp.ZERO = TimeStamp.MIN
TimeStamp.MAX = TimeStamp(2**63 - 1)

TimeSpanUnits = Literal["ns", "us", "ms", "s", "m", "h", "d", "iso"]


def convert_time_units(data: np.ndarray, _from: TimeSpanUnits, to: TimeSpanUnits):
    """Converts the data from one time unit to another.

    :param data: the numpy array to convert.
    :param _from: the units of the data
    :param to: the units to convert to
    :return: the data in the new units
    """
    if _from == "iso":
        data = np.array([datetime.fromisoformat(v).timestamp() for v in data])
        f = TimeSpan.SECOND
    else:
        f = TimeSpan.UNITS.get(_from, TimeSpan.ZERO)
    if f == 0:
        raise ValueError(f"Invalid input time unit {_from}")
    t = TimeSpan.UNITS.get(to, TimeSpan.ZERO)
    if t == 0:
        raise ValueError(f"Invalid output time unit {to}")

    converted = data * f / t
    if to == TimeSpan.SECOND_UNITS:
        return converted.astype(np.float64)
    if to == TimeSpan.NANOSECOND_UNITS:
        return converted.astype(np.int64)
    return converted


class Rate(float):
    """Rate represents a data rate measured in Hz."""

    def __new__(cls, value: CrudeRate):
        if isinstance(value, float):
            return super().__new__(cls, value)
        if isinstance(value, TimeSpan):
            value = 1 / value.seconds
        elif isinstance(value, int):
            value = float(value)
        else:
            raise TypeError(f"Cannot convert {type(value)} to Rate")
        return super().__new__(cls, value)

    @classmethod
    def __get_validators__(cls):
        """Implemented for pydantic validation. Should not be used externally."""
        yield cls.validate

    @classmethod
    def validate(cls, v):
        """Implemented for pydantic validation. Should not be used externally."""
        return cls(v)

    @property
    def period(self) -> TimeSpan:
        """:returns: the period of the rate as a TimeSpan"""
        return TimeSpan(int(1 / self * float(TimeSpan.SECOND)))

    def sample_count(self, time_span: CrudeTimeSpan) -> int:
        """:returns: the number of samples in the given TimeSpan at this rate"""
        return int(TimeSpan(time_span).seconds * self)

    def byte_size(self, time_span: CrudeTimeSpan, density: Density) -> Size:
        """:returns: the amount of bytes occupied by the given TimeSpan at this rate
        and the given sample density.

        :param time_span: TimeSpan - A crude TimeSpan representing the size of the
        data set in time.
        :param density: Density - The sample density of the data.
        """
        return Size(self.sample_count(time_span) * int(density))

    def span(self, sample_count: int) -> TimeSpan:
        """:returns: the TimeSpan that corresponds to the given number of samples at
        this rate."""
        return self.period * sample_count

    def size_span(self, size: Size, density: Density) -> TimeSpan:
        """:returns: the TimeSpan that corresponds to the given number of bytes at this
        rate and the given sample density."""
        if size % density != 0:
            raise ContiguityError(f"Size {size} is not a multiple of density {density}")
        return self.span(int(size / density))

    def __str__(self):
        if self < 1:
            return f"{self.period} per cycle"
        return str(int(self)) + "Hz"

    def __repr__(self):
        return f"Rate({super().__repr__()} Hz)"

    def __mul__(self, rhs: CrudeRate) -> Rate:
        return Rate(super().__mul__(Rate(rhs)))

    HZ: Rate
    """One Hz."""
    KHZ: Rate
    """One kHz"""
    MHZ: Rate
    """One MHz"""


Rate.HZ = Rate(1)
Rate.KHZ = Rate(1000) * Rate.HZ
Rate.MHZ = Rate(1000) * Rate.KHZ


class TimeRange(BaseModel):
    """TimeRange is a range of time marked by a start and end TimeStamp. A TimeRange
    is start inclusive and end exclusive.
    """

    start: TimeStamp
    """The starting TimeStamp of the TimeRange.

    Note that this value is not guaranteed to be before or equal to the ending value.
    To ensure that this is the case, call TimeRange.make_valid().

    In most cases, operations should treat start as inclusive.
    """
    end: TimeStamp
    """The ending TimeStamp of the TimeRange.

    Note that this value is not guaranteed to be after or equal to the ending value. To
    ensure that this is the case, call TimeRange.make_valid()

    In most cases, operations should treat end as exclusive.
    """

    def __init__(
        self,
        start: CrudeTimeStamp | TimeRange,
        end: CrudeTimeStamp | None = None,
        *args,
        **kwargs,
    ):
        if isinstance(start, TimeRange):
            start_ = cast(TimeRange, start)
            start, end = start_.start, start_.end
        end = start if end is None else end
        super().__init__(start=TimeStamp(start), end=TimeStamp(end))

    @classmethod
    def __get_validators__(cls):
        """Implemented for pydantic validation. Should not be used externally."""
        yield cls.validate

    @classmethod
    def validate(cls, v):
        """Implemented for pydantic validation. Should not be used externally."""
        if isinstance(v, TimeRange):
            return cls(v.start, v.end)
        elif isinstance(v, dict):
            return cls(**v)
        return cls(start=v[0], end=v[1])

    @property
    def span(self) -> TimeSpan:
        """:returns: the TimeSpan between the start and end TimeStamps of the TimeRange."""
        return TimeSpan(self.end - self.start)

    def make_valid(self) -> TimeRange:
        """:returns: A copy of the Time Range with its start and end swapped if the
        TimeRange is invalid (i.e. its start is after its end). Otherwise, returns the
        TimeRange itself.
        """
        return self.swap() if not self.valid else self

    def clamp(self, bound: TimeRange) -> TimeRange:
        """Clamps a copy of the TimeRange and returns it, restricting its start and end
        to within the given bound TimeRange. The clamping is performed in a manner such
        that bound.contains(clamped) is guaranteed to return True.

        :param bound: The TimeRange to clamp by.
        :returns: The clamped TimeRange.
        """
        copy = self.copy()
        if bound.start.after(self.start):
            copy.start = bound.start
        if bound.start.after(self.end):
            copy.end = bound.start
        if bound.end.before(self.end):
            copy.end = bound.end
        if bound.end.before(self.start):
            copy.start = bound.end
        return copy

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
        self.copy()
        return TimeRange(start=self.end, end=self.start)

    def copy(self, *args, **kwargs) -> TimeRange:
        """:returns: A copy of the time range."""
        return TimeRange(start=self.start, end=self.end)

    @property
    def valid(self) -> bool:
        """:returns: True if the TimeRange start is before or equal to the TimeRange
        end.
        """
        return self.span >= TimeSpan.ZERO

    def __str__(self) -> str:
        return str(self.start) + " - " + str(self.end)

    def __eq__(self, rhs: object) -> bool:
        if not isinstance(rhs, TimeRange):
            return False
        return self.start == rhs.start and self.end == rhs.end

    # Assigning these values to None prevents pydantic
    # from throwing a nasty missing attribute error.

    MAX: ClassVar[TimeRange] = None  # type: ignore
    """The maximum possible TimeRange, starting at TimeStamp.MIN and ending
    a TimeStamp.MAX
    """
    MIN: ClassVar[TimeRange] = None  # type: ignore
    """The minimum possible TimeRange, starting at TimeStamp.MAX and ending at
    TimeStamp.MIN. Note that this TimeRange is invalid.
    """
    ZERO: ClassVar[TimeRange] = None  # type: ignore
    """A TimeRange starting and ending at the unix epoch in UTC (TimeStamp.ZERO)."""


TimeRange.MAX = TimeRange(TimeStamp.MIN, TimeStamp.MAX)
TimeRange.MIN = TimeRange(TimeStamp.MAX, TimeStamp.MIN)
TimeRange.ZERO = TimeRange(0, 0)


class Density(int):
    """Density is the number of bytes contained in a single sample."""

    def __new__(cls, value: CrudeDensity):
        if isinstance(value, Density):
            return value
        if isinstance(value, int):
            return super().__new__(cls, value)
        raise TypeError(f"Cannot convert {type(value)} to Density")

    @classmethod
    def __get_validators__(cls):
        """Implemented for pydantic validation. Should not be used externally."""
        yield cls.validate

    @classmethod
    def validate(cls, v):
        """Implemented for pydantic validation. Should not be used externally."""
        return cls(v)

    def sample_count(self, size: CrudeSize) -> int:
        """:returns: The number of contained in the given byte Size."""
        return int(size / self)

    def size_span(self, sample_count: int) -> Size:
        """:returns: The number of bytes occupied by the given number of samples."""
        return Size(sample_count * self)

    def __repr__(self):
        return f"Density({super().__repr__()})"

    UNKNOWN: Density
    """An unknown density, represented as a ZERO value."""
    BIT128: Density
    """A density of 128 bits or 16 bytes per sample."""
    BIT64: Density
    """A density of 64 bits or 8 bytes per sample."""
    BIT32: Density
    """A density of 32 bits or 4 bytes per sample."""
    BIT16: Density
    """A density of 16 bits of 2 bytes per sample."""
    BIT8: Density
    """A density of 8 bits or 1 bytes per sample."""


Density.UNKNOWN = Density(0)
Density.BIT128 = Density(16)
Density.BIT64 = Density(8)
Density.BIT32 = Density(4)
Density.BIT16 = Density(2)
Density.BIT8 = Density(1)


class Size(int):
    def __mul__(self, rhs: CrudeSize) -> Size:
        return Size(super(Size, self).__mul__(Size(rhs)))

    def __str__(self) -> str:
        tot_gb = self.trunc(Size.GB)
        tot_mb = self.trunc(Size.MB)
        tot_kb = self.trunc(Size.KB)
        tot_b = self.trunc(Size.BYTE)
        gb = tot_gb
        mb = tot_mb - tot_gb
        kb = tot_kb - tot_mb
        b = tot_b - tot_kb

        v = ""
        if gb != 0:
            v += f"{gb.gb_int}{Size.GB_UNITS} "
        if mb != 0:
            v += f"{mb.mb_int}{Size.MB_UNITS} "
        if kb != 0:
            v += f"{kb.kb_int}{Size.KB_UNITS} "
        if b != 0 or len(v) == 0:
            v += f"{b.bytes}{Size.BYTE_UNITS} "
        return v.strip()

    @property
    def gb(self) -> float:
        """:returns: The decimal number of gigabytes in the Size, including
        fractional gigabytes"""
        return float(self / Size.GB)

    @property
    def gb_int(self) -> int:
        """:returns: The integer number of gigabytes in the Size, NOT including
        fractional gigabytes"""
        return int(self / Size.GB)

    @property
    def mb(self) -> float:
        """:returns: The decimal number of megabytes in the Size, including
        fractional gigabytes"""
        return float(self / Size.MB)

    @property
    def mb_int(self) -> int:
        """:returns: The integer number of megabytes in the Size, NOT including
        fractional gigabytes"""
        return int(self / Size.MB)

    @property
    def kb(self) -> float:
        """:returns: The decimal number of kilobytes in the Size, including
        fractional gigabytes"""
        return float(self / Size.KB)

    @property
    def kb_int(self) -> int:
        """:returns: The integer number of kilobytes in the Size, NOT including
        fractional gigabytes"""
        return int(self / Size.GB)

    @property
    def bytes(self) -> int:
        """:returns: The number of bytes in the Size."""
        return int(self)

    def trunc(self, size: CrudeSize) -> Size:
        """Truncates the Size to th nearest integer of hte given size.

        For example,

        (Size.GIGABYTE + Size.MEGABYTE).trunc(Size.GIGABYTE)

        would return

        Size.GIGABYTE

        :param size: The Size to truncate by.
        """
        return Size(trunc(self / size) * size)

    def __sub__(self, rhs: CrudeSize) -> Size:
        return Size(int(self) - Size(rhs))

    def __add__(self, rhs: CrudeSize) -> Size:
        return Size(int(self) + Size(rhs))

    BYTE: Size
    BYTE_UNITS: SizeUnits
    KB: Size
    KB_UNITS: SizeUnits
    MB: Size
    MB_UNITS: SizeUnits
    GB: Size
    GB_UNITS: SizeUnits


Size.BYTE = Size(1)
Size.BYTE_UNITS = "b"
Size.KB = Size(1024) * Size.BYTE
Size.KB_UNITS = "kb"
Size.MB = Size(1024) * Size.KB
Size.MB_UNITS = "mb"
Size.GB = Size(1024) * Size.MB
Size.GB_UNITS = "gb"

SizeUnits = ["gb", "mb", "kb", "b"]


class DataType(str):
    """DataType represents a data type as a string."""

    def __new__(cls, value: CrudeDataType):
        if isinstance(value, DataType):
            return value

        if isinstance(value, str):
            return super().__new__(cls, value)

        if isinstance(value, np.number):
            value = DataType._FROM_NUMPY.get(np.dtype(value), None)
            if value is not None:
                return value

        if isinstance(value, float):
            return DataType.FLOAT64

        if isinstance(value, int):
            return DataType.INT64

        if isinstance(value, uuid.UUID):
            return DataType.UUID

        if isinstance(value, list):
            if len(value) == 0:
                raise ValueError("Cannot extract a data type from an empty list")

            if isinstance(value[0], TimeStamp):
                return DataType.TIMESTAMP

            if isinstance(value[0], float):
                return DataType.FLOAT64

            if isinstance(value[0], int):
                return DataType.INT64

            if isinstance(value[0], str):
                return DataType.STRING

            if isinstance(value[0], uuid.UUID):
                return DataType.UUID

            if isinstance(value[0], dict):
                return DataType.JSON

            raise TypeError(f"Cannot convert {type(value)} to DataType")

        if np.issctype(value):
            value = DataType._FROM_NUMPY.get(np.dtype(value), None)
            if value is not None:
                return value

        if isinstance(value, dict):
            return DataType.JSON

        raise TypeError(f"Cannot convert {type(value)} to DataType")

    @classmethod
    def __get_validators__(cls):
        """Implemented for pydantic validation. Should not be used externally."""
        yield cls.validate

    @classmethod
    def validate(cls, v):
        """Implemented for pydantic validation. Should not be used externally."""
        return cls(v)

    @classmethod
    def __modify_schema__(cls, field_schema):
        """Implemented for pydantic validation. Should not be used externally."""
        field_schema.update(type="string")

    @property
    def np(self) -> np.dtype:
        """Converts a built-in DataType to a numpy type Scalar Type
        :return: The numpy type
        """
        npt = DataType._TO_NUMPY.get(self, None)
        if npt is None:
            raise TypeError(f"Cannot convert {self} to numpy type")
        return cast(np.dtype, npt)

    @property
    def has_fixed_density(self) -> bool:
        """:returns: True if the DataType has a fixed density"""
        d = DataType._DENSITIES.get(self, None)
        return d is not None and d != Density.UNKNOWN

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

    def __repr__(self):
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
    ALL: tuple[DataType, ...]
    _TO_NUMPY: dict[DataType, DTypeLike]
    _FROM_NUMPY: dict[DTypeLike, DataType]
    _DENSITIES: dict[DataType, Density]


DataType.UUID = DataType("uuid")
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
DataType.JSON = DataType("json")
DataType.STRING = DataType("string")
DataType.ALL = (
    DataType.UUID,
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
    DataType.STRING,
    DataType.JSON,
)

CrudeTimeStamp: TypeAlias = (
    int | TimeStamp | TimeSpan | datetime | timedelta | np.datetime64 | np.int64
)
CrudeTimeSpan: TypeAlias = (
    int | float | TimeSpan | TimeStamp | timedelta | np.timedelta64
)
CrudeRate: TypeAlias = int | float | TimeSpan | Rate
CrudeDensity: TypeAlias = Density | int
CrudeDataType: TypeAlias = DTypeLike | DataType | str | list | np.number
CrudeSize: TypeAlias = int | float | Size

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
}


class Bounds:
    lower: float
    upper: float

    def __init__(self, lower: float, upper: float):
        self.lower = lower
        self.upper = upper

    def contains(self, value: float) -> bool:
        return self.lower < value <= self.upper
