#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


import pytest
from pydantic import BaseModel, ValidationError

from x.color import Color


class TestColor:
    @pytest.mark.parametrize(
        ("value", "expected"),
        [
            ("#ff0000", (255, 0, 0, 255)),
            ("00ff00", (0, 255, 0, 255)),
            ("#ff000080", (255, 0, 0, 128)),
            ("#00000000", (0, 0, 0, 0)),
            ([255, 0, 0], (255, 0, 0, 255)),
            ([255, 0, 0, 0], (255, 0, 0, 0)),
            ([255, 0, 0, 1], (255, 0, 0, 1)),
            ([255, 0, 0, 128], (255, 0, 0, 128)),
            ((255, 0, 0), (255, 0, 0, 255)),
            ({"r": 255, "g": 128, "b": 0, "a": 64}, (255, 128, 0, 64)),
            ({"r": 255, "g": 128, "b": 0}, (255, 128, 0, 255)),
            ("rgb(255, 0, 0)", (255, 0, 0, 255)),
            ("rgba(100, 200, 50, 0.5)", (100, 200, 50, 128)),
            ("rgba(0, 0, 0, 0.3)", (0, 0, 0, 77)),
            ("rgba(0, 0, 0, 0.7)", (0, 0, 0, 179)),
        ],
    )
    def test_parse(self, value: object, expected: tuple[int, int, int, int]) -> None:
        c = Color(value)
        assert (c.r, c.g, c.b, c.a) == expected

    def test_default_alpha_is_opaque(self) -> None:
        assert Color(r=255, g=0, b=0).a == 255

    def test_default_is_opaque_black(self) -> None:
        assert Color() == Color(r=0, g=0, b=0, a=255)

    def test_keyword_alpha_is_a_byte(self) -> None:
        assert Color(r=255, g=0, b=0, a=1).a == 1

    @pytest.mark.parametrize("a", [0.5, 1.0, "1"])
    def test_non_integer_keyword_alpha_rejected(self, a: object) -> None:
        with pytest.raises(ValidationError):
            Color(r=255, g=0, b=0, a=a)

    def test_rgb_string_to_hex(self) -> None:
        assert Color("rgb(255, 0, 0)").hex() == "#ff0000"

    def test_hex_output_no_alpha(self) -> None:
        assert Color(r=255, g=0, b=0).hex() == "#ff0000"

    def test_hex_output_with_alpha(self) -> None:
        assert Color(r=255, g=0, b=0, a=128).hex() == "#ff000080"

    def test_is_zero(self) -> None:
        assert Color(r=0, g=0, b=0, a=0).is_zero
        assert not Color(r=1, a=0).is_zero

    def test_serializes_to_eight_digit_hex(self) -> None:
        assert Color(r=255, g=0, b=1).model_dump_json() == '"#ff0001ff"'
        assert Color(r=1, g=2, b=3, a=4).model_dump_json() == '"#01020304"'
        assert Color(r=0, g=0, b=0, a=0).model_dump_json() == '"#00000000"'

    def test_serializes_nested_color_to_hex(self) -> None:
        class Holder(BaseModel):
            color: Color

        assert Holder(color=Color("#ff0000")).model_dump() == {"color": "#ff0000ff"}

    @pytest.mark.parametrize("a", range(256))
    def test_round_trip_every_alpha(self, a: int) -> None:
        original = Color(r=100, g=200, b=50, a=a)
        assert Color(original.model_dump()) == original
        assert Color.model_validate_json(original.model_dump_json()) == original

    def test_eq_hex(self) -> None:
        assert Color(r=255, g=0, b=0) == "#ff0000"

    def test_eq_array_rgb(self) -> None:
        assert Color(r=255, g=0, b=0) == [255, 0, 0]

    def test_eq_array_rgba(self) -> None:
        assert Color(r=255, g=0, b=0, a=128) == [255, 0, 0, 128]

    def test_eq_tuple(self) -> None:
        assert Color(r=255, g=0, b=0) == (255, 0, 0)

    def test_neq_hex(self) -> None:
        assert Color(r=255, g=0, b=0) != "#00ff00"

    def test_eq_invalid_returns_not_implemented(self) -> None:
        assert Color(r=255, g=0, b=0) != 42

    @pytest.mark.parametrize("value", ["#xyz", "#12345", "#gggggg"])
    def test_invalid_hex(self, value: str) -> None:
        with pytest.raises(ValueError, match="Invalid hex color"):
            Color(value)

    @pytest.mark.parametrize(
        "value",
        [
            "",
            [255, 0, 0, 0.5],
            {"rgba255": [5, 5, 5, 255]},
            {"r": 1, "g": 2, "b": 3, "a": 0.5},
        ],
    )
    def test_legacy_form_rejected(self, value: object) -> None:
        with pytest.raises(ValidationError):
            Color(value)

    def test_bare_hash(self) -> None:
        with pytest.raises(ValueError, match="Invalid hex color"):
            Color("#")

    def test_invalid_array_length(self) -> None:
        with pytest.raises(ValueError, match="Invalid color array length: 2"):
            Color([1, 2])

    def test_rgba_string_alpha_above_1_rejected(self) -> None:
        with pytest.raises(ValueError, match="rgba alpha must be 0-1"):
            Color("rgba(1, 2, 3, 1.5)")

    @pytest.mark.parametrize(
        "value",
        [
            [300, 0, 0, 1],
            [-1, 0, 0, 1],
            [0, 256, 0],
            [255, 0, 0, 300],
            {"r": 300, "g": 0, "b": 0, "a": 1},
        ],
    )
    def test_channel_outside_range_rejected(self, value: object) -> None:
        with pytest.raises(ValidationError):
            Color(value)

    @pytest.mark.parametrize(
        "value",
        [
            [0, 0, 0, -1],
            {"r": 0, "g": 0, "b": 0, "a": -1},
        ],
    )
    def test_negative_alpha_rejected(self, value: object) -> None:
        with pytest.raises(ValidationError):
            Color(value)

    def test_negative_alpha_rejected_on_construction(self) -> None:
        with pytest.raises(ValidationError):
            Color(r=0, g=0, b=0, a=-0.1)

    def test_channel_outside_range_rejected_on_construction(self) -> None:
        with pytest.raises(ValidationError):
            Color(r=300, g=0, b=0)

    @pytest.mark.parametrize("value", [[1.7, 0, 0, 1], [0, 0, 0, 0.5]])
    def test_fractional_channel_rejected(self, value: object) -> None:
        with pytest.raises(ValueError, match="not a whole number"):
            Color(value)
