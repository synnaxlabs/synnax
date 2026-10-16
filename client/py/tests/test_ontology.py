#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time
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
    def test_retrieve_children(self, client: sy.Synnax) -> None:
        name = str(uuid4())
        g = client.groups.create(sy.ontology.ROOT_ID, name)
        assert g.key is not None
        g2 = client.groups.create(sy.group.ontology_id(g.key), name)
        assert g2.key is not None
        children = client.ontology.retrieve_children(sy.group.ontology_id(g.key))
        assert children == [sy.group.ontology_id(g2.key)]

    def test_retrieve_parents(self, client: sy.Synnax) -> None:
        name = str(uuid4())
        g = client.groups.create(sy.ontology.ROOT_ID, name)
        assert g.key is not None
        g2 = client.groups.create(sy.group.ontology_id(g.key), name)
        assert g2.key is not None
        parents = client.ontology.retrieve_parents(sy.group.ontology_id(g2.key))
        assert parents == [sy.group.ontology_id(g.key)]

    def test_remove_children(self, client: sy.Synnax) -> None:
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

    def test_move_children(self, client: sy.Synnax) -> None:
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


def _create_group(client: sy.Synnax, name: str | None = None) -> sy.ontology.ID:
    g = client.groups.create(sy.ontology.ROOT_ID, name or str(uuid4()))
    assert g.key is not None
    return sy.group.ontology_id(g.key)


@pytest.mark.ontology
class TestOntologyRetrieve:
    def test_should_retrieve_a_resource_by_id(self, client: sy.Synnax) -> None:
        name = str(uuid4())
        id = _create_group(client, name)
        resources = client.ontology.retrieve(id)
        assert [(r.id, r.name) for r in resources] == [(id, name)]

    def test_should_filter_by_type(
        self, client: sy.Synnax, virtual_channel: sy.Channel
    ) -> None:
        group = _create_group(client)
        channel = sy.ontology.ID(key=str(virtual_channel.key), type="channel")
        resources = client.ontology.retrieve([group, channel], types=["group"])
        assert [r.id for r in resources] == [group]

    def test_should_retrieve_all_resources_of_a_type(self, client: sy.Synnax) -> None:
        group = _create_group(client)
        resources = client.ontology.retrieve(types=["group"])
        assert group in [r.id for r in resources]
        assert all(r.id.type == "group" for r in resources)

    def test_should_limit_and_offset_the_results(self, client: sy.Synnax) -> None:
        for _ in range(3):
            _create_group(client)
        first = client.ontology.retrieve(types=["group"], limit=2)
        assert len(first) == 2
        second = client.ontology.retrieve(types=["group"], limit=2, offset=1)
        assert len(second) == 2
        assert second[0].id == first[1].id

    def test_should_include_field_data_by_default(self, client: sy.Synnax) -> None:
        id = _create_group(client)
        assert client.ontology.retrieve(id)[0].data is not None

    def test_should_exclude_field_data(self, client: sy.Synnax) -> None:
        id = _create_group(client)
        resources = client.ontology.retrieve(id, exclude_field_data=True)
        assert resources[0].data is None

    def test_should_filter_children_by_type(
        self, client: sy.Synnax, virtual_channel: sy.Channel
    ) -> None:
        parent = _create_group(client)
        child = client.groups.create(parent, str(uuid4()))
        assert child.key is not None
        channel = sy.ontology.ID(key=str(virtual_channel.key), type="channel")
        client.ontology.add_children(parent, channel)
        children = client.ontology.retrieve_children(parent, types=["group"])
        assert children == [sy.group.ontology_id(child.key)]


@pytest.mark.ontology
class TestOntologySearch:
    def test_should_find_a_resource_by_name(self, client: sy.Synnax) -> None:
        name = f"ontology_search_{uuid4().hex}"
        id = _create_group(client, name)
        # The search index updates asynchronously.
        deadline = time.monotonic() + 5
        ids = [r.id for r in client.ontology.search(name)]
        while id not in ids and time.monotonic() < deadline:
            time.sleep(0.05)
            ids = [r.id for r in client.ontology.search(name)]
        assert id in ids

    def test_should_return_nothing_for_an_unknown_term(self, client: sy.Synnax) -> None:
        assert client.ontology.search(f"zzqx{uuid4().hex}") == []
