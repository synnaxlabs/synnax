#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

from synnax.color import Color


class TestColor:
    def test_from_hex_6_char(self):
        c = Color("#ff0000")
        assert c.r == 255
        assert c.g == 0
        assert c.b == 0
        assert c.a == 1.0

    def test_from_hex_8_char(self):
        c = Color("#ff000080")
        assert c.r == 255
        assert c.g == 0
        assert c.b == 0
        assert abs(c.a - 128 / 255) < 0.01

    def test_from_hex_without_hash(self):
        c = Color("00ff00")
        assert c.r == 0
        assert c.g == 255
        assert c.b == 0

    def test_from_array_rgb(self):
        c = Color([255, 0, 0])
        assert c.r == 255
        assert c.a == 1.0

    def test_from_array_rgba(self):
        c = Color([255, 0, 0, 0.5])
        assert c.r == 255
        assert c.a == 0.5

    def test_from_dict(self):
        c = Color({"r": 255, "g": 128, "b": 0, "a": 1.0})
        assert c.r == 255
        assert c.g == 128
        assert c.b == 0
        assert c.a == 1.0

    def test_hex_output_no_alpha(self):
        c = Color(r=255, g=0, b=0, a=1.0)
        assert c.hex() == "#ff0000"

    def test_hex_output_with_alpha(self):
        c = Color(r=255, g=0, b=0, a=0.5)
        h = c.hex()
        assert h.startswith("#ff0000")
        assert len(h) == 9

    def test_is_zero(self):
        assert Color(r=0, g=0, b=0, a=0).is_zero
        assert not Color().is_zero
        assert not Color(r=1).is_zero
        assert not Color(a=0.5).is_zero

    def test_round_trip_dict(self):
        original = Color(r=100, g=200, b=50, a=0.75)
        d = original.model_dump()
        restored = Color(d)
        assert restored == original

    def test_eq_hex(self):
        assert Color(r=255, g=0, b=0, a=1.0) == "#ff0000"

    def test_eq_array_rgb(self):
        assert Color(r=255, g=0, b=0, a=1.0) == [255, 0, 0]

    def test_eq_array_rgba(self):
        assert Color(r=255, g=0, b=0, a=0.5) == [255, 0, 0, 0.5]

    def test_eq_tuple(self):
        assert Color(r=255, g=0, b=0, a=1.0) == (255, 0, 0)

    def test_neq_hex(self):
        assert Color(r=255, g=0, b=0, a=1.0) != "#00ff00"

    def test_eq_invalid_returns_not_implemented(self):
        assert Color(r=255, g=0, b=0, a=1.0) != 42

    def test_invalid_hex(self):
        with pytest.raises(ValueError):
            Color("#xyz")

    def test_invalid_array_length(self):
        with pytest.raises(ValueError):
            Color([1, 2])

    def test_default_alpha_is_opaque(self):
        c = Color(r=255, g=0, b=0)
        assert c.a == 1.0
