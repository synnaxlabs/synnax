#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

import uuid
from datetime import UTC, datetime, timedelta, tzinfo
from fractions import Fraction
from math import trunc
from typing import Any, ClassVar, Literal, NoReturn, TypeAlias, overload

import numpy as np
import pandas as pd
from numpy.typing import DTypeLike
from pydantic import BaseModel, ConfigDict, GetCoreSchemaHandler
from pydantic_core import core_schema

from x.exceptions import ContiguityError

_EPOCH = datetime(1970, 1, 1, tzinfo=UTC)


def _timedelta_nanoseconds(delta: timedelta) -> int:
    return (
        (delta.days * 86_400 + delta.seconds) * 1_000_000 + delta.microseconds
    ) * 1_000


_TIMEDELTA64_UNIT_NANOSECONDS = {
    "W": 604_800 * 10**9,
    "D": 86_400 * 10**9,
    "h": 3_600 * 10**9,
    "m": 60 * 10**9,
    "s": 10**9,
    "ms": 10**6,
    "us": 10**3,
    "ns": 1,
}


def _timedelta64_nanoseconds(delta: np.timedelta64) -> int:
    if np.isnat(delta):
        raise ValueError("Cannot convert NaT to TimeSpan")
    unit, count = np.datetime_data(delta.dtype)
    if unit not in _TIMEDELTA64_UNIT_NANOSECONDS:
        return int(delta.astype("timedelta64[ns]").astype(np.int64))
    return int(delta.astype(np.int64)) * count * _TIMEDELTA64_UNIT_NANOSECONDS[unit]


def _int64(value: int, name: str) -> int:
    if not -(2**63) <= value < 2**63:
        raise ValueError(f"{value} does not fit in a 64-bit {name}")
    return value


def _stamp_nanoseconds(value: CrudeTimeStamp) -> int:
    if isinstance(value, str):
        return int(value)
    if isinstance(value, pd.Timestamp):
        return int(value.value)
    if isinstance(value, datetime):
        return _timedelta_nanoseconds(value.astimezone() - _EPOCH)
    if isinstance(value, timedelta):
        return _timedelta_nanoseconds(value)
    if isinstance(value, np.datetime64):
        if np.isnat(value):
            raise ValueError("Cannot convert NaT to TimeStamp")
        return int(pd.Timestamp(value).value)
    if isinstance(value, (int, float, np.integer, np.floating)):
        return int(value)
    raise TypeError(f"Cannot convert {type(value)} to TimeStamp")


def _span_nanoseconds(value: CrudeTimeSpan) -> int:
    if isinstance(value, str):
        return int(value)
    if isinstance(value, pd.Timedelta):
        return int(value.value)
    if isinstance(value, timedelta):
        return _timedelta_nanoseconds(value)
    if isinstance(value, np.timedelta64):
        return _timedelta64_nanoseconds(value)
    if isinstance(value, (int, float, np.integer, np.floating)):
        return int(value)
    raise TypeError(f"Cannot convert {type(value)} to TimeSpan")


def _operand_error(op: str, lhs: object, rhs: object) -> TypeError:
    return TypeError(
        f"unsupported operand type(s) for {op}: '{type(lhs).__name__}' and "
        f"'{type(rhs).__name__}'"
    )


def _reject_number(op: str, lhs: object, rhs: object) -> None:
    """Raises for a bare number, as int and float would otherwise take it as their
    own.
    """
    if isinstance(rhs, (int, float, np.number)):
        raise _operand_error(op, lhs, rhs)


