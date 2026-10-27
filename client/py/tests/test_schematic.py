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
    return client.projects.create(name=f"schematic-project-{uuid4()}")


@pytest.mark.schematic
class TestSchematic:
    def test_create(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should create a schematic from a name."""
        s = client.schematics.create(project.key, name="Test Schematic")
        assert s.name == "Test Schematic"
        assert s.nodes == []

    def test_create_from_object(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should create a schematic from a Schematic object."""
        s = client.schematics.create(project.key, sy.Schematic(name="From Object"))
        assert s.name == "From Object"

    def test_create_multiple(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should create multiple schematics."""
        schematics = client.schematics.create(
            project.key,
            [sy.Schematic(name="Multi 1"), sy.Schematic(name="Multi 2")],
        )
        assert [s.name for s in schematics] == ["Multi 1", "Multi 2"]

    def test_retrieve_by_key(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should retrieve a schematic by key."""
        created = client.schematics.create(project.key, name="Retrieve Me")
        retrieved = client.schematics.retrieve(key=created.key)
        assert retrieved.key == created.key
        assert retrieved.name == "Retrieve Me"

    def test_retrieve_by_keys(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should retrieve multiple schematics by keys."""
        s1 = client.schematics.create(project.key, name="Schematic A")
        s2 = client.schematics.create(project.key, name="Schematic B")
        retrieved = client.schematics.retrieve(keys=[s1.key, s2.key])
        assert {s.key for s in retrieved} == {s1.key, s2.key}

    def test_retrieve_not_found(self, client: sy.Synnax) -> None:
        """Should raise NotFoundError when the key does not exist."""
        with pytest.raises(NotFoundError):
            client.schematics.retrieve(key=uuid4())

    def test_rename(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should rename a schematic."""
        s = client.schematics.create(project.key, name="Before Rename")
        client.schematics.rename(s.key, "After Rename")
        assert client.schematics.retrieve(key=s.key).name == "After Rename"

    def test_copy(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should copy a schematic under a new key and name."""
        s = client.schematics.create(project.key, name="Original")
        c = client.schematics.copy(s.key, name="Copy")
        assert c.key != s.key
        assert c.name == "Copy"
        assert not c.snapshot
        assert client.schematics.retrieve(key=c.key).name == "Copy"

    def test_copy_snapshot(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should copy a schematic as an immutable snapshot."""
        s = client.schematics.create(project.key, name="Original")
        c = client.schematics.copy(s.key, name="Snapshot", snapshot=True)
        assert c.snapshot

    def test_delete(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should delete a schematic."""
        s = client.schematics.create(project.key, name="To Delete")
        client.schematics.delete(s.key)
        with pytest.raises(NotFoundError):
            client.schematics.retrieve(key=s.key)

    def test_delete_multiple(self, client: sy.Synnax, project: sy.Project) -> None:
        """Should delete multiple schematics."""
        s1 = client.schematics.create(project.key, name="Delete 1")
        s2 = client.schematics.create(project.key, name="Delete 2")
        client.schematics.delete([s1.key, s2.key])
        with pytest.raises(NotFoundError):
            client.schematics.retrieve(keys=[s1.key, s2.key])
