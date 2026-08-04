#  Copyright 2026 Synnax Labs, Inc.
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

from x import telem


@pytest.mark.telem
@pytest.mark.series
class TestSeries:
    def test_construction_from_np(self) -> None:
        """Should correctly construct a series from a primitive numpy array"""
        d = np.array([1, 2, 3, 4], dtype=np.int8)
        s = telem.Series(d)
        assert len(s) == 4
        assert s.data_type == telem.DataType.INT8
        assert s[3] == 4

    def test_construction_from_np_data_type_override(self) -> None:
        """Should correctly convert the numpy arrays data type"""
        d = np.array([1, 2, 3, 4], dtype=np.int8)
        s = telem.Series(d, data_type=telem.DataType.FLOAT64)
        assert len(s) == 4
        assert s.data_type == telem.DataType.FLOAT64
        assert s[3] == 4
        assert s.__array__().dtype == np.float64

    def test_array_with_dtype(self) -> None:
        """Should convert to specified dtype when using __array__"""
        s = telem.Series([1, 2, 3], data_type=telem.DataType.INT64)
        arr = s.__array__(dtype=np.dtype(np.float32))
        assert arr.dtype == np.float32
        assert list(arr) == [1.0, 2.0, 3.0]

    def test_array_with_copy_true(self) -> None:
        """Should return a copy when copy=True"""
        s = telem.Series([1, 2, 3], data_type=telem.DataType.INT64)
        arr1 = s.__array__(copy=True)
        arr2 = s.__array__(copy=True)
        assert not np.shares_memory(arr1, arr2)

    def test_array_with_copy_false_same_dtype(self) -> None:
        """Should not copy when copy=False and no dtype conversion needed"""
        s = telem.Series([1, 2, 3], data_type=telem.DataType.INT64)
        arr = s.__array__(copy=False)
        assert arr.dtype == np.int64

    def test_np_array_with_dtype(self) -> None:
        """Should work with np.array() and dtype parameter (NumPy 2.0 protocol)"""
        s = telem.Series([1, 2, 3], data_type=telem.DataType.INT64)
        arr = np.array(s, dtype=np.float32)
        assert arr.dtype == np.float32

    def test_np_array_with_copy(self) -> None:
        """Should work with np.array() and copy parameter (NumPy 2.0 protocol)"""
        s = telem.Series([1, 2, 3], data_type=telem.DataType.INT64)
        arr = np.array(s, copy=True)
        assert arr.dtype == np.int64
        assert list(arr) == [1, 2, 3]

    def test_construction_from_pd_series(self) -> None:
        """Should correctly construct the array from a pandas series"""
        d = pd.Series([1, 2, 3], dtype=np.float64)
        s = telem.Series(d)
        assert len(s) == 3
        assert s[2] == 3
        assert s.data_type == telem.DataType.FLOAT64

    def test_construction_from_list(self) -> None:
        """Should correctly construct the array from a list"""
        d = [1, 2, 3]
        s = telem.Series(d)
        assert len(s) == 3
        assert s[2] == 3
        assert s.data_type == telem.DataType.INT64

    def test_construction_from_series(self) -> None:
        """Should correctly construct the telem.Series from another telem.Series"""
        d = telem.Series([1, 2, 3], data_type=telem.DataType.INT8, alignment=12)
        s = telem.Series(d)
        assert len(s) == 3
        assert s[2] == 3
        assert s.data_type == telem.DataType.INT8
        assert s.alignment == 12

    def test_construction_from_buffer(self) -> None:
        """Should correctly construct the telem.Series from a buffer"""
        d = telem.Series([1.0, 2.0, 3.0]).data
        s = telem.Series(d, data_type=telem.DataType.FLOAT64)
        assert len(s) == 3
        assert s[2] == 3
        assert s.data_type == telem.DataType.FLOAT64

    def test_construct_from_buffer_no_data_type(self) -> None:
        """Should throw a ValueError"""
        with pytest.raises(ValueError):
            assert telem.Series(b"57678")

    def test_construction_from_bool_list(self) -> None:
        """Should infer BOOL from a list of Python booleans"""
        s = telem.Series([True, False, True, False])
        assert len(s) == 4
        assert s.data_type == telem.DataType.BOOL
        assert s.data == b"\x01\x00\x01\x00"
        assert bool(s[0]) is True
        assert bool(s[1]) is False

    def test_construction_from_np_bool_array(self) -> None:
        """Should infer BOOL from a numpy bool_ array"""
        d = np.array([True, False, True], dtype=np.bool_)
        s = telem.Series(d)
        assert len(s) == 3
        assert s.data_type == telem.DataType.BOOL
        assert s.data == b"\x01\x00\x01"

    def test_bool_normalizes_nonzero_numeric_inputs(self) -> None:
        """Should normalize nonzero numeric inputs to 0x01 when dtype is BOOL"""
        s = telem.Series([0, 1, 42, -3, 0], data_type=telem.DataType.BOOL)
        assert s.data == b"\x00\x01\x01\x01\x00"
        assert bool(s[2]) is True

    def test_bool_to_numeric_cast_is_identity(self) -> None:
        """Casting a BOOL series to a numeric dtype should preserve 0/1 values"""
        s = telem.Series([True, False, True, True], data_type=telem.DataType.BOOL)
        as_int32 = s.to_numpy(dtype=np.dtype(np.int32))
        assert list(as_int32) == [1, 0, 1, 1]
        as_float64 = s.to_numpy(dtype=np.dtype(np.float64))
        assert list(as_float64) == [1.0, 0.0, 1.0, 1.0]

    def test_numeric_to_bool_cast_normalizes_nonzero(self) -> None:
        """Casting a numeric series to BOOL via to_numpy should normalize nonzero values"""
        s = telem.Series([0, 1, 42, -3, 0], data_type=telem.DataType.INT32)
        as_bool = s.to_numpy(dtype=np.dtype(np.bool_))
        assert list(as_bool) == [False, True, True, True, False]

    def test_construction_from_np_timestamp(self) -> None:
        d = telem.Series([telem.TimeStamp.now()])
        assert len(d) == 1

    def test_construction_from_int(self) -> None:
        """Should correctly construct the series from a single integer"""
        d = telem.Series(1)
        assert len(d) == 1
        assert d.data_type == telem.DataType.INT64

    def test_construction_from_int_with_dt(self) -> None:
        """Should correctly set a custom data type on the integer"""
        d = telem.Series(1, data_type=telem.DataType.INT8)
        assert len(d) == 1
        assert d.data_type == telem.DataType.INT8

    def test_construction_from_float(self) -> None:
        """Should correctly construct the series from a single float"""
        d = telem.Series(1.0)
        assert len(d) == 1
        assert d.data_type == telem.DataType.FLOAT64

    def test_construction_from_float_with_dt(self) -> None:
        """Should correctly set a custom data type on the float"""
        d = telem.Series(1.0, data_type=telem.DataType.FLOAT32)
        assert len(d) == 1
        assert d.data_type == telem.DataType.FLOAT32

    def test_construction_from_strings(self) -> None:
        """Should correctly construct the series from a list of strings"""
        d = telem.Series(["hello"])
        assert len(d) == 1
        assert d.data_type == telem.DataType.STRING
        assert d[0] == "hello"

    def test_construction_from_string(self) -> None:
        """Should correctly construct the series from a single string"""
        d = telem.Series("hello")
        assert len(d) == 1
        assert d.data_type == telem.DataType.STRING

    def test_construction_from_dicts(self) -> None:
        """Should correctly construct the series from a list of dicts"""
        d = telem.Series([{"hello": "world"}])
        assert len(d) == 1
        assert d.data_type == telem.DataType.JSON

    def test_size(self) -> None:
        """Should return the correct number of bytes in the buffer"""
        s = telem.Series([1, 2, 3], data_type=telem.DataType.INT16)
        assert s.size == 3 * 2

    def test_astype(self) -> None:
        """Should convert the series to a different data type"""
        s = telem.Series([1, 2, 3], data_type=telem.DataType.INT16)
        s = s.astype(telem.DataType.INT8)
        assert s.size == 3
        assert s[0] == 1

    def test_cast_numeric_as_list(self) -> None:
        """Should correctly convert the series to a builtin list"""
        s = telem.Series([1, 2, 3], data_type=telem.DataType.INT8)
        assert list(s) == [1, 2, 3]

    def test_cast_uuid_as_list(self) -> None:
        """Should correctly convert the series to a builtin list"""
        one = uuid.uuid4()
        two = uuid.uuid4()
        s = telem.Series([one, two], data_type=telem.DataType.UUID)
        list_ = list(s)
        assert list_[0] == one
        assert list_[1] == two

    def test_cast_json_as_list(self) -> None:
        """Should correctly convert the series to a builtin list"""
        s = telem.Series([{"hello": "world"}], data_type=telem.DataType.JSON)
        assert list(s) == [{"hello": "world"}]

    def test_cast_string_as_list(self) -> None:
        """Should correctly convert the series to a builtin list"""
        s = telem.Series(["hello"], data_type=telem.DataType.STRING)
        assert list(s) == ["hello"]

    def test_greater_than(self) -> None:
        """Should correctly compare the series to a scalar"""
        s = telem.Series([1, 2, 3], data_type=telem.DataType.INT8)
        assert list(s.__array__() > 2) == [False, False, True]

    def test_less_than(self) -> None:
        """Should correctly compare the series to a scalar"""
        s = telem.Series([1, 2, 3], data_type=telem.DataType.INT8)
        assert list(s.__array__() < 2) == [True, False, False]

    def test_list_access_numeric(self) -> None:
        """Should correctly access the series by index"""
        s = telem.Series([1, 2, 3], data_type=telem.DataType.INT8)
        assert s[0] == 1

    def test_list_access_string(self) -> None:
        """Should correctly access the series by index"""
        s = telem.Series(["hello", "world"], data_type=telem.DataType.STRING)
        assert s[1] == "world"

    def test_list_access_json(self) -> None:
        """Should correctly access the series by index"""
        s = telem.Series(
            [{"hello": "world"}, {"blue": "dog"}], data_type=telem.DataType.JSON
        )
        assert s[1] == {"blue": "dog"}

    def test_list_access_string_negative(self) -> None:
        """Should correctly access the series by index"""
        s = telem.Series(["hello", "world"], data_type=telem.DataType.STRING)
        assert s[-1] == "world"

    def test_alignment_bounds_default(self) -> None:
        """Should correctly calculate alignment_bounds with default alignment"""
        s = telem.Series([1, 2, 3, 4, 5], data_type=telem.DataType.INT8)
        bounds = s.alignment_bounds
        assert bounds.lower == 0
        assert bounds.upper == 5

    def test_alignment_bounds_with_alignment(self) -> None:
        """Should correctly calculate alignment_bounds with custom alignment"""
        s = telem.Series(
            [1, 2, 3],
            data_type=telem.DataType.INT8,
            alignment=telem.Alignment(2, 10),
        )
        bounds = s.alignment_bounds
        expected_start = (2 << 32) | 10
        assert bounds.lower == float(expected_start)
        assert bounds.upper == float(expected_start + 3)

    def test_alignment_preserved_from_series(self) -> None:
        """Should preserve alignment when constructing from another telem.Series"""
        s1 = telem.Series(
            [1, 2, 3],
            data_type=telem.DataType.INT8,
            alignment=telem.Alignment(1, 2),
        )
        s2 = telem.Series(s1)
        assert s2.alignment == s1.alignment