class TimeStamp(int):
    """TimeStamp represents a 64 bit nanosecond-precision UTC timestamp. The TimeStamp
    constructor accepts a variety of types and will attempt to convert them to a
    TimeStamp. The following types are supported:

    * TimeStamp - Returns a copy of the TimeStamp.
    * TimeSpan - Treats the TimeSpan as a duration since the Unix epoch in UTC.
    * pd.TimeStamp - Converts the pandas TimeStamp to a TimeStamp. If the timestamp is
        not timezone aware, it is assumed to be in UTC.
    * datetime - Converts the datetime to a TimeStamp.  If the datetime is not timezone
        aware, it is assumed to be in the local timezone.
    * timedelta - Treats the timedelta as a duration since the Unix epoch in UTC.
    * np.datetime64 - Treats the datetime64 as a duration since the Unix epoch in UTC.
    * int - Treats the int as a nanosecond duration since the Unix epoch in UTC.
    * str - Parses a decimal string of nanoseconds since the Unix epoch in UTC.
    """

    def __new__(cls, value: CrudeTimeStamp) -> TimeStamp:
        if isinstance(value, TimeStamp):
            return value
        return super().__new__(cls, _int64(_stamp_nanoseconds(value), "TimeStamp"))

    @classmethod
    def __get_pydantic_core_schema__(
        cls, source_type: Any, handler: GetCoreSchemaHandler
    ) -> core_schema.CoreSchema:
        """Implemented for pydantic validation. Should not be used externally."""
        return core_schema.no_info_after_validator_function(cls, handler(int))

    @classmethod
    def validate(cls, value: Any, *args: Any, **kwargs: Any) -> TimeStamp:
        """Implemented for pydantic validation. Should not be used externally."""
        return cls(value)

    @classmethod
    def now(cls) -> TimeStamp:
        """:returns: the current time as a TimeStamp."""
        return TimeStamp(datetime.now(UTC))

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
        return datetime.fromtimestamp(self / TimeSpan.SECOND, UTC).astimezone(tz)

    def after(self, ts: CrudeTimeStamp) -> bool:
        """:returns: True if the TimeStamp is strictly after the given TimeStamp.
        :param ts: The TimeStamp to compare to.

        """
        return super().__gt__(_stamp_nanoseconds(ts))

    def after_eq(self, ts: CrudeTimeStamp) -> bool:
        """:return: True if the TimeStamp is after or equal to the given TimeStamp.
        :param ts: The TimeStamp to compare to.
        """
        return super().__ge__(_stamp_nanoseconds(ts))

    def before(self, ts: CrudeTimeStamp) -> bool:
        """:returns: True if the TimeStamp is strictly before the given TimeStamp.
        :param ts: The TimeStamp to compare to.
        """
        return super().__lt__(_stamp_nanoseconds(ts))

    def before_eq(self, ts: CrudeTimeStamp) -> bool:
        """:returns: True if the TimeStamp is before or equal to the given TimeStamp.
        :param ts: The TimeStamp to compare to.
        """
        return super().__le__(_stamp_nanoseconds(ts))

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

    # The int overloads keep the int signature, but a bare number raises TypeError.
    @overload
    def __add__(self, rhs: TimeStamp) -> NoReturn: ...
    @overload
    def __add__(self, rhs: TimeSpan) -> TimeStamp: ...
    @overload
    def __add__(self, rhs: int) -> int: ...
    def __add__(self, rhs: object) -> int:
        if isinstance(rhs, TimeSpan):
            return TimeStamp(int(self) + int(rhs))
        _reject_number("+", self, rhs)
        return NotImplemented

    def __radd__(self, lhs: int) -> NoReturn:
        raise _operand_error("+", lhs, self)

    @overload
    def __sub__(self, rhs: TimeStamp) -> TimeSpan: ...
    @overload
    def __sub__(self, rhs: TimeSpan) -> TimeStamp: ...
    @overload
    def __sub__(self, rhs: int) -> int: ...
    def __sub__(self, rhs: object) -> int:
        if isinstance(rhs, TimeStamp):
            return TimeSpan(int(self) - int(rhs))
        if isinstance(rhs, TimeSpan):
            return TimeStamp(int(self) - int(rhs))
        _reject_number("-", self, rhs)
        return NotImplemented

    def __rsub__(self, lhs: int) -> NoReturn:
        raise _operand_error("-", lhs, self)

    def __mul__(self, rhs: int) -> NoReturn:
        raise _operand_error("*", self, rhs)

    def __rmul__(self, lhs: int) -> NoReturn:
        raise _operand_error("*", lhs, self)

    def __neg__(self) -> NoReturn:
        raise TypeError("bad operand type for unary -: 'TimeStamp'")

    def __lt__(self, rhs: CrudeTimeStamp) -> bool:
        return self.before(rhs)

    def __le__(self, rhs: CrudeTimeStamp) -> bool:
        return self.before_eq(rhs)

    def __ge__(self, rhs: CrudeTimeStamp) -> bool:
        return self.after_eq(rhs)

    def __gt__(self, rhs: CrudeTimeStamp) -> bool:
        return self.after(rhs)

    def __eq__(self, rhs: object) -> bool:
        if not isinstance(
            rhs, (int, TimeSpan, datetime, timedelta, np.datetime64, np.int64)
        ):
            return False
        return super().__eq__(_stamp_nanoseconds(rhs))

    def __hash__(self) -> int:
        return super().__hash__()

    def __str__(self) -> str:
        return self.datetime().isoformat()

    MIN: TimeStamp
    """The minimum possible value of a TimeStamp"""
    MAX: TimeStamp
    """The maximum possible value of a TimeStamp"""
    ZERO: TimeStamp


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
    * str - Parses a decimal string of nanoseconds.
    """

    def __new__(cls, value: CrudeTimeSpan) -> TimeSpan:
        return super().__new__(cls, _int64(_span_nanoseconds(value), "TimeSpan"))

    @classmethod
    def __get_pydantic_core_schema__(
        cls, source_type: Any, handler: GetCoreSchemaHandler
    ) -> core_schema.CoreSchema:
        """Implemented for pydantic validation. Should not be used externally."""
        return core_schema.no_info_after_validator_function(cls, handler(int))

    @classmethod
    def validate(cls, value: Any, *args: Any, **kwargs: Any) -> TimeSpan:
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

    @staticmethod
    def from_seconds(span: CrudeTimeSpan) -> TimeSpan:
        """An alternative constructor for TimeSpan that parses raw numeric values as
        seconds instead of nanoseconds.
        """
        if isinstance(span, TimeSpan):
            return span
        if isinstance(span, Rate):
            return span.period
        if isinstance(span, (float, int)):
            return TimeSpan.SECOND * span
        return TimeSpan(span)

    @staticmethod
    def to_seconds(dur: CrudeTimeSpan | None) -> float | None:
        if dur is None:
            return None
        return TimeSpan.from_seconds(dur).seconds

    @property
    def rate(self) -> Rate:
        """:returns: The rate of the TimeSpan as a Rate."""
        return Rate(1 / self.seconds)

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
        return _div_toward_zero(int(self), int(TimeSpan.DAY))

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
        return _div_toward_zero(int(self), int(TimeSpan.HOUR))

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
        return _div_toward_zero(int(self), int(TimeSpan.MINUTE))

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
        return _div_toward_zero(int(self), int(TimeSpan.SECOND))

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
        return _div_toward_zero(int(self), int(TimeSpan.MILLISECOND))

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
        return _div_toward_zero(int(self), int(TimeSpan.MICROSECOND))

    @property
    def nanoseconds(self) -> int:
        """:returns: The integer number of nanoseconds in the TimeSpan."""
        return int(self)

    def trunc(self, span: CrudeTimeSpan) -> TimeSpan:
        """Truncates the TimeSpan to the nearest integer multiple of the given span.

        For example,

        (TimeSpan.DAY + TimeSpan.HOUR + TimeSpan.SECOND).trunc(TimeSpan.DAY)

        would return

        TimeSpan.DAY

        :param span: The TimeSpan to truncate by.
        """
        step = int(TimeSpan(span))
        return TimeSpan(_div_toward_zero(int(self), step) * step)

    @property
    def timedelta(self) -> timedelta:
        """:returns: The TimeSpan represented as a datetime.timedelta."""
        return timedelta(seconds=self.seconds)

    # The int overloads keep the int signature, but a bare number raises TypeError.
    @overload
    def __add__(self, rhs: TimeStamp) -> TimeStamp: ...
    @overload
    def __add__(self, rhs: TimeSpan) -> TimeSpan: ...
    @overload
    def __add__(self, rhs: int) -> int: ...
    def __add__(self, rhs: object) -> int:
        if isinstance(rhs, TimeStamp):
            return TimeStamp(int(self) + int(rhs))
        if isinstance(rhs, TimeSpan):
            return TimeSpan(int(self) + int(rhs))
        _reject_number("+", self, rhs)
        return NotImplemented

    def __radd__(self, lhs: int) -> NoReturn:
        raise _operand_error("+", lhs, self)

    @overload
    def __sub__(self, rhs: TimeStamp) -> NoReturn: ...
    @overload
    def __sub__(self, rhs: TimeSpan) -> TimeSpan: ...
    @overload
    def __sub__(self, rhs: int) -> int: ...
    def __sub__(self, rhs: object) -> int:
        if isinstance(rhs, TimeSpan):
            return TimeSpan(int(self) - int(rhs))
        _reject_number("-", self, rhs)
        return NotImplemented

    def __rsub__(self, lhs: int) -> NoReturn:
        raise _operand_error("-", lhs, self)

    def __mul__(self, rhs: float | np.integer | np.floating) -> TimeSpan:
        if isinstance(rhs, TimeStamp):
            raise _operand_error("*", self, rhs)
        if isinstance(rhs, (int, np.integer)):
            return TimeSpan(int(self) * int(rhs))
        return TimeSpan(int(self) * float(rhs))

    # Pyright has no reflected-operand priority, so it types int * TimeSpan as int.
    def __rmul__(self, lhs: int | np.integer) -> TimeSpan:
        return self.__mul__(lhs)

    # Mypy and Pyright reject overloads that make TimeSpan / TimeSpan a float and
    # TimeSpan / number a TimeSpan, as TimeSpan is an int. These annotations follow int.
    def __truediv__(self, rhs: float | np.integer | np.floating) -> float:
        if isinstance(rhs, TimeSpan):
            return float(Fraction(int(self), int(rhs)))
        if isinstance(rhs, TimeStamp):
            raise _operand_error("/", self, rhs)
        if isinstance(rhs, (int, np.integer)):
            return TimeSpan(_div_toward_zero(int(self), int(rhs)))
        return TimeSpan(int(Fraction(int(self)) / Fraction(float(rhs))))

    def __floordiv__(self, rhs: int | np.integer) -> int:
        if isinstance(rhs, TimeSpan):
            return int(self) // int(rhs)
        if isinstance(rhs, TimeStamp):
            raise _operand_error("//", self, rhs)
        if isinstance(rhs, (int, np.integer)):
            return TimeSpan(int(self) // int(rhs))
        return NotImplemented

    @overload
    def __mod__(self, rhs: TimeSpan) -> TimeSpan: ...
    @overload
    def __mod__(self, rhs: int) -> int: ...
    def __mod__(self, rhs: object) -> int:
        if isinstance(rhs, TimeSpan):
            return TimeSpan(int(self) % int(rhs))
        _reject_number("%", self, rhs)
        return NotImplemented

    def __neg__(self) -> TimeSpan:
        return TimeSpan(-int(self))

    def __abs__(self) -> TimeSpan:
        return TimeSpan(abs(int(self)))

    # Omits np.timedelta64: its reflected comparisons return np.bool, not bool.
    def __gt__(self, rhs: _ComparableTimeSpan) -> bool:
        return super().__gt__(_span_nanoseconds(rhs))

    def __ge__(self, rhs: _ComparableTimeSpan) -> bool:
        return super().__ge__(_span_nanoseconds(rhs))

    def __lt__(self, rhs: _ComparableTimeSpan) -> bool:
        return super().__lt__(_span_nanoseconds(rhs))

    def __le__(self, rhs: _ComparableTimeSpan) -> bool:
        return super().__le__(_span_nanoseconds(rhs))

    def __eq__(self, rhs: object) -> bool:
        if isinstance(rhs, (int, float)):
            return int(self) == rhs
        if not isinstance(rhs, (timedelta, np.timedelta64)):
            return NotImplemented
        return super().__eq__(_span_nanoseconds(rhs))

    def __hash__(self) -> int:
        return super().__hash__()

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
TimeSpan.MAX = TimeSpan(2**63 - 1)
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


def _div_toward_zero(a: int, b: int) -> int:
    """Divides integers exactly, truncating toward zero like int(a / b)."""
    q = abs(a) // abs(b)
    return q if (a < 0) == (b < 0) else -q


TimeSpanUnits = Literal["ns", "us", "ms", "s", "m", "h", "d", "iso"]


def convert_time_units(
    data: np.ndarray, *, _from: TimeSpanUnits, to: TimeSpanUnits
) -> np.ndarray:
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

    if to == TimeSpan.NANOSECOND_UNITS and np.issubdtype(data.dtype, np.integer):
        return data.astype(np.int64) * int(f)
    converted = data * f / t
    if to == TimeSpan.SECOND_UNITS:
        return converted.astype(np.float64)
    if to == TimeSpan.NANOSECOND_UNITS:
        return converted.astype(np.int64)
    return converted


class Rate(float):
    """Rate represents a data rate measured in Hz."""

    def __new__(cls, value: CrudeRate) -> Rate:
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
    def __get_pydantic_core_schema__(
        cls, source_type: Any, handler: GetCoreSchemaHandler
    ) -> core_schema.CoreSchema:
        """Implemented for pydantic validation. Should not be used externally."""
        return core_schema.no_info_after_validator_function(cls, handler(float))

    @classmethod
    def validate(cls, v: Any, *args: Any, **kwargs: Any) -> Rate:
        """Implemented for pydantic validation. Should not be used externally."""
        return cls(v)

    @property
    def period(self) -> TimeSpan:
        """:returns: the period of the rate as a TimeSpan"""
        return TimeSpan(int(1 / self * float(TimeSpan.SECOND)))

    def sample_count(self, time_span: CrudeTimeSpan) -> int:
        """:returns: the number of samples in the given TimeSpan at this rate"""
        return int(TimeSpan(time_span).seconds * self)

    def byte_size(self, time_span: CrudeTimeSpan, *, density: Density) -> Size:
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
        # repr gives the shortest decimal that round-trips, so Rate(0.1) is 1/10 and
        # not the binary float just below it.
        rate = Fraction(repr(float(self)))
        return TimeSpan(int(Fraction(sample_count * int(TimeSpan.SECOND)) / rate))

    def size_span(self, size: Size, *, density: Density) -> TimeSpan:
        """:returns: the TimeSpan that corresponds to the given number of bytes at this
        rate and the given sample density."""
        if size % density != 0:
            raise ContiguityError(f"Size {size} is not a multiple of density {density}")
        return self.span(int(size / density))

    def __str__(self) -> str:
        if self < 1:
            return f"{self.period} per cycle"
        return f"{round(self, 2)} Hz"

    def __repr__(self) -> str:
        return f"Rate({super().__repr__()} Hz)"

    # The float overloads keep the float signature, but a bare number raises TypeError.
    @overload
    def __add__(self, rhs: Rate) -> Rate: ...
    @overload
    def __add__(self, rhs: float) -> float: ...
    def __add__(self, rhs: object) -> float:
        if isinstance(rhs, Rate):
            return Rate(float(self) + float(rhs))
        _reject_number("+", self, rhs)
        return NotImplemented

    def __radd__(self, lhs: float) -> NoReturn:
        raise _operand_error("+", lhs, self)

    @overload
    def __sub__(self, rhs: Rate) -> Rate: ...
    @overload
    def __sub__(self, rhs: float) -> float: ...
    def __sub__(self, rhs: object) -> float:
        if isinstance(rhs, Rate):
            return Rate(float(self) - float(rhs))
        _reject_number("-", self, rhs)
        return NotImplemented

    def __rsub__(self, lhs: float) -> NoReturn:
        raise _operand_error("-", lhs, self)

    def __mul__(self, rhs: CrudeRate) -> Rate:
        return Rate(super().__mul__(Rate(rhs)))

    def __rmul__(self, lhs: float) -> Rate:
        return Rate(float(lhs) * float(self))

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

    model_config = ConfigDict(frozen=True)

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
        *args: Any,
        **kwargs: Any,
    ):
        if isinstance(start, TimeRange):
            end = start.end
            start = start.start
        end = start if end is None else end
        super().__init__(start=TimeStamp(start), end=TimeStamp(end))

    @property
    def span(self) -> TimeSpan:
        """:returns: the TimeSpan between the start and end TimeStamps of the
        TimeRange.
        """
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
        start, end = self.start, self.end
        if bound.start.after(self.start):
            start = bound.start
        if bound.start.after(self.end):
            end = bound.start
        if bound.end.before(self.end):
            end = bound.end
        if bound.end.before(self.start):
            start = bound.end
        return TimeRange(start, end)

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

    def overlaps_with(self, tr: TimeRange, *, delta: CrudeTimeSpan = 0) -> bool:
        """:returns: True if the time ranges share at least delta of time. Ranges that
        only touch at a boundary do not overlap.
        :param tr: The time range to compare against.
        :param delta: The minimum shared span. Defaults to zero.
        """
        other = tr.make_valid()
        rng = self.make_valid()
        if rng == other:
            return rng.span >= TimeSpan(delta)
        if other.end == rng.start or rng.end == other.start:
            return False
        start = max(rng.start, other.start)
        end = min(rng.end, other.end)
        if end < start:
            return False
        return TimeSpan(int(end) - int(start)) >= TimeSpan(delta)

    def swap(self) -> TimeRange:
        """:returns: A new TimeRange with the start and end values swapped. Note that
        this may make the TimeRange invalid.
        """
        self.copy()
        return TimeRange(start=self.end, end=self.start)

    def copy(self, *args: Any, **kwargs: Any) -> TimeRange:
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

    def __hash__(self) -> int:
        return hash((self.start, self.end))

    MAX: ClassVar[TimeRange]
    """The maximum possible TimeRange, starting at TimeStamp.MIN and ending
    a TimeStamp.MAX
    """
    MIN: ClassVar[TimeRange]
    """The minimum possible TimeRange, starting at TimeStamp.MAX and ending at
    TimeStamp.MIN. Note that this TimeRange is invalid.
    """
    ZERO: ClassVar[TimeRange]
    """A TimeRange starting and ending at the unix epoch in UTC (TimeStamp.ZERO)."""


TimeRange.MAX = TimeRange(TimeStamp.MIN, TimeStamp.MAX)
TimeRange.MIN = TimeRange(TimeStamp.MAX, TimeStamp.MIN)
TimeRange.ZERO = TimeRange(0, 0)


class Density(int):
    """Density is the number of bytes contained in a single sample."""

    def __new__(cls, value: CrudeDensity) -> Density:
        if isinstance(value, Density):
            return value
        if isinstance(value, int):
            return super().__new__(cls, value)
        raise TypeError(f"Cannot convert {type(value)} to Density")

    @classmethod
    def __get_pydantic_core_schema__(
        cls, source_type: Any, handler: GetCoreSchemaHandler
    ) -> core_schema.CoreSchema:
        """Implemented for pydantic validation. Should not be used externally."""
        return core_schema.no_info_after_validator_function(cls, handler(int))

    @classmethod
    def validate(cls, v: Any, *args: Any, **kwargs: Any) -> Density:
        """Implemented for pydantic validation. Should not be used externally."""
        return cls(v)

    def sample_count(self, size: CrudeSize) -> int:
        """:returns: The number of contained in the given byte Size."""
        return int(size / self)

    def size_span(self, sample_count: int) -> Size:
        """:returns: The number of bytes occupied by the given number of samples."""
        return Size(sample_count * self)

    def __repr__(self) -> str:
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
        return Size(super().__mul__(Size(rhs)))

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
        return int(self / Size.KB)

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
    BYTE_UNITS: str
    KB: Size
    KB_UNITS: str
    MB: Size
    MB_UNITS: str
    GB: Size
    GB_UNITS: str


Size.BYTE = Size(1)
Size.BYTE_UNITS = "b"
Size.KB = Size(1024) * Size.BYTE
Size.KB_UNITS = "kb"
Size.MB = Size(1024) * Size.KB
Size.MB_UNITS = "mb"
Size.GB = Size(1024) * Size.MB
Size.GB_UNITS = "gb"

_SizeUnits = ["gb", "mb", "kb", "b"]


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

        result = DataType._FROM_NUMPY.get(np.dtype(value), None)
        if result is not None:
            return result

        raise TypeError(f"Cannot convert {type(value)} to DataType")

    @staticmethod
    def _from_sample(value: Any) -> DataType:
        if isinstance(value, str):
            return DataType.STRING
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
        if isinstance(value, np.number):
            result = DataType._FROM_NUMPY.get(np.dtype(value), None)
            if result is not None:
                return result
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
        :return: The numpy type
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


DataType.UUID = _const("uuid")
DataType.UNKNOWN = _const("")
DataType.TIMESTAMP = _const("timestamp")
DataType.FLOAT64 = _const("float64")
DataType.FLOAT32 = _const("float32")
DataType.INT64 = _const("int64")
DataType.INT32 = _const("int32")
DataType.INT16 = _const("int16")
DataType.INT8 = _const("int8")
DataType.UINT64 = _const("uint64")
DataType.UINT32 = _const("uint32")
DataType.UINT16 = _const("uint16")
DataType.UINT8 = _const("uint8")
DataType.JSON = _const("json")
DataType.STRING = _const("string")
DataType.BYTES = _const("bytes")
DataType.BOOLEAN = _const("boolean")
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

CrudeTimeStamp: TypeAlias = (
    int | TimeStamp | TimeSpan | datetime | timedelta | np.datetime64 | np.int64 | str
)
CrudeTimeSpan: TypeAlias = (
    int | float | TimeSpan | TimeStamp | timedelta | np.timedelta64 | str
)
_ComparableTimeSpan: TypeAlias = int | float | timedelta
CrudeRate: TypeAlias = int | float | TimeSpan | Rate
CrudeDensity: TypeAlias = Density | int
CrudeDataType: TypeAlias = (
    DTypeLike | DataType | str | list[str] | np.number | int | float
)
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
    np.dtype(np.datetime64): DataType.TIMESTAMP,
    np.dtype("datetime64[ns]"): DataType.TIMESTAMP,
    np.dtype(np.str_): DataType.STRING,
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

    def __str__(self) -> str:
        """Returns a string representation of the Alignment in the format
        'domain-sample'.

        :returns: A string representation of the Alignment.
        """
        return f"{self.domain_index}-{self.sample_index}"

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
