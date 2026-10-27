#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Callable
from typing import cast
from uuid import uuid4

import pytest

import synnax as sy
from synnax import project
from synnax.exceptions import NotFoundError
from x.testutil import assert_eventually


@pytest.mark.project
class TestProject:
    def test_create(self, client: sy.Synnax) -> None:
        """Should create a single project."""
        p = client.projects.create(name="Test Project")
        assert p.key is not None
        assert p.name == "Test Project"

    def test_create_from_object(self, client: sy.Synnax) -> None:
        """Should create a project from a Project object."""
        p = client.projects.create(sy.project.Project(name="From Object"))
        assert p.name == "From Object"

    def test_create_multiple(self, client: sy.Synnax) -> None:
        """Should create multiple projects."""
        projects = client.projects.create(
            [
                sy.project.Project(name="Multi 1"),
                sy.project.Project(name="Multi 2"),
            ]
        )
        assert len(projects) == 2
        assert projects[0].name == "Multi 1"
        assert projects[1].name == "Multi 2"

    def test_retrieve_by_key(self, client: sy.Synnax) -> None:
        """Should retrieve a project by key."""
        created = client.projects.create(name="Retrieve Me")
        retrieved = client.projects.retrieve(key=created.key)
        assert retrieved.key == created.key
        assert retrieved.name == "Retrieve Me"

    def test_retrieve_by_keys(self, client: sy.Synnax) -> None:
        """Should retrieve multiple projects by keys."""
        p1 = client.projects.create(name="Project A")
        p2 = client.projects.create(name="Project B")
        projects = client.projects.retrieve(keys=[p1.key, p2.key])
        assert len(projects) == 2

    def test_retrieve_conditions(self, client: sy.Synnax) -> None:
        """Should keep projects that match every condition and the given keys."""
        p1 = client.projects.create(name="Where A")
        p2 = client.projects.create(name="Where B")
        projects = client.projects.retrieve(
            ~project.key.eq(p1.key), keys=[p1.key, p2.key]
        )
        assert [p.key for p in projects] == [p2.key]
        assert client.projects.retrieve(p1.key).name == "Where A"
        one = client.projects.retrieve(project.key.in_([p2.key]), one=True)
        assert one.key == p2.key

    def test_retrieve_with_search_term(self, client: sy.Synnax) -> None:
        """Should search for projects by name."""
        prefix = f"searchable-project-{uuid4()}"
        client.projects.create(
            [
                sy.project.Project(name=f"{prefix}-1"),
                sy.project.Project(name=f"{prefix}-2"),
            ]
        )

        def check() -> None:
            results = client.projects.retrieve(search_term=prefix)
            assert len(results) >= 2
            assert all(prefix in p.name for p in results)

        assert_eventually(check)

    def test_rename(self, client: sy.Synnax) -> None:
        """Should rename a project."""
        p = client.projects.create(name="Before Rename")
        client.projects.rename(p.key, "After Rename")
        assert client.projects.retrieve(key=p.key).name == "After Rename"

    def test_retrieve_missing_raises(self, client: sy.Synnax) -> None:
        """Should raise NotFoundError when retrieving a missing project by key."""
        with pytest.raises(NotFoundError):
            client.projects.retrieve(key=uuid4())

    def test_retrieve_rejects_author(self, client: sy.Synnax) -> None:
        """Should reject the removed author filter."""
        retrieve = cast(Callable[..., object], client.projects.retrieve)
        with pytest.raises(TypeError):
            retrieve(author=uuid4())

    def test_delete(self, client: sy.Synnax) -> None:
        """Should delete a project."""
        p = client.projects.create(name="To Delete")
        client.projects.delete(p.key)
        with pytest.raises(NotFoundError):
            client.projects.retrieve(key=p.key)

    def test_delete_multiple(self, client: sy.Synnax) -> None:
        """Should delete multiple projects."""
        p1 = client.projects.create(name="Delete 1")
        p2 = client.projects.create(name="Delete 2")
        client.projects.delete([p1.key, p2.key])
        assert client.projects.retrieve(keys=[p1.key, p2.key]) == []
