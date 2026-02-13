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

    def test_create(self, client: sy.Synnax):
        """Should create a group under the root."""
        name = str(uuid4())
        g = client.groups.create(sy.ontology.ROOT_ID, name)
        assert g.key is not None
        assert g.name == name

    def test_create_nested(self, client: sy.Synnax):
        """Should create a group nested under another group."""
        parent = client.groups.create(sy.ontology.ROOT_ID, str(uuid4()))
        child = client.groups.create(parent.ontology_id, str(uuid4()))
        assert child.key is not None
        children = client.ontology.retrieve_children(parent.ontology_id)
        assert len(children) == 1
        assert children[0].name == child.name

    def test_rename(self, client: sy.Synnax):
        """Should rename an existing group."""
        g = client.groups.create(sy.ontology.ROOT_ID, str(uuid4()))
        new_name = str(uuid4())
        client.groups.rename(g.key, new_name)
        resource = client.ontology.retrieve(g.ontology_id)
        assert resource.name == new_name

    def test_delete(self, client: sy.Synnax):
        """Should delete a group."""
        g = client.groups.create(sy.ontology.ROOT_ID, str(uuid4()))
        client.groups.delete([g.key])
        children = client.ontology.retrieve_children(sy.ontology.ROOT_ID)
        assert all(c.id.key != str(g.key) for c in children)

    def test_delete_multiple(self, client: sy.Synnax):
        """Should delete multiple groups at once."""
        g1 = client.groups.create(sy.ontology.ROOT_ID, str(uuid4()))
        g2 = client.groups.create(sy.ontology.ROOT_ID, str(uuid4()))
        client.groups.delete([g1.key, g2.key])
        children = client.ontology.retrieve_children(sy.ontology.ROOT_ID)
        keys = {c.id.key for c in children}
        assert str(g1.key) not in keys
        assert str(g2.key) not in keys
