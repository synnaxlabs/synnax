#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import uuid
from datetime import UTC, datetime, timedelta, timezone

import dateutil
import numpy as np
import pandas as pd
import pytest

from synnax import (
    Alignment,
    ContiguityError,
    CrudeDataType,
    CrudeRate,
    CrudeTimeSpan,
    CrudeTimeStamp,
    DataType,
    Density,
    Rate,
    Size,
    TimeRange,
    TimeSpan,
    TimeSpanUnits,
    TimeStamp,
    convert_time_units,
)

_now = TimeStamp.now()

EST = dateutil.tz.gettz("EST")


@pytest.mark.telem
class TestTimeStamp:
    def test_now(self):
        """Should return the current timestamp"""
        now = TimeStamp.now() + TimeSpan.SECOND
        assert now.datetime() > datetime.now().astimezone()

    @pytest.mark.parametrize(
        "crude, expected",
        [
            (1000, 1000),
            (TimeSpan.MILLISECOND * 2500, 2500000000),
            (105 * TimeSpan.MILLISECOND, 105 * TimeSpan.MILLISECOND),
            (
                datetime.fromtimestamp(105, UTC),
                TimeStamp(105 * TimeSpan.SECOND),
            ),
            (_now, _now),
            (timedelta(seconds=105), TimeStamp(105 * TimeSpan.SECOND)),
            (np.datetime64(1000, "ms"), TimeStamp(1000 * TimeSpan.MILLISECOND)),
            (
                datetime(2022, 2, 22, 15, 41, 50, tzinfo=EST),
                TimeStamp(1645562510000000000),
            ),
            (
                datetime(2022, 2, 22, 15, 41, 50, tzinfo=timezone.utc),
                TimeStamp(1645544510000000000),
            ),
            (
                datetime(2022, 2, 22, 10, 41, 50, tzinfo=EST),
                TimeStamp(1645544510000000000),
            ),
            (
                pd.Timestamp(datetime(2022, 2, 22, 15, 41, 50, tzinfo=EST)),
                TimeStamp(1645562510000000000),
            ),
            (np.int64(1000), TimeStamp(1 * TimeSpan.MICROSECOND)),
        ],
    )
    def test_construction(self, crude: CrudeTimeStamp, expected: TimeStamp):
        """Should initialize a timestamp from a variety of types"""
        delta = TimeSpan(TimeStamp(crude) - TimeStamp(expected))
        assert (
            TimeStamp(crude) == expected
        ), f"""
        Expected: {TimeStamp(expected)}
        Got: {TimeStamp(crude)}
        Delta: {delta}
        """

    def test_after_false(self):
        """Should return true if the timestamp is after the given timestamp"""
        ts = TimeStamp(1000)
        assert not ts > TimeSpan.MICROSECOND

    def test_after_true(self):
        """Should return true if the timestamp is after the given timestamp"""
        ts = TimeStamp(10000)
        assert ts > TimeSpan.MICROSECOND

    def test_after_eq_after(self):
        """Should return true if the timestamp is after or equal to the given
        timestamp"""
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
        """Should return a rng of timestamps between two timestamps"""
        ts1 = TimeStamp(1000)
        ts2 = TimeSpan(2000)
        rng = ts1.span_range(ts2)
        assert rng.span == 2 * TimeSpan.MICROSECOND

    def test_range(self):
        """Should return a rng of timestamps between two timestamps"""
        ts1 = TimeStamp(1000)
        ts2 = TimeStamp(2000)
        rng = ts1.range(ts2)
        assert rng.span == TimeSpan.MICROSECOND

    def test_datetime(self):
        """Should correctly convert the TimeStamp to a datetime in local time."""
        ts1 = TimeStamp(1645562510000000000)
        assert ts1.datetime(tz=timezone.utc) == datetime(
            2022, 2, 22, 20, 41, 50, tzinfo=timezone.utc
        )

    def test_trunc(self):
        """Should correctly return the truncation of a standard TimeSpan divisor"""
        ts1 = TimeStamp(1 * TimeSpan.DAY + 1 * TimeSpan.HOUR)
        assert ts1.trunc(TimeSpan.DAY) == (1 * TimeSpan.DAY)


