#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pathops
import pytest
from fontTools.svgLib.path import parse_path

from compound_icon import CLEARANCE, Cut, compound

PLUS = "M13 7.75v-2.5h3.25v-3.25h2.5v3.25h3.25v2.5h-3.25v3.25h-2.5v-3.25z"
# The live area of the grid, from (2, 2) to (22, 22).
SQUARE = "M2 2h20v20h-20z"


def parse(d: str) -> pathops.Path:
    path = pathops.Path()
    parse_path(d, path.getPen())
    return pathops.simplify(path)


def grow(path: pathops.Path, by: float) -> pathops.Path:
    band = pathops.Path(path)
    band.stroke(2 * by, pathops.LineCap.ROUND_CAP, pathops.LineJoin.ROUND_JOIN, 4)
    band.convertConicsToQuads()
    return pathops.op(path, band, pathops.PathOp.UNION)


def without_plus(d: str) -> pathops.Path:
    return pathops.op(parse(d), parse(PLUS), pathops.PathOp.DIFFERENCE)


class TestCompound:
    """Tests for building a compound icon from a base and a modifier."""

    def test_should_scale_the_base_from_the_bottom_left_corner(self) -> None:
        base = without_plus(compound(SQUARE, PLUS, "circle"))
        x_min, y_min, x_max, y_max = base.bounds
        assert x_min == pytest.approx(2)
        assert y_min == pytest.approx(4)
        assert x_max == pytest.approx(20)
        assert y_max == pytest.approx(22)

    def test_should_scale_the_modifier_to_the_top_right(self) -> None:
        result = parse(compound(SQUARE, "M0 0h12v12h-12z"))
        assert result.contains((13.1, 2.1))
        assert result.contains((21.9, 10.9))
        assert not result.contains((12.9, 6.5))
        assert not result.contains((17.5, 1.9))

    def test_should_center_a_wider_modifier_in_a_square_as_wide(self) -> None:
        result = parse(compound(SQUARE, "M0 0h12v12h-12z", width=11))
        assert result.contains((11.1, 2.1))
        assert result.contains((21.9, 12.9))
        assert not result.contains((10.9, 7.5))

    def test_should_measure_a_modifier_drawn_with_quadratic_curves(self) -> None:
        result = parse(compound(SQUARE, "M0 0Q12 0 12 12Q0 12 0 0Z", "circle"))
        assert result.contains((17.5, 6.5))

    @pytest.mark.parametrize("cut", ["box", "circle", "outline"])
    def test_should_clear_the_base_around_the_modifier(self, cut: Cut) -> None:
        base = without_plus(compound(SQUARE, PLUS, cut))
        near = pathops.op(
            base, grow(parse(PLUS), CLEARANCE - 0.05), pathops.PathOp.INTERSECTION
        )
        assert near.area == pytest.approx(0, abs=1e-3)

    def test_should_round_the_corners_the_cut_makes(self) -> None:
        result = parse(compound(SQUARE, PLUS))
        assert not result.contains((11.2, 4.05))
        assert result.contains((10.5, 4.75))
        assert result.contains((2.05, 21.95))

    def test_should_give_the_cut_ends_of_a_line_base_round_caps(self) -> None:
        line = "M2 6h20v2h-20z"
        assert parse(compound(line, PLUS)).contains((11.1, 7.8))
        capped = parse(compound(line, PLUS, line_width=2))
        assert not capped.contains((11.1, 7.8))
        assert capped.contains((11.2, 8.5))

    def test_should_drop_slivers_the_cut_leaves(self) -> None:
        result = parse(compound("M10.78 4.22h11.22v0.33h-11.22z", PLUS))
        assert result.area == pytest.approx(parse(PLUS).area, abs=1e-2)

    def test_should_keep_small_shapes_away_from_the_cut(self) -> None:
        assert parse(compound("M3 19h0.5v0.5h-0.5z", PLUS)).contains((3.1, 19.5))

    @pytest.mark.parametrize(
        ("modifier", "width", "line_width", "message"),
        [
            ("", 9, None, "modifier has no width"),
            (PLUS, 0, None, "width must be positive"),
            (PLUS, 9, 0, "line width must be positive"),
        ],
    )
    def test_should_reject_input_it_cannot_place(
        self, modifier: str, width: float, line_width: float | None, message: str
    ) -> None:
        with pytest.raises(ValueError, match=message):
            compound(SQUARE, modifier, width=width, line_width=line_width)
