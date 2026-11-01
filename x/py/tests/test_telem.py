#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import operator
import time
import uuid
from collections.abc import Callable, Iterator
from datetime import UTC, datetime, timedelta
from fractions import Fraction

import dateutil
import numpy as np
import pandas as pd
import pytest
from hypothesis import given
from hypothesis import strategies as st
from pydantic import ValidationError

import x.telem as sy
from x.exceptions import ContiguityError

_now = sy.TimeStamp.now()

EST = dateutil.tz.gettz("EST")

_INT64 = st.integers(min_value=-(2**63), max_value=2**63 - 1)
_SPAN = st.integers(min_value=-(2**62), max_value=2**62)
_NONZERO = _SPAN.filter(lambda v: v != 0)
_STAMP = st.integers(min_value=0, max_value=2**62)
_OUT_OF_INT64 = st.integers(min_value=2**63) | st.integers(max_value=-(2**63) - 1)
_SMALL_SPAN = st.integers(min_value=-(2**61), max_value=2**61)
_SMALL_STAMP = st.integers(min_value=0, max_value=2**61)
_FACTOR = st.floats(min_value=-1000, max_value=1000, allow_nan=False)
_BARE_NUMBERS = [
    pytest.param(5, id="int"),
    pytest.param(5.0, id="float"),
    pytest.param(np.int64(5), id="int64"),
    pytest.param(np.float64(5), id="float64"),
]


@pytest.mark.telem
class TestTimeStamp:
    def test_now(self) -> None:
        """Should return the current timestamp"""
        now = sy.TimeStamp.now() + sy.TimeSpan.SECOND
        assert now.datetime() > datetime.now().astimezone()

    @pytest.mark.parametrize(
        ("crude", "expected"),
        [
            (1000, 1000),
            (sy.TimeSpan.MILLISECOND * 2500, 2500000000),
            (105 * sy.TimeSpan.MILLISECOND, 105 * sy.TimeSpan.MILLISECOND),
            (
                datetime.fromtimestamp(105, UTC),
                sy.TimeStamp(105 * sy.TimeSpan.SECOND),
            ),
            (_now, _now),
            (timedelta(seconds=105), sy.TimeStamp(105 * sy.TimeSpan.SECOND)),
            (np.datetime64(1000, "ms"), sy.TimeStamp(1000 * sy.TimeSpan.MILLISECOND)),
            (
                datetime(2022, 2, 22, 15, 41, 50, tzinfo=EST),
                sy.TimeStamp(1645562510000000000),
            ),
            (
                datetime(2022, 2, 22, 15, 41, 50, tzinfo=UTC),
                sy.TimeStamp(1645544510000000000),
            ),
            (
                datetime(2022, 2, 22, 10, 41, 50, tzinfo=EST),
                sy.TimeStamp(1645544510000000000),
            ),
            (
                pd.Timestamp(datetime(2022, 2, 22, 15, 41, 50, tzinfo=EST)),
                sy.TimeStamp(1645562510000000000),
            ),
            (np.int64(1000), sy.TimeStamp(1 * sy.TimeSpan.MICROSECOND)),
        ],
    )
    def test_construction(
        self, crude: sy.CrudeTimeStamp, expected: sy.TimeStamp
    ) -> None:
        """Should initialize a timestamp from a variety of types"""
        delta = sy.TimeSpan(sy.TimeStamp(crude) - sy.TimeStamp(expected))
        assert sy.TimeStamp(crude) == expected, f"""
        Expected: {sy.TimeStamp(expected)}
        Got: {sy.TimeStamp(crude)}
        Delta: {delta}
        """

    @pytest.fixture
    def non_utc_tz(self, monkeypatch: pytest.MonkeyPatch) -> Iterator[None]:
        if not hasattr(time, "tzset"):
            pytest.skip("time.tzset is not available on this platform")
        monkeypatch.setenv("TZ", "America/New_York")
        time.tzset()
        yield
        monkeypatch.undo()
        time.tzset()

    @pytest.mark.usefixtures("non_utc_tz")
    def test_naive_pd_timestamp_is_utc(self) -> None:
        """Should treat a naive pd.Timestamp as UTC regardless of the local timezone"""
        ts = sy.TimeStamp(pd.Timestamp("2022-02-22 15:41:50"))
        assert ts == 1645544510000000000

    @pytest.mark.usefixtures("non_utc_tz")
    def test_aware_pd_timestamp(self) -> None:
        """Should convert an aware pd.Timestamp from its own timezone"""
        ts = sy.TimeStamp(pd.Timestamp("2022-02-22 10:41:50", tz="America/New_York"))
        assert ts == 1645544510000000000

    @pytest.mark.usefixtures("non_utc_tz")
    def test_pd_timestamp_nanoseconds(self) -> None:
        """Should preserve the nanoseconds of a pd.Timestamp"""
        ts = sy.TimeStamp(pd.Timestamp(1645544510123456789))
        assert ts == 1645544510123456789

    @pytest.mark.usefixtures("non_utc_tz")
    def test_naive_pd_timestamp_matches_datetime64(self) -> None:
        """Should convert a naive pd.Timestamp and np.datetime64 to the same value"""
        value = "2022-02-22T15:41:50.123456789"
        assert sy.TimeStamp(pd.Timestamp(value)) == sy.TimeStamp(
            np.datetime64(value, "ns")
        )

    def test_nat_datetime64(self) -> None:
        """Should raise a ValueError for a NaT np.datetime64"""
        with pytest.raises(ValueError, match="NaT"):
            sy.TimeStamp(np.datetime64("NaT"))

    def test_aware_datetime_microseconds(self) -> None:
        """Should convert an aware datetime with exact microseconds"""
        ts = sy.TimeStamp(datetime(2262, 4, 11, 23, 47, 16, 854775, tzinfo=UTC))
        assert ts == 9223372036854775000

    def test_timedelta_microseconds(self) -> None:
        """Should convert a timedelta with exact microseconds"""
        ts = sy.TimeStamp(timedelta(days=19045, microseconds=123457))
        assert ts == 1645488000123457000

    def test_after_false(self) -> None:
        """Should return true if the timestamp is after the given timestamp"""
        ts = sy.TimeStamp(1000)
        assert not ts > sy.TimeSpan.MICROSECOND

    def test_after_true(self) -> None:
        """Should return true if the timestamp is after the given timestamp"""
        ts = sy.TimeStamp(10000)
        assert ts > sy.TimeSpan.MICROSECOND

    def test_after_eq_after(self) -> None:
        """Should return true if the timestamp is after or equal to the given
        timestamp"""
        ts = sy.TimeStamp(1000)
        assert ts >= sy.TimeSpan.MICROSECOND

    def test_after_eq_before(self) -> None:
        """Should return true if the timestamp is after or equal to the given
        timestamp
        """
        ts = sy.TimeStamp(100)
        assert not ts >= sy.TimeSpan.MICROSECOND

    def test_before_false(self) -> None:
        """Should return true if the timestamp is before the given timestamp"""
        ts = sy.TimeStamp(1000)
        assert not ts < sy.TimeSpan.MICROSECOND

    def test_before_true(self) -> None:
        """Should return true if the timestamp is before the given timestamp"""
        ts = sy.TimeStamp(100)
        assert ts < sy.TimeSpan.MICROSECOND

    def test_before_eq_before(self) -> None:
        """Should return true if the timestamp is before or equal to the given
        timestamp
        """
        ts = sy.TimeStamp(100)
        assert ts <= sy.TimeSpan.MICROSECOND

    def test_before_eq_after(self) -> None:
        """Should return true if the timestamp is before or equal to the given
        timestamp
        """
        ts = sy.TimeStamp(1000)
        assert ts <= sy.TimeSpan.MICROSECOND

    def test_add(self) -> None:
        """Should add a timespan to a timestamp"""
        ts = sy.TimeStamp(1000)
        ts += sy.TimeSpan.MICROSECOND
        assert ts == sy.TimeStamp(2000)

    def test_sub(self) -> None:
        """Should subtract a timespan from a timestamp"""
        ts = sy.TimeStamp(2000)
        ts = ts - sy.TimeSpan.MICROSECOND
        assert ts == sy.TimeStamp(1000)

    def test_span_range(self) -> None:
        """Should return a rng of timestamps between two timestamps"""
        ts1 = sy.TimeStamp(1000)
        ts2 = sy.TimeSpan(2000)
        rng = ts1.span_range(ts2)
        assert rng.span == 2 * sy.TimeSpan.MICROSECOND

    def test_range(self) -> None:
        """Should return a rng of timestamps between two timestamps"""
        ts1 = sy.TimeStamp(1000)
        ts2 = sy.TimeStamp(2000)
        rng = ts1.range(ts2)
        assert rng.span == sy.TimeSpan.MICROSECOND

    def test_datetime(self) -> None:
        """Should correctly convert the sy.TimeStamp to a datetime in local time."""
        ts1 = sy.TimeStamp(1645562510000000000)
        assert ts1.datetime(tz=UTC) == datetime(2022, 2, 22, 20, 41, 50, tzinfo=UTC)

    def test_trunc(self) -> None:
        """Should correctly return the truncation of a standard sy.TimeSpan divisor"""
        ts1 = sy.TimeStamp(1 * sy.TimeSpan.DAY + 1 * sy.TimeSpan.HOUR)
        assert ts1.trunc(sy.TimeSpan.DAY) == (1 * sy.TimeSpan.DAY)


