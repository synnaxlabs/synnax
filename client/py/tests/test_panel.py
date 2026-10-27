#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import UUID

import pytest

import synnax as sy
from synnax.exceptions import NotFoundError, ValidationError
from x.testutil import assert_eventually


@pytest.fixture
def parent(client: sy.Synnax) -> sy.ontology.ID:
    """Creates a project to own the panels of a test."""
    proj = client.projects.create(name="Panel Parent")
    return sy.project.ontology_id(proj.key)


@pytest.mark.panel
class TestPanel:
    def test_create(self, client: sy.Synnax, parent: sy.ontology.ID) -> None:
        """Should create a single panel with an empty root leaf."""
        p = client.panels.create(name="Test Panel", parent=parent)
        assert p.name == "Test Panel"
        assert isinstance(p.root, sy.panel.LeafNode)
        assert p.root.tabs == []

    def test_create_from_object(
        self, client: sy.Synnax, parent: sy.ontology.ID
    ) -> None:
        """Should create a panel from a Panel object and keep its tree."""
        tab = sy.panel.ViewTab(type="docs")
        p = client.panels.create(
            sy.Panel(
                name="From Object",
                parent=parent,
                root=sy.panel.LeafNode(tabs=[tab]),
            )
        )
        retrieved = client.panels.retrieve(key=p.key)
        assert isinstance(retrieved.root, sy.panel.LeafNode)
        assert len(retrieved.root.tabs) == 1
        assert retrieved.root.tabs[0].key == tab.key

    def test_create_multiple(self, client: sy.Synnax, parent: sy.ontology.ID) -> None:
        """Should create multiple panels."""
        panels = client.panels.create(
            [
                sy.Panel(name="Multi 1", parent=parent),
                sy.Panel(name="Multi 2", parent=parent),
            ]
        )
        assert [p.name for p in panels] == ["Multi 1", "Multi 2"]

    def test_create_without_parent_raises(self, client: sy.Synnax) -> None:
        """Should raise ValidationError when a panel has no parent."""
        with pytest.raises(ValidationError):
            client.panels.create(sy.Panel(name="Orphan"))

    def test_retrieve_by_key(self, client: sy.Synnax, parent: sy.ontology.ID) -> None:
        """Should retrieve a panel by key."""
        created = client.panels.create(name="Retrieve Me", parent=parent)
        retrieved = client.panels.retrieve(key=created.key)
        assert retrieved.key == created.key
        assert retrieved.name == "Retrieve Me"

    def test_retrieve_by_keys(self, client: sy.Synnax, parent: sy.ontology.ID) -> None:
        """Should retrieve multiple panels by keys."""
        p1 = client.panels.create(name="Panel A", parent=parent)
        p2 = client.panels.create(name="Panel B", parent=parent)
        panels = client.panels.retrieve(keys=[p1.key, p2.key])
        assert {p.key for p in panels} == {p1.key, p2.key}

    def test_retrieve_by_parent(
        self, client: sy.Synnax, parent: sy.ontology.ID
    ) -> None:
        """Should retrieve only the panels that are children of the parent."""
        p1 = client.panels.create(name="Child 1", parent=parent)
        p2 = client.panels.create(name="Child 2", parent=parent)
        other = client.projects.create(name="Other Parent")
        client.panels.create(name="Other", parent=sy.project.ontology_id(other.key))
        panels = client.panels.retrieve(parent=parent)
        assert {p.key for p in panels} == {p1.key, p2.key}

    def test_retrieve_by_parent_without_children(
        self, client: sy.Synnax, parent: sy.ontology.ID
    ) -> None:
        """Should return no panels when the parent has no panel children."""
        assert client.panels.retrieve(parent=parent) == []

    def test_retrieve_with_search_term(
        self, client: sy.Synnax, parent: sy.ontology.ID
    ) -> None:
        """Should search for panels by name."""
        prefix = "searchable-panel"
        created = client.panels.create(
            [
                sy.Panel(name=f"{prefix}-1", parent=parent),
                sy.Panel(name=f"{prefix}-2", parent=parent),
            ]
        )

        def check() -> None:
            results = client.panels.retrieve(search_term=prefix)
            assert {p.key for p in created} <= {p.key for p in results}
            assert all(prefix in p.name for p in results)

        assert_eventually(check)

    def test_retrieve_missing_raises(self, client: sy.Synnax) -> None:
        """Should raise NotFoundError when retrieving a missing panel by key."""
        with pytest.raises(NotFoundError):
            client.panels.retrieve(key=UUID("7c3e9a52-4b1d-4f0e-9a6c-2d8b5e1f3a90"))

    def test_rename(self, client: sy.Synnax, parent: sy.ontology.ID) -> None:
        """Should rename a panel."""
        p = client.panels.create(name="Before Rename", parent=parent)
        client.panels.rename(p.key, "After Rename")
        assert client.panels.retrieve(key=p.key).name == "After Rename"

    def test_delete(self, client: sy.Synnax, parent: sy.ontology.ID) -> None:
        """Should delete a panel."""
        p = client.panels.create(name="To Delete", parent=parent)
        client.panels.delete(p.key)
        with pytest.raises(NotFoundError):
            client.panels.retrieve(key=p.key)

    def test_delete_multiple(self, client: sy.Synnax, parent: sy.ontology.ID) -> None:
        """Should delete multiple panels."""
        p1 = client.panels.create(name="Delete 1", parent=parent)
        p2 = client.panels.create(name="Delete 2", parent=parent)
        client.panels.delete([p1.key, p2.key])
        with pytest.raises(NotFoundError):
            client.panels.retrieve(keys=[p1.key, p2.key])
