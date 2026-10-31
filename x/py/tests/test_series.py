#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import operator
import uuid
from collections.abc import Callable
from typing import Any, Literal, assert_type

import numpy as np
import numpy.typing as npt
import pandas as pd
import pytest
from hypothesis import given
from hypothesis import strategies as st
from pydantic import BaseModel, JsonValue, TypeAdapter, ValidationError

from x import telem
from x.interop import ArrayOperators


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
        with pytest.raises(ValueError, match="data_type must be specified"):
            telem.Series(b"57678")

    def test_construction_from_bool_list(self) -> None:
        """Should infer BOOLEAN from a list of Python booleans"""
        s = telem.Series([True, False, True, False])
        assert len(s) == 4
        assert s.data_type == telem.DataType.BOOLEAN
        assert s.data == b"\x01\x00\x01\x00"
        assert bool(s[0]) is True
        assert bool(s[1]) is False

    def test_construction_from_np_bool_array(self) -> None:
        """Should infer BOOLEAN from a numpy bool_ array"""
        d = np.array([True, False, True], dtype=np.bool_)
        s = telem.Series(d)
        assert len(s) == 3
        assert s.data_type == telem.DataType.BOOLEAN
        assert s.data == b"\x01\x00\x01"

    @pytest.mark.parametrize("value", [np.True_, np.False_])
    def test_construction_from_np_bool_scalar(self, value: np.bool_) -> None:
        """Should infer BOOLEAN from a numpy bool_ scalar"""
        s = telem.Series(value)
        assert s.data_type == telem.DataType.BOOLEAN
        assert len(s) == 1
        assert s.data == bytes([int(value)])
        assert s[0] == value

    def test_bool_normalizes_nonzero_numeric_inputs(self) -> None:
        """Should normalize nonzero numeric inputs to 0x01 when dtype is BOOLEAN"""
        s = telem.Series([0, 1, 42, -3, 0], data_type=telem.DataType.BOOLEAN)
        assert s.data == b"\x00\x01\x01\x01\x00"
        assert bool(s[2]) is True

    def test_bool_to_numeric_cast_is_identity(self) -> None:
        """Casting a BOOLEAN series to a numeric dtype should preserve 0/1 values"""
        s = telem.Series([True, False, True, True], data_type=telem.DataType.BOOLEAN)
        as_int32 = s.to_numpy(dtype=np.dtype(np.int32))
        assert list(as_int32) == [1, 0, 1, 1]
        as_float64 = s.to_numpy(dtype=np.dtype(np.float64))
        assert list(as_float64) == [1.0, 0.0, 1.0, 1.0]

    def test_numeric_to_bool_cast_normalizes_nonzero(self) -> None:
        """Casting a numeric series to BOOLEAN via to_numpy should normalize nonzero
        values
        """
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

    def test_alignment_bounds_with_multiple(self) -> None:
        """Should step alignment_bounds by the alignment multiple of each sample"""
        s = telem.Series(
            [1, 2, 3],
            data_type=telem.DataType.INT8,
            alignment=telem.Alignment(0, 8),
            alignment_multiple=4,
        )
        assert s.alignment_bounds.lower == 8
        assert s.alignment_bounds.upper == 20

    def test_alignment_multiple_preserved_from_series(self) -> None:
        """Should preserve the alignment multiple when constructing from a Series"""
        s1 = telem.Series([1, 2], data_type=telem.DataType.INT8, alignment_multiple=4)
        assert telem.Series(s1).alignment_multiple == 4
        assert telem.Series(telem.MultiSeries([s1])).alignment_multiple == 4

    def test_alignment_multiple_argument_wins(self) -> None:
        """Should prefer an explicit alignment multiple to the source's multiple"""
        s1 = telem.Series([1, 2], data_type=telem.DataType.INT8, alignment_multiple=4)
        assert telem.Series(s1, alignment_multiple=2).alignment_multiple == 2
        multi = telem.MultiSeries([s1])
        assert telem.Series(multi, alignment_multiple=2).alignment_multiple == 2

    def test_astype_keeps_alignment_multiple(self) -> None:
        """Should keep the alignment multiple when converting the data type"""
        s1 = telem.Series([1, 2], data_type=telem.DataType.INT8, alignment_multiple=4)
        converted = s1.astype(telem.DataType.FLOAT64)
        assert converted.alignment_multiple == 4
        assert converted.alignment_bounds.upper == 8

    def test_alignment_preserved_from_series(self) -> None:
        """Should preserve alignment when constructing from another telem.Series"""
        s1 = telem.Series(
            [1, 2, 3],
            data_type=telem.DataType.INT8,
            alignment=telem.Alignment(1, 2),
        )
        s2 = telem.Series(s1)
        assert s2.alignment == s1.alignment

    def test_alignment_override_from_series(self) -> None:
        """Should use a given alignment over the source series' alignment"""
        s1 = telem.Series([1, 2, 3], alignment=telem.Alignment(1, 2))
        assert telem.Series(s1, alignment=5).alignment == 5

    def test_astype_keeps_alignment(self) -> None:
        """Should keep the alignment when casting"""
        s = telem.Series([1, 2, 3], alignment=7)
        assert s.astype(telem.DataType.FLOAT64).alignment == 7

    def test_infer_mixed_numeric_list(self) -> None:
        """Should infer a float series when ints and floats are mixed"""
        s = telem.Series([1, 2.5])
        assert s.data_type == telem.DataType.FLOAT64
        assert list(s) == [1.0, 2.5]

    @pytest.mark.parametrize(
        ("values", "data_type"),
        [
            ([2**53 + 1, np.uint64(1)], telem.DataType.INT64),
            ([np.uint64(2**63), 1], telem.DataType.UINT64),
        ],
    )
    def test_infer_mixed_ints_keeps_values(
        self, values: list[Any], data_type: telem.DataType
    ) -> None:
        """Should pick an integer type that holds every mixed integer exactly"""
        s = telem.Series(values)
        assert s.data_type == data_type
        assert np.asarray(s).tolist() == [int(v) for v in values]

    @pytest.mark.parametrize(
        "values", [[np.uint64(2**63), np.int64(-1)], [2**53 + 1, 0.5]]
    )
    def test_infer_rejects_lossy_promotion(self, values: list[Any]) -> None:
        """Should reject a mix that no inferred type holds exactly"""
        with pytest.raises(TypeError):
            telem.Series(values)

    def test_infer_rejects_bool_and_number_list(self) -> None:
        """Should reject a list that mixes booleans and numbers"""
        with pytest.raises(TypeError):
            telem.Series([True, 2])

    def test_infer_timestamp_scalar(self) -> None:
        """Should infer a timestamp series from a single TimeStamp"""
        assert telem.Series(telem.TimeStamp(5)).data_type == telem.DataType.TIMESTAMP

    def test_uuid_list_rejects_non_uuid(self) -> None:
        """Should reject a value that is not a UUID instead of dropping it"""
        values: list[Any] = [uuid.uuid4(), "not-a-uuid"]
        with pytest.raises(TypeError):
            telem.Series(values, data_type=telem.DataType.UUID)

    def test_json_from_scalar_string(self) -> None:
        """Should encode a bare string as one JSON value, like a one-item list"""
        s = telem.Series("hello", data_type=telem.DataType.JSON)
        assert s.data_type == telem.DataType.JSON
        assert s.data == telem.Series(["hello"], data_type=telem.DataType.JSON).data


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
        with pytest.raises(IndexError):
            s[-4]

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
        assert list(s) == values

    def test_json_iteration_order(self) -> None:
        """Should iterate JSON values in insertion order"""
        values = [{"i": 0}, {"i": 1}, {"i": 2}]
        s = telem.Series(values)
        assert list(s) == values

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
        """Should correctly construct a telem.MultiSeries from multiple
        telem.Series :)
        """
        s1 = telem.Series([1, 2, 3], data_type=telem.DataType.INT8)
        s2 = telem.Series([4, 5, 6], data_type=telem.DataType.INT8)
        s = telem.MultiSeries([s1, s2])
        assert len(s) == 6

    def test_construction_mismatched_data_types(self) -> None:
        """Should throw a ValueError"""
        s1 = telem.Series([1, 2, 3], data_type=telem.DataType.INT8)
        s2 = telem.Series([4, 5, 6], data_type=telem.DataType.INT16)
        with pytest.raises(ValueError, match="must have the same data type"):
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


