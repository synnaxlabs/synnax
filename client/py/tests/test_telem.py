import time
from datetime import datetime, timedelta

from delta import telem


class TestTimeStamp:
    def test_now(self):
        """
        Should return the current timestamp
        """
        now = telem.now() + telem.SECOND
        assert now.time() > datetime.now()

    def test_init_from_datetime(self):
        """
        Should initialize a TimeStamp from a datetime
        """
        dt = datetime(2020, 1, 1, 0, 0, 0)
        ts = telem.TimeStamp(dt)
        assert ts.time() == dt

    def test_init_from_timestamp(self):
        """
        Should initialize a TimeStamp from a timestamp
        """
        now = telem.now()
        ts = telem.TimeStamp(now)
        assert ts == now

    def test_init_from_nanoseconds(self):
        """
        Should initialize a TimeStamp from a nanoseconds value
        """
        ts = telem.TimeStamp(1000)
        assert ts == telem.MICROSECOND

    def test_init_from_timespan(self):
        """
        Should initialize a TimeStamp from a timespan
        """
        ts = telem.TimeStamp(telem.SECOND)
        assert ts == telem.SECOND

    def test_is_zero(self):
        """
        Should return true if the timestamp is zero
        """
        ts = telem.TimeStamp(0)
        assert ts.is_zero()

    def test_after_false(self):
        """
        Should return true if the timestamp is after the given timestamp
        """
        ts = telem.TimeStamp(1000)
        assert not ts.after(telem.MICROSECOND)

    def test_after_true(self):
        """
        Should return true if the timestamp is after the given timestamp
        """
        ts = telem.TimeStamp(10000)
        assert ts.after(telem.MICROSECOND)

    def test_after_eq_after(self):
        """
        Should return true if the timestamp is after or equal to the given timestamp
        """
        ts = telem.TimeStamp(1000)
        assert ts.after_eq(telem.MICROSECOND)

    def test_after_eq_before(self):
        """
        Should return true if the timestamp is after or equal to the given timestamp
        """
        ts = telem.TimeStamp(100)
        assert not ts.after_eq(telem.MICROSECOND)

    def test_before_false(self):
        """
        Should return true if the timestamp is before the given timestamp
        """
        ts = telem.TimeStamp(1000)
        assert not ts.before(telem.MICROSECOND)

    def test_before_true(self):
        """
        Should return true if the timestamp is before the given timestamp
        """
        ts = telem.TimeStamp(100)
        assert ts.before(telem.MICROSECOND)

    def test_before_eq_before(self):
        """
        Should return true if the timestamp is before or equal to the given timestamp
        """
        ts = telem.TimeStamp(100)
        assert ts.before_eq(telem.MICROSECOND)

    def test_before_eq_after(self):
        """
        Should return true if the timestamp is before or equal to the given timestamp
        """
        ts = telem.TimeStamp(1000)
        assert ts.before_eq(telem.MICROSECOND)

    def test_add(self):
        """
        Should add a timespan to a timestamp
        """
        ts = telem.TimeStamp(1000)
        ts += telem.MICROSECOND
        assert ts == telem.TimeStamp(2000)

    def test_sub(self):
        """
        Should subtract a timespan from a timestamp
        """
        ts = telem.TimeStamp(2000)
        ts = ts - telem.MICROSECOND
        assert ts == telem.TimeStamp(1000)

    def test_span_range(self):
        """
        Should return a range of timestamps between two timestamps
        """
        ts1 = telem.TimeStamp(1000)
        ts2 = telem.TimeSpan(2000)
        range = ts1.span_range(ts2)
        assert range.span() == 2 * telem.MICROSECOND

    def test_range(self):
        """
        Should return a range of timestamps between two timestamps
        """
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
    def test_initialize_from_int(self):
        """
        Should initialize a TimeSpan from a timestamp
        """
        ts = telem.TimeSpan(1000)
        assert ts == telem.MICROSECOND

    def test_initialize_from_timedelta(self):
        """
        Should initialize a TimeSpan from a timedelta
        """
        td = timedelta(microseconds=1000)
        ts = telem.TimeSpan(td)
        assert ts == 1000 * telem.MICROSECOND

    def test_initialize_from_timespan(self):
        """
        Should initialize a TimeSpan from a timespan
        """
        ts = 1000 * telem.MICROSECOND
        ts2 = telem.TimeSpan(ts)
        assert ts2 == 1000 * telem.MICROSECOND

    def test_initialize_from_timestamp(self):
        """
        Should initialize a TimeSpan from a timestamp
        """
        ts = telem.TimeStamp(1000)
        ts2 = telem.TimeSpan(ts)
        assert ts2 == telem.MICROSECOND

    def test_seconds(self):
        """
        Should return the number of seconds in the timespan
        """
        ts = telem.SECOND
        assert ts.seconds() == 1

    def test_is_zero(self):
        """
        Should return true if the span is zero
        """
        ts = telem.TimeSpan(0)
        assert ts.is_zero()

    def test_is_not_zero(self):
        """
        Should return false if the span is not zero
        """
        ts = telem.TimeSpan(1)
        assert not ts.is_zero()

    # def test_size_span(self):
    #     """
    #     Should calculate a timespan from a Size and Density
    #     """
    #     # assert span == telem.TimeSpan(2 * telem.SECOND)
