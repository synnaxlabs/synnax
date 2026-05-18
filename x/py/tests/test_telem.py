#  Copyright 2026 Synnax Labs, Inc.
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

import x.telem as sy
from x.exceptions import ContiguityError

_now = sy.TimeStamp.now()

EST = dateutil.tz.gettz("EST")


@pytest.mark.telem
class TestTimeStamp:
    def test_now(self) -> None:
        """Should return the current timestamp"""
        now = sy.TimeStamp.now() + sy.TimeSpan.SECOND
        assert now.datetime() > datetime.now().astimezone()

    @pytest.mark.parametrize(
        "crude, expected",
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
                datetime(2022, 2, 22, 15, 41, 50, tzinfo=timezone.utc),
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
        """Should return true if the timestamp is after or equal to the given timestamp"""
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
        """Should return true if the timestamp is before or equal to the given timestamp"""
        ts = sy.TimeStamp(100)
        assert ts <= sy.TimeSpan.MICROSECOND

    def test_before_eq_after(self) -> None:
        """Should return true if the timestamp is before or equal to the given timestamp"""
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
        assert ts1.datetime(tz=timezone.utc) == datetime(
            2022, 2, 22, 20, 41, 50, tzinfo=timezone.utc
        )

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
        """Should return true if the ranges overlap but a smaller range is not contained"""
        tr = sy.TimeRange(0, 1000)
        tr2 = sy.TimeRange(500, 1500)
        assert not tr.contains(tr2)

    def test_range_contains_range(self) -> None:
        """Should return true if the ranges overlap and the smaller range is contained"""
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
        "crude, expected",
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

    def test_seconds(self) -> None:
        """Should return the number of seconds in the timespan"""
        assert sy.TimeSpan.SECOND.seconds == 1

    def test_timedelta(self) -> None:
        """Should return a timedelta"""
        assert sy.TimeSpan.SECOND.timedelta == timedelta(seconds=1)

    def test_add(self) -> None:
        """Should correctly add two time spans"""
        assert sy.TimeSpan.MICROSECOND + sy.TimeSpan.MICROSECOND == sy.TimeSpan(2000)

    def test_sub(self) -> None:
        """Should correctly subtract two time spans"""
        assert sy.TimeSpan.MICROSECOND - sy.TimeSpan.MICROSECOND == sy.TimeSpan(0)

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
        "span, expected",
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
        "span, expected",
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
        "span, expected",
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
class TestRate:
    @pytest.mark.parametrize(
        "crude, expected",
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
            sy.Rate(timedelta(seconds=1))  # type: ignore

    def test_sample_count(self) -> None:
        """Should return the number of samples"""
        assert sy.Rate(1.0).sample_count(5 * sy.TimeSpan.SECOND) == 5

    def test_byte_size(self) -> None:
        """Should return the number of bytes in the given span"""
        assert sy.Rate(1.0).byte_size(5 * sy.TimeSpan.SECOND, sy.Density.BIT64) == 40

    def test_byte_span(self) -> None:
        """Should return the time span from a byte size"""
        assert (
            sy.Rate(1.0).size_span(sy.Size(40), sy.Density.BIT64)
            == 5 * sy.TimeSpan.SECOND
        )

    def test_byte_span_invalid(self) -> None:
        """Should raise a contiguity error if the size is not a multiple of the density"""
        with pytest.raises(ContiguityError):
            sy.Rate(1.0).size_span(sy.Size(41), sy.Density.BIT64)


@pytest.mark.telem
class TestDataType:
    @pytest.mark.parametrize(
        "crude, expected",
        [
            (np.int8, sy.DataType.INT8),
            (np.int16, sy.DataType.INT16),
            ("int32", sy.DataType.INT32),
            ("int64", sy.DataType.INT64),
            (np.float32(1), sy.DataType.FLOAT32),
            ({"a": 1}, sy.DataType.JSON),
            (["a"], sy.DataType.STRING),
            (uuid.uuid4(), sy.DataType.UUID),
        ],
    )
    def test_construction(self, crude: sy.CrudeDataType, expected: sy.DataType) -> None:
        assert sy.DataType(crude) == expected

    def test_string(self) -> None:
        """Should return the string representation of the data type"""
        assert str(sy.DataType.INT8) == "int8"

    @pytest.mark.parametrize(
        "value, expected",
        [
            (sy.DataType.INT8, np.dtype(np.int8)),
            (sy.DataType.FLOAT32, np.dtype(np.float32)),
        ],
    )
    def test_np(self, value: sy.DataType, expected: np.dtype[np.generic]) -> None:
        """Should return the correct numpy representation of the data type"""
        assert value.np == expected

    @pytest.mark.parametrize(
        "data_type, expected",
        [
            (sy.DataType.STRING, True),
            (sy.DataType.JSON, True),
            (sy.DataType.BYTES, True),
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


@pytest.mark.telem
class TestSize:
    @pytest.mark.parametrize(
        "crude, expected",
        [(1, sy.Size.BYTE), (1.0, sy.Size.BYTE), (sy.Size.BYTE, sy.Size.BYTE)],
    )
    def test_construction(self, crude: sy.CrudeSize, expected: sy.Size) -> None:
        assert sy.Size(crude) == expected

    @pytest.mark.parametrize(
        "value, expected",
        [(sy.Size.GB + sy.Size.MB * 500, "1gb 500mb"), (sy.Size.GB * 0, "0b")],
    )
    def test_str(self, value: sy.Size, expected: str) -> None:
        assert str(value) == expected


@pytest.mark.telem
@pytest.mark.parametrize(
    "data, from_, to, expected",
    [
        (np.array([1, 2, 3]), "s", "ms", 1000),
        (np.array([1, 2, 3]), "ms", "ms", 1),
        (np.array([sy.TimeStamp(0).datetime().isoformat()]), "iso", "ns", 0),
    ],
)
def test_convert_time_units(
    data: np.ndarray,
    from_: sy.TimeSpanUnits,
    to: sy.TimeSpanUnits,
    expected: int | float,
) -> None:
    assert sy.convert_time_units(data, from_, to)[0] == expected


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
        model2 = TestModel(alignment=packed_value)  # type: ignore[arg-type]
        assert model2.alignment.domain_index == 5
        assert model2.alignment.sample_index == 10


@pytest.mark.telem
class TestSecondsLinspace:
    def test_basic(self) -> None:
        result = sy.seconds_linspace(0, 3)
        assert len(result) == 3
        assert result[0] == 0
        assert result[1] == sy.TimeSpan.SECOND
        assert result[2] == 2 * sy.TimeSpan.SECOND

    def test_with_offset(self) -> None:
        result = sy.seconds_linspace(5, 2)
        assert result[0] == 5 * sy.TimeSpan.SECOND
        assert result[1] == 6 * sy.TimeSpan.SECOND

    def test_empty(self) -> None:
        assert sy.seconds_linspace(0, 0) == []