@pytest.mark.telem
@pytest.mark.series
class TestSampleConversion:
    @pytest.mark.parametrize(
        ("value", "expected"),
        [(3, 3.0), (2.5, 2.5), (np.float32(1.5), 1.5), (np.int64(4), 4.0)],
    )
    def test_to_float(self, value: telem.SampleValue, expected: float) -> None:
        """Should convert a numeric sample to a float"""
        assert telem.to_float(value) == expected

    @pytest.mark.parametrize(
        ("value", "expected"),
        [(3, 3), (np.int64(4), 4), (telem.TimeStamp(7), 7)],
    )
    def test_to_int(self, value: telem.SampleValue, expected: int) -> None:
        """Should convert an integer or timestamp sample to an int"""
        assert telem.to_int(value) == expected

    @pytest.mark.parametrize("value", ["a", uuid.uuid4(), {"a": 1}])
    def test_to_float_rejects(self, value: telem.SampleValue) -> None:
        """Should reject a sample that is not numeric"""
        with pytest.raises(TypeError):
            telem.to_float(value)

    @pytest.mark.parametrize("value", [1.5, np.float64(2.0), "a"])
    def test_to_int_rejects(self, value: telem.SampleValue) -> None:
        """Should reject a sample that is not an integer"""
        with pytest.raises(TypeError):
            telem.to_int(value)


