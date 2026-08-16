#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

from x.color import Color


class TestColor:
    def test_from_hex_6_char(self) -> None:
        c = Color("#ff0000")
        assert c.r == 255
        assert c.g == 0
        assert c.b == 0
        assert c.a == 1.0

    def test_from_hex_8_char(self) -> None:
        c = Color("#ff000080")
        assert c.r == 255
        assert c.g == 0
        assert c.b == 0
        assert abs(c.a - 128 / 255) < 0.01

    def test_from_hex_without_hash(self) -> None:
        c = Color("00ff00")
        assert c.r == 0
        assert c.g == 255
        assert c.b == 0

    def test_from_array_rgb(self) -> None:
        c = Color([255, 0, 0])
        assert c.r == 255
        assert c.a == 1.0

    def test_from_array_rgba(self) -> None:
        c = Color([255, 0, 0, 0.5])
        assert c.r == 255
        assert c.a == 0.5

    def test_from_dict(self) -> None:
        c = Color({"r": 255, "g": 128, "b": 0, "a": 1.0})
        assert c.r == 255
        assert c.g == 128
        assert c.b == 0
        assert c.a == 1.0

    def test_from_rgb_string(self) -> None:
        c = Color("rgb(255, 0, 0)")
        assert c.r == 255
        assert c.g == 0
        assert c.b == 0
        assert c.a == 1.0

    def test_from_rgba_string(self) -> None:
        c = Color("rgba(100, 200, 50, 0.5)")
        assert c.r == 100
        assert c.g == 200
        assert c.b == 50
        assert c.a == 0.5

    def test_rgb_string_to_hex(self) -> None:
        assert Color("rgb(255, 0, 0)").hex() == "#ff0000"

    def test_hex_output_no_alpha(self) -> None:
        c = Color(r=255, g=0, b=0, a=1.0)
        assert c.hex() == "#ff0000"

    def test_hex_output_with_alpha(self) -> None:
        c = Color(r=255, g=0, b=0, a=0.5)
        h = c.hex()
        assert h.startswith("#ff0000")
        assert len(h) == 9

    def test_is_zero(self) -> None:
        assert Color(r=0, g=0, b=0, a=0).is_zero
        assert not Color().is_zero
        assert not Color(r=1).is_zero
        assert not Color(a=0.5).is_zero

    def test_round_trip_dict(self) -> None:
        original = Color(r=100, g=200, b=50, a=0.75)
        d = original.model_dump()
        restored = Color(d)
        assert restored == original

    def test_eq_hex(self) -> None:
        assert Color(r=255, g=0, b=0, a=1.0) == "#ff0000"

    def test_eq_array_rgb(self) -> None:
        assert Color(r=255, g=0, b=0, a=1.0) == [255, 0, 0]

    def test_eq_array_rgba(self) -> None:
        assert Color(r=255, g=0, b=0, a=0.5) == [255, 0, 0, 0.5]

    def test_eq_tuple(self) -> None:
        assert Color(r=255, g=0, b=0, a=1.0) == (255, 0, 0)

    def test_neq_hex(self) -> None:
        assert Color(r=255, g=0, b=0, a=1.0) != "#00ff00"

    def test_eq_invalid_returns_not_implemented(self) -> None:
        assert Color(r=255, g=0, b=0, a=1.0) != 42

    def test_invalid_hex(self) -> None:
        with pytest.raises(ValueError, match="Invalid hex color"):
            Color("#xyz")

    def test_invalid_array_length(self) -> None:
        with pytest.raises(ValueError, match="Invalid color array length: 2"):
            Color([1, 2])

    def test_default_alpha_is_opaque(self) -> None:
        c = Color(r=255, g=0, b=0)
        assert c.a == 1.0

    def test_legacy_alpha_255_in_array(self) -> None:
        c = Color([28, 28, 28, 255])
        assert c == Color(r=28, g=28, b=28, a=1.0)

    def test_legacy_alpha_mid_range_in_array(self) -> None:
        c = Color([255, 0, 0, 128])
        assert abs(c.a - 128 / 255) < 0.001

    def test_legacy_alpha_just_above_1_clamps_to_opaque(self) -> None:
        c = Color([255, 0, 0, 1.5])
        assert c.a == 1.0

    def test_legacy_alpha_at_clamp_boundary(self) -> None:
        c = Color([255, 0, 0, 2])
        assert c.a == 1.0

    def test_legacy_alpha_just_above_clamp_boundary_divides(self) -> None:
        c = Color([255, 0, 0, 2.01])
        assert abs(c.a - 2.01 / 255) < 0.001

    def test_alpha_above_255_rejected(self) -> None:
        with pytest.raises(ValueError, match="above the 0-255 scale"):
            Color([255, 0, 0, 300])
        with pytest.raises(ValueError, match="above the 0-255 scale"):
            Color({"r": 5, "g": 5, "b": 5, "a": 300})

    def test_alpha_1_stays_opaque(self) -> None:
        c = Color([255, 0, 0, 1])
        assert c.a == 1.0

    def test_fractional_alpha_untouched(self) -> None:
        c = Color([255, 0, 0, 0.25])
        assert c.a == 0.25

    def test_legacy_alpha_255_in_dict(self) -> None:
        c = Color({"r": 5, "g": 5, "b": 5, "a": 255})
        assert c == Color(r=5, g=5, b=5, a=1.0)

    def test_legacy_rgba255_object(self) -> None:
        c = Color({"rgba255": [5, 5, 5, 255]})
        assert c == Color(r=5, g=5, b=5, a=1.0)

    def test_legacy_rgba255_object_with_fractional_alpha(self) -> None:
        c = Color({"rgba255": [122, 44, 38, 0.5]})
        assert c == Color(r=122, g=44, b=38, a=0.5)
