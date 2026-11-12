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
from x import filter


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

    def test_create_with_key(self, client: sy.Synnax) -> None:
        """Should create a group with the given key."""
        key = uuid4()
        g = client.groups.create(str(uuid4()), scope="channel", key=key)
        assert g.key == key
        assert client.groups.retrieve(key).name == g.name

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

    def test_retrieve_by_parent(self, client: sy.Synnax) -> None:
        """Should keep the groups in a parent, and split them by whether they nest."""
        top = client.groups.create(str(uuid4()), scope="channel")
        parent = sy.group.resource_id(top.key)
        child = client.groups.create(str(uuid4()), parent=parent)
        assert [g.key for g in client.groups.retrieve(parents=parent)] == [child.key]
        keys = [top.key, child.key]
        nested = client.groups.retrieve(key=keys, nested=True)
        assert [g.key for g in nested] == [child.key]
        assert [g.key for g in client.groups.retrieve(key=keys, nested=False)] == [
            top.key
        ]

    def test_retrieve_where(self, client: sy.Synnax) -> None:
        """Should keep groups that match a filter tree and the given keys."""
        g1 = client.groups.create(str(uuid4()), scope="channel")
        g2 = client.groups.create(str(uuid4()), scope="channel")
        where = sy.group.Where(
            not_=sy.group.Where(key=filter.Set[sy.group.Key](eq=g1.key))
        )
        groups = client.groups.retrieve(where=where, keys=[g1.key, g2.key, uuid4()])
        assert [g.key for g in groups] == [g2.key]

    def test_retrieve_not_found(self, client: sy.Synnax) -> None:
        """Should raise NotFoundError when the group does not exist."""
        with pytest.raises(sy.NotFoundError):
            client.groups.retrieve(uuid4())

    def test_retrieve_key_list(self, client: sy.Synnax) -> None:
        """Should return a list for a list of keys, even with one key."""
        a = client.groups.create(str(uuid4()), scope="channel")
        res = client.groups.retrieve(key=[a.key])
        assert [r.key for r in res] == [a.key]

    def test_retrieve_plain_key_beside_fields(self, client: sy.Synnax) -> None:
        """Should return one group for a plain key beside other fields."""
        a = client.groups.create(str(uuid4()), scope="channel")
        assert client.groups.retrieve(key=a.key, not_={"key": uuid4()}).key == a.key
        with pytest.raises(sy.NotFoundError, match=r"Found no group where .+"):
            client.groups.retrieve(key=a.key, not_={"key": a.key})

    def test_retrieve_or(self, client: sy.Synnax) -> None:
        """Should retrieve the groups that match any filter in or_."""
        a = client.groups.create(str(uuid4()), scope="channel")
        b = client.groups.create(str(uuid4()), scope="channel")
        res = client.groups.retrieve(or_=[{"key": a.key}, {"key": [b.key]}])
        assert {r.key for r in res} == {a.key, b.key}

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