_SERIES: TypeAdapter[telem.Series] = TypeAdapter(telem.Series)


class _TypedHolder(BaseModel):
    series: telem.Series[np.float64]


class _UnionHolder(BaseModel):
    series: telem.Series[np.float32 | np.float64]


class _JsonHolder(BaseModel):
    series: telem.Series[JsonValue]


class _NumberHolder(BaseModel):
    series: telem.Series[np.number]


class _Holder(BaseModel):
    series: telem.Series


def _series() -> telem.Series:
    return telem.Series(np.array([1, 2, 3], dtype=np.int64))


def _multi_series() -> telem.MultiSeries:
    return telem.MultiSeries(
        [
            telem.Series(np.array([1, 2], dtype=np.int64)),
            telem.Series(np.array([3], dtype=np.int64)),
        ]
    )


@pytest.mark.telem
@pytest.mark.series
@pytest.mark.parametrize("create", [_series, _multi_series])
class TestArrayOperators:
    def test_comparison(self, create: Callable[[], ArrayOperators]) -> None:
        """Should compare element-wise and return a boolean array"""
        a = create()
        assert np.array_equal(a == 2, [False, True, False])
        assert np.array_equal(a > 1, [False, True, True])

    def test_arithmetic(self, create: Callable[[], ArrayOperators]) -> None:
        """Should apply arithmetic element-wise"""
        assert np.array_equal(create() * 2, [2, 4, 6])

    def test_reflected(self, create: Callable[[], ArrayOperators]) -> None:
        """Should apply a reflected operator when the left operand is a scalar"""
        assert np.array_equal(10 - create(), [9, 8, 7])

    def test_negation(self, create: Callable[[], ArrayOperators]) -> None:
        """Should negate every sample"""
        assert np.array_equal(-create(), [-1, -2, -3])

    @pytest.mark.parametrize(
        "op",
        [
            operator.eq,
            operator.ne,
            operator.lt,
            operator.gt,
            operator.le,
            operator.ge,
            operator.add,
            operator.sub,
            operator.mul,
            operator.truediv,
            operator.floordiv,
            operator.mod,
            operator.pow,
            operator.lshift,
            operator.rshift,
            operator.and_,
            operator.or_,
            operator.xor,
            operator.matmul,
            divmod,
        ],
    )
    def test_binary_matches_numpy(
        self, create: Callable[[], ArrayOperators], op: Callable[[Any, Any], Any]
    ) -> None:
        """Should match NumPy for every binary operator in both operand orders"""
        a = create()
        arr = np.asarray(a)
        # A list on the left makes Python call the reflected operator.
        other = [3, 2, 1]
        assert np.array_equal(op(a, other), op(arr, np.array(other)))
        assert np.array_equal(op(other, a), op(np.array(other), arr))

    @pytest.mark.parametrize(
        "op", [operator.neg, operator.pos, operator.abs, operator.invert]
    )
    def test_unary_matches_numpy(
        self, create: Callable[[], ArrayOperators], op: Callable[[Any], Any]
    ) -> None:
        """Should match NumPy for every unary operator"""
        a = create()
        assert np.array_equal(op(a), op(np.asarray(a)))

    def test_in_place(self, create: Callable[[], ArrayOperators]) -> None:
        """Should rebind an in-place operation to a new array"""
        a: Any = create()
        a += 1
        assert isinstance(a, np.ndarray)
        assert np.array_equal(a, [2, 3, 4])

    def test_unhashable(self, create: Callable[[], ArrayOperators]) -> None:
        """Should not be hashable, like a NumPy array"""
        with pytest.raises(TypeError):
            hash(create())


