#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from datetime import datetime, timedelta, timezone

import numpy as np
import pandas as pd
import pytest
from pytz import timezone as pytz_timezone

from synnax import (
    TimeStamp,
    TimeSpan,
    Density,
    UnparsedTimeStamp,
    TimeRange,
    ContiguityError,
    Rate,
    UnparsedTimeSpan,
    Size,
    UnparsedRate,
)

_now = TimeStamp.now()


class TestTimeStamp:
    def test_now(self):
        """Should return the current timestamp"""
        now = TimeStamp.now() + TimeSpan.SECOND
        assert now.datetime() > datetime.now().astimezone()

    @pytest.mark.parametrize(
        "unparsed, expected",
        [
            (1000, 1000),
            (TimeSpan.MILLISECOND * 2500, 2500000000),
            (105 * TimeSpan.MILLISECOND, 105 * TimeSpan.MILLISECOND),
            (
                datetime.utcfromtimestamp(105).replace(tzinfo=timezone.utc),
                TimeStamp(105 * TimeSpan.SECOND),
            ),
            (_now, _now),
            (timedelta(seconds=105), TimeStamp(105 * TimeSpan.SECOND)),
            (np.datetime64(1000, "ms"), TimeStamp(1000 * TimeSpan.MILLISECOND)),
            (
                datetime(2022, 2, 22, 15, 41, 50, tzinfo=pytz_timezone("EST")),
                TimeStamp(1645562510000000000),
            ),
            (
                datetime(2022, 2, 22, 15, 41, 50, tzinfo=timezone.utc),
                TimeStamp(1645544510000000000),
            ),
            (
                datetime(2022, 2, 22, 10, 41, 50, tzinfo=pytz_timezone("EST")),
                TimeStamp(1645544510000000000),
            ),
            (
                pd.Timestamp(
                    datetime(2022, 2, 22, 15, 41, 50, tzinfo=pytz_timezone("EST"))
                ),
                TimeStamp(1645562510000000000),
            ),
        ],
    )
    def test_init(self, unparsed: UnparsedTimeStamp, expected: TimeStamp):
        """Should initialize a timestamp from a variety of types"""
        assert TimeStamp(unparsed) == expected

    def test_invalid_init(self):
        """Should raise an exception if the timestamp is invalid"""
        with pytest.raises(TypeError):
            TimeStamp("dog")  # type: ignore

    def test_is_zero(self):
        """Should return true if the timestamp is zero"""
        ts = TimeStamp(0)
        assert ts.is_zero()

    def test_after_false(self):
        """Should return true if the timestamp is after the given timestamp"""
        ts = TimeStamp(1000)
        assert not ts > TimeSpan.MICROSECOND

    def test_after_true(self):
        """Should return true if the timestamp is after the given timestamp"""
        ts = TimeStamp(10000)
        assert ts > TimeSpan.MICROSECOND

    def test_after_eq_after(self):
        """Should return true if the timestamp is after or equal to the given timestamp"""
        ts = TimeStamp(1000)
        assert ts >= TimeSpan.MICROSECOND

    def test_after_eq_before(self):
        """Should return true if the timestamp is after or equal to the given timestamp"""
        ts = TimeStamp(100)
        assert not ts >= TimeSpan.MICROSECOND

    def test_before_false(self):
        """Should return true if the timestamp is before the given timestamp"""
        ts = TimeStamp(1000)
        assert not ts < TimeSpan.MICROSECOND

    def test_before_true(self):
        """Should return true if the timestamp is before the given timestamp"""
        ts = TimeStamp(100)
        assert ts < TimeSpan.MICROSECOND

    def test_before_eq_before(self):
        """Should return true if the timestamp is before or equal to the given timestamp"""
        ts = TimeStamp(100)
        assert ts <= TimeSpan.MICROSECOND

    def test_before_eq_after(self):
        """Should return true if the timestamp is before or equal to the given timestamp"""
        ts = TimeStamp(1000)
        assert ts <= TimeSpan.MICROSECOND

    def test_add(self):
        """Should add a timespan to a timestamp"""
        ts = TimeStamp(1000)
        ts += TimeSpan.MICROSECOND
        assert ts == TimeStamp(2000)

    def test_sub(self):
        """Should subtract a timespan from a timestamp"""
        ts = TimeStamp(2000)
        ts = ts - TimeSpan.MICROSECOND
        assert ts == TimeStamp(1000)

    def test_span_range(self):
        """Should return a range of timestamps between two timestamps"""
        ts1 = TimeStamp(1000)
        ts2 = TimeSpan(2000)
        range = ts1.span_range(ts2)
        assert range.span() == 2 * TimeSpan.MICROSECOND

    def test_range(self):
        """Should return a range of timestamps between two timestamps"""
        ts1 = TimeStamp(1000)
        ts2 = TimeStamp(2000)
        range = ts1.range(ts2)
        assert range.span() == TimeSpan.MICROSECOND

    def test_datetime(self):
        """Should correctly convert the TimeStamp to a datetime in local time."""
        ts1 = TimeStamp(1645562510000000000)
        assert ts1.datetime(tzinfo=timezone.utc) == datetime(
            2022, 2, 22, 20, 41, 50, tzinfo=timezone.utc
        )


