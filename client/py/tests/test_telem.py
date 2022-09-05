from datetime import datetime, timedelta

import numpy as np
import pandas as pd
import pytest

from arya import telem, errors

_now = telem.now()


class TestTimeStamp:
    def test_now(self):
        """Should return the current timestamp"""
        now = telem.now() + telem.SECOND
        assert now.time() > datetime.now()

    @pytest.mark.parametrize(
        "unparsed, expected",
        [
            (1000, 1000),
            (105 * telem.MILLISECOND, 105 * telem.MILLISECOND),
            (datetime.utcfromtimestamp(105), telem.TimeStamp(105 * telem.SECOND)),
            (pd.Timestamp(105), telem.TimeStamp(105 * telem.NANOSECOND)),
            (_now, _now),
            (timedelta(seconds=105), telem.TimeStamp(105 * telem.SECOND)),
            (np.datetime64(1000, "ms"), telem.TimeStamp(1000 * telem.MILLISECOND)),
        ],
    )
    def test_init(self, unparsed: telem.UnparsedTimeStamp, expected: telem.TimeStamp):
        assert telem.TimeStamp(unparsed) == expected

    def test_invalid_init(self):
        """Should raise an exception if the timestamp is invalid"""
        with pytest.raises(TypeError):
            telem.TimeStamp(1.25)

    def test_is_zero(self):
        """Should return true if the timestamp is zero"""
        ts = telem.TimeStamp(0)
        assert ts.is_zero()

    def test_after_false(self):
        """Should return true if the timestamp is after the given timestamp"""
        ts = telem.TimeStamp(1000)
        assert not ts > telem.MICROSECOND

    def test_after_true(self):
        """Should return true if the timestamp is after the given timestamp"""
        ts = telem.TimeStamp(10000)
        assert ts > telem.MICROSECOND

    def test_after_eq_after(self):
        """Should return true if the timestamp is after or equal to the given timestamp"""
        ts = telem.TimeStamp(1000)
        assert ts >= telem.MICROSECOND

    def test_after_eq_before(self):
        """Should return true if the timestamp is after or equal to the given timestamp"""
        ts = telem.TimeStamp(100)
        assert not ts >= telem.MICROSECOND

    def test_before_false(self):
        """Should return true if the timestamp is before the given timestamp"""
        ts = telem.TimeStamp(1000)
        assert not ts < telem.MICROSECOND

    def test_before_true(self):
        """Should return true if the timestamp is before the given timestamp"""
        ts = telem.TimeStamp(100)
        assert ts < telem.MICROSECOND

    def test_before_eq_before(self):
        """Should return true if the timestamp is before or equal to the given timestamp"""
        ts = telem.TimeStamp(100)
        assert ts <= telem.MICROSECOND

    def test_before_eq_after(self):
        """Should return true if the timestamp is before or equal to the given timestamp"""
        ts = telem.TimeStamp(1000)
        assert ts <= telem.MICROSECOND

    def test_add(self):
        """Should add a timespan to a timestamp"""
        ts = telem.TimeStamp(1000)
        ts += telem.MICROSECOND
        assert ts == telem.TimeStamp(2000)

    def test_sub(self):
        """Should subtract a timespan from a timestamp"""
        ts = telem.TimeStamp(2000)
        ts = ts - telem.MICROSECOND
        assert ts == telem.TimeStamp(1000)

    def test_span_range(self):
        """Should return a range of timestamps between two timestamps"""
        ts1 = telem.TimeStamp(1000)
        ts2 = telem.TimeSpan(2000)
        range = ts1.span_range(ts2)
        assert range.span() == 2 * telem.MICROSECOND

    def test_range(self):
        """Should return a range of timestamps between two timestamps"""
        ts1 = telem.TimeStamp(1000)
        ts2 = telem.TimeStamp(2000)
        range = ts1.range(ts2)
        assert range.span() == telem.MICROSECOND


