#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


import pytest
from pydantic import BaseModel, TypeAdapter, ValidationError

from x.color import Color


class TestColor:
    @pytest.mark.parametrize(
        ("value", "expected"),
        [
            ("#ff0000", (255, 0, 0, 255)),
            ("00ff00", (0, 255, 0, 255)),
            ("#FF000080", (255, 0, 0, 128)),
            ("#00000000", (0, 0, 0, 0)),
            ("#f00", (255, 0, 0, 255)),
            ("#f008", (255, 0, 0, 136)),
            ("  #ff0000  ", (255, 0, 0, 255)),
            ("rgb(255, 0, 0)", (255, 0, 0, 255)),
            ("rgba(100, 200, 50, 0.5)", (100, 200, 50, 128)),
            ("rgb(100, 200, 50, 0.5)", (100, 200, 50, 128)),
            ("rgba(1, 2, 3)", (1, 2, 3, 255)),
            ("rgb(1 2 3)", (1, 2, 3, 255)),
            ("rgb(1 2 3 / 0.3)", (1, 2, 3, 77)),
            ("rgb(1 2 3 / 70%)", (1, 2, 3, 179)),
            ("rgb(100%, 0%, 50%)", (255, 0, 128, 255)),
            ("RGB(255, 0, 0)", (255, 0, 0, 255)),
            ("hsl(0, 100%, 50%)", (255, 0, 0, 255)),
            ("hsl(120 100 50)", (0, 255, 0, 255)),
            ("hsla(240, 100%, 50%, 0.5)", (0, 0, 255, 128)),
            ("hsl(240deg 100% 50% / 25%)", (0, 0, 255, 64)),
            ("hsl(360, 100%, 50%)", (255, 0, 0, 255)),
            ("hsl(-120, 100%, 50%)", (0, 0, 255, 255)),
            ("hsl(0, 0%, 100%)", (255, 255, 255, 255)),
            ([255, 0, 0], (255, 0, 0, 255)),
            ([255, 0, 0, 1], (255, 0, 0, 1)),
            ((255, 0, 0, 0), (255, 0, 0, 0)),
            ({"r": 255, "g": 128, "b": 0, "a": 64}, (255, 128, 0, 64)),
            ({"r": 255, "g": 128, "b": 0}, (255, 128, 0, 255)),
        ],
    )
    def test_parse(self, value: object, expected: tuple[int, int, int, int]) -> None:
        c = Color(value)
        assert (c.r, c.g, c.b, c.a) == expected

    @pytest.mark.parametrize(
        "value",
        [
            "",
            "#",
            "#ff",
            "#12345",
            "#1234567",
            "#gggggg",
            "red",
            "transparent",
            "rgb(1, 2)",
            "rgb(1, 2, 3, 4, 5)",
            "rgb(256, 0, 0)",
            "rgb(-1, 0, 0)",
            "rgb(101%, 0%, 0%)",
            "rgba(1, 2, 3, 1.5)",
            "rgba(1, 2, 3, -0.1)",
            "rgb(1 2 3 /)",
            "rgb(1, 2, 3 / 0.5)",
            "hsl(0, 101%, 50%)",
            "hsl(0, 100%, -1%)",
            "hsl(x, 100%, 50%)",
            "cmyk(1, 2, 3, 4)",
            [1, 2],
            [255, 0, 0, 0.5],
            [1.7, 0, 0],
            [300, 0, 0],
            {"rgba255": [5, 5, 5, 255]},
            {"r": 1, "g": 2, "b": 3, "a": 0.5},
        ],
    )
    def test_rejects(self, value: object) -> None:
        with pytest.raises(ValidationError):
            Color(value)

    def test_default_is_opaque_black(self) -> None:
        assert Color() == Color(r=0, g=0, b=0, a=255)

    def test_keyword_alpha_defaults_to_opaque(self) -> None:
        assert Color(r=255, g=0, b=0).a == 255

    @pytest.mark.parametrize("a", [0.5, 1.0, "1", -1, 256])
    def test_rejects_keyword_alpha(self, a: object) -> None:
        with pytest.raises(ValidationError):
            Color(r=255, g=0, b=0, a=a)

    @pytest.mark.parametrize(
        ("color", "expected"),
        [
            (Color(r=255, g=0, b=1), "#ff0001"),
            (Color(r=1, g=2, b=3, a=4), "#01020304"),
            (Color(r=0, g=0, b=0, a=0), "#00000000"),
            (Color(r=10, g=11, b=12, a=254), "#0a0b0cfe"),
        ],
    )
    def test_hex(self, color: Color, expected: str) -> None:
        assert color.hex() == expected
        assert TypeAdapter(Color).dump_python(color) == expected
        assert color.model_dump_json() == f'"{expected}"'

    def test_serializes_nested_color_to_hex(self) -> None:
        class Holder(BaseModel):
            color: Color

        assert Holder(color=Color("#ff0000")).model_dump() == {"color": "#ff0000"}

    @pytest.mark.parametrize("a", range(256))
    def test_round_trip_every_alpha(self, a: int) -> None:
        original = Color(r=100, g=200, b=50, a=a)
        assert Color(original.model_dump()) == original
        assert Color.model_validate_json(original.model_dump_json()) == original

    @pytest.mark.parametrize(
        "other", ["#ff0000", "rgb(255, 0, 0)", [255, 0, 0], (255, 0, 0, 255)]
    )
    def test_eq(self, other: object) -> None:
        assert Color(r=255, g=0, b=0) == other

    def test_neq(self) -> None:
        assert Color(r=255, g=0, b=0) != "#00ff00"
        assert Color(r=255, g=0, b=0) != 42
