#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import UUID, uuid4

import pytest

import synnax as sy
from synnax import panel
from synnax.exceptions import NotFoundError, ValidationError
from x.testutil import assert_eventually


class TestPanel:
    def test_defaults(self) -> None:
        """Should create a panel with an empty leaf from only a name."""
        p = panel.Panel(name="Ops")
        assert p.root == panel.LeafNode(tabs=[])

    def test_round_trip(self) -> None:
        """Should parse a split tree of tabs and dump it back unchanged."""
        p = panel.Panel.model_validate(
            {
                "name": "Ops",
                "root": {
                    "variant": "split",
                    "direction": "x",
                    "size": 0.5,
                    "first": {
                        "variant": "leaf",
                        "tabs": [
                            {
                                "variant": "resource",
                                "resource": {"type": "lineplot", "key": "k"},
                            }
                        ],
                    },
                    "last": {
                        "variant": "leaf",
                        "tabs": [{"variant": "view", "type": "docs"}],
                    },
                },
            }
        )
        assert isinstance(p.root, panel.SplitNode)
        assert isinstance(p.root.first, panel.LeafNode)
        assert isinstance(p.root.first.tabs[0], panel.ResourceTab)
        assert isinstance(p.root.last, panel.LeafNode)
        assert isinstance(p.root.last.tabs[0], panel.ViewTab)
        assert panel.Panel.model_validate_json(p.model_dump_json()) == p

    def test_resource_id(self) -> None:
        """Should build the resource ID of a panel."""
        key = uuid4()
        assert panel.resource_id(key).type == panel.TYPE_RESOURCE_ID.type
        assert panel.resource_id(key).key == str(key)
        p = panel.Panel(name="Ops")
        assert hash(p) == hash(p.key)


@pytest.fixture
def project(client: sy.Synnax) -> sy.project.Key:
    """Creates a project to own the panels of a test."""
    return client.projects.create(name="Panel Project").key