class TestTimeRange:
    def test_init_from_datetime(self):
        """
        Should initialize a TimeRange from a datetime
        """
        dt = datetime(2020, 1, 1, 0, 0, 0)
        dt2 = datetime(2021, 1, 1, 0, 0, 0)
        tr = telem.TimeRange(dt, dt2)
        assert tr.start.time() == dt
        assert tr.end.time() == dt2

    def test_span(self):
        """
        Should return a valid TimeSpan
        """
        tr = telem.TimeRange(0, 1000)
        assert tr.span() == telem.TimeSpan(1000)

    def test_is_zero(self):
        """
        Should return true if the range is zero
        """
        tr = telem.TimeRange(0, 0)
        assert tr.is_zero()

    def test_bound_by(self):
        """
        Should return a bound version of the range
        """
        tr = telem.TimeRange(0, 1000)
        bound = tr.bound_by(telem.TimeRange(100, 500))
        assert bound.span() == 400 * telem.NANOSECOND

    def test_contains_stamp(self):
        """
        Should return true if the range contains a timestamp
        """
        tr = telem.TimeRange(0, 1000)
        assert tr.contains_stamp(telem.TimeStamp(500))

    def test_doesnt_contain_stamp(self):
        """
        Should return false if the range doesn't contain a timestamp
        """
        tr = telem.TimeRange(0, 1000)
        assert not tr.contains_stamp(telem.TimeStamp(1500))

    def test_stamp_contains_end_of_range(self):
        """
        Should return false if the timestamp is the same as the end of the range
        """
        tr = telem.TimeRange(0, 1000)
        assert not tr.contains_stamp(telem.TimeStamp(1000))

    def test_stamp_contains_start_of_range(self):
        """
        Should return true if the timestamp is the same as the start of the range
        """
        tr = telem.TimeRange(0, 1000)
        assert tr.contains_stamp(telem.TimeStamp(0))

    def test_range_not_contains_range(self):
        """
        Should return true if the ranges overlap but a smaller range is not contained
        """
        tr = telem.TimeRange(0, 1000)
        tr2 = telem.TimeRange(500, 1500)
        assert not tr.contains_range(tr2)

    def test_range_contains_range(self):
        """
        Should return true if the ranges overlap and the smaller range is contained
        """
        tr = telem.TimeRange(0, 1000)
        tr2 = telem.TimeRange(500, 900)
        assert tr.contains_range(tr2)

    def test_range_contains_equal(self):
        """
        Should return true if the ranges are equal
        """
        tr = telem.TimeRange(0, 1000)
        tr2 = telem.TimeRange(0, 1000)
        assert tr.contains_range(tr2)

    def test_range_overlaps(self):
        """
        Should return true if the ranges overlap
        """
        tr = telem.TimeRange(0, 1000)
        tr2 = telem.TimeRange(500, 900)
        assert tr.overlaps_with(tr2)

    def test_range_overlaps_equal(self):
        """
        Should return true if the ranges are equal
        """
        tr = telem.TimeRange(0, 1000)
        tr2 = telem.TimeRange(0, 1000)
        assert tr.overlaps_with(tr2)

    def test_range_overlaps_false(self):
        """
        Should return false if the ranges don't overlap
        """
        tr = telem.TimeRange(0, 1000)
        tr2 = telem.TimeRange(1500, 2000)
        assert not tr.overlaps_with(tr2)

    def test_range_valid(self):
        """
        Should return true if the range is valid
        """
        tr = telem.TimeRange(0, 1000)
        assert tr.is_valid()

    def test_range_invalid(self):
        """
        Should return false if the range is invalid
        """
        tr = telem.TimeRange(1000, 0)
        assert not tr.is_valid()

    def test_range_swap(self):
        """
        Should swap the start and end times
        """
        tr = telem.TimeRange(1000, 0)
        tr = tr.swap()
        assert tr.start == telem.TimeStamp(0)
        assert tr.end == telem.TimeStamp(1000)


class TestTimeSpan:
    @pytest.mark.parametrize(
        "unparsed, expected",
        [
            (1000, telem.MICROSECOND),
            (timedelta(microseconds=1000), 1000 * telem.MICROSECOND),
            (telem.TimeStamp(1000), telem.MICROSECOND),
        ],
    )
    def test_init(self, unparsed: telem.UnparsedTimeSpan, expected: telem.TimeSpan):
        assert telem.TimeSpan(unparsed) == expected

    def test_seconds(self):
        """Should return the number of seconds in the timespan"""
        assert telem.SECOND.seconds() == 1

    def test_is_zero(self):
        """Should return true if the span is zero"""
        assert telem.TimeSpan(0).is_zero()

    def test_delta(self):
        """Should return a timedelta"""
        assert telem.SECOND.delta() == timedelta(seconds=1)

    def test_add(self):
        """Should correctly add two time spans"""
        assert telem.MICROSECOND + telem.MICROSECOND == telem.TimeSpan(2000)

    def test_sub(self):
        """Should correctly subtract two time spans"""
        assert telem.MICROSECOND - telem.MICROSECOND == telem.TimeSpan(0)

    def test_gt(self):
        """Should correctly compare two time spans"""
        assert telem.MICROSECOND > telem.NANOSECOND

    def test_lt(self):
        """Should correctly compare two time spans"""
        assert telem.NANOSECOND < telem.MICROSECOND

    def test_le(self):
        """Should correctly compare two time spans"""
        assert telem.NANOSECOND <= telem.MICROSECOND


class TestRate:
    @pytest.mark.parametrize(
        "unparsed, expected",
        [
            (1000, telem.Rate(1000.0)),
            (telem.SECOND, telem.Rate(1.0)),
        ],
    )
    def test_init(self, unparsed: telem.UnparsedRate, expected: telem.Rate):
        assert telem.Rate(unparsed) == expected

    def test_invalid_init(self):
        """Should raise an exception if the rate is invalid"""
        with pytest.raises(TypeError):
            telem.Rate(timedelta(seconds=1))

    def test_sample_count(self):
        """Should return the number of samples"""
        assert telem.Rate(1.0).sample_count(5 * telem.SECOND) == 5

    def test_byte_size(self):
        """Should return the number of bytes in the given span"""
        assert telem.Rate(1.0).byte_size(5 * telem.SECOND, telem.BIT64) == 40

    def test_byte_span(self):
        """Should return the time span from a byte size"""
        assert (
                telem.Rate(1.0).size_span(telem.Size(40),
                                          telem.BIT64) == 5 * telem.SECOND
        )

    def test_byte_span_invalid(self):
        """Should raise a contiguity error if the size is not a multiple of the density"""
        with pytest.raises(errors.ContiguityError):
            telem.Rate(1.0).size_span(telem.Size(41), telem.BIT64)