@pytest.mark.telem
class TestTimeRange:
    def test_construction_from_datetime(self) -> None:
        """Should initialize a sy.TimeRange from a datetime"""
        dt = datetime(2020, 1, 1, 0, 0, 0).astimezone()
        dt2 = datetime(2021, 1, 1, 0, 0, 0).astimezone()
        tr = sy.TimeRange(dt, dt2)
        assert tr.start.datetime() == dt
        assert tr.end.datetime() == dt2

    def test_span(self) -> None:
        """Should return a valid sy.TimeSpan"""
        tr = sy.TimeRange(0, 1000)
        assert tr.span == sy.TimeSpan(1000)

    def test_bound_by(self) -> None:
        """Should return a bound version of the range"""
        tr = sy.TimeRange(0, 1000)
        bound = tr.clamp(sy.TimeRange(100, 500))
        assert bound.span == 400 * sy.TimeSpan.NANOSECOND

    def test_contains_stamp(self) -> None:
        """Should return true if the range contains a timestamp"""
        tr = sy.TimeRange(0, 1000)
        assert tr.contains(sy.TimeStamp(500))

    def test_doesnt_contain_stamp(self) -> None:
        """Should return false if the range doesn't contain a timestamp"""
        tr = sy.TimeRange(0, 1000)
        assert not tr.contains(sy.TimeStamp(1500))

    def test_stamp_contains_end_of_range(self) -> None:
        """Should return false if the timestamp is the same as the end of the range"""
        tr = sy.TimeRange(0, 1000)
        assert not tr.contains(sy.TimeStamp(1000))

    def test_stamp_contains_start_of_range(self) -> None:
        """Should return true if the timestamp is the same as the start of the range"""
        tr = sy.TimeRange(0, 1000)
        assert tr.contains(sy.TimeStamp(0))

    def test_range_not_contains_range(self) -> None:
        """Should return true if the ranges overlap but a smaller range is not
        contained
        """
        tr = sy.TimeRange(0, 1000)
        tr2 = sy.TimeRange(500, 1500)
        assert not tr.contains(tr2)

    def test_range_contains_range(self) -> None:
        """Should return true if the ranges overlap and the smaller range is
        contained
        """
        tr = sy.TimeRange(0, 1000)
        tr2 = sy.TimeRange(500, 900)
        assert tr.contains(tr2)

    def test_range_contains_equal(self) -> None:
        """Should return true if the ranges are equal"""
        tr = sy.TimeRange(0, 1000)
        tr2 = sy.TimeRange(0, 1000)
        assert tr.contains(tr2)

    def test_range_overlaps(self) -> None:
        """Should return true if the ranges overlap"""
        tr = sy.TimeRange(0, 1000)
        tr2 = sy.TimeRange(500, 900)
        assert tr.overlaps_with(tr2)

    def test_range_overlaps_equal(self) -> None:
        """Should return true if the ranges are equal"""
        tr = sy.TimeRange(0, 1000)
        tr2 = sy.TimeRange(0, 1000)
        assert tr.overlaps_with(tr2)

    def test_range_overlaps_false(self) -> None:
        """Should return false if the ranges don't overlap"""
        tr = sy.TimeRange(0, 1000)
        tr2 = sy.TimeRange(1500, 2000)
        assert not tr.overlaps_with(tr2)

    def test_range_overlaps_partial(self) -> None:
        """Should overlap a range that starts inside it, in either direction"""
        tr = sy.TimeRange(0, 10)
        tr2 = sy.TimeRange(5, 20)
        assert tr.overlaps_with(tr2)
        assert tr2.overlaps_with(tr)

    def test_range_overlaps_touching(self) -> None:
        """Should not overlap a range that only touches its boundary"""
        assert not sy.TimeRange(0, 10).overlaps_with(sy.TimeRange(10, 20))
        assert not sy.TimeRange(10, 20).overlaps_with(sy.TimeRange(0, 10))

    def test_range_overlaps_delta(self) -> None:
        """Should overlap only when the shared span is at least delta"""
        tr = sy.TimeRange(0, 10)
        tr2 = sy.TimeRange(5, 20)
        assert tr.overlaps_with(tr2, delta=5)
        assert not tr.overlaps_with(tr2, delta=6)

    def test_range_overlaps_equal_delta(self) -> None:
        """Should not overlap an equal range sharing less than delta"""
        tr = sy.TimeRange(0, 10)
        assert tr.overlaps_with(sy.TimeRange(0, 10), delta=10)
        assert not tr.overlaps_with(sy.TimeRange(0, 10), delta=11)

    def test_range_valid(self) -> None:
        """Should return true if the range is valid"""
        tr = sy.TimeRange(0, 1000)
        assert tr.valid

    def test_range_invalid(self) -> None:
        """Should return false if the range is invalid"""
        tr = sy.TimeRange(1000, 0)
        assert not tr.valid

    def test_range_swap(self) -> None:
        """Should swap the start and end times"""
        tr = sy.TimeRange(1000, 0)
        tr = tr.swap()
        assert tr.start == sy.TimeStamp(0)
        assert tr.end == sy.TimeStamp(1000)