@pytest.mark.telem
@pytest.mark.series
class TestSeriesModel:
    def test_round_trip(self) -> None:
        """Should validate a Series from its serialized form"""
        s = telem.Series(
            [1, 2, 3],
            time_range=telem.TimeRange(1, 4),
            alignment=7,
            alignment_multiple=2,
        )
        restored = _SERIES.validate_python(_SERIES.dump_python(s))
        assert restored.data == s.data
        assert restored.data_type == s.data_type
        assert restored.time_range == s.time_range
        assert restored.alignment == s.alignment
        assert restored.alignment_multiple == 2

    def test_json_round_trip(self) -> None:
        """Should round-trip a Series through JSON"""
        s = telem.Series(["a", "b"])
        restored = _SERIES.validate_json(_SERIES.dump_json(s))
        assert list(restored) == ["a", "b"]

    def test_instance(self) -> None:
        """Should accept a Series instance as is"""
        s = _series()
        assert _SERIES.validate_python(s) is s

    def test_model_field(self) -> None:
        """Should validate a Series field of a model from a dict"""
        m = _Holder.model_validate(
            {"series": {"data_type": "float32", "data": telem.Series([1.5]).data}}
        )
        assert m.series.data_type == telem.DataType.FLOAT32

    def test_typed_field(self) -> None:
        """Should accept a typed field whose samples match"""
        m = _TypedHolder.model_validate(
            {"series": {"data_type": "float64", "data": telem.Series([1.5]).data}}
        )
        assert_type(m.series[0], np.float64)
        assert _TypedHolder(series=telem.Series([2.5]).as_(np.float64)).series[0] == 2.5

    @pytest.mark.parametrize(
        "series",
        [telem.Series(["text"]), {"data_type": "string", "data": b""}],
        ids=["instance", "payload"],
    )
    def test_typed_field_mismatch(self, series: object) -> None:
        """Should reject a typed field whose samples do not match"""
        with pytest.raises(ValidationError, match="does not hold samples"):
            _TypedHolder.model_validate({"series": series})

    @pytest.mark.parametrize(
        ("model", "series", "valid"),
        [
            (_UnionHolder, telem.Series([1.5]), True),
            (_UnionHolder, telem.Series([1]), False),
            (_JsonHolder, telem.Series([{"a": 1}]), True),
            (_JsonHolder, telem.Series(["a"]), True),
            (_JsonHolder, telem.Series([uuid.UUID(int=1)]), False),
            (_NumberHolder, telem.Series([1], data_type=telem.DataType.INT32), True),
        ],
        ids=[
            "union",
            "union-mismatch",
            "json",
            "json-string",
            "json-mismatch",
            "superclass",
        ],
    )
    def test_annotated_field(
        self, model: type[BaseModel], series: telem.Series, valid: bool
    ) -> None:
        """Should check a field annotated with a union, JsonValue, or superclass"""
        if valid:
            model.model_validate({"series": series})
        else:
            with pytest.raises(ValidationError, match="does not hold samples"):
                model.model_validate({"series": series})

    def test_unsupported_annotation(self) -> None:
        """Should reject a field annotation that is not a sample type"""
        with pytest.raises(TypeError, match="cannot validate samples"):

            class _LiteralHolder(BaseModel):
                series: telem.Series[Literal["a"]]

    def test_iter(self) -> None:
        """Should iterate over samples, not model fields"""
        assert list(_series()) == [1, 2, 3]


