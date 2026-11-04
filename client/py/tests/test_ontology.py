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


def _create_group(
    client: sy.Synnax, parent: sy.resource.ID | None = None
) -> sy.resource.ID:
    """Creates a group, linked under parent by an ontology edge when one is given."""
    g = client.groups.create(str(uuid4()), scope="channel")
    id = sy.group.resource_id(g.key)
    if parent is not None:
        client.ontology.add_children(parent, id)
    return id


@pytest.mark.ontology
class TestOntology:
    def test_str_of_root_id(self) -> None:
        assert str(sy.ontology.ROOT_ID) == "builtin:root"

    def test_retrieve_children(self, client: sy.Synnax) -> None:
        parent = _create_group(client)
        child = _create_group(client, parent)
        assert client.ontology.retrieve_children(parent) == [child]

    def test_retrieve_parents(self, client: sy.Synnax) -> None:
        parent = _create_group(client)
        child = _create_group(client, parent)
        assert client.ontology.retrieve_parents(child) == [parent]

    def test_remove_children(self, client: sy.Synnax) -> None:
        parent = _create_group(client)
        child = _create_group(client, parent)
        client.ontology.remove_children(parent, child)
        assert client.ontology.retrieve_children(parent) == []

    def test_move_children(self, client: sy.Synnax) -> None:
        from_ = _create_group(client)
        to = _create_group(client)
        child = _create_group(client, from_)
        client.ontology.move_children(from_, to, child)
        assert client.ontology.retrieve_children(to) == [child]
        assert client.ontology.retrieve_children(from_) == []
