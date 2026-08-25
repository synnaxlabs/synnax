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


@pytest.mark.ontology
class TestOntologyID:
    """ID's string form is the wire format the Core parses (e.g. the imex parent
    query param), so it must serialize as type:key."""

    def test_str_is_type_colon_key(self) -> None:
        id = sy.ontology.ID(type="project", key="abc-123")
        assert str(id) == "project:abc-123"

    def test_str_of_root_id(self) -> None:
        assert str(sy.ontology.ROOT_ID) == "builtin:root"

    def test_str_round_trips_through_the_parsing_constructor(self) -> None:
        original = sy.ontology.ID(type="log", key=str(uuid4()))
        assert sy.ontology.ID(str(original)) == original


@pytest.mark.ontology
class TestOntology:
    def test_retrieve_children(self, client: sy.Synnax):
        name = str(uuid4())
        g = client.groups.create(sy.ontology.ROOT_ID, name)
        assert g.key is not None
        g2 = client.groups.create(sy.group.ontology_id(g.key), name)
        assert g2.key is not None
        children = client.ontology.retrieve_children(sy.group.ontology_id(g.key))
        assert children == [sy.group.ontology_id(g2.key)]

    def test_retrieve_parents(self, client: sy.Synnax):
        name = str(uuid4())
        g = client.groups.create(sy.ontology.ROOT_ID, name)
        assert g.key is not None
        g2 = client.groups.create(sy.group.ontology_id(g.key), name)
        assert g2.key is not None
        parents = client.ontology.retrieve_parents(sy.group.ontology_id(g2.key))
        assert parents == [sy.group.ontology_id(g.key)]

    def test_remove_children(self, client: sy.Synnax):
        name = str(uuid4())
        g = client.groups.create(sy.ontology.ROOT_ID, name)
        assert g.key is not None
        g2 = client.groups.create(sy.group.ontology_id(g.key), name)
        assert g2.key is not None
        client.ontology.remove_children(
            sy.group.ontology_id(g.key), sy.group.ontology_id(g2.key)
        )
        children = client.ontology.retrieve_children(sy.group.ontology_id(g.key))
        assert len(children) == 0

    def test_move_children(self, client: sy.Synnax):
        name = str(uuid4())
        g = client.groups.create(sy.ontology.ROOT_ID, name)
        assert g.key is not None
        g2 = client.groups.create(sy.ontology.ROOT_ID, name)
        assert g2.key is not None
        client.ontology.move_children(
            sy.group.ontology_id(g.key),
            sy.group.ontology_id(g2.key),
            sy.group.ontology_id(g.key),
        )
        children = client.ontology.retrieve_children(sy.group.ontology_id(g2.key))
        assert children == [sy.group.ontology_id(g.key)]