class TestTimeRange:
    def test_init_from_datetime(self):
        """Should initialize a TimeRange from a datetime"""
        dt = datetime(2020, 1, 1, 0, 0, 0).astimezone()
        dt2 = datetime(2021, 1, 1, 0, 0, 0).astimezone()
        tr = TimeRange(dt, dt2)
        assert tr.start.datetime() == dt
        assert tr.end.datetime() == dt2

    def test_span(self):
        """Should return a valid TimeSpan"""
        tr = TimeRange(0, 1000)
        assert tr.span() == TimeSpan(1000)

    def test_is_zero(self):
        """Should return true if the range is zero"""
        tr = TimeRange(0, 0)
        assert tr.is_zero()

    def test_bound_by(self):
        """Should return a bound version of the range"""
        tr = TimeRange(0, 1000)
        bound = tr.bound_by(TimeRange(100, 500))
        assert bound.span() == 400 * TimeSpan.NANOSECOND

    def test_contains_stamp(self):
        """Should return true if the range contains a timestamp"""
        tr = TimeRange(0, 1000)
        assert tr.contains_stamp(TimeStamp(500))

    def test_doesnt_contain_stamp(self):
        """Should return false if the range doesn't contain a timestamp"""
        tr = TimeRange(0, 1000)
        assert not tr.contains_stamp(TimeStamp(1500))

    def test_stamp_contains_end_of_range(self):
        """Should return false if the timestamp is the same as the end of the range"""
        tr = TimeRange(0, 1000)
        assert not tr.contains_stamp(TimeStamp(1000))

    def test_stamp_contains_start_of_range(self):
        """Should return true if the timestamp is the same as the start of the range"""
        tr = TimeRange(0, 1000)
        assert tr.contains_stamp(TimeStamp(0))

    def test_range_not_contains_range(self):
        """Should return true if the ranges overlap but a smaller range is not contained"""
        tr = TimeRange(0, 1000)
        tr2 = TimeRange(500, 1500)
        assert not tr.contains_range(tr2)

    def test_range_contains_range(self):
        """Should return true if the ranges overlap and the smaller range is contained"""
        tr = TimeRange(0, 1000)
        tr2 = TimeRange(500, 900)
        assert tr.contains_range(tr2)

    def test_range_contains_equal(self):
        """Should return true if the ranges are equal"""
        tr = TimeRange(0, 1000)
        tr2 = TimeRange(0, 1000)
        assert tr.contains_range(tr2)

    def test_range_overlaps(self):
        """Should return true if the ranges overlap"""
        tr = TimeRange(0, 1000)
        tr2 = TimeRange(500, 900)
        assert tr.overlaps_with(tr2)

    def test_range_overlaps_equal(self):
        """Should return true if the ranges are equal"""
        tr = TimeRange(0, 1000)
        tr2 = TimeRange(0, 1000)
        assert tr.overlaps_with(tr2)

    def test_range_overlaps_false(self):
        """Should return false if the ranges don't overlap"""
        tr = TimeRange(0, 1000)
        tr2 = TimeRange(1500, 2000)
        assert not tr.overlaps_with(tr2)

    def test_range_valid(self):
        """Should return true if the range is valid"""
        tr = TimeRange(0, 1000)
        assert tr.is_valid()

    def test_range_invalid(self):
        """Should return false if the range is invalid"""
        tr = TimeRange(1000, 0)
        assert not tr.is_valid()

    def test_range_swap(self):
        """Should swap the start and end times"""
        tr = TimeRange(1000, 0)
        tr = tr.swap()
        assert tr.start == TimeStamp(0)
        assert tr.end == TimeStamp(1000)