@pytest.mark.telem
@pytest.mark.series
class TestSeriesNarrowing:
    def test_numeric(self) -> None:
        """Should narrow a numeric series to its numpy scalar type"""
        s = telem.Series([1.5, 2.5]).as_(np.float64)
        assert_type(s, telem.Series[np.float64])
        assert_type(s[0], np.float64)
        assert_type(list(s), list[np.float64])
        assert_type(s.to_numpy(), npt.NDArray[np.float64])
        assert s[0] == 1.5
        assert isinstance(s[1], np.float64)

    def test_superclass(self) -> None:
        """Should narrow to a superclass of the sample type"""
        s = telem.Series([1.5]).as_(float)
        assert isinstance(s[0], np.float64)

    def test_same_series(self) -> None:
        """Should return the same series"""
        s = telem.Series([1, 2], data_type=telem.DataType.INT32)
        assert s.as_(np.int32) is s

    @pytest.mark.parametrize(
        ("series", "sample_type", "expected"),
        [
            (telem.Series([True]), np.bool_, np.True_),
            (telem.Series(["a"]), str, "a"),
            (telem.Series([uuid.UUID(int=1)]), uuid.UUID, uuid.UUID(int=1)),
            (telem.Series([telem.TimeStamp(5)]), telem.TimeStamp, telem.TimeStamp(5)),
            (
                telem.Series(b"\x01\x00\x00\x00x", data_type=telem.DataType.BYTES),
                bytes,
                b"x",
            ),
        ],
        ids=["bool", "string", "uuid", "timestamp", "bytes"],
    )
    def test_sample_types(
        self,
        series: telem.Series,
        sample_type: type[telem.SampleValue],
        expected: object,
    ) -> None:
        """Should narrow each data type to the type of its samples"""
        narrowed = series.as_(sample_type)
        assert narrowed[0] == expected
        assert isinstance(narrowed[0], sample_type)

    def test_timestamp_type(self) -> None:
        """Should type the samples of a timestamp series as TimeStamps"""
        s = telem.Series([telem.TimeStamp(5)]).as_(telem.TimeStamp)
        assert_type(s[0], telem.TimeStamp)

    @pytest.mark.parametrize(
        ("series", "sample_type"),
        [
            (telem.Series([1.5]), np.float32),
            (telem.Series([1]), float),
            (telem.Series([telem.TimeStamp(5)]), np.int64),
            (telem.Series([{"a": 1}]), dict),
            (telem.Series(b"", data_type=telem.DataType.UNKNOWN), np.float64),
        ],
        ids=["float-width", "python-float", "timestamp-int", "json", "unknown"],
    )
    def test_mismatch(
        self, series: telem.Series, sample_type: type[telem.SampleValue]
    ) -> None:
        """Should raise a TypeError when the data type does not match"""
        with pytest.raises(TypeError, match="does not hold samples"):
            series.as_(sample_type)

    def test_multi_series(self) -> None:
        """Should narrow every series of a MultiSeries"""
        ms = telem.MultiSeries([telem.Series(["a"]), telem.Series(["b"])]).as_(str)
        assert_type(ms, telem.MultiSeries[str])
        assert_type(ms[0], str)
        assert list(ms) == ["a", "b"]

    def test_multi_series_empty(self) -> None:
        """Should narrow an empty MultiSeries to any sample type"""
        assert len(telem.MultiSeries([]).as_(str)) == 0

    def test_multi_series_mismatch(self) -> None:
        """Should raise a TypeError when the data type does not match"""
        with pytest.raises(TypeError, match="does not hold samples"):
            telem.MultiSeries([telem.Series([1])]).as_(str)

    def test_default(self) -> None:
        """Should type an unnarrowed series with the SampleValue union"""
        s = telem.Series([1])
        assert_type(s, telem.Series[telem.SampleValue])
        assert_type(s[0], telem.SampleValue)


@pytest.mark.telem
@pytest.mark.series
class TestSeriesSamples:
    def test_timestamp(self) -> None:
        """Should return TimeStamps from a timestamp series"""
        s = telem.Series([telem.TimeStamp(1), telem.TimeStamp(2)])
        assert isinstance(s[-1], telem.TimeStamp)
        assert s[-1] == 2
        assert all(isinstance(v, telem.TimeStamp) for v in s)

    def test_json_any_value(self) -> None:
        """Should return any JSON value from a JSON series"""
        data = b"".join(
            len(p).to_bytes(4, "little") + p for p in [b"[1, 2]", b"null", b"3"]
        )
        s = telem.Series(data, data_type=telem.DataType.JSON)
        assert list(s) == [[1, 2], None, 3]

    def test_bytes(self) -> None:
        """Should index and iterate a bytes series"""
        s = telem.Series(
            b"\x01\x00\x00\x00a\x02\x00\x00\x00bc", data_type=telem.DataType.BYTES
        )
        assert s[-1] == b"bc"
        assert list(s) == [b"a", b"bc"]

    def test_uuid_negative_index(self) -> None:
        """Should index a UUID series from the end"""
        s = telem.Series([uuid.UUID(int=1), uuid.UUID(int=2)])
        assert s[-1] == uuid.UUID(int=2)

    @pytest.mark.parametrize("index", [2, -3], ids=["past-end", "before-start"])
    def test_uuid_index_out_of_bounds(self, index: int) -> None:
        """Should raise an IndexError for an index outside a UUID series"""
        s = telem.Series([uuid.UUID(int=1), uuid.UUID(int=2)])
        with pytest.raises(IndexError):
            s[index]

    def test_unknown(self) -> None:
        """Should raise a ValueError when reading a series of unknown data type"""
        s = telem.Series(b"", data_type=telem.DataType.UNKNOWN)
        with pytest.raises(ValueError, match="unknown data type"):
            s[0]

    def test_repr(self) -> None:
        """Should name the data type and length in the representation"""
        assert "len=3" in repr(_series())