@pytest.mark.telem
class TestTimeSpan:
    def test_since(self) -> None:
        """Should return the sy.TimeSpan since the given timestamp"""
        now = sy.TimeStamp.now()
        one_sec_ago = now - 1 * sy.TimeSpan.SECOND
        assert sy.TimeSpan.since(one_sec_ago) < 1002 * sy.TimeSpan.MILLISECOND
        assert sy.TimeSpan.since(one_sec_ago) > 998 * sy.TimeSpan.MILLISECOND

    @pytest.mark.parametrize(
        ("crude", "expected"),
        [
            (1000, sy.TimeSpan.MICROSECOND),
            (timedelta(microseconds=1000), 1000 * sy.TimeSpan.MICROSECOND),
            (sy.TimeStamp(1000), sy.TimeSpan.MICROSECOND),
            (np.timedelta64(1000, "us"), 1000 * sy.TimeSpan.MICROSECOND),
            (pd.Timedelta(1000, "ms"), 1000 * sy.TimeSpan.MILLISECOND),
            (sy.TimeSpan.MICROSECOND * 1000, sy.TimeSpan.MICROSECOND * 1000),
            (np.int64(1000), 1 * sy.TimeSpan.MICROSECOND),
        ],
    )
    def test_construction(self, crude: sy.CrudeTimeSpan, expected: sy.TimeSpan) -> None:
        assert sy.TimeSpan(crude) == expected

    @pytest.mark.parametrize(
        ("crude", "expected"),
        [
            (pd.Timedelta(1, "s") + pd.Timedelta(1, "ns"), 1_000_000_001),
            (pd.Timedelta(days=-3, nanoseconds=7), -3 * 86_400 * 10**9 + 7),
            (np.timedelta64(5, "ns"), 5),
            (np.timedelta64(3, "us"), 3_000),
            (np.timedelta64(2, "ms"), 2_000_000),
            (np.timedelta64(-7, "s"), -7 * 10**9),
            (np.timedelta64(1, "D"), 86_400 * 10**9),
            (np.timedelta64(10**18 + 1, "ns"), 10**18 + 1),
            (timedelta(days=10_000, microseconds=1), 10_000 * 86_400 * 10**9 + 1_000),
            (timedelta(seconds=-1, microseconds=1), -999_999_000),
        ],
    )
    def test_construction_exact(self, crude: sy.CrudeTimeSpan, expected: int) -> None:
        """Should convert durations to exact integer nanoseconds"""
        assert sy.TimeSpan(crude) == expected

    def test_construction_nat(self) -> None:
        """Should reject a NaT timedelta64"""
        with pytest.raises(ValueError, match="NaT"):
            sy.TimeSpan(np.timedelta64("NaT", "ns"))

    @pytest.mark.parametrize(
        "crude",
        [np.timedelta64(200_000, "D"), np.timedelta64(-200_000, "D")],
    )
    def test_construction_out_of_range(self, crude: np.timedelta64) -> None:
        """Should reject a timedelta64 that does not fit in 64-bit nanoseconds"""
        with pytest.raises(ValueError, match="64-bit"):
            sy.TimeSpan(crude)

    def test_construction_max(self) -> None:
        """Should accept the largest timedelta64 that fits in 64-bit nanoseconds"""
        assert sy.TimeSpan(np.timedelta64(2**63 - 1, "ns")) == 2**63 - 1

    def test_seconds(self) -> None:
        """Should return the number of seconds in the timespan"""
        assert sy.TimeSpan.SECOND.seconds == 1

    def test_timedelta(self) -> None:
        """Should return a timedelta"""
        assert sy.TimeSpan.SECOND.timedelta == timedelta(seconds=1)

    def test_add(self) -> None:
        """Should correctly add two time spans"""
        assert sy.TimeSpan(2000) == sy.TimeSpan.MICROSECOND + sy.TimeSpan.MICROSECOND

    def test_sub(self) -> None:
        """Should correctly subtract two time spans"""
        assert sy.TimeSpan(0) == sy.TimeSpan.MICROSECOND - sy.TimeSpan.MICROSECOND

    def test_abs_positive(self) -> None:
        """Should return the same value for positive spans"""
        assert abs(sy.TimeSpan(1000)) == sy.TimeSpan(1000)

    def test_abs_negative(self) -> None:
        """Should return the absolute value for negative spans"""
        assert abs(sy.TimeSpan(-500)) == sy.TimeSpan(500)

    def test_abs_zero(self) -> None:
        """Should return zero for zero span"""
        assert abs(sy.TimeSpan(0)) == sy.TimeSpan(0)

    def test_abs_returns_timespan(self) -> None:
        """Should return a TimeSpan instance"""
        assert isinstance(abs(sy.TimeSpan(-1000)), sy.TimeSpan)

    def test_gt(self) -> None:
        """Should correctly compare two time spans"""
        assert sy.TimeSpan.MICROSECOND > sy.TimeSpan.NANOSECOND

    def test_lt(self) -> None:
        """Should correctly compare two time spans"""
        assert sy.TimeSpan.NANOSECOND < sy.TimeSpan.MICROSECOND

    def test_le(self) -> None:
        """Should correctly compare two time spans"""
        assert sy.TimeSpan.NANOSECOND <= sy.TimeSpan.MICROSECOND

    @pytest.mark.parametrize(
        ("span", "expected"),
        [
            (
                1 * sy.TimeSpan.DAY
                + 10 * sy.TimeSpan.MINUTE
                + 100 * sy.TimeSpan.MILLISECOND,
                "1d 10m 100ms",
            ),
            (10 * sy.TimeSpan.HOUR + 10 * sy.TimeSpan.NANOSECOND, "10h 10ns"),
            (sy.TimeSpan.ZERO, "0ns"),
        ],
    )
    def test_str(self, span: sy.TimeSpan, expected: str) -> None:
        """Should correctly display the sy.TimeSpan as a human-readable string"""
        assert str(span) == expected

    @pytest.mark.parametrize(
        ("span", "expected"),
        [
            (
                1.0,
                1 * sy.TimeSpan.SECOND,
            ),
            (
                1,
                1 * sy.TimeSpan.SECOND,
            ),
            (
                1 * sy.TimeSpan.SECOND,
                1 * sy.TimeSpan.SECOND,
            ),
        ],
    )
    def test_from_seconds(self, span: object, expected: sy.TimeSpan) -> None:
        """It should evaluate pure floats or integers as seconds"""
        abc = sy.TimeSpan.from_seconds(1.0)
        assert abc == sy.TimeSpan(1 * sy.TimeSpan.SECOND)

    @pytest.mark.parametrize(
        ("span", "expected"),
        [
            (1.0, 1.0),
            (1, 1),
            (1 * sy.TimeSpan.MILLISECOND, 0.001),
        ],
    )
    def test_to_seconds(self, span: sy.CrudeTimeSpan, expected: float) -> None:
        """It should evaluate pure floats or integers as seconds"""
        abc = sy.TimeSpan.to_seconds(span)
        assert abc == expected