@pytest.mark.telem
class TestTimeRange:
    def test_construction_from_datetime(self):
        """Should initialize a TimeRange from a datetime"""
        dt = datetime(2020, 1, 1, 0, 0, 0).astimezone()
        dt2 = datetime(2021, 1, 1, 0, 0, 0).astimezone()
        tr = TimeRange(dt, dt2)
        assert tr.start.datetime() == dt
        assert tr.end.datetime() == dt2

    def test_span(self):
        """Should return a valid TimeSpan"""
        tr = TimeRange(0, 1000)
        assert tr.span == TimeSpan(1000)

    def test_bound_by(self):
        """Should return a bound version of the range"""
        tr = TimeRange(0, 1000)
        bound = tr.clamp(TimeRange(100, 500))
        assert bound.span == 400 * TimeSpan.NANOSECOND

    def test_contains_stamp(self):
        """Should return true if the range contains a timestamp"""
        tr = TimeRange(0, 1000)
        assert tr.contains(TimeStamp(500))

    def test_doesnt_contain_stamp(self):
        """Should return false if the range doesn't contain a timestamp"""
        tr = TimeRange(0, 1000)
        assert not tr.contains(TimeStamp(1500))

    def test_stamp_contains_end_of_range(self):
        """Should return false if the timestamp is the same as the end of the range"""
        tr = TimeRange(0, 1000)
        assert not tr.contains(TimeStamp(1000))

    def test_stamp_contains_start_of_range(self):
        """Should return true if the timestamp is the same as the start of the range"""
        tr = TimeRange(0, 1000)
        assert tr.contains(TimeStamp(0))

    def test_range_not_contains_range(self):
        """Should return true if the ranges overlap but a smaller range is not contained"""
        tr = TimeRange(0, 1000)
        tr2 = TimeRange(500, 1500)
        assert not tr.contains(tr2)

    def test_range_contains_range(self):
        """Should return true if the ranges overlap and the smaller range is contained"""
        tr = TimeRange(0, 1000)
        tr2 = TimeRange(500, 900)
        assert tr.contains(tr2)

    def test_range_contains_equal(self):
        """Should return true if the ranges are equal"""
        tr = TimeRange(0, 1000)
        tr2 = TimeRange(0, 1000)
        assert tr.contains(tr2)

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
        assert tr.valid

    def test_range_invalid(self):
        """Should return false if the range is invalid"""
        tr = TimeRange(1000, 0)
        assert not tr.valid

    def test_range_swap(self):
        """Should swap the start and end times"""
        tr = TimeRange(1000, 0)
        tr = tr.swap()
        assert tr.start == TimeStamp(0)
        assert tr.end == TimeStamp(1000)


@pytest.mark.telem
class TestTimeSpan:
    def test_since(self):
        """Should return the TimeSpan since the given timestamp"""
        now = TimeStamp.now()
        one_sec_ago = now - 1 * TimeSpan.SECOND
        assert TimeSpan.since(one_sec_ago) < 1002 * TimeSpan.MILLISECOND
        assert TimeSpan.since(one_sec_ago) > 998 * TimeSpan.MILLISECOND

    @pytest.mark.parametrize(
        "crude, expected",
        [
            (1000, TimeSpan.MICROSECOND),
            (timedelta(microseconds=1000), 1000 * TimeSpan.MICROSECOND),
            (TimeStamp(1000), TimeSpan.MICROSECOND),
            (np.timedelta64(1000, "us"), 1000 * TimeSpan.MICROSECOND),
            (pd.Timedelta(1000, "ms"), 1000 * TimeSpan.MILLISECOND),
            (TimeSpan.MICROSECOND * 1000, TimeSpan.MICROSECOND * 1000),
            (np.int64(1000), 1 * TimeSpan.MICROSECOND),
        ],
    )
    def test_construction(self, crude: CrudeTimeSpan, expected: TimeSpan):
        assert TimeSpan(crude) == expected

    def test_seconds(self):
        """Should return the number of seconds in the timespan"""
        assert TimeSpan.SECOND.seconds == 1

    def test_timedelta(self):
        """Should return a timedelta"""
        assert TimeSpan.SECOND.timedelta == timedelta(seconds=1)

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

    @pytest.mark.parametrize(
        "span, expected",
        [
            (
                1 * TimeSpan.DAY + 10 * TimeSpan.MINUTE + 100 * TimeSpan.MILLISECOND,
                "1d 10m 100ms",
            ),
            (10 * TimeSpan.HOUR + 10 * TimeSpan.NANOSECOND, "10h 10ns"),
            (TimeSpan.ZERO, "0ns"),
        ],
    )
    def test_str(self, span, expected):
        """Should correctly display the TimeSpan as a human-readable string"""
        assert str(span) == expected

    @pytest.mark.parametrize(
        "span, expected",
        [
            (
                1.0,
                1 * TimeSpan.SECOND,
            ),
            (
                1,
                1 * TimeSpan.SECOND,
            ),
            (
                1 * TimeSpan.SECOND,
                1 * TimeSpan.SECOND,
            ),
        ],
    )
    def test_from_seconds(self, span, expected):
        """It should evaluate pure floats or integers as seconds"""
        abc = TimeSpan.from_seconds(1.0)
        assert abc == TimeSpan(1 * TimeSpan.SECOND)

    @pytest.mark.parametrize(
        "span, expected",
        [
            (1.0, 1.0),
            (1, 1),
            (1 * TimeSpan.MILLISECOND, 0.001),
        ],
    )
    def test_to_seconds(self, span, expected):
        """It should evaluate pure floats or integers as seconds"""
        abc = TimeSpan.to_seconds(span)
        assert abc == expected


