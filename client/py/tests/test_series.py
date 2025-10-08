#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import uuid

import numpy as np
import pandas as pd
import pytest

from synnax.telem import DataType, MultiSeries, Series, TimeRange, TimeSpan, TimeStamp


@pytest.mark.telem
@pytest.mark.series
class TestSeries:
    def test_construction_from_np(self):
        """Should correctly construct a series from a primitive numpy array"""
        d = np.array([1, 2, 3, 4], dtype=np.int8)
        s = Series(d)
        assert len(s) == 4
        assert s.data_type == DataType.INT8
        assert s[3] == 4

    def test_construction_from_np_data_type_override(self):
        """Should correctly convert the numpy arrays data type"""
        d = np.array([1, 2, 3, 4], dtype=np.int8)
        s = Series(d, data_type=DataType.FLOAT64)
        assert len(s) == 4
        assert s.data_type == DataType.FLOAT64
        assert s[3] == 4
        assert s.__array__().dtype == np.float64

    def test_construction_from_pd_series(self):
        """Should correctly construct the array from a pandas series"""
        d = pd.Series([1, 2, 3], dtype=np.float64)
        s = Series(d)
        assert len(s) == 3
        assert s[2] == 3
        assert s.data_type == DataType.FLOAT64

    def test_construction_from_list(self):
        """Should correctly construct the array from a list"""
        d = [1, 2, 3]
        s = Series(d)
        assert len(s) == 3
        assert s[2] == 3
        assert s.data_type == DataType.INT64

    def test_construction_from_series(self):
        """Should correctly construct the Series from another Series"""
        d = Series([1, 2, 3], data_type=DataType.INT8, alignment=12)
        s = Series(d)
        assert len(s) == 3
        assert s[2] == 3
        assert s.data_type == DataType.INT8
        assert s.alignment == 12

    def test_construction_from_buffer(self):
        """Should correctly construct the Series from a buffer"""
        d = Series([1.0, 2.0, 3.0]).data
        s = Series(d, data_type=DataType.FLOAT64)
        assert len(s) == 3
        assert s[2] == 3
        assert s.data_type == DataType.FLOAT64

    def test_construct_from_buffer_no_data_type(self):
        """Should throw a ValueError"""
        with pytest.raises(ValueError):
            assert Series(b"57678")

    def test_construction_from_np_timestamp(self):
        d = Series([TimeStamp.now()])
        assert len(d) == 1

    def test_construction_from_int(self):
        """Should correctly construct the series from a single integer"""
        d = Series(1)
        assert len(d) == 1
        assert d.data_type == DataType.INT64

    def test_construction_from_int_with_dt(self):
        """Should correctly set a custom data type on the integer"""
        d = Series(1, data_type=DataType.INT8)
        assert len(d) == 1
        assert d.data_type == DataType.INT8

    def test_construction_from_float(self):
        """Should correctly construct the series from a single float"""
        d = Series(1.0)
        assert len(d) == 1
        assert d.data_type == DataType.FLOAT64

    def test_construction_from_float_with_dt(self):
        """Should correctly set a custom data type on the float"""
        d = Series(1.0, data_type=DataType.FLOAT32)
        assert len(d) == 1
        assert d.data_type == DataType.FLOAT32

    def test_construction_from_strings(self):
        """Should correctly construct the series from a list of strings"""
        d = Series(["hello"])
        assert len(d) == 1
        assert d.data_type == DataType.STRING
        assert d[0] == "hello"

    def test_construction_from_string(self):
        """Should correctly construct the series from a single string"""
        d = Series("hello")
        assert len(d) == 1
        assert d.data_type == DataType.STRING

    def test_construction_from_dicts(self):
        """Should correctly construct the series from a list of dicts"""
        d = Series([{"hello": "world"}])
        assert len(d) == 1
        assert d.data_type == DataType.JSON

    def test_size(self):
        """Should return the correct number of bytes in the buffer"""
        s = Series([1, 2, 3], data_type=DataType.INT16)
        assert s.size == 3 * 2

    def test_astype(self):
        """Should convert the series to a different data type"""
        s = Series([1, 2, 3], data_type=DataType.INT16)
        s = s.astype(DataType.INT8)
        assert s.size == 3
        assert s[0] == 1

    def test_cast_numeric_as_list(self):
        """Should correctly convert the series to a builtin list"""
        s = Series([1, 2, 3], data_type=DataType.INT8)
        assert list(s) == [1, 2, 3]

    def test_cast_uuid_as_list(self):
        """Should correctly convert the series to a builtin list"""
        one = uuid.uuid4()
        two = uuid.uuid4()
        s = Series([one, two], data_type=DataType.UUID)
        list_ = list(s)
        assert list_[0] == one
        assert list_[1] == two

    def test_cast_json_as_list(self):
        """Should correctly convert the series to a builtin list"""
        s = Series([{"hello": "world"}], data_type=DataType.JSON)
        assert list(s) == [{"hello": "world"}]

    def test_cast_string_as_list(self):
        """Should correctly convert the series to a builtin list"""
        s = Series(["hello"], data_type=DataType.STRING)
        assert list(s) == ["hello"]

    def test_greater_than(self):
        """Should correctly compare the series to a scalar"""
        s = Series([1, 2, 3], data_type=DataType.INT8)
        assert all(s > 2) == all([False, False, True])

    def test_less_than(self):
        """Should correctly compare the series to a scalar"""
        s = Series([1, 2, 3], data_type=DataType.INT8)
        assert all(s < 2) == all([True, False, False])

    def test_list_access_numeric(self):
        """Should correctly access the series by index"""
        s = Series([1, 2, 3], data_type=DataType.INT8)
        assert s[0] == 1

    def test_list_access_string(self):
        """Should correctly access the series by index"""
        s = Series(["hello", "world"], data_type=DataType.STRING)
        assert s[1] == "world"

    def test_list_access_json(self):
        """Should correctly access the series by index"""
        s = Series([{"hello": "world"}, {"blue": "dog"}], data_type=DataType.JSON)
        assert s[1] == {"blue": "dog"}

    def test_list_access_string_negative(self):
        """Should correctly access the series by index"""
        s = Series(["hello", "world"], data_type=DataType.STRING)
        assert s[-1] == "world"