@pytest.mark.telem
class TestTimeSpanPrecision:
    big = sy.TimeSpan(2**60 + 1)

    def test_nanoseconds(self) -> None:
        """Should return the exact nanosecond count"""
        assert self.big.nanoseconds == 2**60 + 1

    def test_mul(self) -> None:
        """Should multiply without rounding through a float"""
        assert int(self.big * 3) == (2**60 + 1) * 3

    def test_div(self) -> None:
        """Should divide without rounding through a float"""
        assert int(self.big / 1) == 2**60 + 1
        assert int(sy.TimeSpan(-7) / 2) == -3

    def test_trunc(self) -> None:
        """Should truncate without rounding up"""
        span = sy.TimeSpan(10**18 - 1)
        assert span.trunc(sy.TimeSpan.SECOND) == sy.TimeSpan(999999999 * 10**9)

    def test_str_max(self) -> None:
        """Should format the maximum span exactly"""
        assert str(sy.TimeSpan.MAX) == "106751d 23h 47m 16s 854ms 775us 807ns"


@pytest.mark.telem
class TestTimeArithmetic:
    @given(_SMALL_STAMP, _SMALL_STAMP)
    def test_stamp_minus_stamp(self, a: int, b: int) -> None:
        """Should return the span between two timestamps"""
        diff = sy.TimeStamp(a) - sy.TimeStamp(b)
        assert type(diff) is sy.TimeSpan
        assert int(diff) == a - b

    @given(_SMALL_STAMP, _SMALL_SPAN)
    def test_stamp_and_span(self, stamp: int, span: int) -> None:
        """Should return a timestamp when adding or subtracting a span"""
        ts, sp = sy.TimeStamp(stamp), sy.TimeSpan(span)
        for result, expected in (
            (ts + sp, stamp + span),
            (sp + ts, stamp + span),
            (ts - sp, stamp - span),
        ):
            assert type(result) is sy.TimeStamp
            assert int(result) == expected

    @given(_SMALL_SPAN, _SMALL_SPAN)
    def test_span_and_span(self, a: int, b: int) -> None:
        """Should return a span when adding or subtracting spans"""
        for result, expected in (
            (sy.TimeSpan(a) + sy.TimeSpan(b), a + b),
            (sy.TimeSpan(a) - sy.TimeSpan(b), a - b),
        ):
            assert type(result) is sy.TimeSpan
            assert int(result) == expected

    @given(_SMALL_SPAN)
    def test_neg_and_abs(self, value: int) -> None:
        """Should return a span from negation and abs"""
        for result, expected in (
            (-sy.TimeSpan(value), -value),
            (abs(sy.TimeSpan(value)), abs(value)),
        ):
            assert type(result) is sy.TimeSpan
            assert int(result) == expected

    @given(st.integers(min_value=-(2**40), max_value=2**40), st.integers(-1000, 1000))
    def test_scale_by_int(self, value: int, factor: int) -> None:
        """Should return a span when multiplying by an integer on either side"""
        for result in (sy.TimeSpan(value) * factor, factor * sy.TimeSpan(value)):
            assert type(result) is sy.TimeSpan
            assert int(result) == value * factor

    @given(st.integers(min_value=-(2**40), max_value=2**40), _FACTOR)
    def test_scale_by_float(self, value: int, factor: float) -> None:
        """Should return a span when multiplying by a float"""
        result = sy.TimeSpan(value) * factor
        assert type(result) is sy.TimeSpan
        assert int(result) == int(value * factor)

    @given(_SPAN, _NONZERO)
    def test_div_by_int(self, a: int, b: int) -> None:
        """Should return a span truncated toward zero when dividing by an integer"""
        result = sy.TimeSpan(a) / b
        assert type(result) is sy.TimeSpan
        assert int(result) == int(Fraction(a, b))

    @given(
        st.integers(min_value=-(2**52), max_value=2**52),
        _FACTOR.filter(lambda v: abs(v) >= 1e-3),
    )
    def test_div_by_float(self, a: int, b: float) -> None:
        """Should return a span truncated toward zero when dividing by a float"""
        result = sy.TimeSpan(a) / b
        assert type(result) is sy.TimeSpan
        assert int(result) == int(Fraction(a) / Fraction(b))

    @given(_SPAN, _NONZERO)
    def test_floordiv_by_int(self, a: int, b: int) -> None:
        """Should return a floored span when floor dividing by an integer"""
        result = sy.TimeSpan(a) // b
        assert type(result) is sy.TimeSpan
        assert int(result) == a // b

    @given(_SPAN, _NONZERO)
    def test_div_by_span(self, a: int, b: int) -> None:
        """Should return a float ratio when dividing a span by a span"""
        result = sy.TimeSpan(a) / sy.TimeSpan(b)
        assert type(result) is float
        assert result == float(Fraction(a, b))

    @given(_SPAN, _NONZERO)
    def test_floordiv_by_span(self, a: int, b: int) -> None:
        """Should return an int count when floor dividing a span by a span"""
        result = sy.TimeSpan(a) // sy.TimeSpan(b)
        assert type(result) is int
        assert result == a // b

    @given(_SPAN, _NONZERO)
    def test_mod_by_span(self, a: int, b: int) -> None:
        """Should return a span remainder"""
        result = sy.TimeSpan(a) % sy.TimeSpan(b)
        assert type(result) is sy.TimeSpan
        assert int(result) == a % b

    def test_ratio_of_units(self) -> None:
        """Should count the seconds in a minute"""
        assert sy.TimeSpan.MINUTE / sy.TimeSpan.SECOND == 60.0
        assert sy.TimeSpan.MINUTE // sy.TimeSpan.SECOND == 60

    @pytest.mark.parametrize(
        "operation",
        [
            lambda: sy.TimeStamp(1) + sy.TimeStamp(1),
            lambda: sy.TimeStamp(1) * 2,
            lambda: 2 * sy.TimeStamp(1),
            lambda: -sy.TimeStamp(1),
            lambda: sy.TimeSpan(1) - sy.TimeStamp(1),
        ],
        ids=["stamp + stamp", "stamp * 2", "2 * stamp", "-stamp", "span - stamp"],
    )
    def test_invalid_stamp_operations(self, operation: Callable[[], object]) -> None:
        """Should reject operations that have no meaning for a timestamp"""
        with pytest.raises(TypeError):
            operation()

    @pytest.mark.parametrize("op", [operator.add, operator.sub])
    @pytest.mark.parametrize("number", _BARE_NUMBERS)
    @pytest.mark.parametrize("value", [sy.TimeStamp(10), sy.TimeSpan(10)])
    def test_bare_number_rejected(
        self, op: Callable[[object, object], object], value: int, number: object
    ) -> None:
        """Should reject a bare number after a time value"""
        with pytest.raises(TypeError):
            op(value, number)

    @pytest.mark.parametrize("op", [operator.add, operator.sub])
    @pytest.mark.parametrize("value", [sy.TimeStamp(10), sy.TimeSpan(10)])
    def test_bare_int_before_rejected(
        self, op: Callable[[object, object], object], value: int
    ) -> None:
        """Should reject a bare int before a time value"""
        with pytest.raises(TypeError):
            op(5, value)

    @pytest.mark.parametrize("number", _BARE_NUMBERS)
    def test_bare_number_mod_rejected(self, number: object) -> None:
        """Should reject a bare number as the modulus of a span"""
        with pytest.raises(TypeError):
            operator.mod(sy.TimeSpan(10), number)


