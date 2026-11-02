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
from synnax import ranger, status, view
from synnax.view import RangeView, StatusView
from x.testutil import assert_eventually


@pytest.mark.view
class TestView:
    def test_create(self, client: sy.Synnax) -> None:
        """Should create a single view."""
        label = uuid4()
        v = client.views.create(
            RangeView(name="Test View", where=ranger.Where(labels=[label]))
        )
        assert isinstance(v, RangeView)
        assert v.name == "Test View"
        assert v.where == ranger.Where(labels=[label])

    def test_where_tree(self, client: sy.Synnax) -> None:
        """Should store a where tree."""
        where = status.Where(
            or_=[
                status.Where(variant=sy.filter.Set[status.Variant](in_=["error"])),
                status.Where(not_=status.Where(labels=[uuid4()])),
            ]
        )
        v = client.views.create(StatusView(name="Tree", where=where))
        retrieved = client.views.retrieve(key=v.key)
        assert isinstance(retrieved, StatusView)
        assert retrieved.where == where

    def test_invalid_where(self, client: sy.Synnax) -> None:
        """Should reject a where that the Core cannot run."""
        with pytest.raises(sy.PathError) as exc:
            client.views.create(
                RangeView(
                    name="Bad", where=ranger.Where(name=sy.filter.String(regex="("))
                )
            )
        assert exc.value.path == ["where", "name", "regex"]

    def test_rename(self, client: sy.Synnax) -> None:
        """Should rename a view and keep its where."""
        where = status.Where(variant=sy.filter.Set[status.Variant](eq="error"))
        v = client.views.create(StatusView(name="Before Rename", where=where))
        client.views.rename(v.key, "After Rename")
        renamed = client.views.retrieve(key=v.key)
        assert isinstance(renamed, StatusView)
        assert renamed.name == "After Rename"
        assert renamed.where == where

    def test_create_multiple(self, client: sy.Synnax) -> None:
        """Should create multiple views of different types."""
        views = client.views.create(
            [
                RangeView(name="Multi 1"),
                StatusView(
                    name="Multi 2",
                    where=status.Where(
                        variant=sy.filter.Set[status.Variant](in_=["error"])
                    ),
                ),
            ]
        )
        assert len(views) == 2
        assert isinstance(views[0], RangeView)
        assert isinstance(views[1], StatusView)
        assert views[1].where == status.Where(
            variant=sy.filter.Set[status.Variant](in_=["error"])
        )

    def test_retrieve_by_key(self, client: sy.Synnax) -> None:
        """Should retrieve a view by key."""
        created = client.views.create(
            StatusView(
                name="Retrieve Me",
                where=status.Where(variant=sy.filter.Set[status.Variant](eq="success")),
            )
        )
        assert client.views.retrieve(key=created.key) == created

    def test_retrieve_by_keys(self, client: sy.Synnax) -> None:
        """Should retrieve multiple views by keys."""
        v1 = client.views.create(RangeView(name="View A"))
        v2 = client.views.create(StatusView(name="View B"))
        views = client.views.retrieve(keys=[v1.key, v2.key])
        assert len(views) == 2

    def test_retrieve_with_search_term(self, client: sy.Synnax) -> None:
        """Should search for views by name."""
        prefix = f"searchable-view-{uuid4()}"
        client.views.create(
            [
                RangeView(name=f"{prefix}-1"),
                StatusView(name=f"{prefix}-2"),
            ]
        )

        def check() -> None:
            results = client.views.retrieve(search_term=prefix)
            assert len(results) >= 2
            assert all(prefix in v.name for v in results)

        assert_eventually(check)

    def test_retrieve_conditions(self, client: sy.Synnax) -> None:
        """Should retrieve the views that every condition and filter select."""
        views = client.views.create(
            [
                RangeView(name="Range"),
                StatusView(name="Status"),
            ]
        )
        res = client.views.retrieve(view.type.eq("status"), keys=[v.key for v in views])
        assert len(res) == 1
        assert res[0].name == "Status"

    def test_retrieve_by_bare_key_and_name(self, client: sy.Synnax) -> None:
        """Should retrieve a view by a key or a name passed as the first argument."""
        v = client.views.create(RangeView(name=str(uuid4())))
        assert client.views.retrieve(v.key).name == v.name
        assert client.views.retrieve(v.name).key == v.key
        assert [x.key for x in client.views.retrieve([v.name])] == [v.key]

    def test_retrieve_by_types(self, client: sy.Synnax) -> None:
        """Should retrieve only the views of the given types."""
        views = client.views.create(
            [
                RangeView(name="Range"),
                StatusView(name="Status"),
            ]
        )
        res = client.views.retrieve(keys=[v.key for v in views], types=["range"])
        assert len(res) == 1
        assert res[0].name == "Range"

    def test_delete(self, client: sy.Synnax) -> None:
        """Should delete a view."""
        v = client.views.create(RangeView(name="To Delete"))
        client.views.delete(v.key)
        with pytest.raises(sy.NotFoundError):
            client.views.retrieve(key=v.key)

    def test_delete_multiple(self, client: sy.Synnax) -> None:
        """Should delete multiple views."""
        v1 = client.views.create(RangeView(name="Delete 1"))
        v2 = client.views.create(StatusView(name="Delete 2"))
        client.views.delete([v1.key, v2.key])
        assert client.views.retrieve(keys=[v1.key, v2.key]) == []
