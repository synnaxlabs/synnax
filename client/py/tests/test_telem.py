#  Copyright 2022 Synnax Labs, Inc.
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

import synnax

_now = synnax.now()


class TestTimeStamp:
    def test_now(self):
        """Should return the current timestamp"""
        now = synnax.now() + synnax.SECOND
        assert now.datetime() > datetime.now().astimezone()

    @pytest.mark.parametrize(
        "unparsed, expected",
        [
            (1000, 1000),
            (synnax.MILLISECOND * 2500, 2500000000),
            (105 * synnax.MILLISECOND, 105 * synnax.MILLISECOND),
            (
                datetime.utcfromtimestamp(105).replace(tzinfo=timezone.utc),
                synnax.TimeStamp(105 * synnax.SECOND),
            ),
            (_now, _now),
            (timedelta(seconds=105), synnax.TimeStamp(105 * synnax.SECOND)),
            (np.datetime64(1000, "ms"), synnax.TimeStamp(1000 * synnax.MILLISECOND)),
            (
                datetime(2022, 2, 22, 15, 41, 50, tzinfo=pytz_timezone("EST")),
                synnax.TimeStamp(1645562510000000000),
            ),
            (
                datetime(2022, 2, 22, 15, 41, 50, tzinfo=timezone.utc),
                synnax.TimeStamp(1645544510000000000),
            ),
            (
                datetime(2022, 2, 22, 10, 41, 50, tzinfo=pytz_timezone("EST")),
                synnax.TimeStamp(1645544510000000000),
            ),
            (
                pd.Timestamp(
                    datetime(2022, 2, 22, 15, 41, 50, tzinfo=pytz_timezone("EST"))
                ),
                synnax.TimeStamp(1645562510000000000),
            ),
        ],
    )
    def test_init(self, unparsed: synnax.UnparsedTimeStamp, expected: synnax.TimeStamp):
        """Should initialize a timestamp from a variety of types"""
        assert synnax.TimeStamp(unparsed) == expected

    def test_invalid_init(self):
        """Should raise an exception if the timestamp is invalid"""
        with pytest.raises(TypeError):
            synnax.TimeStamp("dog")

    def test_is_zero(self):
        """Should return true if the timestamp is zero"""
        ts = synnax.TimeStamp(0)
        assert ts.is_zero()

    def test_after_false(self):
        """Should return true if the timestamp is after the given timestamp"""
        ts = synnax.TimeStamp(1000)
        assert not ts > synnax.MICROSECOND

    def test_after_true(self):
        """Should return true if the timestamp is after the given timestamp"""
        ts = synnax.TimeStamp(10000)
        assert ts > synnax.MICROSECOND

    def test_after_eq_after(self):
        """Should return true if the timestamp is after or equal to the given timestamp"""
        ts = synnax.TimeStamp(1000)
        assert ts >= synnax.MICROSECOND

    def test_after_eq_before(self):
        """Should return true if the timestamp is after or equal to the given timestamp"""
        ts = synnax.TimeStamp(100)
        assert not ts >= synnax.MICROSECOND

    def test_before_false(self):
        """Should return true if the timestamp is before the given timestamp"""
        ts = synnax.TimeStamp(1000)
        assert not ts < synnax.MICROSECOND

    def test_before_true(self):
        """Should return true if the timestamp is before the given timestamp"""
        ts = synnax.TimeStamp(100)
        assert ts < synnax.MICROSECOND

    def test_before_eq_before(self):
        """Should return true if the timestamp is before or equal to the given timestamp"""
        ts = synnax.TimeStamp(100)
        assert ts <= synnax.MICROSECOND

    def test_before_eq_after(self):
        """Should return true if the timestamp is before or equal to the given timestamp"""
        ts = synnax.TimeStamp(1000)
        assert ts <= synnax.MICROSECOND

    def test_add(self):
        """Should add a timespan to a timestamp"""
        ts = synnax.TimeStamp(1000)
        ts += synnax.MICROSECOND
        assert ts == synnax.TimeStamp(2000)

    def test_sub(self):
        """Should subtract a timespan from a timestamp"""
        ts = synnax.TimeStamp(2000)
        ts = ts - synnax.MICROSECOND
        assert ts == synnax.TimeStamp(1000)

    def test_span_range(self):
        """Should return a range of timestamps between two timestamps"""
        ts1 = synnax.TimeStamp(1000)
        ts2 = synnax.TimeSpan(2000)
        range = ts1.span_range(ts2)
        assert range.span() == 2 * synnax.MICROSECOND

    def test_range(self):
        """Should return a range of timestamps between two timestamps"""
        ts1 = synnax.TimeStamp(1000)
        ts2 = synnax.TimeStamp(2000)
        range = ts1.range(ts2)
        assert range.span() == synnax.MICROSECOND

    def test_datetime(self):
        """Should correctly convert the TimeStamp to a datetime in local time."""
        ts1 = synnax.TimeStamp(1645562510000000000)
        assert ts1.datetime(tzinfo=timezone.utc) == datetime(
            2022, 2, 22, 20, 41, 50, tzinfo=timezone.utc
        )


