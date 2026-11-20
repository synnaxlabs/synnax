#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import math
import re

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

    def test_round_trip_non_finite_value(self) -> None:
        """Should write a non-finite value as its text and read it back."""
        entry = payload.FloatEntry.model_validate(
            {"label": "LOW", "value": "-Infinity"}
        )
        assert entry.value == -math.inf
        dumped = entry.model_dump_json()
        assert '"value":"-Infinity"' in dumped
        assert payload.FloatEntry.model_validate_json(dumped).value == -math.inf

    def test_model_rejects_none(self) -> None:
        """Should raise a validation error for a value that is not a float."""
        with pytest.raises(ValidationError):
            payload.FloatEntry.model_validate({"label": "a", "value": None})

    def test_python_dump_keeps_float(self) -> None:
        """Should keep a float in a Python-mode dump."""
        entry = payload.FloatEntry.model_validate({"label": "OFF", "value": "NaN"})
        assert math.isnan(entry.model_dump()["value"])

    def test_finite_value_is_a_number(self) -> None:
        """Should write a finite value as a JSON number."""
        entry = payload.FloatEntry(label="ON", value=payload.Float(2.5))
        assert '"value":2.5' in entry.model_dump_json()


class TestInteger:
    def test_round_trip(self) -> None:
        """Should write every digit of an integer as a string and read it back."""
        entry = payload.IntegerEntry.model_validate(
            {"label": "LOW", "value": "-9223372036854775808"}
        )
        assert entry.value == -(2**63)
        dumped = entry.model_dump_json()
        assert '"value":"-9223372036854775808"' in dumped
        assert payload.IntegerEntry.model_validate_json(dumped).value == -(2**63)

    def test_python_dump_keeps_int(self) -> None:
        """Should keep an int in a Python-mode dump."""
        entry = payload.IntegerEntry.model_validate({"label": "LOW", "value": "-12"})
        assert entry.model_dump()["value"] == -12

    @pytest.mark.parametrize(
        ("value", "message"),
        [
            ("9223372036854775808", "9223372036854775808 is out of range for Integer"),
            ("1.5", "cannot parse Integer from '1.5'"),
            ("1_000", "cannot parse Integer from '1_000'"),
            ("+1", "cannot parse Integer from '+1'"),
            (True, "cannot parse Integer from True"),
        ],
    )
    def test_invalid(self, value: str | bool, message: str) -> None:
        """Should reject a value that is not a signed 64-bit integer."""
        with pytest.raises(ValidationError, match=re.escape(message)):
            payload.IntegerEntry.model_validate({"label": "L", "value": value})


class TestUnsigned:
    def test_round_trip(self) -> None:
        """Should write every digit of an unsigned integer as a string."""
        entry = payload.UnsignedEntry.model_validate(
            {"label": "MAX", "value": 2**64 - 1}
        )
        assert '"value":"18446744073709551615"' in entry.model_dump_json()

    @pytest.mark.parametrize(
        ("value", "message"),
        [
            ("-1", "cannot parse Unsigned from '-1'"),
            ("-0", "cannot parse Unsigned from '-0'"),
            ("+1", "cannot parse Unsigned from '+1'"),
            (
                "18446744073709551616",
                "18446744073709551616 is out of range for Unsigned",
            ),
        ],
    )
    def test_invalid(self, value: str, message: str) -> None:
        """Should reject a value that is not an unsigned 64-bit integer."""
        with pytest.raises(ValidationError, match=re.escape(message)):
            payload.UnsignedEntry.model_validate({"label": "L", "value": value})
