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
from synnax import group
from synnax.exceptions import NotFoundError
from synnax.ontology import ID
from synnax.schematic import symbol
from x.testutil import assert_eventually


def _symbol(name: str) -> symbol.Symbol:
    return symbol.Symbol(
        name=name, data=symbol.Spec(svg="<svg></svg>", variant="static")
    )


@pytest.fixture
def parent_group(client: sy.Synnax) -> group.Group:
    root = client.schematics.symbols.retrieve_group()
    return client.groups.create(group.ontology_id(root.key), f"symbols-{uuid4()}")


@pytest.fixture
def parent(parent_group: group.Group) -> ID:
    return group.ontology_id(parent_group.key)


@pytest.mark.schematic
class TestSymbol:
    def test_create(self, client: sy.Synnax, parent: ID) -> None:
        """Should create a single symbol."""
        s = client.schematics.symbols.create(_symbol("Test Symbol"), parent=parent)
        assert s.name == "Test Symbol"
        assert s.data.svg == "<svg></svg>"

    def test_create_multiple(self, client: sy.Synnax, parent: ID) -> None:
        """Should create multiple symbols."""
        symbols = client.schematics.symbols.create(
            [_symbol("Multi 1"), _symbol("Multi 2")], parent=parent
        )
        assert [s.name for s in symbols] == ["Multi 1", "Multi 2"]

    def test_retrieve_by_key(self, client: sy.Synnax, parent: ID) -> None:
        """Should retrieve a symbol by key."""
        created = client.schematics.symbols.create(
            _symbol("Retrieve Me"), parent=parent
        )
        retrieved = client.schematics.symbols.retrieve(key=created.key)
        assert retrieved.key == created.key
        assert retrieved.name == "Retrieve Me"

    def test_retrieve_by_keys(self, client: sy.Synnax, parent: ID) -> None:
        """Should retrieve multiple symbols by keys."""
        s1, s2 = client.schematics.symbols.create(
            [_symbol("Symbol A"), _symbol("Symbol B")], parent=parent
        )
        retrieved = client.schematics.symbols.retrieve(keys=[s1.key, s2.key])
        assert {s.key for s in retrieved} == {s1.key, s2.key}

    def test_retrieve_with_search_term(self, client: sy.Synnax, parent: ID) -> None:
        """Should search for symbols by name."""
        prefix = f"searchable-symbol-{uuid4()}"
        client.schematics.symbols.create(
            [_symbol(f"{prefix}-1"), _symbol(f"{prefix}-2")], parent=parent
        )

        def check() -> None:
            results = client.schematics.symbols.retrieve(search_term=prefix)
            assert len(results) >= 2
            assert all(prefix in s.name for s in results)

        assert_eventually(check)

    def test_retrieve_not_found(self, client: sy.Synnax) -> None:
        """Should raise NotFoundError when the key does not exist."""
        with pytest.raises(NotFoundError):
            client.schematics.symbols.retrieve(key=uuid4())

    def test_rename(self, client: sy.Synnax, parent: ID) -> None:
        """Should rename a symbol."""
        s = client.schematics.symbols.create(_symbol("Before Rename"), parent=parent)
        client.schematics.symbols.rename(s.key, "After Rename")
        assert client.schematics.symbols.retrieve(key=s.key).name == "After Rename"

    def test_delete(self, client: sy.Synnax, parent: ID) -> None:
        """Should delete symbols."""
        s1, s2 = client.schematics.symbols.create(
            [_symbol("Delete 1"), _symbol("Delete 2")], parent=parent
        )
        client.schematics.symbols.delete([s1.key, s2.key])
        with pytest.raises(NotFoundError):
            client.schematics.symbols.retrieve(key=s1.key)

    def test_retrieve_group(self, client: sy.Synnax) -> None:
        """Should retrieve the same symbol group on each call."""
        g = client.schematics.symbols.retrieve_group()
        assert client.schematics.symbols.retrieve_group().key == g.key

    def test_delete_group(
        self, client: sy.Synnax, parent_group: group.Group, parent: ID
    ) -> None:
        """Should delete a group and every symbol in it."""
        s = client.schematics.symbols.create(_symbol("In Group"), parent=parent)
        client.schematics.symbols.delete_group(parent_group.key)
        with pytest.raises(NotFoundError):
            client.schematics.symbols.retrieve(key=s.key)
        with pytest.raises(NotFoundError):
            client.groups.retrieve(parent_group.key)