@pytest.mark.telem
@pytest.mark.series
class TestVariableLengthSeries:
    """Tests for variable-length series encoding (STRING, JSON)."""

    def test_string_multiple_values(self) -> None:
        """Should correctly round-trip multiple string values"""
        values = ["hello", "world", "foo"]
        s = telem.Series(values)
        assert len(s) == 3
        assert s[0] == "hello"
        assert s[1] == "world"
        assert s[2] == "foo"
        assert list(s) == values

    def test_string_empty_strings(self) -> None:
        """Should correctly handle empty strings"""
        values = ["", "", ""]
        s = telem.Series(values)
        assert len(s) == 3
        assert s[0] == ""
        assert s[1] == ""
        assert list(s) == values

    def test_string_mixed_empty_and_nonempty(self) -> None:
        """Should handle a mix of empty and non-empty strings"""
        values = ["hello", "", "world", ""]
        s = telem.Series(values)
        assert len(s) == 4
        assert s[0] == "hello"
        assert s[1] == ""
        assert s[2] == "world"
        assert s[3] == ""
        assert list(s) == values

    def test_string_single_value(self) -> None:
        """Should handle a single string in a list"""
        s = telem.Series(["only"])
        assert len(s) == 1
        assert s[0] == "only"

    def test_string_from_scalar(self) -> None:
        """Should construct from a bare string value"""
        s = telem.Series("hello")
        assert len(s) == 1
        assert s[0] == "hello"
        assert s.data_type == telem.DataType.STRING

    def test_string_multibyte_utf8(self) -> None:
        """Should correctly handle multi-byte UTF-8 characters"""
        values = ["cafe\u0301", "\u00e9", "\U0001f600"]
        s = telem.Series(values)
        assert len(s) == 3
        assert s[0] == "cafe\u0301"
        assert s[1] == "\u00e9"
        assert s[2] == "\U0001f600"
        assert list(s) == values

    def test_string_negative_index(self) -> None:
        """Should support negative indexing"""
        s = telem.Series(["a", "b", "c"])
        assert s[-1] == "c"
        assert s[-2] == "b"
        assert s[-3] == "a"

    def test_string_index_out_of_bounds(self) -> None:
        """Should raise IndexError for out-of-bounds access"""
        s = telem.Series(["a", "b"])
        with pytest.raises(IndexError):
            s[2]

    def test_string_empty_series(self) -> None:
        """Should handle an empty string series"""
        s = telem.Series([], data_type=telem.DataType.STRING)
        assert len(s) == 0
        assert list(s) == []

    def test_string_large_value(self) -> None:
        """Should handle a string longer than 255 bytes"""
        big = "x" * 1000
        s = telem.Series([big, "small"])
        assert len(s) == 2
        assert s[0] == big
        assert s[1] == "small"

    def test_json_multiple_values(self) -> None:
        """Should correctly round-trip multiple JSON values"""
        values = [{"a": "1"}, {"b": "2"}, {"c": "3"}]
        s = telem.Series(values)
        assert len(s) == 3
        assert s[0] == {"a": "1"}
        assert s[1] == {"b": "2"}
        assert s[2] == {"c": "3"}
        assert list(s) == values

    def test_json_from_scalar_dict(self) -> None:
        """Should construct from a bare dict value"""
        s = telem.Series({"key": "val"})
        assert len(s) == 1
        assert s[0] == {"key": "val"}
        assert s.data_type == telem.DataType.JSON

    def test_json_empty_objects(self) -> None:
        """Should handle empty JSON objects"""
        values = [dict[str, str](), dict[str, str](), dict[str, str]()]
        s = telem.Series(values)
        assert len(s) == 3
        assert list(s) == values

    def test_json_negative_index(self) -> None:
        """Should support negative indexing for JSON series"""
        s = telem.Series([{"a": 1}, {"b": 2}, {"c": 3}])
        assert s[-1] == {"c": 3}

    def test_json_empty_series(self) -> None:
        """Should handle an empty JSON series"""
        s = telem.Series([], data_type=telem.DataType.JSON)
        assert len(s) == 0
        assert list(s) == []

    def test_json_index_out_of_bounds(self) -> None:
        """Should raise IndexError for out-of-bounds access"""
        s = telem.Series([{"a": 1}])
        with pytest.raises(IndexError):
            s[1]

    def test_string_len_cached(self) -> None:
        """Should cache the length computation for variable-length series"""
        s = telem.Series(["a", "b", "c"])
        assert len(s) == 3
        assert len(s) == 3

    def test_string_iteration_order(self) -> None:
        """Should iterate in insertion order"""
        values = ["first", "second", "third", "fourth"]
        s = telem.Series(values)
        assert [v for v in s] == values

    def test_json_iteration_order(self) -> None:
        """Should iterate JSON values in insertion order"""
        values = [{"i": 0}, {"i": 1}, {"i": 2}]
        s = telem.Series(values)
        assert [v for v in s] == values

    def test_string_size_bytes(self) -> None:
        """Should report the correct raw byte size including prefixes"""
        s = telem.Series(["ab", "c"])
        # "ab" = 4-byte prefix + 2 bytes, "c" = 4-byte prefix + 1 byte = 11
        assert s.size == 11

    def test_json_size_bytes(self) -> None:
        """Should report the correct raw byte size including prefixes"""
        s = telem.Series([{"a": 1}])
        import json

        encoded = json.dumps({"a": 1}).encode("utf-8")
        assert s.size == 4 + len(encoded)