@pytest.mark.panel
class TestClient:
    def test_create(self, client: sy.Synnax, project: sy.project.Key) -> None:
        """Should create a single panel with an empty root leaf."""
        p = client.panels.create(name="Test Panel", project=project)
        assert p.name == "Test Panel"
        assert isinstance(p.root, sy.panel.LeafNode)
        assert p.root.tabs == []

    def test_create_from_object(
        self, client: sy.Synnax, project: sy.project.Key
    ) -> None:
        """Should create a panel from a Panel object and keep its tree."""
        tab = sy.panel.ViewTab(type="docs")
        p = client.panels.create(
            sy.Panel(
                name="From Object",
                project=project,
                root=sy.panel.LeafNode(tabs=[tab]),
            )
        )
        retrieved = client.panels.retrieve(key=p.key)
        assert isinstance(retrieved.root, sy.panel.LeafNode)
        assert len(retrieved.root.tabs) == 1
        assert retrieved.root.tabs[0].key == tab.key

    def test_create_multiple(self, client: sy.Synnax, project: sy.project.Key) -> None:
        """Should create multiple panels."""
        panels = client.panels.create(
            [
                sy.Panel(name="Multi 1", project=project),
                sy.Panel(name="Multi 2", project=project),
            ]
        )
        assert [p.name for p in panels] == ["Multi 1", "Multi 2"]

    def test_create_without_project_raises(self, client: sy.Synnax) -> None:
        """Should raise ValidationError when a panel has no project."""
        with pytest.raises(
            ValidationError,
            match="must belong to a project",
        ):
            client.panels.create(sy.Panel(name="Orphan"))

    def test_retrieve_by_key(self, client: sy.Synnax, project: sy.project.Key) -> None:
        """Should retrieve a panel by key."""
        created = client.panels.create(name="Retrieve Me", project=project)
        retrieved = client.panels.retrieve(key=created.key)
        assert retrieved.key == created.key
        assert retrieved.name == "Retrieve Me"

    def test_retrieve_by_keys(self, client: sy.Synnax, project: sy.project.Key) -> None:
        """Should retrieve multiple panels by keys."""
        p1 = client.panels.create(name="Panel A", project=project)
        p2 = client.panels.create(name="Panel B", project=project)
        panels = client.panels.retrieve(keys=[p1.key, p2.key])
        assert {p.key for p in panels} == {p1.key, p2.key}

    def test_retrieve_by_group(
        self, client: sy.Synnax, project: sy.project.Key
    ) -> None:
        """Should filter the panels by group."""
        made = client.panels.create(name="Ungrouped", project=project)
        g = client.groups.create("Empty", parent=sy.project.resource_id(project))
        assert client.panels.retrieve(groups=g.key) == []
        ungrouped = client.panels.retrieve(projects=project, grouped=False)
        assert [r.key for r in ungrouped] == [made.key]
        assert client.panels.retrieve(projects=project, grouped=True) == []

    def test_retrieve_where(self, client: sy.Synnax, project: sy.project.Key) -> None:
        """Should keep the panels that match the where and the other filters."""
        a = client.panels.create(name="A", project=project)
        b = client.panels.create(name="B", project=project)
        where = panel.Where(not_=panel.Where(key=sy.filter.Set[panel.Key](eq=a.key)))
        res = client.panels.retrieve(where=where, keys=[a.key, b.key, uuid4()])
        assert [r.key for r in res] == [b.key]

    def test_retrieve_by_project(
        self, client: sy.Synnax, project: sy.project.Key
    ) -> None:
        """Should retrieve only the panels the project owns."""
        p1 = client.panels.create(name="Child 1", project=project)
        p2 = client.panels.create(name="Child 2", project=project)
        other = client.projects.create(name="Other Project")
        client.panels.create(name="Other", project=other.key)
        panels = client.panels.retrieve(projects=[project])
        assert {p.key for p in panels} == {p1.key, p2.key}
        assert all(p.project == project for p in panels)

    def test_retrieve_drafts(self, client: sy.Synnax, project: sy.project.Key) -> None:
        """Should keep a draft's owner and filter by owners and owned."""
        role = client.access.roles.create(sy.Role(name="panel drafts"))
        user = client.users.create(
            username=str(uuid4()), password="test", role=role.key
        )
        shared = client.panels.create(name="Shared", project=project)
        draft = client.panels.create(
            sy.Panel(name="Draft", project=project, owner=user.key)
        )
        assert client.panels.retrieve(key=draft.key).owner == user.key
        drafts = client.panels.retrieve(projects=project, owners=user.key)
        assert [p.key for p in drafts] == [draft.key]
        owned = client.panels.retrieve(projects=project, owned=True)
        assert [p.key for p in owned] == [draft.key]
        unowned = client.panels.retrieve(projects=project, owned=False)
        assert [p.key for p in unowned] == [shared.key]
        visible = client.panels.retrieve(
            projects=project,
            or_=[{"owned": False}, {"owners": user.key}],
        )
        assert {p.key for p in visible} == {shared.key, draft.key}

    def test_retrieve_by_project_without_panels(
        self, client: sy.Synnax, project: sy.project.Key
    ) -> None:
        """Should return no panels when the project owns none."""
        assert client.panels.retrieve(projects=[project]) == []

    def test_retrieve_with_search_term(
        self, client: sy.Synnax, project: sy.project.Key
    ) -> None:
        """Should search for panels by name."""
        prefix = "searchable-panel"
        created = client.panels.create(
            [
                sy.Panel(name=f"{prefix}-1", project=project),
                sy.Panel(name=f"{prefix}-2", project=project),
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

    def test_retrieve_key_list(
        self, client: sy.Synnax, project: sy.project.Key
    ) -> None:
        """Should return a list for a list of keys, even with one key."""
        a = client.panels.create(name="A", project=project)
        res = client.panels.retrieve(key=[a.key])
        assert [r.key for r in res] == [a.key]

    def test_retrieve_plain_key_beside_fields(
        self, client: sy.Synnax, project: sy.project.Key
    ) -> None:
        """Should return one panel for a plain key beside other fields."""
        a = client.panels.create(name="A", project=project)
        assert client.panels.retrieve(key=a.key, not_={"key": uuid4()}).key == a.key
        with pytest.raises(NotFoundError, match=r"Found no panel where .+"):
            client.panels.retrieve(key=a.key, not_={"key": a.key})

    def test_retrieve_or(self, client: sy.Synnax, project: sy.project.Key) -> None:
        """Should retrieve the panels that match any filter in or_."""
        a = client.panels.create(name="A", project=project)
        b = client.panels.create(name="B", project=project)
        res = client.panels.retrieve(or_=[{"key": a.key}, {"key": [b.key]}])
        assert {r.key for r in res} == {a.key, b.key}

    def test_retrieve_one_by_project_without_panels(
        self, client: sy.Synnax, project: sy.project.Key
    ) -> None:
        """Should raise NotFoundError for one panel of a project with no panels."""
        with pytest.raises(NotFoundError, match=r"Found no panel where .+"):
            client.panels.retrieve(projects=[project], one=True)

    def test_retrieve_none_field_by_project_without_panels(
        self, client: sy.Synnax, project: sy.project.Key
    ) -> None:
        """Should reject a field set to None beside a project with no panels."""
        with pytest.raises(ValidationError, match="key is None"):
            client.panels.retrieve(projects=[project], key=None)  # type: ignore[call-overload]

    def test_rename(self, client: sy.Synnax, project: sy.project.Key) -> None:
        """Should rename a panel."""
        p = client.panels.create(name="Before Rename", project=project)
        client.panels.rename(p.key, "After Rename")
        assert client.panels.retrieve(key=p.key).name == "After Rename"

    def test_delete(self, client: sy.Synnax, project: sy.project.Key) -> None:
        """Should delete a panel."""
        p = client.panels.create(name="To Delete", project=project)
        client.panels.delete(p.key)
        with pytest.raises(NotFoundError):
            client.panels.retrieve(key=p.key)

    def test_delete_multiple(self, client: sy.Synnax, project: sy.project.Key) -> None:
        """Should delete multiple panels."""
        p1 = client.panels.create(name="Delete 1", project=project)
        p2 = client.panels.create(name="Delete 2", project=project)
        client.panels.delete([p1.key, p2.key])
        assert client.panels.retrieve(keys=[p1.key, p2.key]) == []
