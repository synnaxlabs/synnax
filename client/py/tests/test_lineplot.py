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
from synnax import lineplot, ranger
from synnax.exceptions import NotFoundError
from x import telem


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
        assert p.ranges == []

    def test_round_trip(self) -> None:
        """Should parse lines, rules, and ranges and dump them back unchanged."""
        range_key = uuid4()
        p = lineplot.LinePlot.model_validate(
            {
                "name": "Pressure",
                "channels": {"x1": 1, "y1": [2, 3]},
                "lines": [
                    {
                        "key": str(uuid4()),
                        "y_axis": "y1",
                        "y_channel": 2,
                        "color": "#ff0000",
                        "detail": "high",
                    }
                ],
                "rules": [
                    {"key": "max", "label": "Max", "axis": "y1", "position": 100}
                ],
                "ranges": [
                    {"variant": "key", "key": str(range_key)},
                    {
                        "variant": "window",
                        "window": {"variant": "dynamic", "time_span": 5},
                    },
                ],
            }
        )
        assert p.channels.y1 == [2, 3]
        assert p.lines[0].detail == "high"
        assert p.lines[0].x_channel == 0
        assert p.rules[0].line_width == 1
        assert p.ranges[0] == ranger.KeyRef(key=range_key)
        assert p.ranges[1] == ranger.WindowRef(
            window=telem.DynamicWindow(time_span=telem.TimeSpan(5))
        )
        assert lineplot.LinePlot.model_validate_json(p.model_dump_json()) == p

    def test_static_window_range(self) -> None:
        """Should select a static window range by its variant."""
        p = lineplot.LinePlot.model_validate(
            {
                "name": "Pressure",
                "ranges": [
                    {
                        "variant": "window",
                        "window": {
                            "variant": "static",
                            "time_range": {"start": 1, "end": 2},
                        },
                    }
                ],
            }
        )
        ref = p.ranges[0]
        assert isinstance(ref, ranger.WindowRef)
        assert isinstance(ref.window, telem.StaticWindow)

    def test_invalid_axis(self) -> None:
        """Should reject a rule on an unknown axis."""
        with pytest.raises(ValidationError):
            lineplot.Rule.model_validate(
                {"key": "a", "label": "A", "axis": "z1", "position": 0}
            )

    def test_resource_id(self) -> None:
        """Should build the resource ID of a line plot."""
        key = uuid4()
        assert lineplot.resource_id(key).type == lineplot.TYPE_RESOURCE_ID.type
        assert lineplot.resource_id(key).key == str(key)
        p = lineplot.LinePlot(name="Pressure")
        assert hash(p) == hash(p.key)


@pytest.fixture
def project(client: sy.Synnax) -> sy.Project:
    return client.projects.create(name="Line Plot Project")


