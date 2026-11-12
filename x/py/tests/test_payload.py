#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import math

import pytest
from pydantic import ValidationError

from x import payload


class TestFloat:
    @pytest.mark.parametrize(
        ("value", "expected"),
        [(1.5, 1.5), ("Infinity", math.inf), ("-Infinity", -math.inf)],
    )
    def test_decode(self, value: float | str, expected: float) -> None:
        """Should decode a float from a number or its text."""
        assert payload.Float(value) == expected

    def test_decode_nan(self) -> None:
        """Should decode NaN from its text."""
        assert math.isnan(payload.Float("NaN"))

    @pytest.mark.parametrize("value", ["inf", "1.5", True, None, [], 10**400])
    def test_decode_invalid(self, value: object) -> None:
        """Should reject a value that is not a float."""
        with pytest.raises(ValueError, match="cannot parse a float"):
            payload.Float(value)

    def test_round_trip_non_finite_fill(self) -> None:
        """Should write a non-finite fill as its text and read it back."""
        read = payload.FloatRead.model_validate({"fill": "-Infinity"})
        assert read.fill == -math.inf
        dumped = read.model_dump_json()
        assert '"fill":"-Infinity"' in dumped
        assert payload.FloatRead.model_validate_json(dumped).fill == -math.inf

    def test_model_rejects_none(self) -> None:
        """Should raise a validation error for a value that is not a float."""
        with pytest.raises(ValidationError):
            payload.FloatEntry.model_validate({"label": "a", "value": None})

    def test_python_dump_keeps_float(self) -> None:
        """Should keep a float in a Python-mode dump."""
        dumped = payload.FloatRead.model_validate({"fill": "NaN"}).model_dump()
        assert math.isnan(dumped["fill"])

    def test_finite_fill_is_a_number(self) -> None:
        """Should write a finite fill as a JSON number."""
        read = payload.FloatRead(fill=payload.Float(2.5))
        assert '"fill":2.5' in read.model_dump_json()


class TestInteger:
    def test_round_trip(self) -> None:
        """Should write every digit of an integer as a string and read it back."""
        read = payload.IntegerRead.model_validate({"fill": "-9223372036854775808"})
        assert read.fill == -(2**63)
        dumped = read.model_dump_json()
        assert '"fill":"-9223372036854775808"' in dumped
        assert payload.IntegerRead.model_validate_json(dumped).fill == -(2**63)

    def test_python_dump_keeps_int(self) -> None:
        """Should keep an int in a Python-mode dump."""
        read = payload.IntegerRead.model_validate({"fill": "-12"})
        assert read.model_dump()["fill"] == -12

    @pytest.mark.parametrize(
        "value", ["9223372036854775808", "1.5", "1_000", "+1", True]
    )
    def test_invalid(self, value: str | bool) -> None:
        """Should reject a value that is not a signed 64-bit integer."""
        with pytest.raises(ValidationError):
            payload.IntegerRead.model_validate({"fill": value})


class TestUnsigned:
    def test_round_trip(self) -> None:
        """Should write every digit of an unsigned integer as a string."""
        read = payload.UnsignedRead.model_validate({"fill": 2**64 - 1})
        assert '"fill":"18446744073709551615"' in read.model_dump_json()

    @pytest.mark.parametrize("value", ["-1", "-0", "+1", "18446744073709551616"])
    def test_invalid(self, value: str) -> None:
        """Should reject a value that is not an unsigned 64-bit integer."""
        with pytest.raises(ValidationError):
            payload.UnsignedRead.model_validate({"fill": value})