@pytest.mark.telem
class TestRate:
    @pytest.mark.parametrize(
        ("crude", "expected"),
        [
            (sy.TimeSpan.MILLISECOND, sy.Rate(1000)),
            (1000, sy.Rate(1000.0)),
        ],
    )
    def test_construction(self, crude: sy.CrudeRate, expected: sy.Rate) -> None:
        assert sy.Rate(crude) == expected

    def test_invalid_init(self) -> None:
        """Should raise an exception if the rate is invalid"""
        with pytest.raises(TypeError):
            sy.Rate(timedelta(seconds=1))  # type: ignore[arg-type]

    def test_sample_count(self) -> None:
        """Should return the number of samples"""
        assert sy.Rate(1.0).sample_count(5 * sy.TimeSpan.SECOND) == 5

    def test_byte_size(self) -> None:
        """Should return the number of bytes in the given span"""
        assert (
            sy.Rate(1.0).byte_size(5 * sy.TimeSpan.SECOND, density=sy.Density.BIT64)
            == 40
        )

    def test_byte_span(self) -> None:
        """Should return the time span from a byte size"""
        assert (
            sy.Rate(1.0).size_span(sy.Size(40), density=sy.Density.BIT64)
            == 5 * sy.TimeSpan.SECOND
        )

    def test_span_exact(self) -> None:
        """Should compute the span without truncating the period first"""
        assert sy.Rate(3).span(3_000_000_000) == 10**9 * sy.TimeSpan.SECOND

    @pytest.mark.parametrize(
        ("rate", "count", "expected"),
        [
            (0.1, 1, 10 * sy.TimeSpan.SECOND),
            (0.5, 1, 2 * sy.TimeSpan.SECOND),
            (2.5, 1, 400 * sy.TimeSpan.MILLISECOND),
            (2.5, 5, 2 * sy.TimeSpan.SECOND),
            (1 / 3, 1, 3 * sy.TimeSpan.SECOND),
        ],
    )
    def test_span_fractional(
        self, rate: float, count: int, expected: sy.TimeSpan
    ) -> None:
        """Should give the decimal answer for a fractional rate"""
        assert sy.Rate(rate).span(count) == expected

    def test_byte_span_invalid(self) -> None:
        """Should raise a contiguity error if the size is not a multiple of the
        density
        """
        with pytest.raises(ContiguityError):
            sy.Rate(1.0).size_span(sy.Size(41), density=sy.Density.BIT64)

    @given(st.floats(0, 1e9), st.floats(0, 1e9))
    def test_rate_and_rate(self, a: float, b: float) -> None:
        """Should return a rate when adding or subtracting rates"""
        for result, expected in (
            (sy.Rate(a) + sy.Rate(b), a + b),
            (sy.Rate(a) - sy.Rate(b), a - b),
        ):
            assert type(result) is sy.Rate
            assert float(result) == expected

    @given(st.floats(0, 1e9), _FACTOR)
    def test_scale(self, rate: float, factor: float) -> None:
        """Should return a rate when multiplying by a number on either side"""
        for result in (sy.Rate(rate) * factor, factor * sy.Rate(rate)):
            assert type(result) is sy.Rate
            assert float(result) == rate * factor

    def test_ratio(self) -> None:
        """Should return a float when dividing a rate by a rate"""
        result = sy.Rate.KHZ / sy.Rate(10)
        assert type(result) is float
        assert result == 100.0

    @pytest.mark.parametrize("op", [operator.add, operator.sub])
    @pytest.mark.parametrize("number", _BARE_NUMBERS)
    def test_bare_number_rejected(
        self, op: Callable[[object, object], object], number: object
    ) -> None:
        """Should reject a bare number after a rate"""
        with pytest.raises(TypeError):
            op(sy.Rate(10), number)

    @pytest.mark.parametrize("op", [operator.add, operator.sub])
    @pytest.mark.parametrize("number", [5, 5.0])
    def test_bare_number_before_rejected(
        self, op: Callable[[object, object], object], number: float
    ) -> None:
        """Should reject a bare int or float before a rate"""
        with pytest.raises(TypeError):
            op(number, sy.Rate(10))


