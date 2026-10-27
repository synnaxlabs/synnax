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

import synnax as sy
from synnax.exceptions import NotFoundError


@pytest.fixture
def project(client: sy.Synnax) -> sy.Project:
    return client.projects.create(name="Line Plot Project")


@pytest.mark.lineplot
class TestLinePlot:
    def test_create(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should create a line plot from a name."""
        p = client.lineplots.create(project.key, name="Plot")
        assert p.name == "Plot"
        assert client.lineplots.retrieve(key=p.key).name == "Plot"

    def test_create_from_object(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should create a line plot from a LinePlot object and keep its fields."""
        p = client.lineplots.create(
            project.key,
            sy.LinePlot(
                name="Configured",
                title=sy.lineplot.Title(visible=True),
                rules=[
                    sy.lineplot.Rule(key="r1", label="Limit", axis="y1", position=10)
                ],
            ),
        )
        retrieved = client.lineplots.retrieve(key=p.key)
        assert retrieved.title.visible
        assert retrieved.rules[0].label == "Limit"
        assert retrieved.rules[0].position == 10

    def test_create_multiple(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should create multiple line plots."""
        plots = client.lineplots.create(
            project.key,
            [sy.LinePlot(name="Multi 1"), sy.LinePlot(name="Multi 2")],
        )
        assert [p.name for p in plots] == ["Multi 1", "Multi 2"]

    def test_retrieve_by_keys(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should retrieve multiple line plots by keys."""
        p1 = client.lineplots.create(project.key, name="A")
        p2 = client.lineplots.create(project.key, name="B")
        plots = client.lineplots.retrieve(keys=[p1.key, p2.key])
        assert {p.key for p in plots} == {p1.key, p2.key}

    def test_retrieve_missing_raises(self, client: sy.Synnax) -> None:
        """Should raise NotFoundError when retrieving a missing line plot by key."""
        with pytest.raises(NotFoundError):
            client.lineplots.retrieve(key=uuid4())

    def test_rename(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should rename a line plot."""
        p = client.lineplots.create(project.key, name="Before")
        client.lineplots.rename(p.key, "After")
        assert client.lineplots.retrieve(key=p.key).name == "After"

    def test_delete(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should delete a line plot."""
        p = client.lineplots.create(project.key, name="To Delete")
        client.lineplots.delete(p.key)
        with pytest.raises(NotFoundError):
            client.lineplots.retrieve(key=p.key)

    def test_delete_multiple(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should delete multiple line plots."""
        p1 = client.lineplots.create(project.key, name="Delete 1")
        p2 = client.lineplots.create(project.key, name="Delete 2")
        client.lineplots.delete([p1.key, p2.key])
        with pytest.raises(NotFoundError):
            client.lineplots.retrieve(keys=[p1.key, p2.key])