class TestTimeRange:
    def test_init_from_datetime(self):
        """Should initialize a TimeRange from a datetime"""
        dt = datetime(2020, 1, 1, 0, 0, 0).astimezone()
        dt2 = datetime(2021, 1, 1, 0, 0, 0).astimezone()
        tr = synnax.TimeRange(dt, dt2)
        assert tr.start.datetime() == dt
        assert tr.end.datetime() == dt2

    def test_span(self):
        """Should return a valid TimeSpan"""
        tr = synnax.TimeRange(0, 1000)
        assert tr.span() == synnax.TimeSpan(1000)

    def test_is_zero(self):
        """Should return true if the range is zero"""
        tr = synnax.TimeRange(0, 0)
        assert tr.is_zero()

    def test_bound_by(self):
        """Should return a bound version of the range"""
        tr = synnax.TimeRange(0, 1000)
        bound = tr.bound_by(synnax.TimeRange(100, 500))
        assert bound.span() == 400 * synnax.NANOSECOND

    def test_contains_stamp(self):
        """Should return true if the range contains a timestamp"""
        tr = synnax.TimeRange(0, 1000)
        assert tr.contains_stamp(synnax.TimeStamp(500))

    def test_doesnt_contain_stamp(self):
        """Should return false if the range doesn't contain a timestamp"""
        tr = synnax.TimeRange(0, 1000)
        assert not tr.contains_stamp(synnax.TimeStamp(1500))

    def test_stamp_contains_end_of_range(self):
        """Should return false if the timestamp is the same as the end of the range"""
        tr = synnax.TimeRange(0, 1000)
        assert not tr.contains_stamp(synnax.TimeStamp(1000))

    def test_stamp_contains_start_of_range(self):
        """Should return true if the timestamp is the same as the start of the range"""
        tr = synnax.TimeRange(0, 1000)
        assert tr.contains_stamp(synnax.TimeStamp(0))

    def test_range_not_contains_range(self):
        """Should return true if the ranges overlap but a smaller range is not contained"""
        tr = synnax.TimeRange(0, 1000)
        tr2 = synnax.TimeRange(500, 1500)
        assert not tr.contains_range(tr2)

    def test_range_contains_range(self):
        """Should return true if the ranges overlap and the smaller range is contained"""
        tr = synnax.TimeRange(0, 1000)
        tr2 = synnax.TimeRange(500, 900)
        assert tr.contains_range(tr2)

    def test_range_contains_equal(self):
        """Should return true if the ranges are equal"""
        tr = synnax.TimeRange(0, 1000)
        tr2 = synnax.TimeRange(0, 1000)
        assert tr.contains_range(tr2)

    def test_range_overlaps(self):
        """Should return true if the ranges overlap"""
        tr = synnax.TimeRange(0, 1000)
        tr2 = synnax.TimeRange(500, 900)
        assert tr.overlaps_with(tr2)

    def test_range_overlaps_equal(self):
        """Should return true if the ranges are equal"""
        tr = synnax.TimeRange(0, 1000)
        tr2 = synnax.TimeRange(0, 1000)
        assert tr.overlaps_with(tr2)

    def test_range_overlaps_false(self):
        """Should return false if the ranges don't overlap"""
        tr = synnax.TimeRange(0, 1000)
        tr2 = synnax.TimeRange(1500, 2000)
        assert not tr.overlaps_with(tr2)

    def test_range_valid(self):
        """Should return true if the range is valid"""
        tr = synnax.TimeRange(0, 1000)
        assert tr.is_valid()

    def test_range_invalid(self):
        """Should return false if the range is invalid"""
        tr = synnax.TimeRange(1000, 0)
        assert not tr.is_valid()

    def test_range_swap(self):
        """Should swap the start and end times"""
        tr = synnax.TimeRange(1000, 0)
        tr = tr.swap()
        assert tr.start == synnax.TimeStamp(0)
        assert tr.end == synnax.TimeStamp(1000)


