#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest
from pydantic import ValidationError

from x import border, spatial


class TestRadius:
    def test_round_trip(self) -> None:
        """Should parse each corner and dump them back unchanged."""
        r = border.Radius.model_validate(
            {
                "top_left": {"x": 1, "y": 2},
                "top_right": {"x": 3, "y": 4},
                "bottom_left": {"x": 5, "y": 6},
                "bottom_right": {"x": 7, "y": 8},
            }
        )
        assert r.bottom_right == spatial.XY(x=7, y=8)
        assert border.Radius.model_validate_json(r.model_dump_json()) == r

    def test_number(self) -> None:
        """Should expand a bare number to every corner."""
        r = border.Radius.model_validate(4)
        corner = spatial.XY(x=4, y=4)
        assert r == border.Radius(
            top_left=corner,
            top_right=corner,
            bottom_left=corner,
            bottom_right=corner,
        )

    def test_number_corners_independent(self) -> None:
        """Should give each corner its own pair when expanding a bare number."""
        r = border.Radius.model_validate(4)
        r.top_left.x = 9
        assert r.top_right == spatial.XY(x=4, y=4)

    def test_pair(self) -> None:
        """Should expand a single pair to every corner."""
        r = border.Radius.model_validate({"x": 4, "y": 8})
        assert r.top_left == r.bottom_right == spatial.XY(x=4, y=8)

    def test_corner_numbers(self) -> None:
        """Should expand per-corner numbers and zero a missing corner."""
        r = border.Radius.model_validate({"top_left": 2, "top_right": {"x": 1, "y": 3}})
        assert r.top_left == spatial.XY(x=2, y=2)
        assert r.top_right == spatial.XY(x=1, y=3)
        assert r.bottom_left == spatial.XY(x=0, y=0)

    def test_invalid(self) -> None:
        """Should reject a radius that is not a number or an object."""
        with pytest.raises(ValidationError):
            border.Radius.model_validate("8px")