@pytest.mark.telem
class TestDataType:
    @pytest.mark.parametrize(
        ("crude", "expected"),
        [
            (np.int8, sy.DataType.INT8),
            (np.int16, sy.DataType.INT16),
            ("int32", sy.DataType.INT32),
            ("int64", sy.DataType.INT64),
            (np.float32(1), sy.DataType.FLOAT32),
            ({"a": 1}, sy.DataType.JSON),
            (["a"], sy.DataType.STRING),
            (uuid.uuid4(), sy.DataType.UUID),
            (np.dtype("datetime64[ns]"), sy.DataType.TIMESTAMP),
            (np.datetime64, sy.DataType.TIMESTAMP),
        ],
    )
    def test_construction(self, crude: sy.CrudeDataType, expected: sy.DataType) -> None:
        assert sy.DataType(crude) == expected

    def test_string(self) -> None:
        """Should return the string representation of the data type"""
        assert str(sy.DataType.INT8) == "int8"

    @pytest.mark.parametrize(
        ("value", "expected"),
        [
            (sy.DataType.INT8, np.dtype(np.int8)),
            (sy.DataType.FLOAT32, np.dtype(np.float32)),
            (sy.DataType.BOOLEAN, np.dtype(np.bool_)),
        ],
    )
    def test_np(self, value: sy.DataType, expected: np.dtype[np.generic]) -> None:
        """Should return the correct numpy representation of the data type"""
        assert value.np == expected

    @pytest.mark.parametrize(
        ("data_type", "expected"),
        [
            (sy.DataType.STRING, True),
            (sy.DataType.JSON, True),
            (sy.DataType.BYTES, True),
            (sy.DataType.BOOLEAN, False),
            (sy.DataType.UUID, False),
            (sy.DataType.TIMESTAMP, False),
            (sy.DataType.FLOAT64, False),
            (sy.DataType.FLOAT32, False),
            (sy.DataType.INT64, False),
            (sy.DataType.INT32, False),
            (sy.DataType.INT16, False),
            (sy.DataType.INT8, False),
            (sy.DataType.UINT64, False),
            (sy.DataType.UINT32, False),
            (sy.DataType.UINT16, False),
            (sy.DataType.UINT8, False),
        ],
    )
    def test_is_variable(self, data_type: sy.DataType, expected: bool) -> None:
        """is_variable should return True only for variable-length data types."""
        assert data_type.is_variable is expected

    def test_bool_density(self) -> None:
        """Should report BIT8 density for BOOLEAN"""
        assert sy.DataType.BOOLEAN.density == sy.Density.BIT8

    def test_all(self) -> None:
        """ALL should contain every data type exactly once."""
        assert set(sy.DataType.ALL) == {
            sy.DataType.UNKNOWN,
            sy.DataType.UINT8,
            sy.DataType.UINT16,
            sy.DataType.UINT32,
            sy.DataType.UINT64,
            sy.DataType.INT8,
            sy.DataType.INT16,
            sy.DataType.INT32,
            sy.DataType.INT64,
            sy.DataType.FLOAT32,
            sy.DataType.FLOAT64,
            sy.DataType.BOOLEAN,
            sy.DataType.TIMESTAMP,
            sy.DataType.UUID,
            sy.DataType.STRING,
            sy.DataType.JSON,
            sy.DataType.BYTES,
        }
        assert len(sy.DataType.ALL) == len(set(sy.DataType.ALL))

    @pytest.mark.parametrize("name", ["float65", "datetime", "f8", "FLOAT64"])
    def test_unknown_name(self, name: str) -> None:
        """Should raise a ValueError for a name that is not a data type."""
        with pytest.raises(ValueError, match="not a valid data type"):
            sy.DataType(name)

    @pytest.mark.parametrize("data_type", sy.DataType.ALL)
    def test_known_name(self, data_type: sy.DataType) -> None:
        """Should construct every data type from its name."""
        assert sy.DataType(str(data_type)) == data_type


@pytest.mark.telem
class TestBounds:
    @pytest.mark.parametrize(
        ("value", "expected"),
        [(-1, False), (0, True), (5, True), (9.5, True), (10, False), (11, False)],
    )
    def test_contains(self, value: float, expected: bool) -> None:
        """Should include the lower bound and exclude the upper bound."""
        assert sy.Bounds(0, 10).contains(value) is expected


@pytest.mark.telem
class TestSize:
    @pytest.mark.parametrize(
        ("crude", "expected"),
        [(1, sy.Size.BYTE), (1.0, sy.Size.BYTE), (sy.Size.BYTE, sy.Size.BYTE)],
    )
    def test_construction(self, crude: sy.CrudeSize, expected: sy.Size) -> None:
        assert sy.Size(crude) == expected

    @pytest.mark.parametrize(
        ("value", "expected"),
        [
            (sy.Size.GB + sy.Size.MB * 500, "1gb 500mb"),
            (sy.Size.GB * 0, "0b"),
            (sy.Size(2048), "2kb"),
        ],
    )
    def test_str(self, value: sy.Size, expected: str) -> None:
        assert str(value) == expected


@pytest.mark.telem
@pytest.mark.parametrize(
    ("data", "from_", "to", "expected"),
    [
        (np.array([1, 2, 3]), "s", "ms", 1000),
        (np.array([1, 2, 3]), "ms", "ms", 1),
        (np.array([sy.TimeStamp(0).datetime().isoformat()]), "iso", "ns", 0),
        (np.array([1_700_000_000_123_456_789]), "ns", "ns", 1_700_000_000_123_456_789),
    ],
)
def test_convert_time_units(
    data: np.ndarray,
    from_: sy.TimeSpanUnits,
    to: sy.TimeSpanUnits,
    expected: int | float,
) -> None:
    assert sy.convert_time_units(data, _from=from_, to=to)[0] == expected