class TestTimeSpan:
    @pytest.mark.parametrize(
        "unparsed, expected",
        [
            (1000, synnax.MICROSECOND),
            (timedelta(microseconds=1000), 1000 * synnax.MICROSECOND),
            (synnax.TimeStamp(1000), synnax.MICROSECOND),
            (np.timedelta64(1000, "us"), 1000 * synnax.MICROSECOND),
            (pd.Timedelta(1000, "us"), 1000 * synnax.MICROSECOND),
        ],
    )
    def test_init(self, unparsed: synnax.UnparsedTimeSpan, expected: synnax.TimeSpan):
        assert synnax.TimeSpan(unparsed) == expected

    def test_seconds(self):
        """Should return the number of seconds in the timespan"""
        assert synnax.SECOND.seconds() == 1

    def test_is_zero(self):
        """Should return true if the span is zero"""
        assert synnax.TimeSpan(0).is_zero()

    def test_delta(self):
        """Should return a timedelta"""
        assert synnax.SECOND.delta() == timedelta(seconds=1)

    def test_add(self):
        """Should correctly add two time spans"""
        assert synnax.MICROSECOND + synnax.MICROSECOND == synnax.TimeSpan(2000)

    def test_sub(self):
        """Should correctly subtract two time spans"""
        assert synnax.MICROSECOND - synnax.MICROSECOND == synnax.TimeSpan(0)

    def test_gt(self):
        """Should correctly compare two time spans"""
        assert synnax.MICROSECOND > synnax.NANOSECOND

    def test_lt(self):
        """Should correctly compare two time spans"""
        assert synnax.NANOSECOND < synnax.MICROSECOND

    def test_le(self):
        """Should correctly compare two time spans"""
        assert synnax.NANOSECOND <= synnax.MICROSECOND


class TestRate:
    @pytest.mark.parametrize(
        "unparsed, expected",
        [
            (1000, synnax.Rate(1000.0)),
            (synnax.SECOND, synnax.Rate(1.0)),
        ],
    )
    def test_init(self, unparsed: synnax.UnparsedRate, expected: synnax.Rate):
        assert synnax.Rate(unparsed) == expected

    def test_invalid_init(self):
        """Should raise an exception if the rate is invalid"""
        with pytest.raises(TypeError):
            synnax.Rate(timedelta(seconds=1))

    def test_sample_count(self):
        """Should return the number of samples"""
        assert synnax.Rate(1.0).sample_count(5 * synnax.SECOND) == 5

    def test_byte_size(self):
        """Should return the number of bytes in the given span"""
        assert synnax.Rate(1.0).byte_size(5 * synnax.SECOND, synnax.BIT64) == 40

    def test_byte_span(self):
        """Should return the time span from a byte size"""
        assert (
            synnax.Rate(1.0).size_span(synnax.Size(40), synnax.BIT64)
            == 5 * synnax.SECOND
        )

    def test_byte_span_invalid(self):
        """Should raise a contiguity error if the size is not a multiple of the density"""
        with pytest.raises(synnax.ContiguityError):
            synnax.Rate(1.0).size_span(synnax.Size(41), synnax.BIT64)