@pytest.mark.telem
class TestRate:
    @pytest.mark.parametrize(
        "crude, expected",
        [
            (TimeSpan.MILLISECOND, Rate(1000)),
            (1000, Rate(1000.0)),
        ],
    )
    def test_construction(self, crude: CrudeRate, expected: Rate):
        assert Rate(crude) == expected

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


@pytest.mark.telem
class TestDataType:
    @pytest.mark.parametrize(
        "crude, expected",
        [
            (np.int8, DataType.INT8),
            (np.int16, DataType.INT16),
            ("int32", DataType.INT32),
            ("int64", DataType.INT64),
            (np.float32(1), DataType.FLOAT32),
            ({"a": 1}, DataType.JSON),
            (["a"], DataType.STRING),
            (uuid.uuid4(), DataType.UUID),
        ],
    )
    def test_construction(self, crude: CrudeDataType, expected: DataType):
        assert DataType(crude) == expected

    def test_string(self):
        """Should return the string representation of the data type"""
        assert str(DataType.INT8) == "int8"

    @pytest.mark.parametrize(
        "value, expected",
        [(DataType.INT8, np.dtype(np.int8)), (DataType.FLOAT32, np.dtype(np.float32))],
    )
    def test_np(self, value, expected):
        """Should return the correct numpy representation of the data type"""
        assert value.np == expected


@pytest.mark.telem
class TestSize:
    @pytest.mark.parametrize(
        "crude, expected", [(1, Size.BYTE), (1.0, Size.BYTE), (Size.BYTE, Size.BYTE)]
    )
    def test_construction(self, crude, expected):
        assert Size(crude) == expected

    @pytest.mark.parametrize(
        "value, expected", [(Size.GB + Size.MB * 500, "1gb 500mb"), (Size.GB * 0, "0b")]
    )
    def test_str(self, value, expected):
        assert str(value) == expected


@pytest.mark.telem
@pytest.mark.parametrize(
    "data, from_, to, expected",
    [
        (np.array([1, 2, 3]), "s", "ms", 1000),
        (np.array([1, 2, 3]), "ms", "ms", 1),
        (np.array([TimeStamp(0).datetime().isoformat()]), "iso", "ns", 0),
    ],
)
def test_convert_time_units(
    data: np.ndarray,
    from_: TimeSpanUnits,
    to: TimeSpanUnits,
    expected: int | float,
):
    assert convert_time_units(data, from_, to)[0] == expected