@pytest.mark.telem
class TestAlignment:
    def test_construction(self) -> None:
        """Should construct the alignment from the given domain and sample indexes"""
        align = sy.Alignment(2, 1)
        assert align.sample_index == 1
        assert align.domain_index == 2

    def test_construction_zero(self) -> None:
        """Should construct a zero alignment"""
        align = sy.Alignment(0, 0)
        assert int(align) == 0

    def test_default_construction(self) -> None:
        """Should construct a zero alignment by default"""
        align = sy.Alignment()
        assert align.domain_index == 0
        assert align.sample_index == 0

    def test_construction_from_packed_int(self) -> None:
        """Should construct alignment from a packed integer value"""
        # Create an alignment and get its packed value
        align1 = sy.Alignment(5, 10)
        packed = int(align1)
        # Reconstruct from the packed value
        align2 = sy.Alignment(packed)
        assert align2.domain_index == 5
        assert align2.sample_index == 10

    def test_construction_from_string(self) -> None:
        """Should reconstruct a packed alignment delivered as a JSON string.

        The Go server marshals the uint64 alignment as a JSON string to avoid
        float64 precision loss; this is the exact value a streamed frame carries.
        """
        align = sy.Alignment(str((5 << 32) | 10))
        assert align.domain_index == 5
        assert align.sample_index == 10

    def test_construction_from_tuple(self) -> None:
        """Should construct alignment from a tuple"""
        align = sy.Alignment((3, 7))
        assert align.domain_index == 3
        assert align.sample_index == 7

    def test_construction_from_alignment(self) -> None:
        """Should return the same alignment when constructing from Alignment"""
        align1 = sy.Alignment(4, 8)
        align2 = sy.Alignment(align1)
        assert align1 is align2  # Should be the same object
        assert align2.domain_index == 4
        assert align2.sample_index == 8

    def test_domain_index_extraction(self) -> None:
        """Should correctly extract the domain index from the packed value"""
        align = sy.Alignment(5, 10)
        assert align.domain_index == 5

    def test_sample_index_extraction(self) -> None:
        """Should correctly extract the sample index from the packed value"""
        align = sy.Alignment(5, 10)
        assert align.sample_index == 10

    def test_add_samples(self) -> None:
        """Should add to the alignment sample index"""
        align = sy.Alignment(2, 1)
        align = align.add_samples(3)
        assert align.sample_index == 4
        assert align.domain_index == 2

    def test_add_samples_overflow(self) -> None:
        """Should handle sample index overflow correctly"""
        align = sy.Alignment(2, 0xFFFFFFFF - 1)
        align = align.add_samples(1)
        assert align.sample_index == 0xFFFFFFFF
        assert align.domain_index == 2

    def test_add_samples_wrap(self) -> None:
        """Should wrap a sample index past 32 bits without changing the domain"""
        align = sy.Alignment(2, 0xFFFFFFFF).add_samples(1)
        assert align.domain_index == 2
        assert align.sample_index == 0

    def test_add(self) -> None:
        """Should add both domain and sample indices"""
        align1 = sy.Alignment(2, 5)
        align2 = sy.Alignment(3, 10)
        result = align1.add(align2)
        assert result.domain_index == 5
        assert result.sample_index == 15

    def test_str(self) -> None:
        """Should return the string representation of the alignment"""
        align = sy.Alignment(5, 7)
        assert str(align) == "5-7"

    def test_repr(self) -> None:
        """Should return the repr representation of the alignment"""
        align = sy.Alignment(5, 7)
        assert repr(align) == "Alignment(5, 7)"

    def test_comparison(self) -> None:
        """Should correctly compare alignments"""
        align1 = sy.Alignment(2, 5)
        align2 = sy.Alignment(2, 10)
        align3 = sy.Alignment(3, 5)

        # Same domain, different sample
        assert align1 < align2
        assert align2 > align1

        # Different domain
        assert align2 < align3
        assert align3 > align2

    def test_equality(self) -> None:
        """Should correctly compare equality of alignments"""
        align1 = sy.Alignment(2, 5)
        align2 = sy.Alignment(2, 5)
        align3 = sy.Alignment(3, 5)

        assert align1 == align2
        assert align1 != align3

    def test_max_values(self) -> None:
        """Should handle maximum values for domain and sample indices"""
        max_uint32 = 0xFFFFFFFF
        align = sy.Alignment(max_uint32, max_uint32)
        assert align.domain_index == max_uint32
        assert align.sample_index == max_uint32

    def test_large_domain_index(self) -> None:
        """Should correctly handle large domain indices"""
        align = sy.Alignment(1000000, 50)
        assert align.domain_index == 1000000
        assert align.sample_index == 50

    def test_int_conversion(self) -> None:
        """Should correctly convert to int"""
        align = sy.Alignment(2, 1)
        int_value = int(align)
        # 2 << 32 | 1 = 8589934593
        expected = (2 << 32) | 1
        assert int_value == expected

    def test_pydantic_validation(self) -> None:
        """Should work with pydantic validation"""
        from pydantic import BaseModel

        class TestModel(BaseModel):
            alignment: sy.Alignment

        model = TestModel(alignment=sy.Alignment(5, 10))
        assert model.alignment.domain_index == 5
        assert model.alignment.sample_index == 10

        # Should also accept int
        packed_value = (5 << 32) | 10
        model2 = TestModel.model_validate({"alignment": packed_value})
        assert model2.alignment.domain_index == 5
        assert model2.alignment.sample_index == 10


@pytest.mark.telem
class TestSecondsLinspace:
    def test_basic(self) -> None:
        result = sy.seconds_linspace(0, count=3)
        assert len(result) == 3
        assert result[0] == 0
        assert result[1] == sy.TimeSpan.SECOND
        assert result[2] == 2 * sy.TimeSpan.SECOND

    def test_with_offset(self) -> None:
        result = sy.seconds_linspace(5, count=2)
        assert result[0] == 5 * sy.TimeSpan.SECOND
        assert result[1] == 6 * sy.TimeSpan.SECOND

    def test_empty(self) -> None:
        assert sy.seconds_linspace(0, count=0) == []