class TestTimeSpan:
    @pytest.mark.parametrize(
        "unparsed, expected",
        [
            (1000, TimeSpan.MICROSECOND),
            (timedelta(microseconds=1000), 1000 * TimeSpan.MICROSECOND),
            (TimeStamp(1000), TimeSpan.MICROSECOND),
            (np.timedelta64(1000, "us"), 1000 * TimeSpan.MICROSECOND),
            (pd.Timedelta(1000, "us"), 1000 * TimeSpan.MICROSECOND),
        ],
    )
    def test_init(self, unparsed: UnparsedTimeSpan, expected: TimeSpan):
        assert TimeSpan(unparsed) == expected

    def test_seconds(self):
        """Should return the number of seconds in the timespan"""
        assert TimeSpan.SECOND.seconds() == 1

    def test_is_zero(self):
        """Should return true if the span is zero"""
        assert TimeSpan(0).is_zero()

    def test_delta(self):
        """Should return a timedelta"""
        assert TimeSpan.SECOND.delta() == timedelta(seconds=1)

    def test_add(self):
        """Should correctly add two time spans"""
        assert TimeSpan.MICROSECOND + TimeSpan.MICROSECOND == TimeSpan(2000)

    def test_sub(self):
        """Should correctly subtract two time spans"""
        assert TimeSpan.MICROSECOND - TimeSpan.MICROSECOND == TimeSpan(0)

    def test_gt(self):
        """Should correctly compare two time spans"""
        assert TimeSpan.MICROSECOND > TimeSpan.NANOSECOND

    def test_lt(self):
        """Should correctly compare two time spans"""
        assert TimeSpan.NANOSECOND < TimeSpan.MICROSECOND

    def test_le(self):
        """Should correctly compare two time spans"""
        assert TimeSpan.NANOSECOND <= TimeSpan.MICROSECOND


class TestRate:
    @pytest.mark.parametrize(
        "unparsed, expected",
        [
            (1000, Rate(1000.0)),
            (TimeSpan.SECOND, Rate(1.0)),
        ],
    )
    def test_init(self, unparsed: UnparsedRate, expected: Rate):
        assert Rate(unparsed) == expected

    def test_invalid_init(self):
        """Should raise an exception if the rate is invalid"""
        with pytest.raises(TypeError):
            Rate(timedelta(seconds=1))  # type: ignore

    def test_sample_count(self):
        """Should return the number of samples"""
        assert Rate(1.0).sample_count(5 * TimeSpan.SECOND) == 5

    def test_byte_size(self):
        """Should return the number of bytes in the given span"""
        assert Rate(1.0).byte_size(5 * TimeSpan.SECOND, Density.BIT64) == 40

    def test_byte_span(self):
        """Should return the time span from a byte size"""
        assert Rate(1.0).size_span(Size(40), Density.BIT64) == 5 * TimeSpan.SECOND

    def test_byte_span_invalid(self):
        """Should raise a contiguity error if the size is not a multiple of the density"""
        with pytest.raises(ContiguityError):
            Rate(1.0).size_span(Size(41), Density.BIT64)