@pytest.mark.telem
@pytest.mark.series
class TestMultiSeries:
    def test_construction_from_multiple_series(self):
        """Should correctly construct a MultiSeries from multiple Series :)"""
        s1 = Series([1, 2, 3], data_type=DataType.INT8)
        s2 = Series([4, 5, 6], data_type=DataType.INT8)
        s = MultiSeries([s1, s2])
        assert len(s) == 6

    def test_construction_mismatched_data_types(self):
        """Should throw a ValueError"""
        s1 = Series([1, 2, 3], data_type=DataType.INT8)
        s2 = Series([4, 5, 6], data_type=DataType.INT16)
        with pytest.raises(ValueError):
            MultiSeries([s1, s2])

    def test_construction_from_none(self):
        """Should throw a ValueError"""
        s = MultiSeries([])
        assert len(s) == 0

    def test_conversion_to_numpy(self):
        """Should correctly convert the MultiSeries to a numpy array"""
        s1 = Series([1, 2, 3], data_type=DataType.INT8)
        s2 = Series([4, 5, 6], data_type=DataType.INT8)
        s = MultiSeries([s1, s2])
        assert len(s.to_numpy()) == 6
        assert s.to_numpy().dtype == np.int8

    def test_time_range(self):
        """Should correctly return the time range of the MultiSeries"""
        s1 = Series(
            data=[1, 2, 3],
            data_type=DataType.INT8,
            time_range=TimeRange(start=1 * TimeSpan.SECOND, end=3 * TimeSpan.SECOND),
        )
        s2 = Series(
            data=[4, 5, 6],
            data_type=DataType.INT8,
            time_range=TimeRange(start=4 * TimeSpan.SECOND, end=6 * TimeSpan.SECOND),
        )
        s = MultiSeries([s1, s2])
        assert s.time_range.start == 1 * TimeSpan.SECOND
        assert s.time_range.end == 6 * TimeSpan.SECOND

    def test_access_by_index(self):
        """Should correctly access the MultiSeries by index"""
        s1 = Series([1, 2, 3], data_type=DataType.INT8)
        s2 = Series([4, 5, 6], data_type=DataType.INT8)
        s = MultiSeries([s1, s2])
        assert s[0] == 1
        assert s[1] == 2
        assert s[2] == 3
        assert s[5] == 6
        assert s[-1] == 6

    def test_conversion_to_list_string(self):
        """Should correctly convert the MultiSeries to a list of strings"""
        s1 = Series(["hello", "world"], data_type=DataType.STRING)
        s2 = Series(["blue", "dog"], data_type=DataType.STRING)
        s = MultiSeries([s1, s2])
        assert list(s) == ["hello", "world", "blue", "dog"]

    def test_conversion_to_list_numeric(self):
        """Should correctly convert the MultiSeries to a list of numbers"""
        s1 = Series([1, 2, 3], data_type=DataType.INT8)
        s2 = Series([4, 5, 6], data_type=DataType.INT8)
        s = MultiSeries([s1, s2])
        assert list(s) == [1, 2, 3, 4, 5, 6]

    def test_conversion_to_list_json(self):
        """Should correctly convert the MultiSeries to a list of dicts"""
        s1 = Series([{"hello": "world"}, {"blue": "dog"}], data_type=DataType.JSON)
        s2 = Series([{"red": "car"}, {"green": "tree"}], data_type=DataType.JSON)
        s = MultiSeries([s1, s2])
        assert list(s) == [
            {"hello": "world"},
            {"blue": "dog"},
            {"red": "car"},
            {"green": "tree"},
        ]