@pytest.mark.telem
class TestStringEncodedInts:
    """Regression coverage for the JSON wire format. The Go server marshals its
    int64/uint64 telemetry primitives as JSON strings to avoid float64 precision
    loss, so each must parse a decimal string back into its packed integer value.
    """

    @pytest.mark.parametrize("typ", [sy.TimeStamp, sy.TimeSpan, sy.Size, sy.Alignment])
    def test_construction_from_string(self, typ: type) -> None:
        """Should construct the primitive from a decimal string."""
        assert typ("1234567890123") == typ(1234567890123)

    @pytest.mark.parametrize("typ", [sy.TimeStamp, sy.TimeSpan, sy.Size, sy.Alignment])
    def test_construction_from_large_string(self, typ: type) -> None:
        """Should parse a value past the float64 safe-integer range without loss."""
        big = (1 << 62) + 7
        assert int(typ(str(big))) == big

    @pytest.mark.parametrize("typ", [sy.TimeStamp, sy.TimeSpan, sy.Alignment])
    def test_pydantic_validation_from_string(self, typ: type) -> None:
        """Should validate a primitive delivered as a JSON string in a model field."""
        from pydantic import create_model

        model_cls = create_model("M", v=(typ, ...))
        big = (1 << 62) + 7
        validated = model_cls.model_validate({"v": str(big)})
        assert int(validated.model_dump()["v"]) == big


@pytest.mark.telem
class TestTimeProperties:
    @given(_INT64)
    def test_nanoseconds_exact(self, value: int) -> None:
        """Should keep every nanosecond of a span"""
        assert sy.TimeSpan(value).nanoseconds == value

    @given(_SPAN, st.integers(min_value=-1000, max_value=1000))
    def test_mul_exact(self, value: int, factor: int) -> None:
        """Should multiply a span by an integer exactly, or reject overflow"""
        product = value * factor
        if -(2**63) <= product < 2**63:
            assert int(sy.TimeSpan(value) * factor) == product
        else:
            with pytest.raises(ValueError, match="64-bit"):
                sy.TimeSpan(value) * factor

    @given(_SPAN, _NONZERO)
    def test_div_truncates(self, a: int, b: int) -> None:
        """Should divide a span by an integer exactly, truncating toward zero"""
        assert int(sy.TimeSpan(a) / b) == int(Fraction(a, b))

    @given(_SPAN, st.integers(min_value=1, max_value=2**40))
    def test_trunc_bounds(self, value: int, step: int) -> None:
        """Should truncate to a multiple of the step no farther than one step away"""
        truncated = int(sy.TimeSpan(value).trunc(step))
        assert truncated % step == 0
        assert abs(value - truncated) < step
        assert abs(truncated) <= abs(value)

    @given(_STAMP)
    def test_timestamp_round_trip(self, value: int) -> None:
        """Should round-trip a timestamp through pandas and NumPy"""
        ts = sy.TimeStamp(value)
        assert sy.TimeStamp(pd.Timestamp(ts, unit="ns", tz=UTC)) == ts
        assert sy.TimeStamp(np.datetime64(value, "ns")) == ts

    @given(st.integers(min_value=1, max_value=100_000), st.integers(0, 10**9))
    def test_rate_span_exact(self, rate: int, count: int) -> None:
        """Should compute an integer rate's span without rounding"""
        assert int(sy.Rate(rate).span(count)) == count * 10**9 // rate

    @given(st.integers(1, 10**6), st.integers(0, 3), st.integers(0, 10**6))
    def test_rate_span_decimal(self, digits: int, places: int, count: int) -> None:
        """Should compute a decimal rate's span as the decimal value written"""
        rate = Fraction(digits, 10**places)
        expected = int(count * 10**9 / rate)
        assert int(sy.Rate(float(rate)).span(count)) == expected

    @given(_STAMP, _STAMP, _STAMP, _STAMP, st.integers(0, 2**62))
    def test_overlaps_with_symmetric(
        self, a: int, b: int, c: int, d: int, delta: int
    ) -> None:
        """Should overlap symmetrically, and when non-empty ranges share time"""
        r1 = sy.TimeRange(min(a, b), max(a, b))
        r2 = sy.TimeRange(min(c, d), max(c, d))
        assert r1.overlaps_with(r2) == r2.overlaps_with(r1)
        if r1.start < r1.end and r2.start < r2.end:
            shares = max(r1.start, r2.start) < min(r1.end, r2.end)
            assert r1.overlaps_with(r2) == shares
        shared = min(r1.end, r2.end) - max(r1.start, r2.start)
        if r1.overlaps_with(r2, delta=delta):
            assert r1.overlaps_with(r2)
            assert shared >= delta

    @given(_OUT_OF_INT64)
    def test_out_of_range_rejected(self, value: int) -> None:
        """Should reject a value that does not fit in 64 bits at construction"""
        with pytest.raises(ValueError, match="64-bit TimeStamp"):
            sy.TimeStamp(value)
        with pytest.raises(ValueError, match="64-bit TimeSpan"):
            sy.TimeSpan(value)

    @given(_INT64)
    def test_hash_matches_int(self, value: int) -> None:
        """Should hash a timestamp and a span like the equal int"""
        assert hash(sy.TimeStamp(value)) == hash(value)
        assert hash(sy.TimeSpan(value)) == hash(value)
        assert len({sy.TimeStamp(value), sy.TimeSpan(value), value}) == 1

    @given(_SPAN, st.floats(allow_nan=False, allow_infinity=False))
    def test_span_float_eq_matches_hash(self, value: int, other: float) -> None:
        """Should only equal a float whose hash matches"""
        span = sy.TimeSpan(value)
        assert (span == other) == (value == other)
        if span == other:
            assert hash(span) == hash(other)

    @given(_STAMP, st.integers(min_value=2**63))
    def test_compare_with_out_of_range_int(self, value: int, big: int) -> None:
        """Should compare against an int that does not fit in 64 bits"""
        assert sy.TimeStamp(value) < big
        assert sy.TimeSpan(value) < big
        assert sy.TimeStamp(value) != big

    @given(_STAMP, _STAMP)
    def test_time_range_hash_matches_eq(self, start: int, end: int) -> None:
        """Should hash equal time ranges to the same value"""
        assert hash(sy.TimeRange(start, end)) == hash(sy.TimeRange(start, end))
        assert len({sy.TimeRange(start, end), sy.TimeRange(start, end)}) == 1

    def test_time_range_frozen(self) -> None:
        """Should reject a change to a hashable time range"""
        tr = sy.TimeRange(0, 10)
        with pytest.raises(ValidationError):
            sy.TimeRange.__setattr__(tr, "end", sy.TimeStamp(20))
