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


@pytest.mark.group
class TestGroupClient:
    """Tests for the group client."""

    def test_create(self, client: sy.Synnax) -> None:
        """Should create a top-level group in a scope."""
        name = str(uuid4())
        g = client.groups.create(name, scope="channel")
        assert g.key is not None
        assert g.name == name
        assert g.scope == "channel"
        assert g.parent is None

    def test_create_nested(self, client: sy.Synnax) -> None:
        """Should create a group nested under another group with its scope."""
        parent = client.groups.create(str(uuid4()), scope="channel")
        child = client.groups.create(
            str(uuid4()), parent=sy.group.resource_id(parent.key)
        )
        assert child.scope == "channel"
        assert child.parent == sy.group.resource_id(parent.key)

    def test_retrieve(self, client: sy.Synnax) -> None:
        """Should retrieve a group by its key."""
        g = client.groups.create(str(uuid4()), scope="channel")
        assert client.groups.retrieve(g.key) == g

    def test_retrieve_multiple(self, client: sy.Synnax) -> None:
        """Should retrieve multiple groups by their keys."""
        g1 = client.groups.create(str(uuid4()), scope="channel")
        g2 = client.groups.create(str(uuid4()), scope="channel")
        groups = client.groups.retrieve(keys=[g1.key, g2.key])
        assert sorted(g.name for g in groups) == sorted([g1.name, g2.name])

    def test_retrieve_not_found(self, client: sy.Synnax) -> None:
        """Should raise NotFoundError when the group does not exist."""
        with pytest.raises(sy.NotFoundError):
            client.groups.retrieve(uuid4())

    def test_rename(self, client: sy.Synnax) -> None:
        """Should rename an existing group."""
        g = client.groups.create(str(uuid4()), scope="channel")
        new_name = str(uuid4())
        client.groups.rename(g.key, new_name)
        assert client.groups.retrieve(g.key).name == new_name

    def test_set_parent(self, client: sy.Synnax) -> None:
        """Should move a group under another group and back to the top level."""
        parent = client.groups.create(str(uuid4()), scope="channel")
        g = client.groups.create(str(uuid4()), scope="channel")
        client.groups.set_parent([g.key], sy.group.resource_id(parent.key))
        assert client.groups.retrieve(g.key).parent == sy.group.resource_id(parent.key)
        client.groups.set_parent([g.key])
        assert client.groups.retrieve(g.key).parent is None

    def test_delete(self, client: sy.Synnax) -> None:
        """Should delete a group."""
        g = client.groups.create(str(uuid4()), scope="channel")
        client.groups.delete([g.key])
        with pytest.raises(sy.NotFoundError):
            client.groups.retrieve(g.key)

    def test_delete_multiple(self, client: sy.Synnax) -> None:
        """Should delete multiple groups at once."""
        g1 = client.groups.create(str(uuid4()), scope="channel")
        g2 = client.groups.create(str(uuid4()), scope="channel")
        client.groups.delete([g1.key, g2.key])
        assert client.groups.retrieve(keys=[g1.key, g2.key]) == []
