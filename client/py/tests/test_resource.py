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
from pydantic import ValidationError

import synnax as sy
from x.testutil import assert_eventually

MISSING_LABEL = sy.label.resource_id(uuid4())


def _token() -> str:
    return f"res{uuid4().hex[:12]}"


def _create_group(client: sy.Synnax, name: str) -> sy.resource.ID:
    return sy.group.resource_id(client.groups.create(name, scope="channel").key)


def _create_label(client: sy.Synnax, name: str) -> sy.resource.ID:
    return sy.label.resource_id(client.labels.create(name=name, color="#E87E04").key)


@pytest.mark.resource
class TestSearch:
    """Tests for searching the resources of every type."""

    def test_merges_types_closest_match_first(self, client: sy.Synnax) -> None:
        """Should merge the matches of every type, closest match first."""
        token = _token()
        exact = _create_group(client, token)
        partial = _create_label(client, f"{token} pump")

        def check() -> None:
            assert client.resources.search(token) == [
                sy.resource.Resource(id=exact, name=token),
                sy.resource.Resource(id=partial, name=f"{token} pump"),
            ]

        assert_eventually(check)

    def test_searches_only_the_given_types(self, client: sy.Synnax) -> None:
        """Should search only the resources of the given types."""
        token = _token()
        _create_group(client, token)
        label = _create_label(client, token)

        def check() -> None:
            res = client.resources.search(token, types=["label"])
            assert [r.id for r in res] == [label]

        assert_eventually(check)

    def test_pages_with_a_limit_and_an_offset(self, client: sy.Synnax) -> None:
        """Should page through the matches with a limit and an offset."""
        token = _token()
        for suffix in ["a", "b", "c"]:
            _create_group(client, f"{token} {suffix}")

        def check() -> None:
            assert len(client.resources.search(token, limit=2)) == 2

        assert_eventually(check)
        assert len(client.resources.search(token, limit=2, offset=2)) == 1

    def test_refuses_a_type_the_core_cannot_search(self, client: sy.Synnax) -> None:
        """Should raise a ValidationError for a type the Core cannot search."""
        with pytest.raises(
            sy.ValidationError, match='types: unknown resource type "builtin"'
        ):
            client.resources.search("pump", types=["builtin"])


@pytest.mark.resource
class TestRetrieve:
    """Tests for retrieving the names of resources by ID."""

    def test_retrieves_a_single_resource(self, client: sy.Synnax) -> None:
        """Should retrieve the name of a single resource."""
        name = _token()
        id = _create_label(client, name)
        assert client.resources.retrieve(id) == sy.resource.Resource(id=id, name=name)

    def test_retrieves_mixed_types_in_order(self, client: sy.Synnax) -> None:
        """Should retrieve resources of several types in the order of the IDs."""
        group_name, label_name = _token(), _token()
        group = _create_group(client, group_name)
        label = _create_label(client, label_name)
        assert client.resources.retrieve([str(label), group]) == [
            sy.resource.Resource(id=label, name=label_name),
            sy.resource.Resource(id=group, name=group_name),
        ]

    def test_leaves_out_an_id_that_names_no_resource(self, client: sy.Synnax) -> None:
        """Should leave out an ID that names no resource."""
        name = _token()
        label = _create_label(client, name)
        res = client.resources.retrieve([MISSING_LABEL, label])
        assert [r.name for r in res] == [name]

    def test_raises_not_found_for_a_missing_single_id(self, client: sy.Synnax) -> None:
        """Should raise a NotFoundError when a single ID names no resource."""
        with pytest.raises(sy.NotFoundError, match="Resource .* not found"):
            client.resources.retrieve(MISSING_LABEL)


@pytest.mark.resource
class TestID:
    """ID's string form is the wire format the Core parses (e.g. the imex parent
    query param), so it must serialize as type:key."""

    def test_str_is_type_colon_key(self) -> None:
        id = sy.resource.ID(type="project", key="abc-123")
        assert str(id) == "project:abc-123"

    def test_str_round_trips_through_the_parsing_constructor(self) -> None:
        original = sy.resource.ID(type="log", key=str(uuid4()))
        assert sy.resource.ID(str(original)) == original

    def test_copy_constructor(self) -> None:
        original = sy.resource.ID(type="channel", key="12")
        assert sy.resource.ID(original) == original

    def test_invalid_type(self) -> None:
        with pytest.raises(ValidationError, match="Input should be 'arc'"):
            sy.resource.ID("dog:1")
