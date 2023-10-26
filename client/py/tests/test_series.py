#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
import numpy as np
import pandas as pd
import pytest

from synnax.telem import DataType, Series, TimeStamp


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
        d = Series([1, 2, 3], data_type=DataType.INT8)
        s = Series(d)
        assert len(s) == 3
        assert s[2] == 3
        assert s.data_type == DataType.INT8

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

    def test_cast_as_list(self):
        """Should correctly convert the series to a builtin list"""
        s = Series([1, 2, 3], data_type=DataType.INT8)
        assert list(s) == [1, 2, 3]

    def test_greater_than(self):
        """Should correctly compare the series to a scalar"""
        s = Series([1, 2, 3], data_type=DataType.INT8)
        assert all(s > 2) == all([False, False, True])

    def test_less_than(self):
        """Should correctly compare the series to a scalar"""
        s = Series([1, 2, 3], data_type=DataType.INT8)
        assert all(s < 2) == all([True, False, False])
