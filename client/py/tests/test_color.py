#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

from synnax.color import Color, construct, is_crude


@pytest.mark.color
class TestColor:
    """Tests for Color parsing and conversion."""

    class TestFromHex:
        """Tests for hex string parsing."""

        def test_six_char_hex(self):
            c = Color.model_validate("#ff0000")
            assert c.r == 255
            assert c.g == 0
            assert c.b == 0
            assert c.a == 1.0

        def test_eight_char_hex_with_alpha(self):
            c = Color.model_validate("#00ff0080")
            assert c.r == 0
            assert c.g == 255
            assert c.b == 0
            assert abs(c.a - 0.502) < 0.01

        def test_empty_string_gives_zero(self):
            c = Color.model_validate("#")
            assert c.r == 0 and c.g == 0 and c.b == 0 and c.a == 0.0

        def test_rejects_invalid_length(self):
            with pytest.raises(ValueError):
                Color.model_validate("#fff")

    class TestFromDict:
        """Tests for dict input."""

        def test_dict_input(self):
            c = Color.model_validate({"r": 128, "g": 64, "b": 32, "a": 0.5})
            assert c.r == 128
            assert c.g == 64
            assert c.b == 32
            assert c.a == 0.5

        def test_zero_color_dict(self):
            c = Color.model_validate({"r": 0, "g": 0, "b": 0, "a": 0.0})
            assert c.r == 0 and c.g == 0 and c.b == 0 and c.a == 0.0

        def test_keyword_args(self):
            c = Color(r=255, g=128, b=64, a=0.5)
            assert c.r == 255
            assert c.g == 128
            assert c.b == 64
            assert c.a == 0.5

    class TestFromArray:
        """Tests for list/tuple input."""

        def test_list_input(self):
            c = Color.model_validate([255, 128, 64, 0.5])
            assert c.r == 255
            assert c.g == 128
            assert c.b == 64
            assert c.a == 0.5

        def test_tuple_input(self):
            c = Color.model_validate((100, 50, 25, 0.75))
            assert c.r == 100
            assert c.g == 50
            assert c.b == 25
            assert c.a == 0.75

    class TestHex:
        """Tests for hex output."""

        def test_full_alpha_six_char(self):
            c = Color(r=255, g=128, b=64, a=1.0)
            assert c.hex(include_alpha=False) == "#ff8040"

        def test_partial_alpha_eight_char(self):
            c = Color(r=255, g=0, b=0, a=0.5)
            hex_str = c.hex()
            assert hex_str.startswith("#ff0000")
            assert len(hex_str) == 9

    class TestIsZero:
        """Tests for zero detection."""

        def test_zero_color(self):
            c = Color(r=0, g=0, b=0, a=0.0)
            assert c.is_zero() is True

        def test_non_zero_color(self):
            c = Color(r=1, g=0, b=0, a=0.0)
            assert c.is_zero() is False

    class TestEquality:
        """Tests for equality comparison."""

        def test_color_equals_color(self):
            c1 = Color.model_validate("#ff0000")
            c2 = Color(r=255, g=0, b=0, a=1.0)
            assert c1 == c2

        def test_color_equals_hex_string(self):
            c = Color(r=255, g=0, b=0, a=1.0)
            assert c == "#ff0000ff"

        def test_color_equals_tuple(self):
            c = Color.model_validate("#ff0000")
            assert c == (255, 0, 0, 1.0)

    class TestHash:
        """Tests for hash consistency."""

        def test_equal_colors_same_hash(self):
            c1 = Color.model_validate("#ff0000")
            c2 = Color(r=255, g=0, b=0, a=1.0)
            assert hash(c1) == hash(c2)

        def test_can_use_in_set(self):
            s = {Color.model_validate("#ff0000"), Color.model_validate("#00ff00")}
            assert len(s) == 2
            s.add(Color(r=255, g=0, b=0, a=1.0))
            assert len(s) == 2


@pytest.mark.color
class TestCrude:
    """Tests for Crude type coercion."""

    class TestIsCrude:
        """Tests for is_crude helper function."""

        def test_hex_string_is_crude(self):
            assert is_crude("#ff0000") is True

        def test_rgb_tuple_is_crude(self):
            assert is_crude((255, 0, 0)) is True

        def test_rgba_tuple_is_crude(self):
            assert is_crude((255, 0, 0, 1.0)) is True

        def test_dict_is_crude(self):
            assert is_crude({"r": 255, "g": 0, "b": 0, "a": 1.0}) is True

        def test_color_is_crude(self):
            c = Color(r=255, g=0, b=0, a=1.0)
            assert is_crude(c) is True

        def test_invalid_string_not_crude(self):
            assert is_crude("not a color") is False

        def test_invalid_tuple_length_not_crude(self):
            assert is_crude((255, 0)) is False

        def test_none_not_crude(self):
            assert is_crude(None) is False

    class TestConstruct:
        """Tests for construct helper function."""

        def test_construct_from_hex(self):
            c = construct("#ff0000")
            assert c.r == 255 and c.g == 0 and c.b == 0

        def test_construct_from_rgb_tuple(self):
            c = construct((128, 64, 32))
            assert c.r == 128
            assert c.g == 64
            assert c.b == 32
            assert c.a == 1.0

        def test_construct_from_rgba_tuple(self):
            c = construct((128, 64, 32, 0.5))
            assert c.r == 128
            assert c.g == 64
            assert c.b == 32
            assert c.a == 0.5

        def test_construct_from_dict(self):
            c = construct({"r": 100, "g": 50, "b": 25, "a": 0.75})
            assert c.r == 100
            assert c.g == 50
            assert c.b == 25
            assert c.a == 0.75

        def test_construct_from_color(self):
            original = Color(r=255, g=128, b=64, a=0.5)
            c = construct(original)
            assert c == original

    class TestRGBSupport:
        """Tests for RGB tuple support (3-element without alpha)."""

        def test_rgb_tuple_defaults_alpha(self):
            c = Color.model_validate((255, 128, 64))
            assert c.r == 255
            assert c.g == 128
            assert c.b == 64
            assert c.a == 1.0

        def test_rgb_list_defaults_alpha(self):
            c = Color.model_validate([100, 50, 25])
            assert c.r == 100
            assert c.g == 50
            assert c.b == 25
            assert c.a == 1.0

        def test_rgb_tuple_equals_rgba_with_full_alpha(self):
            c1 = Color.model_validate((255, 0, 0))
            c2 = Color.model_validate((255, 0, 0, 1.0))
            assert c1 == c2
