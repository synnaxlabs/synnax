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

import synnax as sy
from synnax import lineplot
from synnax.exceptions import NotFoundError


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


@pytest.fixture
def project(client: sy.Synnax) -> sy.Project:
    return client.projects.create(name="Line Plot Project")


@pytest.mark.lineplot
class TestClient:
    def test_create(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should create a line plot from a name."""
        p = client.line_plots.create(project.key, name="Plot")
        assert p.name == "Plot"
        assert client.line_plots.retrieve(key=p.key).name == "Plot"

    def test_create_from_object(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should create a line plot from a LinePlot object and keep its fields."""
        p = client.line_plots.create(
            project.key,
            sy.LinePlot(
                name="Configured",
                title=sy.lineplot.Title(visible=True),
                rules=[
                    sy.lineplot.Rule(key="r1", label="Limit", axis="y1", position=10)
                ],
            ),
        )
        retrieved = client.line_plots.retrieve(key=p.key)
        assert retrieved.title.visible
        assert retrieved.rules[0].label == "Limit"
        assert retrieved.rules[0].position == 10

    def test_create_multiple(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should create multiple line plots."""
        plots = client.line_plots.create(
            project.key,
            [sy.LinePlot(name="Multi 1"), sy.LinePlot(name="Multi 2")],
        )
        assert [p.name for p in plots] == ["Multi 1", "Multi 2"]

    def test_retrieve_by_keys(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should retrieve multiple line plots by keys."""
        p1 = client.line_plots.create(project.key, name="A")
        p2 = client.line_plots.create(project.key, name="B")
        plots = client.line_plots.retrieve(keys=[p1.key, p2.key])
        assert {p.key for p in plots} == {p1.key, p2.key}

    def test_retrieve_missing_raises(self, client: sy.Synnax) -> None:
        """Should raise NotFoundError when retrieving a missing line plot by key."""
        with pytest.raises(NotFoundError):
            client.line_plots.retrieve(key=uuid4())

    def test_rename(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should rename a line plot."""
        p = client.line_plots.create(project.key, name="Before")
        client.line_plots.rename(p.key, "After")
        assert client.line_plots.retrieve(key=p.key).name == "After"

    def test_delete(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should delete a line plot."""
        p = client.line_plots.create(project.key, name="To Delete")
        client.line_plots.delete(p.key)
        with pytest.raises(NotFoundError):
            client.line_plots.retrieve(key=p.key)

    def test_delete_multiple(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should delete multiple line plots."""
        p1 = client.line_plots.create(project.key, name="Delete 1")
        p2 = client.line_plots.create(project.key, name="Delete 2")
        client.line_plots.delete([p1.key, p2.key])
        with pytest.raises(NotFoundError):
            client.line_plots.retrieve(keys=[p1.key, p2.key])