@pytest.mark.telem
@pytest.mark.series
class TestMultiSeries:
    def test_construction_from_multiple_series(self) -> None:
        """Should correctly construct a telem.MultiSeries from multiple telem.Series :)"""
        s1 = telem.Series([1, 2, 3], data_type=telem.DataType.INT8)
        s2 = telem.Series([4, 5, 6], data_type=telem.DataType.INT8)
        s = telem.MultiSeries([s1, s2])
        assert len(s) == 6

    def test_construction_mismatched_data_types(self) -> None:
        """Should throw a ValueError"""
        s1 = telem.Series([1, 2, 3], data_type=telem.DataType.INT8)
        s2 = telem.Series([4, 5, 6], data_type=telem.DataType.INT16)
        with pytest.raises(ValueError):
            telem.MultiSeries([s1, s2])

    def test_construction_from_none(self) -> None:
        """Should throw a ValueError"""
        s = telem.MultiSeries([])
        assert len(s) == 0

    def test_conversion_to_numpy(self) -> None:
        """Should correctly convert the telem.MultiSeries to a numpy array"""
        s1 = telem.Series([1, 2, 3], data_type=telem.DataType.INT8)
        s2 = telem.Series([4, 5, 6], data_type=telem.DataType.INT8)
        s = telem.MultiSeries([s1, s2])
        assert len(s.to_numpy()) == 6
        assert s.to_numpy().dtype == np.int8

    def test_array_with_dtype(self) -> None:
        """Should convert to specified dtype when using __array__"""
        s1 = telem.Series([1, 2, 3], data_type=telem.DataType.INT8)
        s2 = telem.Series([4, 5, 6], data_type=telem.DataType.INT8)
        s = telem.MultiSeries([s1, s2])
        arr = s.__array__(dtype=np.dtype(np.float32))
        assert arr.dtype == np.float32
        assert list(arr) == [1.0, 2.0, 3.0, 4.0, 5.0, 6.0]

    def test_np_array_with_dtype(self) -> None:
        """Should work with np.array() and dtype parameter (NumPy 2.0 protocol)"""
        s1 = telem.Series([1, 2, 3], data_type=telem.DataType.INT8)
        s2 = telem.Series([4, 5, 6], data_type=telem.DataType.INT8)
        s = telem.MultiSeries([s1, s2])
        arr = np.array(s, dtype=np.float64)
        assert arr.dtype == np.float64

    def test_np_array_with_copy(self) -> None:
        """Should work with np.array() and copy parameter (NumPy 2.0 protocol)"""
        s1 = telem.Series([1, 2, 3], data_type=telem.DataType.INT8)
        s2 = telem.Series([4, 5, 6], data_type=telem.DataType.INT8)
        s = telem.MultiSeries([s1, s2])
        arr = np.array(s, copy=True)
        assert arr.dtype == np.int8
        assert list(arr) == [1, 2, 3, 4, 5, 6]

    def test_time_range(self) -> None:
        """Should correctly return the time range of the telem.MultiSeries"""
        s1 = telem.Series(
            data=[1, 2, 3],
            data_type=telem.DataType.INT8,
            time_range=telem.TimeRange(
                start=1 * telem.TimeSpan.SECOND, end=3 * telem.TimeSpan.SECOND
            ),
        )
        s2 = telem.Series(
            data=[4, 5, 6],
            data_type=telem.DataType.INT8,
            time_range=telem.TimeRange(
                start=4 * telem.TimeSpan.SECOND, end=6 * telem.TimeSpan.SECOND
            ),
        )
        s = telem.MultiSeries([s1, s2])
        assert s.time_range is not None
        assert s.time_range.start == 1 * telem.TimeSpan.SECOND
        assert s.time_range.end == 6 * telem.TimeSpan.SECOND

    def test_access_by_index(self) -> None:
        """Should correctly access the telem.MultiSeries by index"""
        s1 = telem.Series([1, 2, 3], data_type=telem.DataType.INT8)
        s2 = telem.Series([4, 5, 6], data_type=telem.DataType.INT8)
        s = telem.MultiSeries([s1, s2])
        assert s[0] == 1
        assert s[1] == 2
        assert s[2] == 3
        assert s[5] == 6
        assert s[-1] == 6

    def test_conversion_to_list_string(self) -> None:
        """Should correctly convert the telem.MultiSeries to a list of strings"""
        s1 = telem.Series(["hello", "world"], data_type=telem.DataType.STRING)
        s2 = telem.Series(["blue", "dog"], data_type=telem.DataType.STRING)
        s = telem.MultiSeries([s1, s2])
        assert list(s) == ["hello", "world", "blue", "dog"]

    def test_conversion_to_list_numeric(self) -> None:
        """Should correctly convert the telem.MultiSeries to a list of numbers"""
        s1 = telem.Series([1, 2, 3], data_type=telem.DataType.INT8)
        s2 = telem.Series([4, 5, 6], data_type=telem.DataType.INT8)
        s = telem.MultiSeries([s1, s2])
        assert list(s) == [1, 2, 3, 4, 5, 6]

    def test_conversion_to_list_json(self) -> None:
        """Should correctly convert the telem.MultiSeries to a list of dicts"""
        s1 = telem.Series(
            [{"hello": "world"}, {"blue": "dog"}], data_type=telem.DataType.JSON
        )
        s2 = telem.Series(
            [{"red": "car"}, {"green": "tree"}], data_type=telem.DataType.JSON
        )
        s = telem.MultiSeries([s1, s2])
        assert list(s) == [
            {"hello": "world"},
            {"blue": "dog"},
            {"red": "car"},
            {"green": "tree"},
        ]

    def test_alignment_from_first_series(self) -> None:
        """Should return the alignment of the first series"""
        s1 = telem.Series(
            [1, 2, 3], data_type=telem.DataType.INT8, alignment=telem.Alignment(1, 5)
        )
        s2 = telem.Series(
            [4, 5, 6], data_type=telem.DataType.INT8, alignment=telem.Alignment(2, 10)
        )
        ms = telem.MultiSeries([s1, s2])
        assert ms.alignment == telem.Alignment(1, 5)

    def test_alignment_empty_multiseries(self) -> None:
        """Should return telem.Alignment(0, 0) for empty telem.MultiSeries"""
        ms = telem.MultiSeries([])
        assert ms.alignment == telem.Alignment(0, 0)

    def test_alignment_bounds_multiseries(self) -> None:
        """Should correctly calculate alignment_bounds from first to last series"""
        s1 = telem.Series(
            [1, 2, 3],
            data_type=telem.DataType.INT8,
            alignment=telem.Alignment(1, 0),
        )
        s2 = telem.Series(
            [4, 5],
            data_type=telem.DataType.INT8,
            alignment=telem.Alignment(1, 10),
        )
        ms = telem.MultiSeries([s1, s2])
        bounds = ms.alignment_bounds
        assert bounds.lower == s1.alignment_bounds.lower
        assert bounds.upper == s2.alignment_bounds.upper

    def test_alignment_bounds_empty_multiseries(self) -> None:
        """Should return Bounds(0, 0) for empty telem.MultiSeries"""
        ms = telem.MultiSeries([])
        bounds = ms.alignment_bounds
        assert bounds.lower == 0
        assert bounds.upper == 0

    def test_empty_multiseries_to_numpy(self) -> None:
        """Should return an empty numpy array for empty telem.MultiSeries"""
        ms = telem.MultiSeries([])
        arr = ms.to_numpy()
        assert len(arr) == 0
        assert arr.dtype == np.float64

    def test_empty_multiseries_to_numpy_with_dtype(self) -> None:
        """Should return an empty numpy array with specified dtype"""
        ms = telem.MultiSeries([])
        arr = ms.to_numpy(dtype=np.dtype(np.int32))
        assert len(arr) == 0
        assert arr.dtype == np.int32

    def test_single_series_copy_false(self) -> None:
        """Should respect copy=False for single series MultiSeries"""
        s = telem.Series([1, 2, 3], data_type=telem.DataType.FLOAT64)
        ms = telem.MultiSeries([s])
        arr1 = ms.__array__(copy=False)
        arr2 = ms.__array__(copy=False)
        assert np.shares_memory(arr1, arr2)

    def test_single_series_copy_true(self) -> None:
        """Should create a copy when copy=True for single series MultiSeries"""
        s = telem.Series([1, 2, 3], data_type=telem.DataType.FLOAT64)
        ms = telem.MultiSeries([s])
        arr1 = ms.__array__(copy=True)
        arr2 = ms.__array__(copy=True)
        assert not np.shares_memory(arr1, arr2)