@pytest.mark.telem
class TestAlignment:
    def test_construction(self):
        """Should construct the alignment from the given domain and sample indexes"""
        align = Alignment(2, 1)
        assert align.sample_index == 1
        assert align.domain_index == 2

    def test_construction_zero(self):
        """Should construct a zero alignment"""
        align = Alignment(0, 0)
        assert int(align) == 0

    def test_default_construction(self):
        """Should construct a zero alignment by default"""
        align = Alignment()
        assert align.domain_index == 0
        assert align.sample_index == 0

    def test_construction_from_packed_int(self):
        """Should construct alignment from a packed integer value"""
        # Create an alignment and get its packed value
        align1 = Alignment(5, 10)
        packed = int(align1)
        # Reconstruct from the packed value
        align2 = Alignment(packed)
        assert align2.domain_index == 5
        assert align2.sample_index == 10

    def test_construction_from_tuple(self):
        """Should construct alignment from a tuple"""
        align = Alignment((3, 7))
        assert align.domain_index == 3
        assert align.sample_index == 7

    def test_construction_from_alignment(self):
        """Should return the same alignment when constructing from Alignment"""
        align1 = Alignment(4, 8)
        align2 = Alignment(align1)
        assert align1 is align2  # Should be the same object
        assert align2.domain_index == 4
        assert align2.sample_index == 8

    def test_domain_index_extraction(self):
        """Should correctly extract the domain index from the packed value"""
        align = Alignment(5, 10)
        assert align.domain_index == 5

    def test_sample_index_extraction(self):
        """Should correctly extract the sample index from the packed value"""
        align = Alignment(5, 10)
        assert align.sample_index == 10

    def test_add_samples(self):
        """Should add to the alignment sample index"""
        align = Alignment(2, 1)
        align = align.add_samples(3)
        assert align.sample_index == 4
        assert align.domain_index == 2

    def test_add_samples_overflow(self):
        """Should handle sample index overflow correctly"""
        align = Alignment(2, 0xFFFFFFFF - 1)
        align = align.add_samples(1)
        assert align.sample_index == 0xFFFFFFFF
        assert align.domain_index == 2

    def test_add(self):
        """Should add both domain and sample indices"""
        align1 = Alignment(2, 5)
        align2 = Alignment(3, 10)
        result = align1.add(align2)
        assert result.domain_index == 5
        assert result.sample_index == 15

    def test_str(self):
        """Should return the string representation of the alignment"""
        align = Alignment(5, 7)
        assert str(align) == "5-7"

    def test_repr(self):
        """Should return the repr representation of the alignment"""
        align = Alignment(5, 7)
        assert repr(align) == "Alignment(5, 7)"

    def test_comparison(self):
        """Should correctly compare alignments"""
        align1 = Alignment(2, 5)
        align2 = Alignment(2, 10)
        align3 = Alignment(3, 5)

        # Same domain, different sample
        assert align1 < align2
        assert align2 > align1

        # Different domain
        assert align2 < align3
        assert align3 > align2

    def test_equality(self):
        """Should correctly compare equality of alignments"""
        align1 = Alignment(2, 5)
        align2 = Alignment(2, 5)
        align3 = Alignment(3, 5)

        assert align1 == align2
        assert align1 != align3

    def test_max_values(self):
        """Should handle maximum values for domain and sample indices"""
        max_uint32 = 0xFFFFFFFF
        align = Alignment(max_uint32, max_uint32)
        assert align.domain_index == max_uint32
        assert align.sample_index == max_uint32

    def test_large_domain_index(self):
        """Should correctly handle large domain indices"""
        align = Alignment(1000000, 50)
        assert align.domain_index == 1000000
        assert align.sample_index == 50

    def test_int_conversion(self):
        """Should correctly convert to int"""
        align = Alignment(2, 1)
        int_value = int(align)
        # 2 << 32 | 1 = 8589934593
        expected = (2 << 32) | 1
        assert int_value == expected

    def test_pydantic_validation(self):
        """Should work with pydantic validation"""
        from pydantic import BaseModel

        class TestModel(BaseModel):
            alignment: Alignment

        model = TestModel(alignment=Alignment(5, 10))
        assert model.alignment.domain_index == 5
        assert model.alignment.sample_index == 10

        # Should also accept int
        packed_value = (5 << 32) | 10
        model2 = TestModel(alignment=packed_value)
        assert model2.alignment.domain_index == 5
        assert model2.alignment.sample_index == 10