@pytest.mark.lineplot
class TestClient:
    def test_create(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should create a line plot from a name."""
        p = client.line_plots.create(name="Plot", project=project.key)
        assert p.name == "Plot"
        assert p.project == project.key
        assert client.line_plots.retrieve(key=p.key).name == "Plot"

    def test_create_from_object(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should create a line plot from a LinePlot object and keep its fields."""
        p = client.line_plots.create(
            sy.LinePlot(
                name="Configured",
                title=sy.lineplot.Title(visible=True),
                rules=[
                    sy.lineplot.Rule(key="r1", label="Limit", axis="y1", position=10)
                ],
                project=project.key,
            ),
        )
        retrieved = client.line_plots.retrieve(key=p.key)
        assert retrieved.title.visible
        assert retrieved.rules[0].label == "Limit"
        assert retrieved.rules[0].position == 10

    def test_create_multiple(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should create multiple line plots."""
        plots = client.line_plots.create(
            [
                sy.LinePlot(name="Multi 1", project=project.key),
                sy.LinePlot(name="Multi 2", project=project.key),
            ],
        )
        assert [p.name for p in plots] == ["Multi 1", "Multi 2"]

    def test_retrieve_by_keys(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should retrieve multiple line plots by keys."""
        p1 = client.line_plots.create(name="A", project=project.key)
        p2 = client.line_plots.create(name="B", project=project.key)
        plots = client.line_plots.retrieve(keys=[p1.key, p2.key])
        assert {p.key for p in plots} == {p1.key, p2.key}

    def test_retrieve_by_project(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should retrieve only the line plots the project owns."""
        a = client.line_plots.create(name="Child A", project=project.key)
        b = client.line_plots.create(name="Child B", project=project.key)
        other = client.projects.create(name="Other Project")
        client.line_plots.create(name="Other", project=other.key)
        retrieved = client.line_plots.retrieve(projects=[project.key])
        assert {r.key for r in retrieved} == {a.key, b.key}

    def test_retrieve_by_group(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should filter the line plots by group."""
        made = client.line_plots.create(name="Ungrouped", project=project.key)
        g = client.groups.create("Empty", parent=sy.project.resource_id(project.key))
        assert client.line_plots.retrieve(groups=g.key) == []
        ungrouped = client.line_plots.retrieve(projects=project.key, grouped=False)
        assert [r.key for r in ungrouped] == [made.key]
        assert client.line_plots.retrieve(projects=project.key, grouped=True) == []

    def test_retrieve_where(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should keep the line plots that match the where and the other filters."""
        a = client.line_plots.create(name="A", project=project.key)
        b = client.line_plots.create(name="B", project=project.key)
        where = lineplot.Where(
            not_=lineplot.Where(key=sy.filter.Set[lineplot.Key](eq=a.key))
        )
        res = client.line_plots.retrieve(where=where, keys=[a.key, b.key, uuid4()])
        assert [r.key for r in res] == [b.key]

    def test_retrieve_missing_raises(self, client: sy.Synnax) -> None:
        """Should raise NotFoundError when retrieving a missing line plot by key."""
        with pytest.raises(NotFoundError):
            client.line_plots.retrieve(key=uuid4())

    def test_retrieve_key_list(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should return a list for a list of keys, even with one key."""
        a = client.line_plots.create(name="A", project=project.key)
        res = client.line_plots.retrieve(key=[a.key])
        assert [r.key for r in res] == [a.key]

    def test_retrieve_plain_key_beside_fields(
        self, client: sy.Synnax, project: sy.Project
    ) -> None:
        """Should return one line plot for a plain key beside other fields."""
        a = client.line_plots.create(name="A", project=project.key)
        assert client.line_plots.retrieve(key=a.key, not_={"key": uuid4()}).key == a.key
        with pytest.raises(NotFoundError, match=r"Found no line plot where .+"):
            client.line_plots.retrieve(key=a.key, not_={"key": a.key})

    def test_retrieve_or(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should retrieve the line plots that match any filter in or_."""
        a = client.line_plots.create(name="A", project=project.key)
        b = client.line_plots.create(name="B", project=project.key)
        res = client.line_plots.retrieve(or_=[{"key": a.key}, {"key": [b.key]}])
        assert {r.key for r in res} == {a.key, b.key}

    def test_rename(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should rename a line plot."""
        p = client.line_plots.create(name="Before", project=project.key)
        client.line_plots.rename(p.key, "After")
        assert client.line_plots.retrieve(key=p.key).name == "After"

    def test_delete(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should delete a line plot."""
        p = client.line_plots.create(name="To Delete", project=project.key)
        client.line_plots.delete(p.key)
        with pytest.raises(NotFoundError):
            client.line_plots.retrieve(key=p.key)

    def test_delete_multiple(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should delete multiple line plots."""
        p1 = client.line_plots.create(name="Delete 1", project=project.key)
        p2 = client.line_plots.create(name="Delete 2", project=project.key)
        client.line_plots.delete([p1.key, p2.key])
        assert client.line_plots.retrieve(keys=[p1.key, p2.key]) == []
