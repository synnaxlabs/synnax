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

    def test_missing_corner(self) -> None:
        """Should reject a radius without every corner."""
        with pytest.raises(ValidationError):
            border.Radius.model_validate({"top_left": {"x": 1, "y": 2}})