_INT64 = st.integers(min_value=-(2**63), max_value=2**63 - 1)
_FLOAT = st.floats(allow_nan=False, allow_infinity=False)
_TEXT = st.text(max_size=20)
_JSON = st.dictionaries(_TEXT, st.integers() | _TEXT | st.booleans(), max_size=4)


@pytest.mark.telem
@pytest.mark.series
class TestSeriesProperties:
    @given(st.lists(_INT64, min_size=1, max_size=50))
    def test_int_round_trip(self, values: list[int]) -> None:
        """Should round-trip a list of int64 values"""
        s = telem.Series(values)
        assert s.data_type == telem.DataType.INT64
        assert np.asarray(s).tolist() == values

    @given(st.lists(_FLOAT, min_size=1, max_size=50))
    def test_float_round_trip(self, values: list[float]) -> None:
        """Should round-trip a list of float64 values"""
        assert np.asarray(telem.Series(values)).tolist() == values

    @given(st.lists(st.integers(-(2**31), 2**31) | _FLOAT, min_size=1, max_size=50))
    def test_mixed_numbers_keep_values(self, values: list[int | float]) -> None:
        """Should keep every value of a mixed int and float list"""
        s = telem.Series(values)
        assert np.asarray(s).tolist() == [float(v) for v in values]

    @given(
        st.lists(
            st.integers(0, 2**63 - 1) | st.integers(0, 2**64 - 1).map(np.uint64),
            min_size=1,
            max_size=50,
        )
    )
    def test_mixed_int_kinds_round_trip(self, values: list[Any]) -> None:
        """Should round-trip mixed int64 and uint64 values exactly"""
        assert np.asarray(telem.Series(values)).tolist() == [int(v) for v in values]

    @given(st.lists(_TEXT, min_size=1, max_size=20))
    def test_string_round_trip(self, values: list[str]) -> None:
        """Should round-trip a list of strings"""
        assert list(telem.Series(values)) == values

    @given(st.lists(_JSON, min_size=1, max_size=10))
    def test_json_round_trip(self, values: list[dict[str, Any]]) -> None:
        """Should round-trip a list of JSON objects"""
        assert list(telem.Series(values)) == values

    @given(st.lists(st.uuids(), min_size=1, max_size=20))
    def test_uuid_round_trip(self, values: list[uuid.UUID]) -> None:
        """Should round-trip a list of UUIDs"""
        assert list(telem.Series(values)) == values

    @given(st.lists(_INT64, min_size=1, max_size=20), st.integers(0, 2**32 - 1))
    def test_alignment_survives_copy_and_cast(
        self, values: list[int], align: int
    ) -> None:
        """Should keep the alignment through a copy and a cast"""
        s = telem.Series(values, alignment=align)
        assert telem.Series(s).alignment == align
        assert s.astype(telem.DataType.FLOAT64).alignment == align

    @given(st.lists(_INT64, max_size=20), st.integers(0, 2**64 - 21))
    def test_alignment_bounds_exact(self, values: list[int], align: int) -> None:
        """Should keep every bit of a 64-bit alignment in the bounds"""
        bounds = telem.Series(
            values, data_type="int64", alignment=align
        ).alignment_bounds
        assert bounds.lower == align
        assert bounds.upper == align + len(values)

    @given(st.lists(_INT64, min_size=1, max_size=20))
    def test_datetime64_round_trip(self, values: list[int]) -> None:
        """Should round-trip a datetime64[ns] array as timestamps"""
        s = telem.Series(np.array(values, dtype="datetime64[ns]"))
        assert s.data_type == telem.DataType.TIMESTAMP
        assert np.asarray(s).tolist() == values
