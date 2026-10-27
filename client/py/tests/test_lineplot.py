#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import uuid4

import pytest
from pydantic import ValidationError

from synnax import lineplot


class TestLinePlot:
    def test_defaults(self) -> None:
        """Should create a plot with default axes from only a name."""
        p = lineplot.LinePlot(name="Pressure")
        assert p.axes.y1.key == "y1"
        assert p.axes.y1.label_direction == "y"
        assert p.axes.x1.label_direction == "x"
        assert p.legend.position.root.x == "left"
        assert p.title.level == "h4"
        assert p.channels.x1 == 0
        assert p.ranges.custom is None

    def test_round_trip(self) -> None:
        """Should parse lines, rules, and ranges and dump them back unchanged."""
        p = lineplot.LinePlot.model_validate(
            {
                "name": "Pressure",
                "channels": {"x1": 1, "y1": [2, 3]},
                "lines": [{"key": "2", "color": "#ff0000", "detail": "high"}],
                "rules": [
                    {"key": "max", "label": "Max", "axis": "y1", "position": 100}
                ],
                "ranges": {"x1": ["r1"], "custom": {"variant": "dynamic", "span": 5}},
            }
        )
        assert p.channels.y1 == [2, 3]
        assert p.lines[0].detail == "high"
        assert p.rules[0].line_width == 1
        assert isinstance(p.ranges.custom, lineplot.DynamicCustomRange)
        assert lineplot.LinePlot.model_validate_json(p.model_dump_json()) == p

    def test_static_custom_range(self) -> None:
        """Should select a static custom range by its variant."""
        r = lineplot.Ranges.model_validate(
            {"custom": {"variant": "static", "start": 1, "end": 2}}
        )
        assert isinstance(r.custom, lineplot.StaticCustomRange)

    def test_invalid_axis(self) -> None:
        """Should reject a rule on an unknown axis."""
        with pytest.raises(ValidationError):
            lineplot.Rule(key="a", label="A", axis="z1", position=0)  # type: ignore[arg-type]

    def test_ontology_id(self) -> None:
        """Should build the ontology ID of a line plot."""
        key = uuid4()
        assert lineplot.ontology_id(key).type == lineplot.ONTOLOGY_TYPE.type
        assert lineplot.ontology_id(key).key == str(key)
        p = lineplot.LinePlot(name="Pressure")
        assert hash(p) == hash(p.key)
