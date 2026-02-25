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
class TestOntology:
    def test_retrieve_basic(self, client: sy.Synnax):
        name = str(uuid4())
        g = client.groups.create(sy.ontology.ROOT_ID, name)
        assert g.key is not None
        g2 = client.ontology.retrieve(sy.group.ontology_id(g.key))
        assert g2.name == name

    def test_retrieve_children(self, client: sy.Synnax):
        name = str(uuid4())
        g = client.groups.create(sy.ontology.ROOT_ID, name)
        assert g.key is not None
        g2 = client.groups.create(sy.group.ontology_id(g.key), name)
        assert g2.key is not None
        children = client.ontology.retrieve_children(sy.group.ontology_id(g.key))
        assert len(children) == 1
        assert children[0].name == name

    def test_retrieve_parents(self, client: sy.Synnax):
        name = str(uuid4())
        g = client.groups.create(sy.ontology.ROOT_ID, name)
        assert g.key is not None
        g2 = client.groups.create(sy.group.ontology_id(g.key), name)
        assert g2.key is not None
        parents = client.ontology.retrieve_parents(sy.group.ontology_id(g2.key))
        assert len(parents) == 1
        assert parents[0].name == name

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
        assert len(children) == 1
        assert children[0].name == name
